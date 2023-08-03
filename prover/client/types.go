package client

import (
	"scroll-tech/common/types/message"
)

// ProverLoginRequest defines the request structure for login API
type ProverLoginRequest struct {
	PublicKey  string `json:"public_key"`
	ProverName string `json:"prover_name"`
}

// ProverLoginResponse defines the response structure for login API
type ProverLoginResponse struct {
	ErrCode int    `json:"errcode,omitempty"`
	ErrMsg  string `json:"errmsg,omitempty"`
	Data    *struct {
		Time  string `json:"time"`
		Token string `json:"token"`
	} `json:"data,omitempty"`
}

// GetTaskRequest defines the request structure for GetTask API
type GetTaskRequest struct {
	ProverVersion string            `json:"prover_version"`
	ProverHeight  uint64            `json:"prover_height"`
	TaskType      message.ProofType `json:"task_type"`
}

// GetTaskResponse defines the response structure for GetTask API
type GetTaskResponse struct {
	ErrCode int             `json:"errcode,omitempty"`
	ErrMsg  string          `json:"errmsg,omitempty"`
	Data    message.TaskMsg `json:"data,omitempty"`
}

// SubmitProofRequest defines the request structure for the SubmitProof API.
type SubmitProofRequest struct {
	TaskID    string             `json:"task_id"`
	Status    message.RespStatus `json:"status"`
	Error     string             `json:"error"`
	TaskType  message.ProofType  `json:"task_type"`
	Signature string             `json:"signature"`
	Proof     string             `json:"proof"`
}

// SubmitProofResponse defines the response structure for the SubmitProof API.
type SubmitProofResponse struct {
	ErrCode int    `json:"errcode,omitempty"`
	ErrMsg  string `json:"errmsg,omitempty"`
	Success bool   `json:"success"`
}
