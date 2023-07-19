package logic

import (
	"context"
	"math/big"

	"gorm.io/gorm"

	"scroll-tech/prover-stats-api/internal/orm"
)

// ProverTaskLogic deals the prover task logic with orm.
type ProverTaskLogic struct {
	proverTaskOrm *orm.ProverTask
}

// NewProverTaskLogic provides a ProverTaskLogic with database instance.
func NewProverTaskLogic(db *gorm.DB) *ProverTaskLogic {
	return &ProverTaskLogic{
		proverTaskOrm: orm.NewProverTask(db),
	}
}

// GetTasksByProver returns tasks by given prover's public key and page, page size.
func (p *ProverTaskLogic) GetTasksByProver(ctx context.Context, pubkey string, page, pageSize int) ([]*orm.ProverTask, error) {
	offset := (page - 1) * pageSize
	limit := pageSize
	return p.proverTaskOrm.GetProverTasksByProver(ctx, pubkey, offset, limit)
}

// GetTotalRewards returns prover's total rewards by given public key.
func (p *ProverTaskLogic) GetTotalRewards(ctx context.Context, pubkey string) (*big.Int, error) {
	totalReward, err := p.proverTaskOrm.GetProverTotalReward(ctx, pubkey)
	if err != nil {
		return nil, err
	}
	return totalReward, nil
}

// GetTask returns ProverTask by given task id.
func (p *ProverTaskLogic) GetTask(ctx context.Context, taskID string) (*orm.ProverTask, error) {
	task, err := p.proverTaskOrm.GetProverTasksByHash(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return task, nil
}
