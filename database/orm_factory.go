package database

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" //nolint:golint

	"scroll-tech/database/orm"
)

// OrmFactory include all ormFactory interface
type OrmFactory interface {
	// TODO: add more orm intreface at here
	orm.ProveTaskOrm
	orm.BlockResultOrm
	orm.Layer1MessageOrm
	orm.Layer2MessageOrm
	orm.RollupResultOrm
	GetDB() *sqlx.DB
	Beginx() (*sqlx.Tx, error)
	Close() error
}

type ormFactory struct {
	orm.ProveTaskOrm
	orm.BlockResultOrm
	orm.Layer1MessageOrm
	orm.Layer2MessageOrm
	orm.RollupResultOrm
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
		ProveTaskOrm:     orm.NewProveTaskOrm(db),
		BlockResultOrm:   orm.NewBlockResultOrm(db),
		Layer1MessageOrm: orm.NewLayer1MessageOrm(db),
		Layer2MessageOrm: orm.NewLayer2MessageOrm(db),
		RollupResultOrm:  orm.NewRollupResultOrm(db),
		DB:               db,
	}, nil
}

func (o *ormFactory) GetDB() *sqlx.DB {
	return o.DB
}

func (o *ormFactory) Beginx() (*sqlx.Tx, error) {
	return o.DB.Beginx()
}
