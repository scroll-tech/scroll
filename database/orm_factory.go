package database

import (
	"time"

	"scroll-tech/database/cache"
	"scroll-tech/database/orm"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" //nolint:golint
)

// OrmFactory include all ormFactory interface
type OrmFactory interface {
	orm.BlockTraceOrm
	orm.BlockBatchOrm
	orm.L1MessageOrm
	orm.L2MessageOrm
	orm.SessionInfoOrm
	cache.CacheOrm
	GetDB() *sqlx.DB
	Beginx() (*sqlx.Tx, error)
	Close() error
}

type ormFactory struct {
	orm.BlockTraceOrm
	orm.BlockBatchOrm
	orm.L1MessageOrm
	orm.L2MessageOrm
	orm.SessionInfoOrm
	*sqlx.DB
	// cache interface.
	cache.CacheOrm
}

// NewOrmFactory create an ormFactory factory include all ormFactory interface
func NewOrmFactory(cfg *DBConfig) (OrmFactory, error) {
	// Initialize sql/sqlx
	db, err := sqlx.Open(cfg.DriverName, cfg.DSN)
	if err != nil {
		return nil, err
	}

	db.SetMaxIdleConns(cfg.MaxOpenNum)
	db.SetMaxIdleConns(cfg.MaxIdleNum)
	if err = db.Ping(); err != nil {
		return nil, err
	}

	// Create redis client.
	cacheOrm, err := cache.NewRedisClient(cfg.RedisConfig.RedisURL, time.Duration(cfg.RedisConfig.TraceExpireSec)*time.Second)
	if err != nil {
		return nil, err
	}

	return &ormFactory{
		BlockTraceOrm:  orm.NewBlockTraceOrm(db, cacheOrm),
		BlockBatchOrm:  orm.NewBlockBatchOrm(db),
		L1MessageOrm:   orm.NewL1MessageOrm(db),
		L2MessageOrm:   orm.NewL2MessageOrm(db),
		SessionInfoOrm: orm.NewSessionInfoOrm(db),
		CacheOrm:       cacheOrm,
		DB:             db,
	}, nil
}

func (o *ormFactory) GetDB() *sqlx.DB {
	return o.DB
}

func (o *ormFactory) Beginx() (*sqlx.Tx, error) {
	return o.DB.Beginx()
}
