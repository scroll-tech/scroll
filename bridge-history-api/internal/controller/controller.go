package controller

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"bridge-history-api/internal/logic"
	"bridge-history-api/internal/model"
)

var (
	// HistoryCtrler is controller instance
	HistoryCtrler      *HistoryController
	initControllerOnce sync.Once
)

// InitController inits Controller with database
func InitController(db *gorm.DB) {
	initControllerOnce.Do(func() {
		HistoryCtrler = NewHistoryController(db)
	})
}

// HistoryController contains the query claimable txs service
type HistoryController struct {
	Service logic.HistoryLogic
}

// NewHistoryController return HistoryController instance
func NewHistoryController(db *gorm.DB) *HistoryController {
	return &HistoryController{
		Service: logic.NewHistoryLogic(db),
	}
}

// GetAllClaimableTxsByAddr defines the http get method behavior
func (c *HistoryController) GetAllClaimableTxsByAddr(ctx *gin.Context) {
	var req model.QueryByAddressRequest
	if err := ctx.ShouldBind(&req); err != nil {
		model.RenderJSON(ctx, 400, err, nil)
		return
	}
	txs, total, err := c.Service.GetClaimableTxsByAddress(ctx, common.HexToAddress(req.Address), req.Offset, req.Limit)
	if err != nil {
		model.RenderJSON(ctx, 500, err, nil)
		return
	}

	model.RenderJSON(ctx, 200, nil, &model.ResultData{Result: txs, Total: total})
}

// GetAllTxsByAddr defines the http get method behavior
func (c *HistoryController) GetAllTxsByAddr(ctx *gin.Context) {
	var req model.QueryByAddressRequest
	if err := ctx.ShouldBind(&req); err != nil {
		model.RenderJSON(ctx, 400, err, nil)
		return
	}
	message, total, err := c.Service.GetTxsByAddress(ctx, common.HexToAddress(req.Address), req.Offset, req.Limit)
	if err != nil {
		model.RenderJSON(ctx, 500, err, nil)
		return
	}
	model.RenderJSON(ctx, 200, nil, &model.ResultData{Result: message, Total: total})
}

// PostQueryTxsByHash defines the http post method behavior
func (c *HistoryController) PostQueryTxsByHash(ctx *gin.Context) {
	var req model.QueryByHashRequest
	if err := ctx.ShouldBind(&req); err != nil {
		model.RenderJSON(ctx, 400, err, nil)
		return
	}
	result, err := c.Service.GetTxsByHashes(ctx, req.Txs)
	if err != nil {
		model.RenderJSON(ctx, 500, err, nil)
		return
	}
	model.RenderJSON(ctx, 200, nil, &model.ResultData{Result: result, Total: 0})
}
