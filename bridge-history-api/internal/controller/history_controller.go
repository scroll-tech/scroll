package controller

import (
	"errors"
	"reflect"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"

	"bridge-history-api/internal/logic"
	"bridge-history-api/internal/types"
)

const (
	cacheKeyPrefixClaimableTxsByAddr = "claimableTxsByAddr:"
	cacheKeyPrefixQueryTxsByHash     = "queryTxsByHash:"
)

// HistoryController contains the query claimable txs service
type HistoryController struct {
	historyLogic *logic.HistoryLogic
	cache        *cache.Cache
	singleFlight singleflight.Group
}

// NewHistoryController return HistoryController instance
func NewHistoryController(db *gorm.DB) *HistoryController {
	return &HistoryController{
		historyLogic: logic.NewHistoryLogic(db),
		cache:        cache.New(30*time.Second, 10*time.Minute),
	}
}

// GetAllClaimableTxsByAddr defines the http get method behavior
func (c *HistoryController) GetAllClaimableTxsByAddr(ctx *gin.Context) {
	var req types.QueryByAddressRequest
	if err := ctx.ShouldBind(&req); err != nil {
		types.RenderFailure(ctx, types.ErrParameterInvalidNo, err)
		return
	}

	cacheKey := cacheKeyPrefixClaimableTxsByAddr + req.Address
	if cachedData, found := c.cache.Get(cacheKey); found {
		if resultData, ok := cachedData.(*types.ResultData); ok {
			types.RenderSuccess(ctx, resultData)
			return
		}
		// Log error for unexpected type, then fetch data from the database.
		log.Error("unexpected type in cache", "expected", "*types.ResultData", "got", reflect.TypeOf(cachedData))
	}

	result, err, _ := c.singleFlight.Do(cacheKey, func() (interface{}, error) {
		txs, total, err := c.historyLogic.GetClaimableTxsByAddress(ctx, common.HexToAddress(req.Address))
		if err != nil {
			return nil, err
		}
		resultData := &types.ResultData{Result: txs, Total: total}
		c.cache.Set(cacheKey, resultData, cache.DefaultExpiration)
		return resultData, nil
	})

	if err != nil {
		types.RenderFailure(ctx, types.ErrGetClaimablesFailure, err)
		return
	}

	if resultData, ok := result.(*types.ResultData); ok {
		types.RenderSuccess(ctx, resultData)
	} else {
		log.Error("unexpected type from singleflight", "expected", "*types.ResultData", "got", reflect.TypeOf(result))
		types.RenderFailure(ctx, types.ErrGetClaimablesFailure, errors.New("unexpected error"))
	}
}

// GetAllTxsByAddr defines the http get method behavior
//func (c *HistoryController) GetAllTxsByAddr(ctx *gin.Context) {
//	var req types.QueryByAddressRequest
//	if err := ctx.ShouldBind(&req); err != nil {
//		types.RenderJSON(ctx, types.ErrParameterInvalidNo, err, nil)
//		return
//	}
//	offset := (req.Page - 1) * req.PageSize
//	limit := req.PageSize
//	message, total, err := c.historyLogic.GetTxsByAddress(ctx, common.HexToAddress(req.Address), offset, limit)
//	if err != nil {
//		types.RenderFailure(ctx, types.ErrGetTxsByAddrFailure, err)
//      return
//  }
//  types.RenderSuccess(ctx, &types.ResultData{Result: message, Total: total})
//}

// PostQueryTxsByHash defines the http post method behavior
func (c *HistoryController) PostQueryTxsByHash(ctx *gin.Context) {
	var req types.QueryByHashRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		types.RenderFailure(ctx, types.ErrParameterInvalidNo, err)
		return
	}

	if len(req.Txs) > 10 {
		types.RenderFailure(ctx, types.ErrParameterInvalidNo, errors.New("the number of hashes in the request exceeds the allowed maximum"))
		return
	}
	hashesMap := make(map[string]struct{}, len(req.Txs))
	results := make([]*types.TxHistoryInfo, 0, len(req.Txs))
	uncachedHashes := make([]string, 0, len(req.Txs))
	for _, hash := range req.Txs {
		if _, exists := hashesMap[hash]; exists {
			// Skip duplicate tx hash values.
			continue
		}
		hashesMap[hash] = struct{}{}

		cacheKey := cacheKeyPrefixQueryTxsByHash + hash
		if cachedData, found := c.cache.Get(cacheKey); found {
			if txInfo, ok := cachedData.(*types.TxHistoryInfo); ok {
				results = append(results, txInfo)
			} else {
				log.Error("unexpected type in cache", "expected", "*types.TxHistoryInfo", "got", reflect.TypeOf(cachedData))
				uncachedHashes = append(uncachedHashes, hash)
			}
		} else {
			uncachedHashes = append(uncachedHashes, hash)
		}
	}

	releaseChan := make(chan struct{})
	defer close(releaseChan)
	doSingleFlight := func(hashes []string) {
		for _, hash := range hashes {
			_, err, _ := c.singleFlight.Do(hash, func() (interface{}, error) {
				<-releaseChan
				return nil, nil
			})
			// Add this check to fix golint: log and continue.
			if err != nil {
				log.Warn("unexpected error", "err", err)
			}
		}
	}

	if len(uncachedHashes) > 0 {
		go doSingleFlight(uncachedHashes)

		dbResults, err := c.historyLogic.GetTxsByHashes(ctx, uncachedHashes)
		if err != nil {
			types.RenderFailure(ctx, types.ErrGetTxsByHashFailure, err)
			return
		}

		for _, result := range dbResults {
			results = append(results, result)
			cacheKey := cacheKeyPrefixQueryTxsByHash + result.Hash
			c.cache.Set(cacheKey, result, cache.DefaultExpiration)
		}
	}

	resultData := &types.ResultData{Result: results, Total: uint64(len(results))}
	types.RenderSuccess(ctx, resultData)
}
