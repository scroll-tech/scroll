package store

import (
	"github.com/jmoiron/sqlx"
	// postgres driver
	_ "github.com/lib/pq"

	"scroll-tech/store/config"
)

// NewConnection create db connection
func NewConnection(cfg *config.DBConfig) (*sqlx.DB, error) {
	// Initialize sql/sqlx
	db, err := sqlx.Open(cfg.DriverName, cfg.DSN)
	if err != nil {
		return nil, err
	}

	db.SetMaxIdleConns(cfg.MaxOpenNum)
	db.SetMaxIdleConns(cfg.MaxIdleNUm)
	return db, db.Ping()
}
