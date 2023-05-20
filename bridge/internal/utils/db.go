package utils

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"scroll-tech/bridge/internal/config"
)

func InitDB(config *config.DBConfig) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(config.DSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxOpenConns(config.MaxOpenNum)
	sqlDB.SetMaxIdleConns(config.MaxIdleNum)

	if err = sqlDB.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

func CloseDB(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if err := sqlDB.Close(); err != nil {
		return err
	}
	return nil
}
