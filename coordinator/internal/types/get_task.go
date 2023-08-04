package types

// GetTaskParameter for ProverTasks request parameter
type GetTaskParameter struct {
	ProverVersion string `form:"prover_version" json:"prover_version" binding:"required"`
	ProverHeight  int    `form:"prover_height" json:"prover_height"`
	TaskType      int    `form:"task_type" json:"task_type"`
}

// GetTaskSchema the schema data return to prover for get prover task
type GetTaskSchema struct {
	TaskID    string `json:"task_id"`
	TaskType  int    `json:"task_type"`
	ProofData string `json:"proof_data"`
}
