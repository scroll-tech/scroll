package utils

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/utils"

	"bridge-history-api/config"
)

type gormLogger struct {
	gethLogger log.Logger
}

func (g *gormLogger) LogMode(level logger.LogLevel) logger.Interface {
	return g
}

func (g *gormLogger) Info(_ context.Context, msg string, data ...interface{}) {
	infoMsg := fmt.Sprintf(msg, data...)
	g.gethLogger.Info("gorm", "info message", infoMsg)
}

func (g *gormLogger) Warn(_ context.Context, msg string, data ...interface{}) {
	warnMsg := fmt.Sprintf(msg, data...)
	g.gethLogger.Warn("gorm", "warn message", warnMsg)
}

func (g *gormLogger) Error(_ context.Context, msg string, data ...interface{}) {
	errMsg := fmt.Sprintf(msg, data...)
	g.gethLogger.Error("gorm", "err message", errMsg)
}

func (g *gormLogger) Trace(_ context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	sql, rowsAffected := fc()
	g.gethLogger.Debug("gorm", "line", utils.FileWithLineNum(), "cost", elapsed, "sql", sql, "rowsAffected", rowsAffected, "err", err)
}

// InitDB init the db handler
func InitDB(config *config.DBConfig) (*gorm.DB, error) {
	tmpGormLogger := gormLogger{
		gethLogger: log.Root(),
	}

	db, err := gorm.Open(postgres.Open(config.DSN), &gorm.Config{
		Logger: &tmpGormLogger,
		NowFunc: func() time.Time {
			// why set time to UTC.
			// if now set this, the inserted data time will use local timezone. like 2023-07-18 18:24:00 CST+8
			// but when inserted, store to postgres is 2023-07-18 18:24:00 UTC+0 the timezone is incorrect.
			// As mysql dsn user:pass@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local, we cant set
			// the timezone by loc=Local. but postgres's dsn don't have loc option to set timezone, so just need set the gorm option like that.
			t, err := nowUTC()
			if err != nil {
				log.Error("Can not get UTC time: ", "err", err)
			}
			return t
		},
	})
	if err != nil {
		return nil, err
	}

	sqlDB, pingErr := Ping(db)
	if pingErr != nil {
		return nil, pingErr
	}

	sqlDB.SetMaxOpenConns(config.MaxOpenNum)
	sqlDB.SetMaxIdleConns(config.MaxIdleNum)

	return db, nil
}

// Ping check db status
func Ping(db *gorm.DB) (*sql.DB, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	if err = sqlDB.Ping(); err != nil {
		return nil, err
	}
	return sqlDB, nil
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

// nowUTC get the utc time.Now
func nowUTC() (time.Time, error) {
	utc, err := time.LoadLocation("")
	if err != nil {
		return time.Time{}, err
	}
	return time.Now().In(utc), nil
}
