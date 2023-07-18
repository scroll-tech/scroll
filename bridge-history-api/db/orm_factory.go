package db

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"
	_ "github.com/lib/pq" //nolint:golint
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/utils"

	"bridge-history-api/config"
	"bridge-history-api/db/orm"
)

// UtilDBOrm provide combined db operations
type UtilDBOrm struct {
	db *gorm.DB
}

type OrmFactory struct {
	*orm.L1CrossMsg
	*orm.L2CrossMsg
	*orm.RelayedMsg
	*orm.L2SentMsg
	*orm.RollupBatch
	*UtilDBOrm
	Db *gorm.DB
}

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

// NewOrmFactory init the db handler
func NewOrmFactory(config *config.DBConfig) (*OrmFactory, error) {
	tmpGormLogger := gormLogger{
		gethLogger: log.Root(),
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

	return &OrmFactory{
		L1CrossMsg:  orm.NewL1CrossMsg(db),
		L2CrossMsg:  orm.NewL2CrossMsg(db),
		RelayedMsg:  orm.NewRelayedMsg(db),
		L2SentMsg:   orm.NewL2SentMsg(db),
		RollupBatch: orm.NewRollupBatch(db),
		UtilDBOrm:   NewUtilDBOrm(db),
		Db:          db,
	}, nil
}

// Close close the db handler. notice the db handler only can close when then program exit.
func (o *OrmFactory) Close() error {
	sqlDB, err := o.Db.DB()
	if err != nil {
		return err
	}
	if err := sqlDB.Close(); err != nil {
		return err
	}
	return nil
}

// UtilDBOrm return the util db orm
func NewUtilDBOrm(db *gorm.DB) *UtilDBOrm {
	return &UtilDBOrm{
		db: db,
	}
}

func (u *UtilDBOrm) GetTotalCrossMsgCountByAddress(sender string) (uint64, error) {
	var count int64
	err := u.db.Table("cross_message").
		Where("sender = ? AND deleted_at IS NULL", sender).
		Count(&count).
		Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
	}
	return uint64(count), err
}

func (u *UtilDBOrm) GetCrossMsgsByAddressWithOffset(sender string, offset int, limit int) ([]orm.CrossMsg, error) {
	var messages []orm.CrossMsg
	err := u.db.Table("cross_message").
		Where("sender = ? AND deleted_at IS NULL", sender).
		Order("block_timestamp DESC NULLS FIRST, id DESC").
		Limit(limit).
		Offset(offset).
		Find(&messages).
		Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
	}
	return messages, err
}
