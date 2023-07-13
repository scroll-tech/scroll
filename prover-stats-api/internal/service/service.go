package service

import (
	"context"
	"math/big"

	"scroll-tech/prover-stats-api/internal/orm"
)

type ProverTaskService struct {
	db *orm.ProverTask
}

func NewProverTaskService(db *orm.ProverTask) *ProverTaskService {
	return &ProverTaskService{db: db}
}

func (p *ProverTaskService) GetTasksByProver(pubkey string) ([]*orm.ProverTask, error) {
	return p.db.GetProverTasksByProver(context.Background(), pubkey)
}

func (p *ProverTaskService) GetTotalRewards(pubkey string) (*big.Int, error) {
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

func (p *ProverTaskService) GetTask(taskID string) (*orm.ProverTask, error) {
	tasks, err := p.db.GetProverTasksByHashes(context.Background(), []string{taskID})
	if err != nil {
		return nil, err
	}
	if len(tasks) > 0 {
		return tasks[0], nil
	}
	return nil, nil
}
