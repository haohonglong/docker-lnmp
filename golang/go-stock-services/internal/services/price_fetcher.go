package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go-stock-services/internal/pubsub"
	"go-stock-services/models"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// PriceFetcher 股票价格获取服务
type PriceFetcher struct {
	db            *gorm.DB
	redis         *redis.Client
	ctx           context.Context
	isRunning     bool
	httpClient    *http.Client
	cfg           PriceFetcherConfig
	tradeService  *TradeService
	orderMonitor  *OrderMonitor
}

// PriceFetcherConfig 价格获取服务配置
type PriceFetcherConfig struct {
	FetchInterval      time.Duration // 百度API抓取间隔（价格数据更新，默认5分钟）
	OrderCheckInterval time.Duration // 委托检查间隔（默认10秒）
	APIBaseURL         string        // 百度股市通API地址
	PHPAPIURL          string        // PHP后端API基础地址（备用，当前使用Go原生TradeService）
}

// NewPriceFetcher 创建价格获取服务实例
func NewPriceFetcher(db *gorm.DB, redisClient *redis.Client, cfg ...PriceFetcherConfig) *PriceFetcher {
	config := PriceFetcherConfig{
		FetchInterval:      5 * time.Minute,  // 默认5分钟从百度API获取价格
		OrderCheckInterval: 10 * time.Second,  // 默认10秒检查委托
	}
	if len(cfg) > 0 {
		config = cfg[0]
	}

	return &PriceFetcher{
		db:           db,
		redis:        redisClient,
		ctx:          context.Background(),
		isRunning:    false,
		httpClient:   &http.Client{Timeout: 15 * time.Second},
		cfg:          config,
		tradeService: NewTradeService(db, redisClient),
		orderMonitor: NewOrderMonitor(db, redisClient),
	}
}

// Start 启动价格获取服务
func (pf *PriceFetcher) Start() {
	if pf.isRunning {
		log.Println("[PriceFetcher] 服务已在运行中")
		return
	}

	pf.isRunning = true
	log.Printf("[PriceFetcher] 启动股票价格获取服务 (价格获取间隔: %v, 委托检查间隔: %v)", pf.cfg.FetchInterval, pf.cfg.OrderCheckInterval)

	// 启动两个独立的循环
	go pf.priceFetchLoop()
	go pf.orderCheckLoop()
}

// Stop 停止价格获取服务
func (pf *PriceFetcher) Stop() {
	pf.isRunning = false
	log.Println("[PriceFetcher] 停止价格获取服务...")
}

// priceFetchLoop 价格获取循环：定期从百度股市通API获取价格并更新DB/缓存
func (pf *PriceFetcher) priceFetchLoop() {
	// 第一次启动立即获取一次
	pf.fetchAndUpdatePrices()

	ticker := time.NewTicker(pf.cfg.FetchInterval)
	defer ticker.Stop()

	for range ticker.C {
		if !pf.isRunning {
			break
		}
		// 非交易时间跳过（周一至周五 9:15~15:30）
		if !pf.isTradingTime() {
			continue
		}
		pf.fetchAndUpdatePrices()
	}
}

// orderCheckLoop 委托检查循环：定期用缓存中的最新价格检查委托匹配
func (pf *PriceFetcher) orderCheckLoop() {
	// 启动时等第一次价格获取完成后再检查
	time.Sleep(3 * time.Second)
	log.Println("[OrderChecker] 委托检查循环已启动")

	ticker := time.NewTicker(pf.cfg.OrderCheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		if !pf.isRunning {
			break
		}
		// 非交易时间跳过
		if !pf.isTradingTime() {
			continue
		}
		pf.checkAllOrders()
	}
}

// isTradingTime 判断当前是否在A股交易时间（周一至周五 9:15~15:30）
func (pf *PriceFetcher) isTradingTime() bool {
	now := time.Now()
	weekday := now.Weekday()
	if weekday == time.Saturday || weekday == time.Sunday {
		return false
	}
	hour, min, _ := now.Clock()
	minutes := hour*60 + min
	// 9:15 = 555, 15:30 = 930
	return minutes >= 555 && minutes < 930
}

// fetchAndUpdatePrices 从百度API获取所有股票价格，仅在价格变化时更新DB和缓存
func (pf *PriceFetcher) fetchAndUpdatePrices() {
	startTime := time.Now()

	// 从数据库获取所有股票代码
	var stocks []models.StockInfo
	result := pf.db.Select("stock_id, stock_code, stock_name, exchange, market, stock_price").Find(&stocks)
	if result.Error != nil {
		log.Printf("[PriceFetcher] 获取股票列表失败: %v", result.Error)
		return
	}

	if len(stocks) == 0 {
		log.Println("[PriceFetcher] 数据库中没有股票数据")
		return
	}

	// 并发获取股票价格（使用goroutine池控制并发数）
	concurrencyLimit := 5
	semaphore := make(chan struct{}, concurrencyLimit)
	results := make(chan *priceResult, len(stocks))

	for _, stock := range stocks {
		go func(s models.StockInfo) {
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			data, err := pf.fetchStockPriceFromAPI(s.StockCode)
			results <- &priceResult{
				stockID:   s.StockID,
				stockCode: s.StockCode,
				stockName: s.StockName,
				data:      data,
				err:       err,
			}
		}(stock)
	}

	// 收集结果：仅价格变化时写DB和推前端
	successCount := 0
	failCount := 0
	changedCount := 0
	for i := 0; i < len(stocks); i++ {
		result := <-results
		if result.err != nil {
			failCount++
		} else if result.data != nil {
			successCount++
			// 价格变化时才写DB和推前端
			if !pf.isPriceChanged(result.stockID, result.data.StockPrice) {
				continue
			}
			changedCount++
			pf.updateStockPriceInDB(result.stockID, result.stockCode, result.stockName, result.data)
			pf.updateStockPriceCache(result.stockCode, result.data)
		}
	}

	elapsed := time.Since(startTime)

	// 只在有价格变化时才发布日志和刷新前端
	if changedCount > 0 {
		log.Printf("[PriceFetcher] 价格更新: 获取 %d, 变化 %d, 失败 %d, 耗时: %v", successCount, changedCount, failCount, elapsed)

		// 发布批次完成事件，前端据此一次性刷新列表
		pubsub.PublishLog(pf.redis, "batch_complete", 0, "", "", "", fmt.Sprintf("价格变化 %d 支", changedCount))
	}
}

// checkAllOrders 用缓存中的最新价格检查所有委托是否满足成交条件
func (pf *PriceFetcher) checkAllOrders() {
	// 扫描所有委托
	pattern := "stock_orders:*:*"
	keys, err := pf.redis.Keys(pf.ctx, pattern).Result()
	if err != nil {
		log.Printf("[PriceFetcher] 扫描委托键失败: %v", err)
		return
	}

	if len(keys) == 0 {
		return
	}

	today := time.Now().Format("2006-01-02")

	for _, key := range keys {
		orders, err := pf.redis.HGetAll(pf.ctx, key).Result()
		if err != nil {
			continue
		}

		for orderID, orderJSON := range orders {
			var order models.RedisOrder
			if err := json.Unmarshal([]byte(orderJSON), &order); err != nil {
				// 无法解析的也清理掉
				pf.redis.HDel(pf.ctx, key, orderID)
				continue
			}

			// 先检查是否过期（非今天的委托）
			if len(order.CreatedAt) >= 10 {
				orderDate := order.CreatedAt[:10]
				if orderDate != today {
					pf.redis.HDel(pf.ctx, key, orderID)
					log.Printf("[OrderChecker] 清理过期委托: %s 股票:%s (日期: %s)", orderID, order.StockCode, orderDate)
					if count, _ := pf.redis.HLen(pf.ctx, key).Result(); count == 0 {
						pf.redis.Del(pf.ctx, key)
					}
					continue
				}
			}

			// 从Redis缓存获取该股票的最新价格
			priceData, err := pf.getCachedStockPrice(order.StockCode)
			if err != nil {
				// 缓存未命中，实时从API获取
				log.Printf("[OrderChecker] 缓存未命中股票 %s 价格，实时从API获取", order.StockCode)
				apiData, apiErr := pf.fetchStockPriceFromAPI(order.StockCode)
				if apiErr != nil {
					log.Printf("[OrderChecker] 实时获取股票 %s 价格也失败: %v，跳过此委托", order.StockCode, apiErr)
					continue
				}
				// 成功获取后写入缓存，后续检查可用
				pf.updateStockPriceCache(order.StockCode, apiData)
				priceData = apiData
			}

			// 检查是否满足成交条件
			if models.CheckOrderMatch(order.OrderType, float64(order.OrderPrice), priceData.StockPrice) {
				log.Printf("[OrderChecker] 委托匹配成功! 订单: %s, 股票: %s, 类型: %s, 委托价: %.2f, 当前价: %.2f",
					orderID, order.StockCode, order.OrderType, float64(order.OrderPrice), priceData.StockPrice)

				// 执行交易
				go pf.executeTrade(order, priceData)

				// 从Redis中移除委托
				pf.redis.HDel(pf.ctx, key, orderID)

				// 如果hash为空，删除整个key
				if count, _ := pf.redis.HLen(pf.ctx, key).Result(); count == 0 {
					pf.redis.Del(pf.ctx, key)
				}
			}
		}
	}

	// 始终检查过期委托（不依赖价格变化）
	pf.orderMonitor.CheckAllOrders()
}

// getCachedStockPrice 从Redis缓存获取股票价格数据
func (pf *PriceFetcher) getCachedStockPrice(stockCode string) (*StockPriceData, error) {
	priceKey := fmt.Sprintf("stock_price:%s", stockCode)
	priceJSON, err := pf.redis.Get(pf.ctx, priceKey).Result()
	if err != nil {
		return nil, fmt.Errorf("缓存中无股票 %s 价格数据", stockCode)
	}

	var data StockPriceData
	if err := json.Unmarshal([]byte(priceJSON), &data); err != nil {
		return nil, fmt.Errorf("解析股票 %s 缓存价格失败", stockCode)
	}
	return &data, nil
}

// BaiduStockAPIResponse 百度股市通API响应结构
type BaiduStockAPIResponse struct {
	Result []struct {
		DisplayData struct {
			ResultData struct {
				TplData struct {
					Result struct {
						Name      string `json:"name"`
						Code      string `json:"code"`
						Exchange  string `json:"exchange"`
						StockType string `json:"stockType"`
						MinuteData struct {
							Update struct {
								Text string `json:"text"`
							} `json:"update"`
							PriceInfo  interface{} `json:"priceinfo"`
							PankouInfos struct {
								OriginPankou map[string]interface{} `json:"origin_pankou"`
								List         []struct {
									Value string `json:"value"`
								} `json:"list"`
							} `json:"pankouinfos"`
						} `json:"minute_data"`
					} `json:"result"`
				} `json:"tplData"`
			} `json:"resultData"`
		} `json:"DisplayData"`
	} `json:"Result"`
}

// StockPriceData 解析后的股票价格数据
type StockPriceData struct {
	StockCode  string  `json:"stock_code"`
	StockName  string  `json:"stock_name"`
	Market     string  `json:"market"`
	Exchange   string  `json:"exchange"`
	StockPrice float64 `json:"stock_price"`
	Open       float64 `json:"open"`
	Close      float64 `json:"close"`
	Lup        float64 `json:"lup"`
	Ldown      float64 `json:"ldown"`
	Highest    float64 `json:"highest"`
	Lowest     float64 `json:"lowest"`
	Average    float64 `json:"average"`
	Change     float64 `json:"change"`
	Amplitude  float64 `json:"amplitude"`
	Volume     int64   `json:"volume"`
	Amount     float64 `json:"amount"`
	UpdatedAt  string  `json:"updated_at"`
}

// fetchStockPriceFromAPI 从百度股市通API获取股票价格
func (pf *PriceFetcher) fetchStockPriceFromAPI(stockCode string) (*StockPriceData, error) {
	params := url.Values{
		"openapi":     {"1"},
		"dspName":     {"iphone"},
		"tn":          {"tangram"},
		"client":      {"app"},
		"query":       {stockCode},
		"code":        {stockCode},
		"word":        {stockCode},
		"resource_id": {"5429"},
		"ma_ver":      {"4"},
		"finClientType": {"pc"},
	}

	apiURL := pf.cfg.APIBaseURL + "?" + params.Encode()

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头，模拟浏览器
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.198 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")

	resp, err := pf.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API返回状态码: %d", resp.StatusCode)
	}

	var apiResp BaiduStockAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("解析API响应失败: %w", err)
	}

	// 检查响应数据
	if len(apiResp.Result) < 2 {
		return nil, fmt.Errorf("API返回数据格式异常")
	}

	tplResult := apiResp.Result[1].DisplayData.ResultData.TplData.Result
	minuteData := tplResult.MinuteData
	originPankou := minuteData.PankouInfos.OriginPankou

	// 检查日期是否为今天
	dateStr := minuteData.Update.Text
	if dateStr != "" {
		yearPrefix := fmt.Sprintf("%d-", time.Now().Year())
		fullDate := yearPrefix + dateStr
		datePart := strings.Split(fullDate, " ")[0]
		if datePart != time.Now().Format("2006-01-02") {
			return nil, fmt.Errorf("闭市时间，跳过 %s", stockCode)
		}
	}

	// 解析价格数据
	data := &StockPriceData{
		StockCode:  tplResult.Code,
		StockName:  tplResult.Name,
		Market:     tplResult.StockType,
		Exchange:   apiResp.Result[0].DisplayData.ResultData.TplData.Result.Exchange,
		StockPrice: getFloatFromMap(originPankou, "currentPrice"),
		Open:       getFloatFromMap(originPankou, "open"),
		Close:      getFloatFromMap(originPankou, "preClose"),
		Lup:        getFloatFromMap(originPankou, "limitUp"),
		Ldown:      getFloatFromMap(originPankou, "limitDown"),
		Highest:    getFloatFromMap(originPankou, "high"),
		Lowest:     getFloatFromMap(originPankou, "low"),
		Change:     getFloatFromMap(originPankou, "turnoverRatio"),
		Amplitude:  getFloatFromMap(originPankou, "amplitudeRatio"),
		Volume:     getInt64FromMap(originPankou, "volume"),
		Amount:     getFloatFromMap(originPankou, "amount"),
	}

	// 获取均价（从list[13]）
	if len(minuteData.PankouInfos.List) > 13 {
		if v, err := strconv.ParseFloat(minuteData.PankouInfos.List[13].Value, 64); err == nil {
			data.Average = v
		}
	}

	data.UpdatedAt = fmt.Sprintf("%d-", time.Now().Year()) + dateStr

	// 验证关键数据
	if data.Open == 0 || data.Volume == 0 {
		return nil, fmt.Errorf("股票 %s 数据异常（开盘价或成交量为0）", stockCode)
	}

	return data, nil
}

// isPriceChanged 检查股票价格是否发生变化（对比Redis缓存的上次价格）
func (pf *PriceFetcher) isPriceChanged(stockID uint, currentPrice float64) bool {
	lastPriceKey := fmt.Sprintf("stock:last_price:%d", stockID)
	lastPriceStr, err := pf.redis.Get(pf.ctx, lastPriceKey).Result()
	if err == redis.Nil {
		// Redis中无缓存（首次/新一天），视为价格变化
		return true
	}
	if err != nil {
		// Redis读取失败，保守起见视为变化
		log.Printf("[PriceFetcher] 读取上次价格缓存失败 %d(%s): %v", stockID, "", err)
		return true
	}

	lastPrice, err := strconv.ParseFloat(lastPriceStr, 64)
	if err != nil {
		return true
	}

	return currentPrice != lastPrice
}

// cacheLastPrice 缓存股票最新价格到Redis（当日有效，次日凌晨过期）
func (pf *PriceFetcher) cacheLastPrice(stockID uint, price float64) {
	lastPriceKey := fmt.Sprintf("stock:last_price:%d", stockID)
	// 计算到今天结束的剩余时间
	now := time.Now()
	endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	ttl := endOfDay.Sub(now) + time.Second
	if ttl < time.Minute {
		ttl = 24 * time.Hour // 容错
	}

	if err := pf.redis.Set(pf.ctx, lastPriceKey, strconv.FormatFloat(price, 'f', 2, 64), ttl).Err(); err != nil {
		log.Printf("[PriceFetcher] 缓存上次价格失败 %d: %v", stockID, err)
	}
}

// updateStockPriceInDB 更新数据库中的股票价格（与PHP StockModel::update + StockDetailModel::add 逻辑对齐）
// 调用前已确认价格变化，此函数直接执行写入
func (pf *PriceFetcher) updateStockPriceInDB(stockID uint, stockCode string, stockName string, data *StockPriceData) {
	stockLabel := fmt.Sprintf("%d - %s(%s)", stockID, stockName, stockCode)

	// 解析API返回的时间
	apiTime, err := time.ParseInLocation("2006-01-02 15:04:05", data.UpdatedAt, time.Local)
	if err != nil {
		log.Printf("[PriceFetcher] 解析API时间失败 %s: %v, 使用当前时间", data.UpdatedAt, err)
		apiTime = time.Now()
	}

	// 1. 更新 stocks 表（与PHP StockModel::update 一致）
	updates := map[string]interface{}{
		"stock_price": data.StockPrice,
		"open":        data.Open,
		"close":       data.Close,
		"lup":         data.Lup,
		"ldown":       data.Ldown,
		"highest":     data.Highest,
		"lowest":      data.Lowest,
		"average":     data.Average,
		"change":      data.Change,
		"amplitude":   data.Amplitude,
		"volume":      data.Volume,
		"amount":      data.Amount,
		"updated_at":  data.UpdatedAt,
	}

	if err := pf.db.Model(&models.StockInfo{}).Where("stock_id = ?", stockID).Updates(updates).Error; err != nil {
		log.Printf("[PriceFetcher] 更新股票 %s 价格失败: %v", stockLabel, err)
		return
	}

	// 2. 去重检查：同一 stock_id + created_at 不重复写入（与PHP hasBothStockIdAndDate 一致）
	var existingCount int64
	pf.db.Model(&models.StockPrice{}).Where("stock_id = ? AND created_at = ?", stockID, data.UpdatedAt).Count(&existingCount)
	if existingCount > 0 {
		log.Printf("[PriceFetcher] 股票 %s 流水已存在(%s)，跳过插入", stockLabel, data.UpdatedAt)
		// 即使流水已存在，也更新缓存价格
		pf.cacheLastPrice(stockID, data.StockPrice)
		return
	}

	// 3. 插入 stock_prices 流水记录（与PHP StockDetailModel::add 一致）
	dateParts := strings.Split(data.UpdatedAt, " ")
	stockDateAt := apiTime
	stockTimeAt := ""
	if len(dateParts) > 1 {
		stockTimeAt = dateParts[1]
	}

	priceRecord := models.StockPrice{
		StockID:     stockID,
		StockPrice:  data.StockPrice,
		StockDateAt: stockDateAt,
		StockTimeAt: stockTimeAt,
		Open:        data.Open,
		Close:       data.Close,
		Lup:         data.Lup,
		Ldown:       data.Ldown,
		Highest:     data.Highest,
		Lowest:      data.Lowest,
		Average:     data.Average,
		Change:      data.Change,
		Amplitude:   data.Amplitude,
		Volume:      data.Volume,
		Amount:      data.Amount,
		CreatedAt:   apiTime,
	}

	if err := pf.db.Create(&priceRecord).Error; err != nil {
		log.Printf("[PriceFetcher] 插入股票 %s 价格流水失败: %v", stockLabel, err)
		return
	}

	// 4. 同步更新 stock_daily_ 表（与PHP StockDateModel 逻辑对齐）
	datePart := dateParts[0] // 日期部分 "2006-01-02"
	stockDateAtParsed, _ := time.ParseInLocation("2006-01-02", datePart, time.Local)

	// 查询当天是否已有记录
	var existingDaily models.StockDaily
	dailyResult := pf.db.Where("stock_id = ? AND stock_date_at = ?", stockID, stockDateAtParsed).First(&existingDaily)

	if dailyResult.Error != nil {
		// 不存在 → 创建新记录（与PHP StockDateModel::create 一致）
		newDaily := models.StockDaily{
			StockID:     stockID,
			StockPrice:  data.StockPrice,
			StockDateAt: stockDateAtParsed,
			Open:        data.Open,
			Close:       data.Close,
			Lup:         data.Lup,
			Ldown:       data.Ldown,
			Highest:     data.Highest,
			Lowest:      data.Lowest,
			Average:     data.Average,
			Change:      data.Change,
			Amplitude:   data.Amplitude,
			Volume:      data.Volume,
			Amount:      data.Amount,
			CreatedAt:   apiTime,
			UpdatedAt:   apiTime,
		}
		if err := pf.db.Create(&newDaily).Error; err != nil {
			log.Printf("[PriceFetcher] 插入股票 %s stock_daily 失败: %v", stockLabel, err)
		} else {
			log.Printf("[PriceFetcher] 股票 %s stock_daily 新增记录(%s)", stockLabel, datePart)
		}
	} else {
		// 已存在 → 比较关键字段是否有变化，有变化才更新（与PHP Stock.php 第435-448行一致）
		if existingDaily.Open != data.Open ||
			existingDaily.Highest != data.Highest ||
			existingDaily.Lowest != data.Lowest ||
			existingDaily.Average != data.Average ||
			existingDaily.Amplitude != data.Amplitude ||
			existingDaily.Change != data.Change {
			dailyUpdates := map[string]interface{}{
				"stock_price": data.StockPrice,
				"open":        data.Open,
				"close":       data.Close,
				"lup":         data.Lup,
				"ldown":       data.Ldown,
				"highest":     data.Highest,
				"lowest":      data.Lowest,
				"average":     data.Average,
				"change":      data.Change,
				"amplitude":   data.Amplitude,
				"volume":      data.Volume,
				"amount":      data.Amount,
				"updated_at":  apiTime,
			}
			if err := pf.db.Model(&existingDaily).Updates(dailyUpdates).Error; err != nil {
				log.Printf("[PriceFetcher] 更新股票 %s stock_daily 失败: %v", stockLabel, err)
			} else {
				log.Printf("[PriceFetcher] 股票 %s stock_daily 更新记录(%s)", stockLabel, datePart)
			}
		}
	}

	// 5. 更新Redis缓存价格
	pf.cacheLastPrice(stockID, data.StockPrice)
	log.Printf("[PriceFetcher] 股票 %s 价格变化 → 已更新(%.2f)", stockLabel, data.StockPrice)
	pubsub.PublishLog(pf.redis, "price_changed", stockID, stockCode, stockName, fmt.Sprintf("%.2f", data.StockPrice), fmt.Sprintf("价格变化 → 已更新(%.2f)", data.StockPrice))
}

// updateStockPriceCache 更新Redis价格缓存
func (pf *PriceFetcher) updateStockPriceCache(stockCode string, data *StockPriceData) {
	priceKey := fmt.Sprintf("stock_price:%s", stockCode)

	priceJSON, err := json.Marshal(data)
	if err != nil {
		log.Printf("[PriceFetcher] 序列化价格数据失败: %v", err)
		return
	}

	// 缓存5分钟
	if err := pf.redis.Set(pf.ctx, priceKey, priceJSON, 5*time.Minute).Err(); err != nil {
		log.Printf("[PriceFetcher] 更新Redis价格缓存失败: %v", err)
	}
}

// executeTrade 通过Go原生TradeService执行交易（不再调用PHP HTTP接口）
func (pf *PriceFetcher) executeTrade(order models.RedisOrder, priceData *StockPriceData) {
	log.Printf("[PriceFetcher] 执行交易: 订单=%s, 用户=%s, 股票=%s, 类型=%s, 委托价=%.2f, 成交价=%.2f, 数量=%d",
		order.OrderID, order.UserID, order.StockCode, order.OrderType, float64(order.OrderPrice), priceData.StockPrice, int(order.OrderQuantity))

	result := pf.tradeService.ExecuteTrade(order, priceData)
	if result.Status == 1 {
		log.Printf("[PriceFetcher] 交易执行成功: 订单=%s", order.OrderID)
	} else {
		log.Printf("[PriceFetcher] 交易执行失败: 订单=%s, 原因=%s", order.OrderID, result.Message)
		// 买入余额不足或卖出股票不足时，通知对应用户
		if order.OrderType == "1" && result.Message == "余额不足, 请先充值" {
			pubsub.PublishLogForUser(pf.redis, "order_failed", 0, order.StockCode, order.StockName,
				fmt.Sprintf("%.2f", priceData.StockPrice),
				fmt.Sprintf("买入 %s 失败: 余额不足，请先充值（委托价%.2f，需金额%.2f）", order.StockName, float64(order.OrderPrice), priceData.StockPrice*float64(order.OrderQuantity)),
				order.UserID)
		} else if order.OrderType == "2" && result.Message == "剩股票数量不够" {
			pubsub.PublishLogForUser(pf.redis, "order_failed", 0, order.StockCode, order.StockName,
				fmt.Sprintf("%.2f", priceData.StockPrice),
				fmt.Sprintf("卖出 %s 失败: 持股数量不足（委托数量%d）", order.StockName, int(order.OrderQuantity)),
				order.UserID)
		} else {
			pubsub.PublishLogForUser(pf.redis, "order_failed", 0, order.StockCode, order.StockName,
				fmt.Sprintf("%.2f", priceData.StockPrice),
				fmt.Sprintf("委托 %s 失败: %s", order.StockName, result.Message),
				order.UserID)
		}
	}
}

// GetStockPrice 获取股票当前价格（优先从缓存）
func (pf *PriceFetcher) GetStockPrice(stockCode string) (float64, error) {
	priceKey := fmt.Sprintf("stock_price:%s", stockCode)
	priceData, err := pf.redis.Get(pf.ctx, priceKey).Result()

	if err == nil {
		var data StockPriceData
		if json.Unmarshal([]byte(priceData), &data) == nil {
			return data.StockPrice, nil
		}
	}

	// 缓存未命中，实时获取
	log.Printf("[PriceFetcher] 缓存未命中，实时获取股票 %s 价格", stockCode)
	data, err := pf.fetchStockPriceFromAPI(stockCode)
	if err != nil {
		return 0, err
	}
	return data.StockPrice, nil
}

// GetStatus 获取服务状态
func (pf *PriceFetcher) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"is_running":           pf.isRunning,
		"fetch_interval":       pf.cfg.FetchInterval.String(),
		"order_check_interval": pf.cfg.OrderCheckInterval.String(),
		"timestamp":            time.Now().Format("2006-01-02 15:04:05"),
	}
}

// priceResult 价格获取结果
type priceResult struct {
	stockID   uint
	stockCode string
	stockName string
	data      *StockPriceData
	err       error
}

// 辅助函数：从 map[string]interface{} 中安全获取 float64
func getFloatFromMap(m map[string]interface{}, key string) float64 {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case float64:
			return v
		case string:
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				return f
			}
		case json.Number:
			if f, err := v.Float64(); err == nil {
				return f
			}
		}
	}
	return 0
}

// 辅助函数：从 map[string]interface{} 中安全获取 int64
func getInt64FromMap(m map[string]interface{}, key string) int64 {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case float64:
			return int64(v)
		case string:
			if i, err := strconv.ParseInt(v, 10, 64); err == nil {
				return i
			}
		case json.Number:
			if i, err := strconv.ParseInt(string(v), 10, 64); err == nil {
				return i
			}
		}
	}
	return 0
}

// 初始化（不再需要手动设置随机种子，Go 1.20+自动初始化）
