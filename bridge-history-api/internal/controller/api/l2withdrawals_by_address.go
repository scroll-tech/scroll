package api

import (
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"

	"scroll-tech/bridge-history-api/internal/logic"
	"scroll-tech/bridge-history-api/internal/types"
)

// L2WithdrawalsByAddressController the controller of GetL2WithdrawalsByAddress
type L2WithdrawalsByAddressController struct {
	historyLogic *logic.HistoryLogic
}

// NewL2WithdrawalsByAddressController create new L2WithdrawalsByAddressController
func NewL2WithdrawalsByAddressController(db *gorm.DB, redisClient *redis.Client) *L2WithdrawalsByAddressController {
	return &L2WithdrawalsByAddressController{
		historyLogic: logic.NewHistoryLogic(db, redisClient),
	}
}

// GetL2WithdrawalsByAddress defines the http get method behavior
func (c *L2WithdrawalsByAddressController) GetL2WithdrawalsByAddress(ctx *gin.Context) {
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

	resultData := &types.ResultData{Results: pagedTxs, Total: total}
	types.RenderSuccess(ctx, resultData)
}
