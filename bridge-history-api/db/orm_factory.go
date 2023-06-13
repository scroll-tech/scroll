package db

import (
	"database/sql"
	"errors"

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
	orm.L2SentMsgOrm
	orm.BridgeBatchOrm
	GetCrossMsgsByAddressWithOffset(sender string, offset int64, limit int64) ([]*orm.CrossMsg, error)
	GetDB() *sqlx.DB
	Beginx() (*sqlx.Tx, error)
	Close() error
}

type ormFactory struct {
	orm.L1CrossMsgOrm
	orm.L2CrossMsgOrm
	orm.RelayedMsgOrm
	orm.L2SentMsgOrm
	orm.BridgeBatchOrm
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
		L1CrossMsgOrm:  orm.NewL1CrossMsgOrm(db),
		L2CrossMsgOrm:  orm.NewL2CrossMsgOrm(db),
		RelayedMsgOrm:  orm.NewRelayedMsgOrm(db),
		L2SentMsgOrm:   orm.NewL2SentMsgOrm(db),
		BridgeBatchOrm: orm.NewBridgeBatchOrm(db),
		DB:             db,
	}, nil
}

func (o *ormFactory) GetDB() *sqlx.DB {
	return o.DB
}

func (o *ormFactory) Beginx() (*sqlx.Tx, error) {
	return o.DB.Beginx()
}

func (o *ormFactory) GetCrossMsgsByAddressWithOffset(sender string, offset int64, limit int64) ([]*orm.CrossMsg, error) {
	para := sender
	var results []*orm.CrossMsg
	rows, err := o.DB.Queryx(`SELECT * FROM cross_message WHERE sender = $1 AND NOT is_deleted ORDER BY block_timestamp DESC NULLS FIRST, id DESC LIMIT $2 OFFSET $3;`, para, limit, offset)
	if err != nil || rows == nil {
		return nil, err
	}
	for rows.Next() {
		msg := &orm.CrossMsg{}
		if err = rows.StructScan(msg); err != nil {
			break
		}
		results = append(results, msg)
	}
	if len(results) == 0 && errors.Is(err, sql.ErrNoRows) {
	} else if err != nil {
		return nil, err
	}
	return results, nil
}
