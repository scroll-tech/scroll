package types

// GetTaskParameter for ProverTasks request parameter
type GetTaskParameter struct {
	ProverHeight uint64   `form:"prover_height" json:"prover_height"`
	TaskType     int      `form:"task_type" json:"task_type"`
	VK           string   `form:"vk" json:"vk"`   // will be deprecated after all go_prover offline
	VKs          []string `form:"vks" json:"vks"` // for rust_prover that supporting multi-circuits
}

// GetTaskSchema the schema data return to prover for get prover task
type GetTaskSchema struct {
	UUID         string `json:"uuid"`
	TaskID       string `json:"task_id"`
	TaskType     int    `json:"task_type"`
	TaskData     string `json:"task_data"`
	HardForkName string `json:"hard_fork_name"`
}
