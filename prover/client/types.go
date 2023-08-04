package client

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"scroll-tech/common/types/message"
)

// Response the response schema
type Response struct {
	ErrCode int         `json:"errcode,omitempty"`
	ErrMsg  string      `json:"errmsg,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// RenderJSON renders response with json
func RenderJSON(ctx *gin.Context, errCode int, err error, data interface{}) {
	var errMsg string
	if err != nil {
		errMsg = err.Error()
	}
	renderData := Response{
		ErrCode: errCode,
		ErrMsg:  errMsg,
		Data:    data,
	}
	ctx.JSON(http.StatusOK, renderData)
}

// ChallengeResponse defines the response structure for random API
type ChallengeResponse struct {
	ErrCode int    `json:"errcode,omitempty"`
	ErrMsg  string `json:"errmsg,omitempty"`
	Data    *struct {
		Time  string `json:"time"`
		Token string `json:"token"`
	} `json:"data,omitempty"`
}

// LoginRequest defines the request structure for login API
type LoginRequest struct {
	Message struct {
		Challenge  string `json:"challenge"`
		ProverName string `json:"prover_name"`
	} `json:"message"`
	Signature string `json:"signature"`
}

// LoginResponse defines the response structure for login API
type LoginResponse struct {
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
	Message message.ProofDetail `json:"message"`
}

// SubmitProofResponse defines the response structure for the SubmitProof API.
type SubmitProofResponse struct {
	ErrCode int    `json:"errcode,omitempty"`
	ErrMsg  string `json:"errmsg,omitempty"`
}
