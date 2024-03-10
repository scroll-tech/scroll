package client

import (
	"errors"

	"scroll-tech/common/types/message"
)

// ErrCoordinatorConnect connect to coordinator error
var ErrCoordinatorConnect = errors.New("connect coordinator error")

// ChallengeResponse defines the response structure for random API
type ChallengeResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
	Data    *struct {
		Time  string `json:"time"`
		Token string `json:"token"`
	} `json:"data,omitempty"`
}

// LoginRequest defines the request structure for login API
type LoginRequest struct {
	Message struct {
		Challenge     string `json:"challenge"`
		ProverName    string `json:"prover_name"`
		ProverVersion string `json:"prover_version"`
	} `json:"message"`
	Signature string `json:"signature"`
}

// LoginResponse defines the response structure for login API
type LoginResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
	Data    *struct {
		Time  string `json:"time"`
		Token string `json:"token"`
	} `json:"data"`
}

// GetTaskRequest defines the request structure for GetTask API
type GetTaskRequest struct {
	ForkBlockNumber uint64            `json:"fork_block_number"`
	TaskType        message.ProofType `json:"task_type"`
	ProverHeight    uint64            `json:"prover_height,omitempty"`
	VK              string            `json:"vk"`
}

// GetTaskResponse defines the response structure for GetTask API
type GetTaskResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
	Data    *struct {
		UUID     string `json:"uuid"`
		TaskID   string `json:"task_id"`
		TaskType int    `json:"task_type"`
		TaskData string `json:"task_data"`
	} `json:"data"`
}

// SubmitProofRequest defines the request structure for the SubmitProof API.
type SubmitProofRequest struct {
	UUID        string `json:"uuid"`
	TaskID      string `json:"task_id"`
	TaskType    int    `json:"task_type"`
	Status      int    `json:"status"`
	Proof       string `json:"proof"`
	FailureType int    `json:"failure_type,omitempty"`
	FailureMsg  string `json:"failure_msg,omitempty"`
}

// SubmitProofResponse defines the response structure for the SubmitProof API.
type SubmitProofResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}
