package controller

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/patrickmn/go-cache"
	"github.com/scroll-tech/go-ethereum/log"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"

	"bridge-history-api/internal/logic"
	"bridge-history-api/internal/types"
)

const (
	cacheKeyPrefixL2ClaimableWithdrawalsByAddr = "l2ClaimableWithdrawalsByAddr:"
	cacheKeyPrefixL2WithdrawalsByAddr          = "l2WithdrawalsByAddr:"
	cacheKeyPrefixTxsByAddr                    = "txsByAddr:"
	cacheKeyPrefixQueryTxsByHashes             = "queryTxsByHashes:"
)

// HistoryController contains the query claimable txs service
type HistoryController struct {
	historyLogic *logic.HistoryLogic
	redis        *redis.Client
	cache        *cache.Cache
	singleFlight singleflight.Group
	cacheMetrics *cacheMetrics
}

// NewHistoryController return HistoryController instance
func NewHistoryController(db *gorm.DB, redis *redis.Client) *HistoryController {
	return &HistoryController{
		historyLogic: logic.NewHistoryLogic(db),
		redis:        redis,
		cache:        cache.New(30*time.Second, 10*time.Minute),
		cacheMetrics: initCacheMetrics(),
	}
}

// GetL2ClaimableWithdrawalsByAddress defines the http get method behavior
func (c *HistoryController) GetL2ClaimableWithdrawalsByAddress(ctx *gin.Context) {
	var req types.QueryByAddressRequest
	if err := ctx.ShouldBind(&req); err != nil {
		types.RenderFailure(ctx, types.ErrParameterInvalidNo, err)
		return
	}

	cacheKey := cacheKeyPrefixL2ClaimableWithdrawalsByAddr + req.Address
	pagedTxs, total, isHit, err := c.getCachedTxsInfo(ctx, cacheKey, uint64(req.Page), uint64(req.PageSize))
	if err != nil {
		log.Error("failed to get cached tx info", "cached key", cacheKey, "page", req.Page, "page size", req.PageSize, "error", err)
		types.RenderFailure(ctx, types.ErrGetL2ClaimableWithdrawalsError, err)
		return
	}

	if isHit {
		c.cacheMetrics.cacheHits.WithLabelValues("GetL2ClaimableWithdrawalsByAddress").Inc()
		log.Info("cache hit", "request", req)
		resultData := &types.ResultData{Result: pagedTxs, Total: total}
		types.RenderSuccess(ctx, resultData)
		return
	}

	c.cacheMetrics.cacheMisses.WithLabelValues("GetL2ClaimableWithdrawalsByAddress").Inc()
	log.Info("cache miss", "request", req)

	result, err, _ := c.singleFlight.Do(cacheKey, func() (interface{}, error) {
		var txs []*types.TxHistoryInfo
		txs, err = c.historyLogic.GetL2ClaimableWithdrawalsByAddress(ctx, req.Address)
		if err != nil {
			return nil, err
		}
		return txs, nil
	})
	if err != nil {
		types.RenderFailure(ctx, types.ErrGetL2ClaimableWithdrawalsError, err)
		return
	}

	txs, ok := result.([]*types.TxHistoryInfo)
	if !ok {
		log.Error("unexpected type from singleflight", "expected", "[]*types.TxHistoryInfo", "got", reflect.TypeOf(result))
		types.RenderFailure(ctx, types.ErrGetL2ClaimableWithdrawalsError, errors.New("unexpected error"))
		return
	}

	err = c.cacheTxsInfo(ctx, cacheKey, txs)
	if err != nil {
		log.Error("failed to cache txs info", "key", cacheKey, "err", err)
		types.RenderFailure(ctx, types.ErrGetL2ClaimableWithdrawalsError, err)
		return
	}

	pagedTxs, total, isHit, err = c.getCachedTxsInfo(ctx, cacheKey, uint64(req.Page), uint64(req.PageSize))
	if err != nil {
		log.Error("failed to get cached tx info", "cached key", cacheKey, "page", req.Page, "page size", req.PageSize, "error", err)
		types.RenderFailure(ctx, types.ErrGetL2ClaimableWithdrawalsError, err)
		return
	}

	if !isHit {
		log.Error("cache miss after write, expect hit", "cached key", cacheKey, "page", req.Page, "page size", req.PageSize, "error", err)
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

	cacheKey := cacheKeyPrefixL2WithdrawalsByAddr + req.Address
	pagedTxs, total, isHit, err := c.getCachedTxsInfo(ctx, cacheKey, uint64(req.Page), uint64(req.PageSize))
	if err != nil {
		log.Error("failed to get cached tx info", "cached key", cacheKey, "page", req.Page, "page size", req.PageSize, "error", err)
		types.RenderFailure(ctx, types.ErrGetL2WithdrawalsError, err)
		return
	}

	if isHit {
		c.cacheMetrics.cacheHits.WithLabelValues("GetL2WithdrawalsByAddress").Inc()
		log.Info("cache hit", "request", req)
		resultData := &types.ResultData{Result: pagedTxs, Total: total}
		types.RenderSuccess(ctx, resultData)
		return
	}

	c.cacheMetrics.cacheMisses.WithLabelValues("GetL2WithdrawalsByAddress").Inc()
	log.Info("cache miss", "request", req)

	result, err, _ := c.singleFlight.Do(cacheKey, func() (interface{}, error) {
		var txs []*types.TxHistoryInfo
		txs, err = c.historyLogic.GetL2WithdrawalsByAddress(ctx, req.Address)
		if err != nil {
			return nil, err
		}
		return txs, nil
	})
	if err != nil {
		types.RenderFailure(ctx, types.ErrGetL2WithdrawalsError, err)
		return
	}

	txs, ok := result.([]*types.TxHistoryInfo)
	if !ok {
		log.Error("unexpected type from singleflight", "expected", "[]*types.TxHistoryInfo", "got", reflect.TypeOf(result))
		types.RenderFailure(ctx, types.ErrGetL2WithdrawalsError, errors.New("unexpected error"))
		return
	}

	err = c.cacheTxsInfo(ctx, cacheKey, txs)
	if err != nil {
		log.Error("failed to cache txs info", "key", cacheKey, "err", err)
		types.RenderFailure(ctx, types.ErrGetL2WithdrawalsError, err)
		return
	}

	pagedTxs, total, isHit, err = c.getCachedTxsInfo(ctx, cacheKey, uint64(req.Page), uint64(req.PageSize))
	if err != nil {
		log.Error("failed to get cached tx info", "cached key", cacheKey, "page", req.Page, "page size", req.PageSize, "error", err)
		types.RenderFailure(ctx, types.ErrGetL2WithdrawalsError, err)
		return
	}

	if !isHit {
		log.Error("cache miss after write, expect hit", "cached key", cacheKey, "page", req.Page, "page size", req.PageSize, "error", err)
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

	cacheKey := cacheKeyPrefixTxsByAddr + req.Address
	pagedTxs, total, isHit, err := c.getCachedTxsInfo(ctx, cacheKey, uint64(req.Page), uint64(req.PageSize))
	if err != nil {
		log.Error("failed to get cached tx info", "cached key", cacheKey, "page", req.Page, "page size", req.PageSize, "error", err)
		types.RenderFailure(ctx, types.ErrGetTxsError, err)
		return
	}

	if isHit {
		c.cacheMetrics.cacheHits.WithLabelValues("GetTxsByAddress").Inc()
		log.Info("cache hit", "request", req)
		resultData := &types.ResultData{Result: pagedTxs, Total: total}
		types.RenderSuccess(ctx, resultData)
		return
	}

	c.cacheMetrics.cacheMisses.WithLabelValues("GetTxsByAddress").Inc()
	log.Info("cache miss", "request", req)

	result, err, _ := c.singleFlight.Do(cacheKey, func() (interface{}, error) {
		var txs []*types.TxHistoryInfo
		txs, err = c.historyLogic.GetTxsByAddress(ctx, req.Address)
		if err != nil {
			return nil, err
		}
		return txs, nil
	})
	if err != nil {
		types.RenderFailure(ctx, types.ErrGetTxsError, err)
		return
	}

	txs, ok := result.([]*types.TxHistoryInfo)
	if !ok {
		log.Error("unexpected type from singleflight", "expected", "[]*types.TxHistoryInfo", "got", reflect.TypeOf(result))
		types.RenderFailure(ctx, types.ErrGetTxsError, errors.New("unexpected error"))
		return
	}

	err = c.cacheTxsInfo(ctx, cacheKey, txs)
	if err != nil {
		log.Error("failed to cache txs info", "key", cacheKey, "err", err)
		types.RenderFailure(ctx, types.ErrGetTxsError, err)
		return
	}

	pagedTxs, total, isHit, err = c.getCachedTxsInfo(ctx, cacheKey, uint64(req.Page), uint64(req.PageSize))
	if err != nil {
		log.Error("failed to get cached tx info", "cached key", cacheKey, "page", req.Page, "page size", req.PageSize, "error", err)
		types.RenderFailure(ctx, types.ErrGetTxsError, err)
		return
	}

	if !isHit {
		log.Error("cache miss after write, expect hit", "cached key", cacheKey, "page", req.Page, "page size", req.PageSize, "error", err)
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
		types.RenderFailure(ctx, types.ErrParameterInvalidNo, errors.New("the number of hashes in the request exceeds the allowed maximum of 100"))
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

		cacheKey := cacheKeyPrefixQueryTxsByHashes + hash
		cachedData, err := c.redis.Get(ctx, cacheKey).Bytes()
		if err == nil {
			c.cacheMetrics.cacheHits.WithLabelValues("PostQueryTxsByHashes").Inc()
			// Log cache hit along with tx hash.
			log.Info("cache hit", "tx hash", hash)
			if len(cachedData) == 0 {
				continue
			} else {
				var txInfo types.TxHistoryInfo
				if err = json.Unmarshal(cachedData, &txInfo); err != nil {
					log.Error("failed to unmarshal cached data", "error", err)
					uncachedHashes = append(uncachedHashes, hash)
				} else {
					results = append(results, &txInfo)
				}
			}
		} else if err == redis.Nil {
			c.cacheMetrics.cacheMisses.WithLabelValues("PostQueryTxsByHashes").Inc()
			// Log cache miss along with tx hash.
			log.Info("cache miss", "tx hash", hash)
			uncachedHashes = append(uncachedHashes, hash)
		} else {
			log.Error("failed to get data from Redis", "error", err)
			uncachedHashes = append(uncachedHashes, hash)
		}
	}

	if len(uncachedHashes) > 0 {
		dbResults, err := c.historyLogic.GetTxsByHashes(ctx, uncachedHashes)
		if err != nil {
			types.RenderFailure(ctx, types.ErrGetTxsByHashError, err)
			return
		}

		resultMap := make(map[string]*types.TxHistoryInfo)
		for _, result := range dbResults {
			results = append(results, result)
			resultMap[result.Hash] = result
		}

		for _, hash := range uncachedHashes {
			cacheKey := cacheKeyPrefixQueryTxsByHashes + hash
			result, found := resultMap[hash]
			if found {
				jsonData, err := json.Marshal(result)
				if err != nil {
					log.Error("failed to marshal data", "error", err)
				} else {
					if err := c.redis.Set(ctx, cacheKey, jsonData, 30*time.Minute).Err(); err != nil {
						log.Error("failed to set data to Redis", "error", err)
					}
				}
			} else {
				if err := c.redis.Set(ctx, cacheKey, "", 30*time.Minute).Err(); err != nil {
					log.Error("failed to set data to Redis", "error", err)
				}
			}
		}
	}

	resultData := &types.ResultData{Result: results, Total: uint64(len(results))}
	types.RenderSuccess(ctx, resultData)
}

func (c *HistoryController) getCachedTxsInfo(ctx context.Context, cacheKey string, pageNum, pageSize uint64) ([]*types.TxHistoryInfo, uint64, bool, error) {
	start := int64(pageNum * pageSize)
	end := int64((pageNum+1)*pageSize - 1)

	total, err := c.redis.ZCard(ctx, cacheKey).Result()
	if err != nil {
		if err == redis.Nil {
			// Key does not exist, cache miss.
			return nil, 0, false, nil
		}
		log.Error("failed to get zcard result", "error", err)
		return nil, 0, false, err
	}

	values, err := c.redis.ZRange(ctx, cacheKey, start, end).Result()
	if err != nil {
		if err == redis.Nil {
			// Key does not exist, cache miss.
			return nil, 0, false, nil
		}
		log.Error("failed to get zrange result", "error", err)
		return nil, 0, false, err
	}

	var pagedTxs []*types.TxHistoryInfo
	for _, v := range values {
		var tx types.TxHistoryInfo
		err := json.Unmarshal([]byte(v), &tx)
		if err != nil {
			log.Error("failed to unmarshal transaction data", "error", err)
			return nil, 0, false, err
		}
		pagedTxs = append(pagedTxs, &tx)
	}

	return pagedTxs, uint64(total), true, nil
}

func (c *HistoryController) cacheTxsInfo(ctx context.Context, cacheKey string, txs []*types.TxHistoryInfo) error {
	err := c.redis.Watch(ctx, func(tx *redis.Tx) error {
		pipe := tx.Pipeline()

		// The transactions are sorted, thus we set the score as their indices.
		for i, tx := range txs {
			if err := pipe.ZAdd(ctx, cacheKey, &redis.Z{Score: float64(i), Member: tx}).Err(); err != nil {
				log.Error("failed to add transaction to sorted set", "error", err)
				return err
			}
		}

		if err := pipe.Expire(ctx, cacheKey, 30*time.Minute).Err(); err != nil {
			log.Error("failed to set expiry time", "error", err)
			return err
		}

		_, err := pipe.Exec(ctx)
		if err != nil {
			log.Error("failed to execute transaction", "error", err)
			return err
		}

		return nil
	}, cacheKey)

	if err != nil {
		log.Error("failed to execute transaction", "error", err)
		return err
	}

	return nil
}
