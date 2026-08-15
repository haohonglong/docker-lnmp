package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"go-stock-services/models"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// OrderService 当日委托服务（与PHP StockOrderModel一致的操作）
type OrderService struct {
	db    *gorm.DB
	redis *redis.Client
	ctx   context.Context
}

// NewOrderService 创建委托服务实例
func NewOrderService(db *gorm.DB, redisClient *redis.Client) *OrderService {
	return &OrderService{
		db:    db,
		redis: redisClient,
		ctx:   context.Background(),
	}
}

// CreateOrder 创建委托（与PHP StockOrderModel::create一致）
func (os *OrderService) CreateOrder(order *models.RedisOrder) (string, error) {
	if err := os.validateOrder(order); err != nil {
		return "", err
	}

	// 生成委托ID
	if order.OrderID == "" {
		order.OrderID = models.GenerateOrderID()
	}
	order.CreatedAt = time.Now().Format("2006-01-02 15:04:05")
	order.Status = "0" // 待成交

	// 存入Redis hash
	key := models.OrderKey(order.UserID, order.StockID)
	orderJSON, err := json.Marshal(order)
	if err != nil {
		return "", fmt.Errorf("序列化委托数据失败: %w", err)
	}

	if err := os.redis.HSet(os.ctx, key, order.OrderID, orderJSON).Err(); err != nil {
		return "", fmt.Errorf("存储委托到Redis失败: %w", err)
	}

	log.Printf("[OrderService] 创建委托成功: ID=%s, 用户=%s, 股票=%s, 类型=%s, 价格=%.2f, 数量=%d",
		order.OrderID, order.UserID, order.StockCode, order.OrderType, float64(order.OrderPrice), int(order.OrderQuantity))

	return order.OrderID, nil
}

// validateOrder 验证委托数据
func (os *OrderService) validateOrder(order *models.RedisOrder) error {
	if order.UserID == "" {
		return fmt.Errorf("用户ID不能为空")
	}
	if order.StockCode == "" {
		return fmt.Errorf("股票代码不能为空")
	}
	if order.OrderType != "1" && order.OrderType != "2" {
		return fmt.Errorf("委托类型必须为1(买入)或2(卖出)")
	}
	if order.OrderPrice <= 0 {
		return fmt.Errorf("委托价格必须大于0")
	}
	if order.OrderQuantity <= 0 {
		return fmt.Errorf("委托数量必须大于0")
	}
	if order.StockID == "" {
		return fmt.Errorf("股票ID不能为空")
	}
	return nil
}

// CancelOrder 取消委托（与PHP StockOrderModel::cancel一致）
func (os *OrderService) CancelOrder(userid, stockID, orderID string) error {
	key := models.OrderKey(userid, stockID)

	// 检查委托是否存在
	exists, err := os.redis.HExists(os.ctx, key, orderID).Result()
	if err != nil {
		return fmt.Errorf("查询委托失败: %w", err)
	}
	if !exists {
		return fmt.Errorf("委托不存在")
	}

	// 删除委托
	if err := os.redis.HDel(os.ctx, key, orderID).Err(); err != nil {
		return fmt.Errorf("取消委托失败: %w", err)
	}

	// 如果hash为空，删除整个key
	if count, _ := os.redis.HLen(os.ctx, key).Result(); count == 0 {
		os.redis.Del(os.ctx, key)
	}

	log.Printf("[OrderService] 取消委托成功: ID=%s, 用户=%s, 股票ID=%s", orderID, userid, stockID)
	return nil
}

// GetUserOrders 获取用户的所有委托（与PHP StockOrderModel::getUserOrders一致）
func (os *OrderService) GetUserOrders(userid string) ([]models.RedisOrder, error) {
	pattern := models.OrderPattern(userid)
	keys, err := os.redis.Keys(os.ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("查询用户委托失败: %w", err)
	}

	var orders []models.RedisOrder
	for _, key := range keys {
		stockOrders, err := os.redis.HGetAll(os.ctx, key).Result()
		if err != nil {
			continue
		}

		for _, orderJSON := range stockOrders {
			var order models.RedisOrder
			if err := json.Unmarshal([]byte(orderJSON), &order); err != nil {
				continue
			}
			orders = append(orders, order)
		}
	}

	return orders, nil
}

// GetOrderByID 根据ID获取委托详情
func (os *OrderService) GetOrderByID(userid, stockID, orderID string) (*models.RedisOrder, error) {
	key := models.OrderKey(userid, stockID)
	orderJSON, err := os.redis.HGet(os.ctx, key, orderID).Result()
	if err != nil {
		return nil, fmt.Errorf("未找到委托 %s: %w", orderID, err)
	}

	var order models.RedisOrder
	if err := json.Unmarshal([]byte(orderJSON), &order); err != nil {
		return nil, fmt.Errorf("解析委托数据失败: %w", err)
	}

	return &order, nil
}

// CleanExpiredOrders 清理过期委托（与PHP一致）
func (os *OrderService) CleanExpiredOrders() (int, error) {
	keys, err := os.redis.Keys(os.ctx, models.AllOrdersPattern()).Result()
	if err != nil {
		return 0, err
	}

	today := time.Now().Format("2006-01-02")
	cleanedCount := 0

	for _, key := range keys {
		stockOrders, err := os.redis.HGetAll(os.ctx, key).Result()
		if err != nil {
			continue
		}

		for orderID, orderJSON := range stockOrders {
			var order models.RedisOrder
			if err := json.Unmarshal([]byte(orderJSON), &order); err != nil {
				continue
			}

			if len(order.CreatedAt) >= 10 && order.CreatedAt[:10] != today {
				os.redis.HDel(os.ctx, key, orderID)
				cleanedCount++
			}
		}

		if count, _ := os.redis.HLen(os.ctx, key).Result(); count == 0 {
			os.redis.Del(os.ctx, key)
		}
	}

	return cleanedCount, nil
}
