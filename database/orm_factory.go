package database

import (
	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" //nolint:golint
	"scroll-tech/database/cache"
	"time"

	"scroll-tech/database/orm"
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

	var (
		rcache cache.CacheOrm
		rCfg   = cfg.RedisConfig
	)
	if rCfg != nil {
		rcache = cache.NewRedisClient(&redis.Options{
			Addr:     rCfg.Addr,
			Password: rCfg.Password,
		}, time.Minute)
	}

	return &ormFactory{
		BlockTraceOrm:  orm.NewBlockTraceOrm(db, rcache),
		BlockBatchOrm:  orm.NewBlockBatchOrm(db),
		L1MessageOrm:   orm.NewL1MessageOrm(db),
		L2MessageOrm:   orm.NewL2MessageOrm(db),
		SessionInfoOrm: orm.NewSessionInfoOrm(db),
		CacheOrm:       rcache,
		DB:             db,
	}, nil
}

func (o *ormFactory) GetDB() *sqlx.DB {
	return o.DB
}

func (o *ormFactory) Beginx() (*sqlx.Tx, error) {
	return o.DB.Beginx()
}
