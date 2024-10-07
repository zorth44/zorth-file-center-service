package database

import (
	"fmt"

	"github.com/zorth44/zorth-file-center-service/config"
	"github.com/zorth44/zorth-file-center-service/internal/model"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func InitDB(cfg config.DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	return db, nil
}

func InitDatabase(db *gorm.DB) error {
	return db.AutoMigrate(&model.File{}, &model.ActivityLog{}, &model.ShareLink{})
}
