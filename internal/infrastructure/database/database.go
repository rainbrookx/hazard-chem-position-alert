package database

import (
	"fmt"

	"github.com/rainbrookx/hazard-chem-position-alert/internal/infrastructure/config"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func InitDatabase(cfg config.DatabaseConfig) (*gorm.DB, error) {
	switch cfg.Default {
	case "sqlite":
		db, err := gorm.Open(sqlite.Open(cfg.SQLite.DSN), &gorm.Config{})
		if err != nil {
			return nil, fmt.Errorf("sqlite 数据库加载失败：%w", err)
		}
		return db, nil
	default:
		return nil, fmt.Errorf("没有[%s]类型的数据库", cfg.Default)
	}
}
