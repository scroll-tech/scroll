package api

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"

	"scroll-tech/bridge-history-api/internal/logic"
	"scroll-tech/bridge-history-api/internal/types"
)

// maxRequestBodyBytes fits 100 tx hashes with ample headroom.
const maxRequestBodyBytes = 64 * 1024

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
	body, err := readBodyWithSizeLimit(ctx, maxRequestBodyBytes)
	if err != nil {
		types.RenderFailure(ctx, types.ErrParameterInvalidNo, err)
		return
	}

	var req types.QueryByHashRequest
	if err := binding.JSON.BindBody(body, &req); err != nil {
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

func readBodyWithSizeLimit(ctx *gin.Context, limit int64) ([]byte, error) {
	// Reject early if the declared size is too large.
	if ctx.Request.ContentLength > limit {
		return nil, &http.MaxBytesError{Limit: limit}
	}
	// Read to EOF to catch trailing data past the limit.
	return io.ReadAll(http.MaxBytesReader(ctx.Writer, ctx.Request.Body, limit))
}
