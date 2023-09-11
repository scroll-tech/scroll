package controller

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"bridge-history-api/internal/logic"
	"bridge-history-api/internal/types"
)

// HistoryController contains the query claimable txs service
type HistoryController struct {
	historyLogic *logic.HistoryLogic
}

// NewHistoryController return HistoryController instance
func NewHistoryController(db *gorm.DB) *HistoryController {
	return &HistoryController{
		historyLogic: logic.NewHistoryLogic(db),
	}
}

// GetAllClaimableTxsByAddr defines the http get method behavior
func (c *HistoryController) GetAllClaimableTxsByAddr(ctx *gin.Context) {
	var req types.QueryByAddressRequest
	if err := ctx.ShouldBind(&req); err != nil {
		types.RenderFailure(ctx, types.ErrParameterInvalidNo, err)
		return
	}
	offset := (req.Page - 1) * req.PageSize
	limit := req.PageSize
	txs, total, err := c.historyLogic.GetClaimableTxsByAddress(ctx, common.HexToAddress(req.Address), offset, limit)
	if err != nil {
		types.RenderFailure(ctx, types.ErrGetClaimablesFailure, err)
		return
	}

	types.RenderSuccess(ctx, &types.ResultData{Result: txs, Total: total})
}

// GetAllTxsByAddr defines the http get method behavior
func (c *HistoryController) GetAllTxsByAddr(ctx *gin.Context) {
	var req types.QueryByAddressRequest
	if err := ctx.ShouldBind(&req); err != nil {
		types.RenderJSON(ctx, types.ErrParameterInvalidNo, err, nil)
		return
	}
	offset := (req.Page - 1) * req.PageSize
	limit := req.PageSize
	message, total, err := c.historyLogic.GetTxsByAddress(ctx, common.HexToAddress(req.Address), offset, limit)
	if err != nil {
		types.RenderFailure(ctx, types.ErrGetTxsByAddrFailure, err)
		return
	}
	types.RenderSuccess(ctx, &types.ResultData{Result: message, Total: total})
}

// PostQueryTxsByHash defines the http post method behavior
func (c *HistoryController) PostQueryTxsByHash(ctx *gin.Context) {
	var req types.QueryByHashRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		types.RenderFailure(ctx, types.ErrParameterInvalidNo, err)
		return
	}
	result, err := c.historyLogic.GetTxsByHashes(ctx, req.Txs)
	if err != nil {
		types.RenderFailure(ctx, types.ErrGetTxsByHashFailure, err)
		return
	}
	types.RenderSuccess(ctx, &types.ResultData{Result: result, Total: 0})
}
