package database

import (
	"log"
	"go-stock-services/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB 数据库连接实例
var DB *gorm.DB

// InitDB 初始化数据库连接
func InitDB(cfg *config.Config) error {
	var err error

	// 创建MySQL连接
	DB, err = gorm.Open(mysql.Open(cfg.GetMySQLDSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return err
	}

	log.Println("MySQL connected successfully")

	return nil
}

// GetDB 获取数据库连接
func GetDB() *gorm.DB {
	return DB
}