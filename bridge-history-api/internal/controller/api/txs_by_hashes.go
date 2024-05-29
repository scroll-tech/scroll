package api

import (
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"

	"scroll-tech/bridge-history-api/internal/logic"
	"scroll-tech/bridge-history-api/internal/types"
)

// TxsByHashesController the controller of PostQueryTxsByHashes
type TxsByHashesController struct {
	historyLogic *logic.HistoryLogic
}

// NewTxsByHashesController create a new TxsByHashesController
func NewTxsByHashesController(db *gorm.DB, redisClient *redis.Client) *TxsByHashesController {
	return &TxsByHashesController{
		historyLogic: logic.NewHistoryLogic(db, redisClient),
	}
}

// PostQueryTxsByHashes query the txs by hashes
func (c *TxsByHashesController) PostQueryTxsByHashes(ctx *gin.Context) {
	var req types.QueryByHashRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		types.RenderFailure(ctx, types.ErrParameterInvalidNo, err)
		return
	}

	results, err := c.historyLogic.GetTxsByHashes(ctx, req.Txs)
	if err != nil {
		types.RenderFailure(ctx, types.ErrGetTxsByHashError, err)
		return
	}

	resultData := &types.ResultData{Results: results, Total: uint64(len(results))}
	types.RenderSuccess(ctx, resultData)
}
