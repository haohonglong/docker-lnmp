package services

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"go-stock-services/models"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// OrderCleaner 委托清理服务
// 定期清理Redis中过期的委托订单（与PHP StockOrderModel::cleanExpiredOrders一致）
type OrderCleaner struct {
	db        *gorm.DB
	redis     *redis.Client
	ctx       context.Context
	isRunning bool
}

// NewOrderCleaner 创建委托清理服务实例
func NewOrderCleaner(db *gorm.DB, redisClient *redis.Client) *OrderCleaner {
	return &OrderCleaner{
		db:        db,
		redis:     redisClient,
		ctx:       context.Background(),
		isRunning: false,
	}
}

// Start 启动委托清理服务
func (oc *OrderCleaner) Start() {
	if oc.isRunning {
		log.Println("[OrderCleaner] 服务已在运行中")
		return
	}

	oc.isRunning = true
	log.Println("[OrderCleaner] 启动委托清理服务...")

	go oc.cleanLoop()
}

// Stop 停止委托清理服务
func (oc *OrderCleaner) Stop() {
	oc.isRunning = false
	log.Println("[OrderCleaner] 停止委托清理服务...")
}

// cleanLoop 清理循环
func (oc *OrderCleaner) cleanLoop() {
	// 启动时先清理一次
	oc.cleanExpiredOrders()

	// 每小时清理一次
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		if !oc.isRunning {
			break
		}
		oc.cleanExpiredOrders()
	}
}

// cleanExpiredOrders 清理过期委托（与PHP逻辑一致：清理非今天的委托）
func (oc *OrderCleaner) cleanExpiredOrders() {
	startTime := time.Now()
	log.Printf("[OrderCleaner] 开始清理过期委托 [%s]", startTime.Format("2006-01-02 15:04:05"))

	keys, err := oc.redis.Keys(oc.ctx, models.AllOrdersPattern()).Result()
	if err != nil {
		log.Printf("[OrderCleaner] 获取委托键失败: %v", err)
		return
	}

	if len(keys) == 0 {
		return
	}

	today := time.Now().Format("2006-01-02")
	cleanedCount := 0

	for _, key := range keys {
		stockOrders, err := oc.redis.HGetAll(oc.ctx, key).Result()
		if err != nil {
			continue
		}

		for orderID, orderJSON := range stockOrders {
			var order models.RedisOrder
			if err := json.Unmarshal([]byte(orderJSON), &order); err != nil {
				// 无法解析的也清理掉
				oc.redis.HDel(oc.ctx, key, orderID)
				cleanedCount++
				continue
			}

			// 检查委托创建日期
			if len(order.CreatedAt) >= 10 {
				orderDate := order.CreatedAt[:10]
				if orderDate != today {
					if err := oc.redis.HDel(oc.ctx, key, orderID).Err(); err == nil {
						cleanedCount++
						log.Printf("[OrderCleaner] 清理过期委托: ID=%s, 用户=%s, 股票=%s, 日期=%s",
							orderID, order.UserID, order.StockCode, orderDate)
					}
				}
			}
		}

		// 如果hash为空，删除整个key
		if count, err := oc.redis.HLen(oc.ctx, key).Result(); err == nil && count == 0 {
			oc.redis.Del(oc.ctx, key)
		}
	}

	elapsed := time.Since(startTime)
	if cleanedCount > 0 {
		log.Printf("[OrderCleaner] 过期委托清理完成: 清理 %d 条, 耗时: %v", cleanedCount, elapsed)
	}
}

// GetStatus 获取服务状态
func (oc *OrderCleaner) GetStatus() map[string]interface{} {
	keys, _ := oc.redis.Keys(oc.ctx, models.AllOrdersPattern()).Result()

	// 统计过期委托数量
	today := time.Now().Format("2006-01-02")
	expiredCount := 0
	for _, key := range keys {
		orders, err := oc.redis.HGetAll(oc.ctx, key).Result()
		if err != nil {
			continue
		}
		for _, orderJSON := range orders {
			var order models.RedisOrder
			if json.Unmarshal([]byte(orderJSON), &order) == nil {
				if len(order.CreatedAt) >= 10 && order.CreatedAt[:10] != today {
					expiredCount++
				}
			}
		}
	}

	return map[string]interface{}{
		"is_running":     oc.isRunning,
		"total_keys":     len(keys),
		"expired_orders": expiredCount,
		"last_clean":     time.Now().Format("2006-01-02 15:04:05"),
	}
}
