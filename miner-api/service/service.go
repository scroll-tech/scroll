package service

import (
	"context"

	"scroll-tech/miner-api/internal/orm"
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

func (p *ProverTaskService) GetTotalRewards(pubkey string) (uint64, error) {
	tasks, err := p.db.GetProverTasksByProver(context.Background(), pubkey)
	if err != nil {
		return 0, err
	}
	var rewards uint64
	for _, task := range tasks {
		rewards += task.Reward
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
