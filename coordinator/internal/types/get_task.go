package types

// GetTaskParameter for ProverTasks request parameter
type GetTaskParameter struct {
	ProverHeight uint64 `form:"prover_height" json:"prover_height"`
	TaskTypes    []int  `form:"task_types" json:"task_types"`
}

// GetTaskSchema the schema data return to prover for get prover task
type GetTaskSchema struct {
	UUID         string `json:"uuid"`
	TaskID       string `json:"task_id"`
	TaskType     int    `json:"task_type"`
	TaskData     string `json:"task_data"`
	HardForkName string `json:"hard_fork_name"`
}
