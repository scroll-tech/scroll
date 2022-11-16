package database

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" //nolint:golint

	"scroll-tech/database/orm"
)

// OrmFactory include all ormFactory interface
type OrmFactory interface {
	orm.BlockTraceOrm
	orm.BlockBatchOrm
	orm.L1MessageOrm
	orm.L2MessageOrm
	GetDB() *sqlx.DB
	Beginx() (*sqlx.Tx, error)
	Close() error
}

type ormFactory struct {
	orm.BlockTraceOrm
	orm.BlockBatchOrm
	orm.L1MessageOrm
	orm.L2MessageOrm
	*sqlx.DB
}

// NewOrmFactory create an ormFactory factory include all ormFactory interface
func NewOrmFactory(cfg *DBConfig) (OrmFactory, error) {
	// Initialize sql/sqlx
	db, err := sqlx.Open(cfg.DriverName, cfg.DSN)
	if err != nil {
		return nil, err
	}

	db.SetMaxIdleConns(cfg.MaxOpenNum)
	db.SetMaxIdleConns(cfg.MaxIdleNUm)
	if err = db.Ping(); err != nil {
		return nil, err
	}

	return &ormFactory{
		BlockTraceOrm: orm.NewBlockTraceOrm(db),
		BlockBatchOrm: orm.NewBlockBatchOrm(db),
		L1MessageOrm:  orm.NewL1MessageOrm(db),
		L2MessageOrm:  orm.NewL2MessageOrm(db),
		DB:            db,
	}, nil
}

func (o *ormFactory) GetDB() *sqlx.DB {
	return o.DB
}

func (o *ormFactory) Beginx() (*sqlx.Tx, error) {
	return o.DB.Beginx()
}
