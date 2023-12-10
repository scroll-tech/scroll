package api

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"

	"scroll-tech/bridge-history-api/internal/logic"
	"scroll-tech/bridge-history-api/internal/types"
)

// HistoryController contains the query claimable txs service
type HistoryController struct {
	historyLogic *logic.HistoryLogic
}

// NewHistoryController return HistoryController instance
func NewHistoryController(db *gorm.DB, redis *redis.Client) *HistoryController {
	return &HistoryController{
		historyLogic: logic.NewHistoryLogic(db, redis),
	}
}

// GetL2UnclaimedWithdrawalsByAddress defines the http get method behavior
func (c *HistoryController) GetL2UnclaimedWithdrawalsByAddress(ctx *gin.Context) {
	var req types.QueryByAddressRequest
	if err := ctx.ShouldBind(&req); err != nil {
		types.RenderFailure(ctx, types.ErrParameterInvalidNo, err)
		return
	}

	pagedTxs, total, err := c.historyLogic.GetL2UnclaimedWithdrawalsByAddress(ctx, req.Address, req.Page, req.PageSize)
	if err != nil {
		types.RenderFailure(ctx, types.ErrGetL2ClaimableWithdrawalsError, err)
		return
	}

	resultData := &types.ResultData{Result: pagedTxs, Total: total}
	types.RenderSuccess(ctx, resultData)
}

// GetL2WithdrawalsByAddress defines the http get method behavior
func (c *HistoryController) GetL2WithdrawalsByAddress(ctx *gin.Context) {
	var req types.QueryByAddressRequest
	if err := ctx.ShouldBind(&req); err != nil {
		types.RenderFailure(ctx, types.ErrParameterInvalidNo, err)
		return
	}

	pagedTxs, total, err := c.historyLogic.GetL2WithdrawalsByAddress(ctx, req.Address, req.Page, req.PageSize)
	if err != nil {
		types.RenderFailure(ctx, types.ErrGetL2WithdrawalsError, err)
		return
	}

	resultData := &types.ResultData{Result: pagedTxs, Total: total}
	types.RenderSuccess(ctx, resultData)
}

// GetTxsByAddress defines the http get method behavior
func (c *HistoryController) GetTxsByAddress(ctx *gin.Context) {
	var req types.QueryByAddressRequest
	if err := ctx.ShouldBind(&req); err != nil {
		types.RenderFailure(ctx, types.ErrParameterInvalidNo, err)
		return
	}

	pagedTxs, total, err := c.historyLogic.GetTxsByAddress(ctx, req.Address, req.Page, req.PageSize)
	if err != nil {
		types.RenderFailure(ctx, types.ErrGetTxsError, err)
		return
	}

	resultData := &types.ResultData{Result: pagedTxs, Total: total}
	types.RenderSuccess(ctx, resultData)
}

// PostQueryTxsByHashes defines the http post method behavior
func (c *HistoryController) PostQueryTxsByHashes(ctx *gin.Context) {
	var req types.QueryByHashRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		types.RenderFailure(ctx, types.ErrParameterInvalidNo, err)
		return
	}

	if len(req.Txs) > 100 {
		types.RenderFailure(ctx, types.ErrParameterInvalidNo, errors.New("number of hashes exceeds the allowed maximum (100)"))
		return
	}

	results, err := c.historyLogic.GetTxsByHashes(ctx, req.Txs)
	if err != nil {
		types.RenderFailure(ctx, types.ErrGetTxsByHashError, err)
		return
	}

	resultData := &types.ResultData{Result: results, Total: uint64(len(results))}
	types.RenderSuccess(ctx, resultData)
}
