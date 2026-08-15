package models

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// FlexFloat 可兼容JSON中字符串或数字类型的float64
type FlexFloat float64

func (f *FlexFloat) UnmarshalJSON(data []byte) error {
	// 先尝试作为数字解析
	var v float64
	if err := json.Unmarshal(data, &v); err == nil {
		*f = FlexFloat(v)
		return nil
	}
	// 再尝试作为字符串解析
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		parsed, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return err
		}
		*f = FlexFloat(parsed)
		return nil
	}
	return fmt.Errorf("FlexFloat: cannot unmarshal %s", string(data))
}

// FlexInt 可兼容JSON中字符串或数字类型的int
type FlexInt int

func (f *FlexInt) UnmarshalJSON(data []byte) error {
	// 先尝试作为数字解析
	var v float64
	if err := json.Unmarshal(data, &v); err == nil {
		*f = FlexInt(int(v))
		return nil
	}
	// 再尝试作为字符串解析
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		parsed, err := strconv.Atoi(s)
		if err != nil {
			return err
		}
		*f = FlexInt(parsed)
		return nil
	}
	return fmt.Errorf("FlexInt: cannot unmarshal %s", string(data))
}

// RedisOrder 委托订单（与PHP Redis中的数据结构一致）
// Redis存储: key=stock_orders:{userid}:{stock_id}, field={order_id}, value=JSON
// 注意：PHP存入Redis时order_price和order_quantity可能是字符串，因此使用FlexFloat/FlexInt兼容
type RedisOrder struct {
	OrderID      string    `json:"order_id"`
	UserID       string    `json:"userid"`
	StockID      string    `json:"stock_id"`
	SimulationID string    `json:"simulation_id"`
	OrderType    string    `json:"order_type"`     // 1=买入, 2=卖出
	OrderPrice   FlexFloat `json:"order_price"`
	OrderQuantity FlexInt   `json:"order_quantity"`
	StockCode    string    `json:"stock_code"`
	StockName    string    `json:"stock_name"`
	Exchange     string    `json:"exchange"`
	Status       string    `json:"status"`         // 0=待成交, 1=已成交, 2=已取消
	CreatedAt    string    `json:"created_at"`
}

// GenerateOrderID 生成唯一的委托ID（与PHP一致的逻辑）
func GenerateOrderID() string {
	return fmt.Sprintf("%d%d", time.Now().UnixNano()/100000, time.Now().Nanosecond()%10000)
}

// OrderKey 生成Redis委托存储键名
func OrderKey(userid, stockID string) string {
	return fmt.Sprintf("stock_orders:%s:%s", userid, stockID)
}

// OrderPattern 生成Redis委托模式匹配
func OrderPattern(userid string) string {
	return fmt.Sprintf("stock_orders:%s:*", userid)
}

// AllOrdersPattern 全部委托模式匹配
func AllOrdersPattern() string {
	return "stock_orders:*:*"
}
