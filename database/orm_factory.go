package database

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" //nolint:golint
	"github.com/scroll-tech/go-ethereum/log"

	commondatabase "scroll-tech/common/database"
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
	db, err := openDB(cfg)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.MaxOpenNum)
	db.SetMaxIdleConns(cfg.MaxIdleNum)
	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &ormFactory{
		db: db,
	}, nil
}

// openDB opens the sqlx handle. With IAM auth it connects through a
// token-refreshing connector; otherwise it opens the DSN directly.
func openDB(cfg *DBConfig) (*sqlx.DB, error) {
	if cfg.UseIAMAuth {
		log.Info("connecting to database with AWS RDS IAM auth", "region", cfg.AWSRegion)
		connector, err := commondatabase.NewRDSIAMConnector(context.Background(), cfg.DSN, cfg.AWSRegion)
		if err != nil {
			return nil, err
		}
		return sqlx.NewDb(sql.OpenDB(connector), "pgx"), nil
	}
	log.Info("connecting to database with password auth")
	return sqlx.Open(cfg.DriverName, cfg.DSN)
}

func (o *ormFactory) GetDB() *sqlx.DB {
	return o.db
}

func (o *ormFactory) Beginx() (*sqlx.Tx, error) {
	return o.db.Beginx()
}
