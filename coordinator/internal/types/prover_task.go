package types

// ProverTaskParameter for ProverTasks request parameter
type ProverTaskParameter struct {
	ProverVersion int `form:"prover_version" json:"prover_version" binding:"required"`
	ProverHeight  int `form:"prover_height" json:"prover_height" binding:"required"`
	ProofType     int `form:"proof_type" json:"proof_type"`
}

// ProverTaskSchema the schema data return to prover for get prover task
type ProverTaskSchema struct {
	TaskID    string `json:"task_id"`
	ProofType int    `json:"proof_type"`
	ProofData string `json:"proof_data"`
}
