package types

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	// Success indicates that the operation was successful.
	Success = 0
	// InternalServerError represents a fatal error occurring on the server.
	InternalServerError = 500
	// ErrParameterInvalidNo represents an error when the parameters are invalid.
	ErrParameterInvalidNo = 40001
	// ErrGetL2ClaimableWithdrawalsError represents an error when trying to get L2 claimable withdrawal transactions.
	ErrGetL2ClaimableWithdrawalsError = 40002
	// ErrGetL2WithdrawalsError represents an error when trying to get L2 withdrawal transactions by address.
	ErrGetL2WithdrawalsError = 40003
	// ErrGetTxsError represents an error when trying to get transactions by address.
	ErrGetTxsError = 40004
	// ErrGetTxsByHashError represents an error when trying to get transactions by hash list.
	ErrGetTxsByHashError = 40005
)

// QueryByAddressRequest the request parameter of address api
type QueryByAddressRequest struct {
	Address  string `form:"address" binding:"required"`
	Page     int    `form:"page" binding:"required"`
	PageSize int    `form:"page_size" binding:"required"`
}

// QueryByHashRequest the request parameter of hash api
type QueryByHashRequest struct {
	Txs []string `raw:"txs" binding:"required"`
}

// ResultData contains return txs and total
type ResultData struct {
	Result []*TxHistoryInfo `json:"result"`
	Total  uint64           `json:"total"`
}

// Response the response schema
type Response struct {
	ErrCode int         `json:"errcode"`
	ErrMsg  string      `json:"errmsg"`
	Data    interface{} `json:"data"`
}

// Finalized the schema of tx finalized infos
type Finalized struct {
	Hash        string `json:"hash"`
	BlockNumber uint64 `json:"blockNumber"`
}

// UserClaimInfo the schema of tx claim infos
type UserClaimInfo struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Value      string `json:"value"`
	Nonce      string `json:"nonce"`
	Message    string `json:"message"`
	Proof      string `json:"proof"`
	BatchIndex string `json:"batch_index"`
	Claimable  bool   `json:"claimable"`
}

// TxHistoryInfo the schema of tx history infos
type TxHistoryInfo struct {
	Hash        string         `json:"hash"`
	MsgHash     string         `json:"msgHash"`
	Amount      string         `json:"amount"`
	IsL1        bool           `json:"isL1"`
	L1Token     string         `json:"l1Token"`
	L2Token     string         `json:"l2Token"`
	BlockNumber uint64         `json:"blockNumber"`
	TxStatus    int            `json:"txStatus"`
	FinalizeTx  *Finalized     `json:"finalizeTx"`
	ClaimInfo   *UserClaimInfo `json:"claimInfo"`
	CreatedAt   *time.Time     `json:"createdTime"`
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

// RenderSuccess renders success response with json
func RenderSuccess(ctx *gin.Context, data interface{}) {
	RenderJSON(ctx, Success, nil, data)
}

// RenderFailure renders failure response with json
func RenderFailure(ctx *gin.Context, errCode int, err error) {
	RenderJSON(ctx, errCode, err, nil)
}

// RenderFatal renders fatal response with json
func RenderFatal(ctx *gin.Context, err error) {
	var errMsg string
	if err != nil {
		errMsg = err.Error()
	}
	renderData := Response{
		ErrCode: InternalServerError,
		ErrMsg:  errMsg,
		Data:    nil,
	}
	ctx.Set("errcode", InternalServerError)
	ctx.JSON(http.StatusInternalServerError, renderData)
}
