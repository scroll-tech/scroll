package logic

import (
	"context"
	"math/big"

	"gorm.io/gorm"

	"scroll-tech/prover-stats-api/internal/orm"
)

type ProverTaskLogic struct {
	proverTaskOrm *orm.ProverTask
}

func NewProverTaskLogic(db *gorm.DB) *ProverTaskLogic {
	return &ProverTaskLogic{
		proverTaskOrm: orm.NewProverTask(db),
	}
}

func (p *ProverTaskLogic) GetTasksByProver(ctx context.Context, pubkey string, page, pageSize int) ([]*orm.ProverTask, error) {
	offset := (page - 1) * pageSize
	limit := pageSize
	return p.proverTaskOrm.GetProverTasksByProver(ctx, pubkey, offset, limit)
}

func (p *ProverTaskLogic) GetTotalRewards(ctx context.Context, pubkey string) (*big.Int, error) {
	totalReward, err := p.proverTaskOrm.GetProverTotalReward(ctx, pubkey)
	if err != nil {
		return nil, err
	}
	return totalReward, nil
}

func (p *ProverTaskLogic) GetTask(ctx context.Context, taskID string) (*orm.ProverTask, error) {
	task, err := p.proverTaskOrm.GetProverTasksByHash(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return task, nil
}
