package store

import (
	"github.com/jmoiron/sqlx"

	"scroll-tech/store/config"
	"scroll-tech/store/orm"
)

// OrmFactory include all ormFactory interface
type OrmFactory interface {
	orm.BlockResultOrm
	// TODO: add more orm intreface at here
	orm.Layer1MessageOrm
	orm.Layer2MessageOrm
	orm.RollupResultOrm
	Close() error
}

type ormFactory struct {
	orm.BlockResultOrm
	orm.Layer1MessageOrm
	orm.Layer2MessageOrm
	orm.RollupResultOrm
	*sqlx.DB
}

// NewOrmFactory create an ormFactory factory include all ormFactory interface
func NewOrmFactory(cfg *config.DBConfig) (OrmFactory, error) {
	db, err := NewConnection(cfg)
	if err != nil {
		return nil, err
	}

	return ormFactory{
		BlockResultOrm:   orm.NewBlockResultOrm(db),
		Layer1MessageOrm: orm.NewLayer1MessageOrm(db),
		Layer2MessageOrm: orm.NewLayer2MessageOrm(db),
		RollupResultOrm:  orm.NewRollupResultOrm(db),
		DB:               db,
	}, nil
}
