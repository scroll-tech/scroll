package database

import (
	"fmt"
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
	cache.Cache
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
	cache.Cache
}

// NewOrmFactory create an ormFactory factory include all ormFactory interface
func NewOrmFactory(cfg *DBConfig) (OrmFactory, error) {
	pCfg, rCfg := cfg.DB, cfg.RedisConfig
	if rCfg == nil {
		return nil, fmt.Errorf("redis config is empty")
	}
	// Initialize sql/sqlx
	db, err := sqlx.Open(pCfg.DriverName, pCfg.DSN)
	if err != nil {
		return nil, err
	}

	db.SetMaxIdleConns(pCfg.MaxOpenNum)
	db.SetMaxIdleConns(pCfg.MaxIdleNum)
	if err = db.Ping(); err != nil {
		return nil, err
	}

	// Create redis client.
	cacheOrm, err := cache.NewRedisClient(rCfg)
	if err != nil {
		return nil, err
	}

	return &ormFactory{
		BlockTraceOrm:  orm.NewBlockTraceOrm(db, cacheOrm),
		BlockBatchOrm:  orm.NewBlockBatchOrm(db, cacheOrm),
		L1MessageOrm:   orm.NewL1MessageOrm(db, cacheOrm),
		L2MessageOrm:   orm.NewL2MessageOrm(db, cacheOrm),
		SessionInfoOrm: orm.NewSessionInfoOrm(db, cacheOrm),
		Cache:          cacheOrm,
		DB:             db,
	}, nil
}

func (o *ormFactory) GetDB() *sqlx.DB {
	return o.DB
}

func (o *ormFactory) Beginx() (*sqlx.Tx, error) {
	return o.DB.Beginx()
}
