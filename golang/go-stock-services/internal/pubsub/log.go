package pubsub

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
)

// LogMessage 日志消息结构
type LogMessage struct {
	Type      string `json:"type"`       // log类型: price_changed, price_unchanged, info, error, order_failed
	StockID   uint   `json:"stock_id"`   // 股票ID
	StockCode string `json:"stock_code"` // 股票代码
	StockName string `json:"stock_name"` // 股票名称
	Price     string `json:"price"`      // 价格
	Message   string `json:"message"`    // 消息内容
	Timestamp string `json:"timestamp"`  // 时间戳
	UserID    string `json:"user_id"`    // 用户ID（用于委托失败等定向通知）
}

// PublishLog 发布日志到Redis频道
func PublishLog(redisClient *redis.Client, logType string, stockID uint, stockCode string, stockName string, price string, message string) {
	PublishLogForUser(redisClient, logType, stockID, stockCode, stockName, price, message, "")
}

// PublishLogForUser 发布日志到Redis频道（带用户ID，用于定向通知）
func PublishLogForUser(redisClient *redis.Client, logType string, stockID uint, stockCode string, stockName string, price string, message string, userID string) {
	msg := LogMessage{
		Type:      logType,
		StockID:   stockID,
		StockCode: stockCode,
		StockName: stockName,
		Price:     price,
		Message:   message,
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		UserID:    userID,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[PubSub] 序列化日志消息失败: %v", err)
		return
	}

	if err := redisClient.Publish(context.Background(), "stock:log", string(data)).Err(); err != nil {
		log.Printf("[PubSub] 发布日志到Redis失败: %v", err)
	}
}
