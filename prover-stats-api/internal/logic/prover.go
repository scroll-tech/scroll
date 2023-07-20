package logic

import (
	"gorm.io/gorm"
	"scroll-tech/prover-stats-api/internal/orm"
)

type ProverLogic struct {
	proverOrm *orm.Prover
}

func NewProverLogic(db *gorm.DB) *ProverLogic {
	return &ProverLogic{proverOrm: orm.NewProver(db)}
}

func (p *ProverLogic) ListProvers() {

}
