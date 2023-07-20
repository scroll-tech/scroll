package orm

import "gorm.io/gorm"

type Prover struct {
	db *gorm.DB
}

func NewProver(db *gorm.DB) *Prover {
	return &Prover{db: db}
}

func (p *Prover) ListProvers() {

}
