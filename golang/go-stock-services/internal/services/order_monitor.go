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

// OrderMonitor 委托监控服务
// 由 PriceFetcher 在每次价格获取完成后触发检查
type OrderMonitor struct {
	db    *gorm.DB
	redis *redis.Client
	ctx   context.Context
}

// NewOrderMonitor 创建委托监控服务实例
func NewOrderMonitor(db *gorm.DB, redisClient *redis.Client) *OrderMonitor {
	return &OrderMonitor{
		db:    db,
		redis: redisClient,
		ctx:   context.Background(),
	}
}

// CheckAllOrders 检查所有未完成的委托（由 PriceFetcher 调用）
func (om *OrderMonitor) CheckAllOrders() {
	keys, err := om.redis.Keys(om.ctx, models.AllOrdersPattern()).Result()
	if err != nil {
		log.Printf("[OrderMonitor] 扫描委托键失败: %v", err)
		return
	}

	if len(keys) == 0 {
		log.Printf("[OrderMonitor] 没有委托键")
		return
	}

	log.Printf("[OrderMonitor] 发现 %d 个委托键需要检查", len(keys))

	checkedCount := 0
	expiredCount := 0

	for _, key := range keys {
		orders, err := om.redis.HGetAll(om.ctx, key).Result()
		if err != nil {
			log.Printf("[OrderMonitor] 获取委托键 %s 失败: %v", key, err)
			continue
		}

		if len(orders) == 0 {
			log.Printf("[OrderMonitor] 委托键 %s 为空，删除", key)
			om.redis.Del(om.ctx, key)
			continue
		}

		for orderID, orderJSON := range orders {
			var order models.RedisOrder
			if err := json.Unmarshal([]byte(orderJSON), &order); err != nil {
				log.Printf("[OrderMonitor] 解析委托数据失败: %v, JSON: %s", err, orderJSON)
				continue
			}

			checkedCount++

			// 清理过期委托（非当天的）
			if om.isOrderExpired(&order) {
				om.redis.HDel(om.ctx, key, orderID)
				if count, _ := om.redis.HLen(om.ctx, key).Result(); count == 0 {
					om.redis.Del(om.ctx, key)
				}
				expiredCount++
				log.Printf("[OrderMonitor] 清理过期委托: %s 股票:%s (创建于: %s)", orderID, order.StockCode, order.CreatedAt)
			}
		}
	}

	log.Printf("[OrderMonitor] 委托检查完成: 检查=%d, 过期清理=%d", checkedCount, expiredCount)
}

// isOrderExpired 检查委托是否过期（非当天）
func (om *OrderMonitor) isOrderExpired(order *models.RedisOrder) bool {
	if len(order.CreatedAt) >= 10 {
		orderDate := order.CreatedAt[:10]
		today := time.Now().Format("2006-01-02")
		return orderDate != today
	}
	return false
}

// GetStatus 获取服务状态
func (om *OrderMonitor) GetStatus() map[string]interface{} {
	keys, _ := om.redis.Keys(om.ctx, models.AllOrdersPattern()).Result()

	return map[string]interface{}{
		"order_keys": len(keys),
		"last_check": time.Now().Format("2006-01-02 15:04:05"),
	}
}
