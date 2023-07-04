package utils

import (
	"context"
	"time"

	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"scroll-tech/bridge/internal/config"
)

type gormLogger struct {
	gethLogger log.Logger
}

func (g *gormLogger) LogMode(level logger.LogLevel) logger.Interface {
	return g
}

func (g *gormLogger) Info(_ context.Context, msg string, data ...interface{}) {
	g.gethLogger.Info(msg, data)
}

func (g *gormLogger) Warn(_ context.Context, msg string, data ...interface{}) {
	g.gethLogger.Warn(msg, data)
}

func (g *gormLogger) Error(_ context.Context, msg string, data ...interface{}) {
	g.gethLogger.Error(msg, data)
}

func (g *gormLogger) Trace(_ context.Context, begin time.Time, fc func() (string, int64), err error) {
	rows, sql := fc()
	g.gethLogger.Trace(rows, sql)
}

// InitDB init the db handler
func InitDB(config *config.DBConfig, gethLogger log.Logger) (*gorm.DB, error) {
	tmpGormLogger := gormLogger{
		gethLogger: gethLogger,
	}

	db, err := gorm.Open(postgres.Open(config.DSN), &gorm.Config{
		Logger: &tmpGormLogger,
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

// CloseDB close the db handler. notice the db handler only can close when then program exit.
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
