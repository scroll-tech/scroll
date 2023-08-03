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

// ProverTasksRequest defines the request structure for ProverTasks API
type ProverTasksRequest struct {
	ProverVersion string            `json:"prover_version"`
	ProverHeight  uint64            `json:"prover_height"`
	ProofType     message.ProofType `json:"proof_type"`
}

// ProverTasksResponse defines the response structure for ProverTasks API
type ProverTasksResponse struct {
	ErrCode int    `json:"errcode,omitempty"`
	ErrMsg  string `json:"errmsg,omitempty"`
	Data    *struct {
		TaskID    string            `json:"task_id"`
		ProofType message.ProofType `json:"proof_type"`
		ProofData string            `json:"proof_data"`
	} `json:"data,omitempty"`
}

// SubmitProofRequest defines the request structure for the SubmitProof API.
type SubmitProofRequest struct {
	TaskID    string             `json:"task_id"`
	Status    message.RespStatus `json:"status"`
	Error     string             `json:"error"`
	ProofType message.ProofType  `json:"proof_type"`
	Signature string             `json:"signature"`
	Proof     string             `json:"proof"`
}

// SubmitProofResponse defines the response structure for the SubmitProof API.
type SubmitProofResponse struct {
	ErrCode int    `json:"errcode,omitempty"`
	ErrMsg  string `json:"errmsg,omitempty"`
}
