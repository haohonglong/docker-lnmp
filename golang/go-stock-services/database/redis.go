package database

import (
	"context"
	"log"

	"go-stock-services/config"

	"github.com/go-redis/redis/v8"
)

// Redis Redis连接实例
var Redis *redis.Client
var ctx = context.Background()

// InitRedis 初始化Redis连接
func InitRedis(cfg *config.Config) error {
	Redis = redis.NewClient(&redis.Options{
		Addr:     cfg.GetRedisAddr(),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	// 测试连接
	_, err := Redis.Ping(ctx).Result()
	if err != nil {
		return err
	}

	log.Println("Redis connected successfully")
	return nil
}

// GetRedis 获取Redis连接
func GetRedis() *redis.Client {
	return Redis
}

// GetContext 获取Redis上下文
func GetContext() context.Context {
	return ctx
}
