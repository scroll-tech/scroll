package db

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" //nolint:golint

	"bridge-history-api/config"
	"bridge-history-api/db/orm"
)

// OrmFactory include all ormFactory interface
type OrmFactory interface {
	orm.L1CrossMsgOrm
	orm.L2CrossMsgOrm
	orm.RelayedMsgOrm
	GetDB() *sqlx.DB
	Beginx() (*sqlx.Tx, error)
	Close() error
}

type ormFactory struct {
	orm.L1CrossMsgOrm
	orm.L2CrossMsgOrm
	orm.RelayedMsgOrm
	*sqlx.DB
}

// NewOrmFactory create an ormFactory factory include all ormFactory interface
func NewOrmFactory(cfg *config.Config) (OrmFactory, error) {
	// Initialize sql/sqlx
	db, err := sqlx.Open(cfg.DB.DriverName, cfg.DB.DSN)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.DB.MaxOpenNum)
	db.SetMaxIdleConns(cfg.DB.MaxIdleNum)
	if err = db.Ping(); err != nil {
		return nil, err
	}

	return &ormFactory{
		L1CrossMsgOrm: orm.NewL1CrossMsgOrm(db),
		L2CrossMsgOrm: orm.NewL2CrossMsgOrm(db),
		RelayedMsgOrm: orm.NewRelayedMsgOrm(db),
		DB:            db,
	}, nil
}

func (o *ormFactory) GetDB() *sqlx.DB {
	return o.DB
}

func (o *ormFactory) Beginx() (*sqlx.Tx, error) {
	return o.DB.Beginx()
}
