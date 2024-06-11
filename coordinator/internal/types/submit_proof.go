package types

// SubmitProofParameter the SubmitProof api request parameter
type SubmitProofParameter struct {
	// TODO when prover have upgrade, need change this field to required
	UUID         string `form:"uuid" json:"uuid"`
	TaskID       string `form:"task_id" json:"task_id" binding:"required"`
	TaskType     int    `form:"task_type" json:"task_type" binding:"required"`
	Status       int    `form:"status" json:"status"`
	Proof        string `form:"proof" json:"proof"`
	FailureType  int    `form:"failure_type" json:"failure_type"`
	FailureMsg   string `form:"failure_msg" json:"failure_msg"`
	HardForkName string `form:"hard_fork_name" json:"hard_fork_name"`
}
