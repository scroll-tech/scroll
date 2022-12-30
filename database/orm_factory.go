package database

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" //nolint:golint

	"scroll-tech/common/viper"

	"scroll-tech/database/orm"
)

// OrmFactory include all ormFactory interface
type OrmFactory interface {
	orm.BlockTraceOrm
	orm.BlockBatchOrm
	orm.L1MessageOrm
	orm.L2MessageOrm
	orm.SessionInfoOrm
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
}

// NewOrmFactory create an ormFactory factory include all ormFactory interface
func NewOrmFactory(vp *viper.Viper) (OrmFactory, error) {
	// Initialize sql/sqlx
	driverName := vp.GetString("driver_name")
	dsn := vp.GetString("dsn")
	db, err := sqlx.Open(driverName, dsn)
	if err != nil {
		return nil, err
	}

	maxOpenNum := vp.GetInt("max_open_num")
	maxIdleNum := vp.GetInt("max_idle_num")
	db.SetMaxIdleConns(maxOpenNum)
	db.SetMaxIdleConns(maxIdleNum)
	if err = db.Ping(); err != nil {
		return nil, err
	}

	return &ormFactory{
		BlockTraceOrm:  orm.NewBlockTraceOrm(db),
		BlockBatchOrm:  orm.NewBlockBatchOrm(db),
		L1MessageOrm:   orm.NewL1MessageOrm(db),
		L2MessageOrm:   orm.NewL2MessageOrm(db),
		SessionInfoOrm: orm.NewSessionInfoOrm(db),
		DB:             db,
	}, nil
}

func (o *ormFactory) GetDB() *sqlx.DB {
	return o.DB
}

func (o *ormFactory) Beginx() (*sqlx.Tx, error) {
	return o.DB.Beginx()
}
