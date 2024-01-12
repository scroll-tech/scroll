package types

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"scroll-tech/bridge-history-api/internal/orm"
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
	Page     uint64 `form:"page" binding:"required,min=1"`
	PageSize uint64 `form:"page_size" binding:"required,min=1,max=100"`
}

// QueryByHashRequest the request parameter of hash api
type QueryByHashRequest struct {
	Txs []string `json:"txs" binding:"required,min=1,max=100"`
}

// ResultData contains return txs and total
type ResultData struct {
	Results []*TxHistoryInfo `json:"results"`
	Total   uint64           `json:"total"`
}

// Response the response schema
type Response struct {
	ErrCode int         `json:"errcode"`
	ErrMsg  string      `json:"errmsg"`
	Data    interface{} `json:"data"`
}

// CounterpartChainTx is the schema of counterpart chain tx info
type CounterpartChainTx struct {
	Hash        string `json:"hash"`
	BlockNumber uint64 `json:"block_number"`
}

// ClaimInfo is the schema of tx claim info
type ClaimInfo struct {
	From      string         `json:"from"`
	To        string         `json:"to"`
	Value     string         `json:"value"`
	Nonce     string         `json:"nonce"`
	Message   string         `json:"message"`
	Proof     L2MessageProof `json:"proof"`
	Claimable bool           `json:"claimable"`
}

// L2MessageProof is the schema of L2 message proof
type L2MessageProof struct {
	BatchIndex  string `json:"batch_index"`
	MerkleProof string `json:"merkle_proof"`
}

// TxHistoryInfo the schema of tx history infos
type TxHistoryInfo struct {
	Hash               string              `json:"hash"`
	ReplayTxHash       string              `json:"replay_tx_hash"`
	RefundTxHash       string              `json:"refund_tx_hash"`
	MessageHash        string              `json:"message_hash"`
	TokenType          orm.TokenType       `json:"token_type"`    // 0: unknown, 1: eth, 2: erc20, 3: erc721, 4: erc1155
	TokenIDs           []string            `json:"token_ids"`     // only for erc721 and erc1155
	TokenAmounts       []string            `json:"token_amounts"` // for eth and erc20, the length is 1, for erc721 and erc1155, the length could be > 1
	MessageType        orm.MessageType     `json:"message_type"`  // 0: unknown, 1: layer 1 message, 2: layer 2 message
	L1TokenAddress     string              `json:"l1_token_address"`
	L2TokenAddress     string              `json:"l2_token_address"`
	BlockNumber        uint64              `json:"block_number"`
	TxStatus           orm.TxStatusType    `json:"tx_status"` // 0: sent, 1: sent failed, 2: relayed, 3: failed relayed, 4: relayed reverted, 5: skipped, 6: dropped
	CounterpartChainTx *CounterpartChainTx `json:"counterpart_chain_tx"`
	ClaimInfo          *ClaimInfo          `json:"claim_info"`
	BlockTimestamp     uint64              `json:"block_timestamp"`
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
