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
	Deal            float64 `json:"deal"`
	DealTotal       float64 `json:"deal_total"`
	DealType        string  `json:"deal_type"`
	CommissionRate  float64 `json:"commission_rate"`
	TransferFeeRate float64 `json:"transfer_fee_rate"`
	StampTaxRate    float64 `json:"stamp_tax_rate"`
	MinCommission   float64 `json:"min_commission"`
	Exchange        string  `json:"exchange"`
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

	if totalShares <= 0 {
		return 0
	}

	netInvestment := (totalBuyAmount + totalBuyFee) - (totalSellAmount - totalSellFee)
	costPrice := netInvestment / totalShares

	return math.Round(costPrice*10000) / 10000
}

// ExecuteTrade 执行交易（Go原生实现PHP StockSimulationModel::trade的完整逻辑）
func (ts *TradeService) ExecuteTrade(order models.RedisOrder, priceData *StockPriceData) TradeResult {
	now := time.Now()
	createdAt := now.Format("2006-01-02 15:04:05")

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
	var costPrice float64
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
			costPrice = 0
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
	if gone == 0 {
		// 查询历史交易记录
		var trades []TradeRecord
		ts.db.Table("stock_trades").
			Select("deal, deal_total, deal_type, commission_rate, transfer_fee_rate, stamp_tax_rate, min_commission, exchange").
			Where("simulation_id = ? AND (gone = '0' OR gone = '' OR gone IS NULL) AND deal_type IN ('1', '2')", order.SimulationID).
			Find(&trades)

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
		})

		costPrice = CalculateStockCost(trades, 5.00, 0.00001)
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
		"updated_at": createdAt,
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
		"created_at": createdAt,
	}).Error; err != nil {
		tx.Rollback()
		log.Printf("[TradeService] 写入用户日志失败: %v", err)
		return TradeResult{Status: 0, Message: "写入用户日志失败"}
	}

	// 7c. stock_prices 去重检查，获取或插入 stock_prices_id
	var stockPricesID uint

	// 解析API时间
	apiTime, err := time.ParseInLocation("2006-01-02 15:04:05", priceData.UpdatedAt, time.Local)
	if err != nil {
		apiTime = now
	}

	dateParts := strings.Split(priceData.UpdatedAt, " ")
	stockTimeAt := ""
	if len(dateParts) > 1 {
		stockTimeAt = dateParts[1]
	}

	// 检查是否已存在
	var existingPrice models.StockPrice
	result := tx.Where("stock_id = ? AND created_at = ?", order.StockID, priceData.UpdatedAt).First(&existingPrice)
	if result.Error == nil && existingPrice.ID > 0 {
		// 已存在，使用现有ID
		stockPricesID = existingPrice.ID
	} else {
		// 不存在，插入新记录
		stockIDUint := uint(0)
		// 将 order.StockID (string) 转为 uint
		fmt.Sscanf(order.StockID, "%d", &stockIDUint)

		newPrice := models.StockPrice{
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
		if err := tx.Create(&newPrice).Error; err != nil {
			tx.Rollback()
			log.Printf("[TradeService] 插入stock_prices失败: %v", err)
			return TradeResult{Status: 0, Message: "插入股票价格流水失败"}
		}
		stockPricesID = newPrice.ID
	}

	// 7d. 更新 stock_simulations
	simUpdates := map[string]interface{}{
		"deal":         priceData.StockPrice,
		"deal_type":    order.OrderType,
		"deal_total":   dealTotal,
		"total_tax":    totalTax,
		"stock_remain": number,
		"updated_at":   createdAt,
	}
	if gone == 1 {
		// 清仓时cost设为NULL
		simUpdates["cost"] = nil
	} else {
		simUpdates["cost"] = costPrice
	}

	if err := tx.Model(&models.StockSimulation{}).
		Where("userid = ? AND stock_id = ? AND simulation_id = ?", order.UserID, order.StockID, order.SimulationID).
		Updates(simUpdates).Error; err != nil {
		tx.Rollback()
		log.Printf("[TradeService] 更新stock_simulations失败: %v", err)
		return TradeResult{Status: 0, Message: "更新模拟交易记录失败"}
	}

	// 7e. 插入 stock_trades
	var costPtr *float64
	if gone == 0 {
		costPtr = &costPrice
	}
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
		CreatedAt:       now,
		UpdatedAt:       now,
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
