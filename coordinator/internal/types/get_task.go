package types

// GetTaskParameter for ProverTasks request parameter
type GetTaskParameter struct {
	ProverHeight int    `form:"prover_height" json:"prover_height"`
	TaskType     int    `form:"task_type" json:"task_type"`
	VK           string `form:"vk" json:"vk"`
}

// GetTaskSchema the schema data return to prover for get prover task
type GetTaskSchema struct {
	TaskID   string `json:"task_id"`
	TaskType int    `json:"task_type"`
	TaskData string `json:"task_data"`
}
