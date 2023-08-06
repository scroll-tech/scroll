package client

import (
	"scroll-tech/common/types/message"
)

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
	TaskType     message.ProofType `json:"task_type"`
	ProverHeight uint64            `json:"prover_height,omitempty"`
}

// GetTaskResponse defines the response structure for GetTask API
type GetTaskResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
	Data    *struct {
		TaskID   string `json:"task_id"`
		TaskType int    `json:"task_type"`
		TaskData string `json:"task_data"`
	} `json:"data"`
}

// SubmitProofRequest defines the request structure for the SubmitProof API.
type SubmitProofRequest struct {
	TaskID   string `json:"task_id"`
	TaskType int    `json:"task_type"`
	Status   int    `json:"status"`
	Proof    string `json:"proof"`
}

// SubmitProofResponse defines the response structure for the SubmitProof API.
type SubmitProofResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}
