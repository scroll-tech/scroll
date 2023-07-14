package logic

import (
	"context"
	"gorm.io/gorm"
	"math/big"

	"scroll-tech/prover-stats-api/internal/orm"
)

type ProverTaskLogic struct {
	db *orm.ProverTask
}

func NewProverTaskLogic(db *gorm.DB) *ProverTaskLogic {
	return &ProverTaskLogic{db: orm.NewProverTask(db)}
}

func (p *ProverTaskLogic) GetTasksByProver(pubkey string) ([]*orm.ProverTask, error) {
	return p.db.GetProverTasksByProver(context.Background(), pubkey)
}

func (p *ProverTaskLogic) GetTotalRewards(pubkey string) (*big.Int, error) {
	tasks, err := p.db.GetProverTasksByProver(context.Background(), pubkey)
	if err != nil {
		return nil, err
	}
	rewards := new(big.Int)
	for _, task := range tasks {
		rewards.Add(rewards, task.Reward.BigInt())
	}
	return rewards, nil
}

func (p *ProverTaskLogic) GetTask(taskID string) (*orm.ProverTask, error) {
	tasks, err := p.db.GetProverTasksByHashes(context.Background(), []string{taskID})
	if err != nil {
		return nil, err
	}
	if len(tasks) > 0 {
		return tasks[0], nil
	}
	return nil, nil
}
