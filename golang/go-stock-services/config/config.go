package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config 应用配置
type Config struct {
	// MySQL 配置
	MySQLHost     string
	MySQLPort     int
	MySQLDatabase string
	MySQLUsername string
	MySQLPassword string
	MySQLCharset  string

	// Redis 配置
	RedisHost     string
	RedisPort     int
	RedisPassword string
	RedisDB       int

	// 服务配置
	OrderCheckInterval   time.Duration
	PriceUpdateInterval  time.Duration
	CleanupInterval      time.Duration
	MaxConcurrentChecks  int
	LogLevel             string

	// HTTP 服务配置
	HTTPPort        int
	HTTPReadTimeout time.Duration
	HTTPWriteTimeout time.Duration

	// 股票API配置
	StockAPIURL string

	// PHP后端配置
	PHPAPIURL string
}

// LoadConfig 加载配置
func LoadConfig() *Config {
	// 尝试加载 .env 文件
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found, using environment variables")
	}

	config := &Config{
		// MySQL 配置
		MySQLHost:     getEnvRequired("MYSQL_HOST"),
		MySQLPort:     getEnvAsIntRequired("MYSQL_PORT"),
		MySQLDatabase: getEnvRequired("MYSQL_DATABASE"),
		MySQLUsername: getEnvRequired("MYSQL_USERNAME"),
		MySQLPassword: getEnvRequired("MYSQL_PASSWORD"),
		MySQLCharset:  getEnvRequired("MYSQL_CHARSET"),

		// Redis 配置
		RedisHost:     getEnvRequired("REDIS_HOST"),
		RedisPort:     getEnvAsIntRequired("REDIS_PORT"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvAsIntRequired("REDIS_DB"),

		// 服务配置
		OrderCheckInterval:  getEnvAsDurationRequired("ORDER_CHECK_INTERVAL") * time.Second,
		PriceUpdateInterval: getEnvAsDurationRequired("PRICE_UPDATE_INTERVAL") * time.Second,
		CleanupInterval:     getEnvAsDurationRequired("CLEANUP_INTERVAL") * time.Second,
		MaxConcurrentChecks: getEnvAsIntRequired("MAX_CONCURRENT_CHECKS"),
		LogLevel:            getEnvRequired("LOG_LEVEL"),

		// HTTP 服务配置
		HTTPPort:         getEnvAsIntRequired("HTTP_PORT"),
		HTTPReadTimeout:  getEnvAsDurationRequired("HTTP_READ_TIMEOUT") * time.Second,
		HTTPWriteTimeout: getEnvAsDurationRequired("HTTP_WRITE_TIMEOUT") * time.Second,

		// 股票API配置
		StockAPIURL: getEnvRequired("STOCK_API_URL"),

		// PHP后端配置
		PHPAPIURL: getEnvRequired("PHP_API_URL"),
	}

	return config
}

// getEnv 获取环境变量，如果不存在则返回默认值（仅用于可选配置）
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvRequired 获取必填环境变量，不存在则报错退出
func getEnvRequired(key string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	log.Fatalf("缺少必填环境变量: %s", key)
	return ""
}

// getEnvAsIntRequired 获取必填整数环境变量，不存在则报错退出
func getEnvAsIntRequired(key string) int {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("缺少必填环境变量: %s", key)
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		log.Fatalf("环境变量 %s 的值 %q 不是有效整数: %v", key, value, err)
	}
	return intValue
}

// getEnvAsDurationRequired 获取必填时间间隔环境变量（秒），不存在则报错退出
func getEnvAsDurationRequired(key string) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("缺少必填环境变量: %s", key)
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		log.Fatalf("环境变量 %s 的值 %q 不是有效整数: %v", key, value, err)
	}
	return time.Duration(intValue)
}

// getEnvAsInt 获取环境变量并转换为整数
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvAsDuration 获取环境变量并转换为时间间隔（秒）
func getEnvAsDuration(key string, defaultValue int) time.Duration {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return time.Duration(intValue)
		}
	}
	return time.Duration(defaultValue)
}

// GetMySQLDSN 获取MySQL连接字符串
func (c *Config) GetMySQLDSN() string {
	return c.MySQLUsername + ":" + c.MySQLPassword + "@tcp(" + c.MySQLHost + ":" + strconv.Itoa(c.MySQLPort) + ")/" + c.MySQLDatabase + "?charset=" + c.MySQLCharset + "&parseTime=True&loc=Local"
}

// GetRedisAddr 获取Redis地址
func (c *Config) GetRedisAddr() string {
	return c.RedisHost + ":" + strconv.Itoa(c.RedisPort)
}