package database

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" //nolint:golint
)

// OrmFactory include all ormFactory interface
type OrmFactory interface {
	GetDB() *sqlx.DB
	Beginx() (*sqlx.Tx, error)
}

type ormFactory struct {
	db *sqlx.DB
}

// NewOrmFactory create an ormFactory factory include all ormFactory interface
func NewOrmFactory(cfg *DBConfig) (OrmFactory, error) {
	// Initialize sql/sqlx
	db, err := sqlx.Open(cfg.DriverName, cfg.DSN)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.MaxOpenNum)
	db.SetMaxIdleConns(cfg.MaxIdleNum)
	if err = db.Ping(); err != nil {
		return nil, err
	}

	return &ormFactory{
		db: db,
	}, nil
}

func (o *ormFactory) GetDB() *sqlx.DB {
	return o.db
}

func (o *ormFactory) Beginx() (*sqlx.Tx, error) {
	return o.db.Beginx()
}
