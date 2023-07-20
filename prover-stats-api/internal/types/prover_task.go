package types

import "time"

// ProverTasksPaginationParameter for GetTasksByProver request
type ProverTasksPaginationParameter struct {
	PublicKey string `form:"public_key" json:"public_key" binding:"required"`
	Page      int    `form:"page" json:"page" binding:"required"`
	PageSize  int    `form:"page_size" json:"page_size" binding:"required"`
}

// ProverTaskSchema is response for ProverTask
type ProverTaskSchema struct {
	TaskID        string    `json:"task_id"`
	ProverName    string    `json:"prover_name"`
	TaskType      string    `json:"task_type"`
	ProvingStatus string    `json:"proving_status"`
	FailureType   string    `json:"failure_type"`
	Reward        string    `json:"reward"`
	CreatedAt     time.Time `json:"created_at"`
}

// ProverTotalRewardsParameter for GetTotalRewards request
type ProverTotalRewardsParameter struct {
	PublicKey string `form:"public_key" json:"public_key" binding:"required"`
}

// ProverTotalRewardsSchema is response for GetTotalRewards
type ProverTotalRewardsSchema struct {
	Rewards string `json:"rewards"`
}

// ProverTaskParameter for GetTask request
type ProverTaskParameter struct {
	TaskID string `form:"task_id" json:"task_id" binding:"required"`
}
