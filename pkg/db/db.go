package db

import (
	"fmt"

	"github.com/example/masking-tool-backend/config"
	"gorm.io/gorm"

	// database drivers
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
)

var Conn *gorm.DB

func Init() error {
	cfg := config.Config
	var dsn string
	var dialector gorm.Dialector
	if cfg.DBType == "mysql" {
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)
		dialector = mysql.Open(dsn)
	} else {
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode)
		dialector = postgres.Open(dsn)
	}
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return err
	}
	Conn = db
	return nil
}
