package services

import (
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"go-stock-services/internal/pubsub"
	"go-stock-services/models"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// TradeService 交易服务（Go原生实现，替代PHP的StockSimulationModel::trade）
type TradeService struct {
	db    *gorm.DB
	redis *redis.Client
}

// NewTradeService 创建交易服务实例
func NewTradeService(db *gorm.DB, redis *redis.Client) *TradeService {
	return &TradeService{db: db, redis: redis}
}

// TradeResult 交易结果
type TradeResult struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

// TransactionFee 交易费用明细
type TransactionFee struct {
	Commission  float64 `json:"commission"`
	TransferFee float64 `json:"transfer_fee"`
	StampTax    float64 `json:"stamp_tax"`
	TotalFee    float64 `json:"total_fee"`
}

// CalculateTransactionFee 计算单笔交易的税费（与PHP calculateTransactionFee一致）
func CalculateTransactionFee(amount float64, dealType string, exchange string, commissionRate float64, transferFeeRate float64, stampTaxRate float64, minCommission float64) TransactionFee {
	// 佣金 (最低5元)
	commission := math.Max(amount*commissionRate, minCommission)

	// 过户费 (双向收取，仅上海交易所 'SH' 收取)
	transferFee := 0.0
	if exchange == "SH" {
		transferFee = amount * transferFeeRate
	}

	// 印花税 (仅卖出收取)
	stampTax := 0.0
	if dealType == "2" {
		stampTax = amount * stampTaxRate
	}

	return TransactionFee{
		Commission:  commission,
		TransferFee: transferFee,
		StampTax:    stampTax,
		TotalFee:    commission + transferFee + stampTax,
	}
}

// TradeRecord 交易记录（用于成本价计算）
type TradeRecord struct {
	Deal            float64  `json:"deal"`
	DealTotal       float64  `json:"deal_total"`
	DealType        string   `json:"deal_type"`
	CommissionRate  float64  `json:"commission_rate"`
	TransferFeeRate float64  `json:"transfer_fee_rate"`
	StampTaxRate    float64  `json:"stamp_tax_rate"`
	MinCommission   float64  `json:"min_commission"`
	Exchange        string   `json:"exchange"`
	Cost            *float64 `json:"cost"`          // 台账中已存在的成本真值（可能按交割单回填过）
	StockRemain     int      `json:"stock_remain"`  // 台账中该笔交易后的持仓
	TotalTax        float64  `json:"total_tax"`     // 台账中已存在的真实总费用（佣金+过户费+印花税等）
	HasTotalTax     bool     `json:"has_total_tax"` // 该笔是否已有回填的 total_tax
}

// CalculateStockCost 计算股票多次买或卖后的平均持仓成本价（与PHP calculateStockCost一致）
func CalculateStockCost(trades []TradeRecord, minCommission float64, transferFeeRate float64) float64 {
	totalShares := 0.0
	totalBuyAmount := 0.0
	totalBuyFee := 0.0
	totalSellAmount := 0.0
	totalSellFee := 0.0

	for _, trade := range trades {
		mc := trade.MinCommission
		if mc == 0 {
			mc = minCommission
		}
		cr := trade.CommissionRate
		if cr == 0 {
			cr = 0.001
		}
		tfr := trade.TransferFeeRate
		if tfr == 0 {
			tfr = transferFeeRate
		}
		str := trade.StampTaxRate
		if str == 0 {
			str = 0.001
		}

		price := trade.Deal
		shares := trade.DealTotal
		dealType := trade.DealType
		exchange := trade.Exchange
		if exchange == "" {
			exchange = "SH"
		}

		amount := price * shares

		if trade.HasTotalTax {
			// 优先使用台账中已存在的 total_tax（按真实交割单回填过，再按费率重算会失真）
			if dealType == "1" {
				// 买入
				totalShares += shares
				totalBuyAmount += amount
				totalBuyFee += trade.TotalTax
			} else {
				// 卖出
				totalShares -= shares
				totalSellAmount += amount
				totalSellFee += trade.TotalTax
			}
		} else {
			fee := CalculateTransactionFee(amount, dealType, exchange, cr, tfr, str, mc)

			if dealType == "1" {
				// 买入
				totalShares += shares
				totalBuyAmount += amount
				totalBuyFee += (fee.Commission + fee.TransferFee)
			} else {
				// 卖出
				totalShares -= shares
				totalSellAmount += amount
				totalSellFee += fee.TotalFee
			}
		}
	}

	if totalShares <= 0 {
		return 0
	}

	netInvestment := (totalBuyAmount + totalBuyFee) - (totalSellAmount - totalSellFee)
	costPrice := netInvestment / totalShares

	return math.Round(costPrice*10000) / 10000
}

// ExecuteTrade 执行交易（Go原生实现PHP StockSimulationModel::trade的完整逻辑）
func (ts *TradeService) ExecuteTrade(order models.RedisOrder, priceData *StockPriceData) TradeResult {
	// 截断到秒，避免 time.Time 的子秒被 MySQL datetime 列四舍五入产生 ±1s 偏差
	now := time.Now().Truncate(time.Second)

	// 解析API行情时间：所有流水表（user/logs/stock_prices/stocks/stock_simulations/stock_trades）统一用它，便于按日期关联
	apiTime, err := time.ParseInLocation("2006-01-02 15:04:05", priceData.UpdatedAt, time.Local)
	if err != nil {
		apiTime = now
	}

	log.Printf("[TradeService] 执行交易: 订单=%s, 用户=%s, 股票=%s, 类型=%s, 委托价=%.2f, 成交价=%.2f, 数量=%d",
		order.OrderID, order.UserID, order.StockCode, order.OrderType, float64(order.OrderPrice), priceData.StockPrice, int(order.OrderQuantity))

	// 1. 查询 simulation 记录
	var simulation models.StockSimulation
	if err := ts.db.Where("simulation_id = ?", order.SimulationID).First(&simulation).Error; err != nil {
		log.Printf("[TradeService] 查询simulation失败: %v", err)
		return TradeResult{Status: 0, Message: "查询模拟案例失败"}
	}

	commissionRate := simulation.CommissionRate
	transferFeeRate := simulation.TransferFeeRate
	stampTaxRate := simulation.StampTaxRate
	minCommission := simulation.MinCommission

	// 2. 获取用户钱包余额
	var wallet float64
	if err := ts.db.Table("user").Where("id = ?", order.UserID).Select("wallet").Scan(&wallet).Error; err != nil {
		log.Printf("[TradeService] 查询用户钱包失败: %v", err)
		return TradeResult{Status: 0, Message: "查询用户钱包失败"}
	}

	dealTotal := int(order.OrderQuantity)
	number := simulation.StockRemain // 剩余股票数量

	// 3. 获取交易所信息
	exchange := order.Exchange
	if exchange == "" {
		exchange = "SH"
		var stock models.StockInfo
		if err := ts.db.Where("stock_id = ?", order.StockID).Select("exchange").Scan(&stock).Error; err == nil {
			if stock.Exchange != "" {
				exchange = stock.Exchange
			}
		}
	}

	// 4. 计算税费
	amount := priceData.StockPrice * float64(dealTotal)
	fee := CalculateTransactionFee(amount, order.OrderType, exchange, commissionRate, transferFeeRate, stampTaxRate, minCommission)
	totalTax := fee.TotalFee

	// 5. 买入/卖出逻辑
	gone := 0
	var costPrice *float64 // nil 表示无成本（清仓/首笔卖出）
	var cause, content string

	if order.OrderType == "1" { // 买入
		number += dealTotal
		amountWithTax := amount + totalTax
		wallet -= amountWithTax

		if wallet < 0 {
			return TradeResult{Status: 0, Message: "余额不足, 请先充值"}
		}

		cause = "交易-买入"
		content = fmt.Sprintf("%s(%s)[%s] %d股（%.2f/股）-%.2f(税费%.2f)，余额:%.2f",
			order.StockName, order.StockCode, order.SimulationID, dealTotal, priceData.StockPrice, amountWithTax, totalTax, wallet)

	} else if order.OrderType == "2" { // 卖出
		number -= dealTotal

		if number < 0 {
			return TradeResult{Status: 0, Message: "剩股票数量不够"}
		} else if number == 0 { // 清仓
			gone = 1
			costPrice = nil
		}

		amountNet := amount - totalTax
		wallet += amountNet

		cause = "交易-卖出"
		content = fmt.Sprintf("%s(%s)[%s] %d股（%.2f/股）+%.2f(税费%.2f)，余额:%.2f",
			order.StockName, order.StockCode, order.SimulationID, dealTotal, priceData.StockPrice, amountNet, totalTax, wallet)

	} else {
		return TradeResult{Status: 0, Message: "非法交易类别！必须是1或2"}
	}

	// 6. 不清仓时计算成本价
	// 以 stock_trades 台账为准：取当前持仓周期内最后一笔已存在交易的 cost 与持仓数
	// （cost 可能按真实交割单手工回填过，属于用户真值，绝不能用费率重算覆盖）
	if gone == 0 {
		// 6a. 最后一笔已有 cost（含回填真值）：基于已存在的 cost 增量计算，不再重算历史税费
		var lastTrade models.StockTrade
		result := ts.db.Table("stock_trades").
			Select("id, cost, stock_remain").
			Where("simulation_id = ? AND (gone = '0' OR gone = '' OR gone IS NULL) AND deal_type IN ('1', '2')", order.SimulationID).
			Order("id DESC").Limit(1).Find(&lastTrade)
		// 用 RowsAffected 准确判断是否有记录，不依赖 Select 是否包含主键
		hasLast := result.RowsAffected > 0

		if hasLast && lastTrade.Cost != nil && lastTrade.StockRemain > 0 {
			baseCost := *lastTrade.Cost
			baseRemain := float64(lastTrade.StockRemain)

			if order.OrderType == "1" {
				// 买入：新成本 = (上笔成本×上笔持仓 + 本次成交额[含本次税费]) / 新总持仓
				cost := (baseCost*baseRemain + amount + totalTax) / (baseRemain + float64(dealTotal))
				costPrice = &cost
			} else {
				// 卖出：摊薄成本价 = (上笔成本×上笔持仓 − 本次卖出净额) / 新持仓
				cost := (baseCost*baseRemain - (amount - totalTax)) / (baseRemain - float64(dealTotal))
				costPrice = &cost
			}
		} else if hasLast {
			// 6b. 有历史但最后一笔 cost 缺失（脏数据）：全量兜底，且优先使用台账中已存在的 total_tax
			var trades []TradeRecord
			ts.db.Table("stock_trades").
				Select("deal, deal_total, deal_type, commission_rate, transfer_fee_rate, stamp_tax_rate, min_commission, exchange, cost, stock_remain, total_tax").
				Where("simulation_id = ? AND (gone = '0' OR gone = '' OR gone IS NULL) AND deal_type IN ('1', '2')", order.SimulationID).
				Find(&trades)

			// 标记每笔是否已有回填的 total_tax（有则不再按费率重算）
			for i := range trades {
				if trades[i].TotalTax != 0 {
					trades[i].HasTotalTax = true
				}
			}

			// 添加当前交易
			trades = append(trades, TradeRecord{
				Deal:            priceData.StockPrice,
				DealTotal:       float64(dealTotal),
				DealType:        order.OrderType,
				CommissionRate:  commissionRate,
				TransferFeeRate: transferFeeRate,
				StampTaxRate:    stampTaxRate,
				MinCommission:   minCommission,
				Exchange:        exchange,
				TotalTax:        totalTax,
				HasTotalTax:     true,
			})

			cp := CalculateStockCost(trades, 5.00, 0.00001)
			costPrice = &cp
		} else {
			// 6c. 首笔买入（当前周期无任何交易）：成本 = 本次总投入(含税费) / 股数
			if order.OrderType == "1" {
				cost := (amount + totalTax) / float64(dealTotal)
				costPrice = &cost
			}
			// 首笔卖出（无历史交易）：cost 保持 nil，与 PHP 端一致
		}
	}

	// 7. 开启事务
	tx := ts.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 7a. 更新用户钱包
	if err := tx.Table("user").Where("id = ?", order.UserID).Updates(map[string]interface{}{
		"wallet":     wallet,
		"updated_at": apiTime,
	}).Error; err != nil {
		tx.Rollback()
		log.Printf("[TradeService] 更新用户钱包失败: %v", err)
		return TradeResult{Status: 0, Message: "更新用户金额异常"}
	}

	// 7b. 写入用户日志 (对应PHP的LogModel)
	if err := tx.Table("logs").Create(map[string]interface{}{
		"uid":        order.UserID,
		"cause":      cause,
		"content":    content,
		"created_at": apiTime,
	}).Error; err != nil {
		tx.Rollback()
		log.Printf("[TradeService] 写入用户日志失败: %v", err)
		return TradeResult{Status: 0, Message: "写入用户日志失败"}
	}

	// 7c. stock_prices 获取或更新/插入（以最新价格为准）
	var stockPricesID uint

	dateParts := strings.Split(priceData.UpdatedAt, " ")
	stockTimeAt := ""
	if len(dateParts) > 1 {
		stockTimeAt = dateParts[1]
	}

	stockIDUint := uint(0)
	fmt.Sscanf(order.StockID, "%d", &stockIDUint)

	priceRecord := models.StockPrice{
		StockID:     stockIDUint,
		StockPrice:  priceData.StockPrice,
		StockDateAt: apiTime,
		StockTimeAt: stockTimeAt,
		Open:        priceData.Open,
		Close:       priceData.Close,
		Lup:         priceData.Lup,
		Ldown:       priceData.Ldown,
		Highest:     priceData.Highest,
		Lowest:      priceData.Lowest,
		Average:     priceData.Average,
		Change:      priceData.Change,
		Amplitude:   priceData.Amplitude,
		Volume:      priceData.Volume,
		Amount:      priceData.Amount,
		CreatedAt:   apiTime,
	}

	// 检查是否已存在同时间的记录（created_at 用行情时间 apiTime，与 stock_trades.created_at 保持一致）
	var existingPrice models.StockPrice
	apiTimeStr := apiTime.Format("2006-01-02 15:04:05")
	result := tx.Where("stock_id = ? AND created_at = ?", order.StockID, apiTimeStr).First(&existingPrice)
	if result.Error == nil && existingPrice.ID > 0 {
		// 已存在，更新为最新价格
		priceRecord.ID = existingPrice.ID
		if err := tx.Model(&existingPrice).Updates(map[string]interface{}{
			"stock_price": priceData.StockPrice,
			"open":        priceData.Open,
			"close":       priceData.Close,
			"lup":         priceData.Lup,
			"ldown":       priceData.Ldown,
			"highest":     priceData.Highest,
			"lowest":      priceData.Lowest,
			"average":     priceData.Average,
			"change":      priceData.Change,
			"amplitude":   priceData.Amplitude,
			"volume":      priceData.Volume,
			"amount":      priceData.Amount,
		}).Error; err != nil {
			tx.Rollback()
			log.Printf("[TradeService] 更新stock_prices失败: %v", err)
			return TradeResult{Status: 0, Message: "更新股票价格流水失败"}
		}
		stockPricesID = existingPrice.ID
	} else {
		// 不存在，插入新记录
		if err := tx.Create(&priceRecord).Error; err != nil {
			tx.Rollback()
			log.Printf("[TradeService] 插入stock_prices失败: %v", err)
			return TradeResult{Status: 0, Message: "插入股票价格流水失败"}
		}
		stockPricesID = priceRecord.ID
	}

	// 7c2. 同步更新 stocks 表的 stock_price（保证 stocks 和 stock_prices 一致）
	if err := tx.Model(&models.StockInfo{}).Where("stock_id = ?", stockIDUint).Updates(map[string]interface{}{
		"stock_price": priceData.StockPrice,
		"open":        priceData.Open,
		"close":       priceData.Close,
		"lup":         priceData.Lup,
		"ldown":       priceData.Ldown,
		"highest":     priceData.Highest,
		"lowest":      priceData.Lowest,
		"average":     priceData.Average,
		"change":      priceData.Change,
		"amplitude":   priceData.Amplitude,
		"volume":      priceData.Volume,
		"amount":      priceData.Amount,
		"updated_at":  apiTime,
	}).Error; err != nil {
		tx.Rollback()
		log.Printf("[TradeService] 更新stocks价格失败: %v", err)
		return TradeResult{Status: 0, Message: "更新股票价格失败"}
	}

	// 7d. 更新 stock_simulations
	simUpdates := map[string]interface{}{
		"deal":         priceData.StockPrice,
		"deal_type":    order.OrderType,
		"deal_total":   dealTotal,
		"total_tax":    totalTax,
		"stock_remain": number,
		"updated_at":   apiTime,
	}
	// 清仓/首笔卖出时 cost 为 NULL，其余为计算所得成本价
	simUpdates["cost"] = costPrice

	if err := tx.Model(&models.StockSimulation{}).
		Where("userid = ? AND stock_id = ? AND simulation_id = ?", order.UserID, order.StockID, order.SimulationID).
		Updates(simUpdates).Error; err != nil {
		tx.Rollback()
		log.Printf("[TradeService] 更新stock_simulations失败: %v", err)
		return TradeResult{Status: 0, Message: "更新模拟交易记录失败"}
	}

	// 7e. 插入 stock_trades
	costPtr := costPrice
	tradeRemark := fmt.Sprintf("委托自动成交: 委托价%.2f, 成交价%.2f", float64(order.OrderPrice), priceData.StockPrice)

	newTrade := models.StockTrade{
		SimulationID:    order.SimulationID,
		StockPricesID:   stockPricesID,
		Deal:            priceData.StockPrice,
		Cost:            costPtr,
		CommissionRate:  commissionRate,
		TransferFeeRate: transferFeeRate,
		StampTaxRate:    stampTaxRate,
		MinCommission:   minCommission,
		TotalTax:        totalTax,
		DealTotal:       dealTotal,
		StockRemain:     number,
		DealType:        order.OrderType,
		Gone:            "0",
		TradeRemark:     tradeRemark,
		Exchange:        exchange,
		CreatedAt:       apiTime,
		UpdatedAt:       apiTime,
	}
	if err := tx.Create(&newTrade).Error; err != nil {
		tx.Rollback()
		log.Printf("[TradeService] 插入stock_trades失败: %v", err)
		return TradeResult{Status: 0, Message: "插入交易记录失败"}
	}

	// 7f. 清仓时更新所有历史交易记录的gone标记
	if gone == 1 {
		if err := tx.Model(&models.StockTrade{}).
			Where("simulation_id = ? AND (gone = '0' OR gone = '' OR gone IS NULL)", order.SimulationID).
			Update("gone", "1").Error; err != nil {
			tx.Rollback()
			log.Printf("[TradeService] 清仓更新gone失败: %v", err)
			return TradeResult{Status: 0, Message: "清仓失败"}
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		log.Printf("[TradeService] 事务提交失败: %v", err)
		return TradeResult{Status: 0, Message: "交易提交失败"}
	}

	log.Printf("[TradeService] 交易执行成功: 订单=%s, 用户=%s, 类型=%s, 成交价=%.2f, 数量=%d, 税费=%.2f",
		order.OrderID, order.UserID, order.OrderType, priceData.StockPrice, dealTotal, totalTax)

	pubsub.PublishLog(ts.redis, "info", 0, order.StockCode, order.StockName,
		fmt.Sprintf("%.2f", priceData.StockPrice),
		fmt.Sprintf("委托自动成交: %s 委托价%.2f 成交价%.2f", order.StockName, float64(order.OrderPrice), priceData.StockPrice))

	return TradeResult{Status: 1, Message: "交易成功"}
}
