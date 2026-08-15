package models

import (
	"time"
)

// StockInfo 对应 stocks 表
type StockInfo struct {
	StockID      uint      `json:"stock_id" gorm:"primaryKey;column:stock_id"`
	StockCode    string    `json:"stock_code" gorm:"column:stock_code"`
	StockName    string    `json:"stock_name" gorm:"column:stock_name"`
	Market       string    `json:"market" gorm:"column:market"`
	Exchange     string    `json:"exchange" gorm:"column:exchange"`
	StockPrice   float64   `json:"stock_price" gorm:"column:stock_price"`
	Open         float64   `json:"open" gorm:"column:open"`
	Close        float64   `json:"close" gorm:"column:close"`
	Lup          float64   `json:"lup" gorm:"column:lup"`
	Ldown        float64   `json:"ldown" gorm:"column:ldown"`
	Highest      float64   `json:"highest" gorm:"column:highest"`
	Lowest       float64   `json:"lowest" gorm:"column:lowest"`
	Average      float64   `json:"average" gorm:"column:average"`
	Change       float64   `json:"change" gorm:"column:change"`
	Amplitude    float64   `json:"amplitude" gorm:"column:amplitude"`
	Volume       int64     `json:"volume" gorm:"column:volume"`
	Amount       float64   `json:"amount" gorm:"column:amount"`
	Status       string    `json:"status" gorm:"column:status"`
	StockRemark  string    `json:"stock_remark" gorm:"column:stock_remark"`
	CreatedAt    time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"column:updated_at"`
}

// TableName 指定表名
func (StockInfo) TableName() string {
	return "stocks"
}

// StockPrice 对应 stock_prices 表
type StockPrice struct {
	ID          uint      `json:"id" gorm:"primaryKey;column:id"`
	StockID     uint      `json:"stock_id" gorm:"column:stock_id"`
	StockPrice  float64   `json:"stock_price" gorm:"column:stock_price"`
	StockDateAt time.Time `json:"stock_date_at" gorm:"column:stock_date_at"`
	StockTimeAt string    `json:"stock_time_at" gorm:"column:stock_time_at"`
	Open        float64   `json:"open" gorm:"column:open"`
	Close       float64   `json:"close" gorm:"column:close"`
	Lup         float64   `json:"lup" gorm:"column:lup"`
	Ldown       float64   `json:"ldown" gorm:"column:ldown"`
	Highest     float64   `json:"highest" gorm:"column:highest"`
	Lowest      float64   `json:"lowest" gorm:"column:lowest"`
	Average     float64   `json:"average" gorm:"column:average"`
	Change      float64   `json:"change" gorm:"column:change"`
	Amplitude   float64   `json:"amplitude" gorm:"column:amplitude"`
	Volume      int64     `json:"volume" gorm:"column:volume"`
	Amount      float64   `json:"amount" gorm:"column:amount"`
	CreatedAt   time.Time `json:"created_at" gorm:"column:created_at"`
}

// TableName
func (StockPrice) TableName() string {
	return "stock_prices"
}

type StockDaily struct {
	ID          uint      `json:"id" gorm:"primaryKey;column:id"`
	StockID     uint      `json:"stock_id" gorm:"column:stock_id"`
	StockPrice  float64   `json:"stock_price" gorm:"column:stock_price"`
	StockDateAt time.Time `json:"stock_date_at" gorm:"column:stock_date_at"`
	Open        float64   `json:"open" gorm:"column:open"`
	Close       float64   `json:"close" gorm:"column:close"`
	Lup         float64   `json:"lup" gorm:"column:lup"`
	Ldown       float64   `json:"ldown" gorm:"column:ldown"`
	Highest     float64   `json:"highest" gorm:"column:highest"`
	Lowest      float64   `json:"lowest" gorm:"column:lowest"`
	Average     float64   `json:"average" gorm:"column:average"`
	Change      float64   `json:"change" gorm:"column:change"`
	Amplitude   float64   `json:"amplitude" gorm:"column:amplitude"`
	Volume      int64     `json:"volume" gorm:"column:volume"`
	Amount      float64   `json:"amount" gorm:"column:amount"`
	CreatedAt   time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"column:updated_at"`
}

func (StockDaily) TableName() string {
	return "stock_daily_"
}

// StockSimulation 对应 stock_simulations 表
type StockSimulation struct {
	ID                uint      `json:"id" gorm:"primaryKey;column:id"`
	SimulationID      string    `json:"simulation_id" gorm:"column:simulation_id"`
	UserID            uint      `json:"userid" gorm:"column:userid"`
	StockID           uint      `json:"stock_id" gorm:"column:stock_id"`
	CaseName          string    `json:"case_name" gorm:"column:case_name"`
	CommissionRate    float64   `json:"commission_rate" gorm:"column:commission_rate"`
	TransferFeeRate   float64   `json:"transfer_fee_rate" gorm:"column:transfer_fee_rate"`
	StampTaxRate      float64   `json:"stamp_tax_rate" gorm:"column:stamp_tax_rate"`
	MinCommission     float64   `json:"min_commission" gorm:"column:min_commission"`
	TotalTax          float64   `json:"total_tax" gorm:"column:total_tax"`
	Deal              float64   `json:"deal" gorm:"column:deal"`
	DealType          string    `json:"deal_type" gorm:"column:deal_type"`
	Cost              *float64  `json:"cost" gorm:"column:cost"`
	DealTotal         int       `json:"deal_total" gorm:"column:deal_total"`
	StockRemain       int       `json:"stock_remain" gorm:"column:stock_remain"`
	Stars             int       `json:"stars" gorm:"column:stars"`
	SimulationRemark  string    `json:"simulation_remark" gorm:"column:simulation_remark"`
	CreatedAt         time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt         time.Time `json:"updated_at" gorm:"column:updated_at"`
}

// TableName
func (StockSimulation) TableName() string {
	return "stock_simulations"
}

// StockTrade 对应 stock_trades 表
type StockTrade struct {
	ID               uint      `json:"id" gorm:"primaryKey;column:id"`
	SimulationID     string    `json:"simulation_id" gorm:"column:simulation_id"`
	StockPricesID    uint      `json:"stock_prices_id" gorm:"column:stock_prices_id"`
	Deal             float64   `json:"deal" gorm:"column:deal"`
	Cost             *float64  `json:"cost" gorm:"column:cost"`
	CommissionRate   float64   `json:"commission_rate" gorm:"column:commission_rate"`
	TransferFeeRate  float64   `json:"transfer_fee_rate" gorm:"column:transfer_fee_rate"`
	StampTaxRate     float64   `json:"stamp_tax_rate" gorm:"column:stamp_tax_rate"`
	MinCommission    float64   `json:"min_commission" gorm:"column:min_commission"`
	TotalTax         float64   `json:"total_tax" gorm:"column:total_tax"`
	DealTotal        int       `json:"deal_total" gorm:"column:deal_total"`
	StockRemain      int       `json:"stock_remain" gorm:"column:stock_remain"`
	DealType         string    `json:"deal_type" gorm:"column:deal_type"`
	Gone             string    `json:"gone" gorm:"column:gone"`
	TradeRemark      string    `json:"trade_remark" gorm:"column:trade_remark"`
	Exchange         string    `json:"exchange" gorm:"column:exchange"`
	CreatedAt        time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt        time.Time `json:"updated_at" gorm:"column:updated_at"`
}

// TableName
func (StockTrade) TableName() string {
	return "stock_trades"
}

// CheckOrderMatch 检查委托是否满足成交条件
func CheckOrderMatch(orderType string, orderPrice float64, currentPrice float64) bool {
	if orderType == "1" {
		// 买入：当前价 <= 委托价
		return currentPrice <= orderPrice
	}
	// 卖出：当前价 >= 委托价
	return currentPrice >= orderPrice
}
