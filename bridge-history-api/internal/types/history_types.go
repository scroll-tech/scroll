package types

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	// Success shows OK.
	Success = 0
	// InternalServerError shows a fatal error in the server
	InternalServerError = 500
	// ErrParameterInvalidNo is invalid params
	ErrParameterInvalidNo = 40001
	// ErrGetClaimablesFailure is getting all claimables txs error
	ErrGetClaimablesFailure = 40002
	// ErrGetTxsByHashFailure is getting txs by hash list error
	ErrGetTxsByHashFailure = 40003
	// ErrGetTxsByAddrFailure is getting txs by address error
	ErrGetTxsByAddrFailure = 40004
	// ErrGetWithdrawRootByBatchIndexFailure is getting withdraw root by batch index error
	ErrGetWithdrawRootByBatchIndexFailure = 40005
)

// QueryByAddressRequest the request parameter of address api
type QueryByAddressRequest struct {
	Address  string `form:"address" binding:"required"`
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=10"`
}

// QueryByHashRequest the request parameter of hash api
type QueryByHashRequest struct {
	Txs []string `raw:"txs" binding:"required"`
}

// QueryByBatchIndexRequest the request parameter of batch index api
type QueryByBatchIndexRequest struct {
	// BatchIndex can not be 0, because we dont decode the genesis block
	BatchIndex uint64 `form:"batch_index" binding:"required"`
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
	Hash           string     `json:"hash"`
	Amount         string     `json:"amount"`
	To             string     `json:"to"` // useless
	IsL1           bool       `json:"isL1"`
	BlockNumber    uint64     `json:"blockNumber"`
	BlockTimestamp *time.Time `json:"blockTimestamp"` // uselesss
}

// UserClaimInfo the schema of tx claim infos
type UserClaimInfo struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Value      string `json:"value"`
	Nonce      string `json:"nonce"`
	BatchHash  string `json:"batch_hash"`
	Message    string `json:"message"`
	Proof      string `json:"proof"`
	BatchIndex string `json:"batch_index"`
}

// TxHistoryInfo the schema of tx history infos
type TxHistoryInfo struct {
	Hash           string         `json:"hash"`
	MsgHash        string         `json:"msgHash"`
	Amount         string         `json:"amount"`
	To             string         `json:"to"` // useless
	IsL1           bool           `json:"isL1"`
	L1Token        string         `json:"l1Token"`
	L2Token        string         `json:"l2Token"`
	BlockNumber    uint64         `json:"blockNumber"`
	BlockTimestamp *time.Time     `json:"blockTimestamp"` // useless
	FinalizeTx     *Finalized     `json:"finalizeTx"`
	ClaimInfo      *UserClaimInfo `json:"claimInfo"`
	CreatedAt      *time.Time     `json:"createdTime"`
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
