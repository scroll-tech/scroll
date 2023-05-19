package utils

import (
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"scroll-tech/bridge/internal/config"
)

func InitDB(config *config.DBConfig) (*gorm.DB, error) {
	logLevel := logger.Silent
	if config.ShowSql {
		logLevel = logger.Info
	}
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold: config.SlowSqlThreshold, // Slow SQL threshold
			LogLevel:      logLevel,                // Log level
		},
	)

	db, err := gorm.Open(postgres.Open(config.DSN), &gorm.Config{
		Logger: newLogger,
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
	sqlDB.Close()
	return nil
}
