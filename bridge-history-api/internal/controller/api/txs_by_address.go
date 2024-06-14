package api

import (
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"

	"scroll-tech/bridge-history-api/internal/logic"
	"scroll-tech/bridge-history-api/internal/types"
)

// TxsByAddressController the controller of GetTxsByAddress
type TxsByAddressController struct {
	historyLogic *logic.HistoryLogic
}

// NewTxsByAddressController create new TxsByAddressController
func NewTxsByAddressController(db *gorm.DB, redisClient *redis.Client) *TxsByAddressController {
	return &TxsByAddressController{
		historyLogic: logic.NewHistoryLogic(db, redisClient),
	}
}

// GetTxsByAddress defines the http get method behavior
func (c *TxsByAddressController) GetTxsByAddress(ctx *gin.Context) {
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

	resultData := &types.ResultData{Results: pagedTxs, Total: total}
	types.RenderSuccess(ctx, resultData)
}
