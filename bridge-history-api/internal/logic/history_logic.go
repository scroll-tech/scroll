package logic

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"

	"scroll-tech/bridge-history-api/internal/orm"
	"scroll-tech/bridge-history-api/internal/types"
)

const (
	cacheKeyPrefixL2ClaimableWithdrawalsByAddr = "l2ClaimableWithdrawalsByAddr:"
	cacheKeyPrefixL2WithdrawalsByAddr          = "l2WithdrawalsByAddr:"
	cacheKeyPrefixTxsByAddr                    = "txsByAddr:"
	cacheKeyPrefixQueryTxsByHashes             = "queryTxsByHashes:"
	cacheKeyExpiredTime                        = 1 * time.Minute
)

// HistoryLogic services.
type HistoryLogic struct {
	crossMessageOrm *orm.CrossMessage
	batchEventOrm   *orm.BatchEvent
	redis           *redis.Client
	singleFlight    singleflight.Group
	cacheMetrics    *cacheMetrics
}

// NewHistoryLogic returns bridge history services.
func NewHistoryLogic(db *gorm.DB, redis *redis.Client) *HistoryLogic {
	logic := &HistoryLogic{
		crossMessageOrm: orm.NewCrossMessage(db),
		batchEventOrm:   orm.NewBatchEvent(db),
		redis:           redis,
		cacheMetrics:    initCacheMetrics(),
	}
	return logic
}

// GetL2UnclaimedWithdrawalsByAddress gets all unclaimed withdrawal txs under given address.
func (h *HistoryLogic) GetL2UnclaimedWithdrawalsByAddress(ctx context.Context, address string, page, pageSize uint64) ([]*types.TxHistoryInfo, uint64, error) {
	cacheKey := cacheKeyPrefixL2ClaimableWithdrawalsByAddr + address
	pagedTxs, total, isHit, err := h.getCachedTxsInfo(ctx, cacheKey, page, pageSize)
	if err != nil {
		log.Error("failed to get cached tx info", "cached key", cacheKey, "page", page, "page size", pageSize, "error", err)
		return nil, 0, err
	}

	if isHit {
		h.cacheMetrics.cacheHits.WithLabelValues("GetL2UnclaimedWithdrawalsByAddress").Inc()
		log.Info("cache hit", "cache key", cacheKey)
		return pagedTxs, total, nil
	}

	h.cacheMetrics.cacheMisses.WithLabelValues("GetL2UnclaimedWithdrawalsByAddress").Inc()
	log.Info("cache miss", "cache key", cacheKey)

	result, err, _ := h.singleFlight.Do(cacheKey, func() (interface{}, error) {
		var messages []*orm.CrossMessage
		messages, err = h.crossMessageOrm.GetL2UnclaimedWithdrawalsByAddress(ctx, address)
		if err != nil {
			return nil, err
		}
		return messages, nil
	})
	if err != nil {
		log.Error("failed to get l2 claimable withdrawals by address", "address", address, "error", err)
		return nil, 0, err
	}

	messages, ok := result.([]*orm.CrossMessage)
	if !ok {
		log.Error("unexpected type", "expected", "[]*types.TxHistoryInfo", "got", reflect.TypeOf(result), "address", address)
		return nil, 0, errors.New("unexpected error")
	}

	return h.processAndCacheTxHistoryInfo(ctx, cacheKey, messages, page, pageSize)
}

// GetL2WithdrawalsByAddress gets all withdrawal txs under given address.
func (h *HistoryLogic) GetL2WithdrawalsByAddress(ctx context.Context, address string, page, pageSize uint64) ([]*types.TxHistoryInfo, uint64, error) {
	cacheKey := cacheKeyPrefixL2WithdrawalsByAddr + address
	pagedTxs, total, isHit, err := h.getCachedTxsInfo(ctx, cacheKey, page, pageSize)
	if err != nil {
		log.Error("failed to get cached tx info", "cached key", cacheKey, "page", page, "page size", pageSize, "error", err)
		return nil, 0, err
	}

	if isHit {
		h.cacheMetrics.cacheHits.WithLabelValues("GetL2WithdrawalsByAddress").Inc()
		log.Info("cache hit", "cache key", cacheKey)
		return pagedTxs, total, nil
	}

	h.cacheMetrics.cacheMisses.WithLabelValues("GetL2WithdrawalsByAddress").Inc()
	log.Info("cache miss", "cache key", cacheKey)

	result, err, _ := h.singleFlight.Do(cacheKey, func() (interface{}, error) {
		var messages []*orm.CrossMessage
		messages, err = h.crossMessageOrm.GetL2WithdrawalsByAddress(ctx, address)
		if err != nil {
			return nil, err
		}
		return messages, nil
	})
	if err != nil {
		log.Error("failed to get l2 withdrawals by address", "address", address, "error", err)
		return nil, 0, err
	}

	messages, ok := result.([]*orm.CrossMessage)
	if !ok {
		log.Error("unexpected type", "expected", "[]*types.TxHistoryInfo", "got", reflect.TypeOf(result), "address", address)
		return nil, 0, errors.New("unexpected error")
	}

	return h.processAndCacheTxHistoryInfo(ctx, cacheKey, messages, page, pageSize)
}

// GetTxsByAddress gets tx infos under given address.
func (h *HistoryLogic) GetTxsByAddress(ctx context.Context, address string, page, pageSize uint64) ([]*types.TxHistoryInfo, uint64, error) {
	cacheKey := cacheKeyPrefixTxsByAddr + address
	pagedTxs, total, isHit, err := h.getCachedTxsInfo(ctx, cacheKey, page, pageSize)
	if err != nil {
		log.Error("failed to get cached tx info", "cached key", cacheKey, "page", page, "page size", pageSize, "error", err)
		return nil, 0, err
	}

	if isHit {
		h.cacheMetrics.cacheHits.WithLabelValues("GetTxsByAddress").Inc()
		log.Info("cache hit", "cache key", cacheKey)
		return pagedTxs, total, nil
	}

	h.cacheMetrics.cacheMisses.WithLabelValues("GetTxsByAddress").Inc()
	log.Info("cache miss", "cache key", cacheKey)

	result, err, _ := h.singleFlight.Do(cacheKey, func() (interface{}, error) {
		var messages []*orm.CrossMessage
		messages, err = h.crossMessageOrm.GetTxsByAddress(ctx, address)
		if err != nil {
			return nil, err
		}
		return messages, nil
	})
	if err != nil {
		log.Error("failed to get txs by address", "address", address, "error", err)
		return nil, 0, err
	}

	messages, ok := result.([]*orm.CrossMessage)
	if !ok {
		log.Error("unexpected type", "expected", "[]*types.TxHistoryInfo", "got", reflect.TypeOf(result), "address", address)
		return nil, 0, errors.New("unexpected error")
	}

	return h.processAndCacheTxHistoryInfo(ctx, cacheKey, messages, page, pageSize)
}

// GetTxsByHashes gets tx infos under given tx hashes.
func (h *HistoryLogic) GetTxsByHashes(ctx context.Context, txHashes []string) ([]*types.TxHistoryInfo, error) {
	hashesMap := make(map[string]struct{}, len(txHashes))
	results := make([]*types.TxHistoryInfo, 0, len(txHashes))
	uncachedHashes := make([]string, 0, len(txHashes))
	for _, hash := range txHashes {
		if _, exists := hashesMap[hash]; exists {
			// Skip duplicate tx hash values.
			continue
		}
		hashesMap[hash] = struct{}{}

		cacheKey := cacheKeyPrefixQueryTxsByHashes + hash
		cachedData, err := h.redis.Get(ctx, cacheKey).Bytes()
		if err == nil {
			h.cacheMetrics.cacheHits.WithLabelValues("PostQueryTxsByHashes").Inc()
			log.Info("cache hit", "cache key", cacheKey)
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
			h.cacheMetrics.cacheMisses.WithLabelValues("PostQueryTxsByHashes").Inc()
			log.Info("cache miss", "cache key", cacheKey)
			uncachedHashes = append(uncachedHashes, hash)
		} else {
			log.Error("failed to get data from Redis", "error", err)
			uncachedHashes = append(uncachedHashes, hash)
		}
	}

	if len(uncachedHashes) > 0 {
		messages, err := h.crossMessageOrm.GetMessagesByTxHashes(ctx, uncachedHashes)
		if err != nil {
			log.Error("failed to get messages by tx hashes", "hashes", uncachedHashes)
			return nil, err
		}

		var txHistories []*types.TxHistoryInfo
		for _, message := range messages {
			txHistories = append(txHistories, getTxHistoryInfo(message))
		}

		resultMap := make(map[string]*types.TxHistoryInfo)
		for _, result := range txHistories {
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
					if err := h.redis.Set(ctx, cacheKey, jsonData, cacheKeyExpiredTime).Err(); err != nil {
						log.Error("failed to set data to Redis", "error", err)
					}
				}
			} else {
				// tx hash not found, which is also a valid result, cache empty string.
				if err := h.redis.Set(ctx, cacheKey, "", cacheKeyExpiredTime).Err(); err != nil {
					log.Error("failed to set data to Redis", "error", err)
				}
			}
		}
	}
	return results, nil
}

func getTxHistoryInfo(message *orm.CrossMessage) *types.TxHistoryInfo {
	txHistory := &types.TxHistoryInfo{
		MsgHash:   message.MessageHash,
		Amount:    message.TokenAmounts,
		L1Token:   message.L1TokenAddress,
		L2Token:   message.L2TokenAddress,
		IsL1:      orm.MessageType(message.MessageType) == orm.MessageTypeL1SentMessage,
		TxStatus:  message.TxStatus,
		CreatedAt: &message.CreatedAt,
	}
	if txHistory.IsL1 {
		txHistory.Hash = message.L1TxHash
		txHistory.BlockNumber = message.L1BlockNumber
		txHistory.FinalizeTx = &types.Finalized{
			Hash:        message.L2TxHash,
			BlockNumber: message.L2BlockNumber,
		}
	} else {
		txHistory.Hash = message.L2TxHash
		txHistory.BlockNumber = message.L2BlockNumber
		txHistory.FinalizeTx = &types.Finalized{
			Hash:        message.L1TxHash,
			BlockNumber: message.L1BlockNumber,
		}
		if orm.RollupStatusType(message.RollupStatus) == orm.RollupStatusTypeFinalized {
			txHistory.ClaimInfo = &types.UserClaimInfo{
				From:       message.MessageFrom,
				To:         message.MessageTo,
				Value:      message.MessageValue,
				Nonce:      strconv.FormatUint(message.MessageNonce, 10),
				Message:    message.MessageData,
				Proof:      common.Bytes2Hex(message.MerkleProof),
				BatchIndex: strconv.FormatUint(message.BatchIndex, 10),
				Claimable:  true,
			}
		}
	}
	return txHistory
}

func (h *HistoryLogic) getCachedTxsInfo(ctx context.Context, cacheKey string, pageNum, pageSize uint64) ([]*types.TxHistoryInfo, uint64, bool, error) {
	start := int64((pageNum - 1) * pageSize)
	end := start + int64(pageSize)

	total, err := h.redis.ZCard(ctx, cacheKey).Result()
	if err != nil {
		log.Error("failed to get zcard result", "error", err)
		return nil, 0, false, err
	}

	if total == 0 {
		return nil, 0, false, nil
	}

	values, err := h.redis.ZRange(ctx, cacheKey, start, end).Result()
	if err != nil {
		log.Error("failed to get zrange result", "error", err)
		return nil, 0, false, err
	}

	if len(values) == 0 {
		return nil, 0, false, nil
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

func (h *HistoryLogic) cacheTxsInfo(ctx context.Context, cacheKey string, txs []*types.TxHistoryInfo) error {
	_, err := h.redis.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		// The transactions are sorted, thus we set the score as their indices.
		for i, tx := range txs {
			txBytes, err := json.Marshal(tx)
			if err != nil {
				log.Error("failed to marshal transaction to json", "error", err)
				return err
			}
			if err := pipe.ZAdd(ctx, cacheKey, &redis.Z{Score: float64(i), Member: txBytes}).Err(); err != nil {
				log.Error("failed to add transaction to sorted set", "error", err)
				return err
			}
		}
		if err := pipe.Expire(ctx, cacheKey, cacheKeyExpiredTime).Err(); err != nil {
			log.Error("failed to set expiry time", "error", err)
			return err
		}
		return nil
	})

	if err != nil {
		log.Error("failed to execute transaction", "error", err)
		return err
	}
	return nil
}

func (h *HistoryLogic) processAndCacheTxHistoryInfo(ctx context.Context, cacheKey string, messages []*orm.CrossMessage, page, pageSize uint64) ([]*types.TxHistoryInfo, uint64, error) {
	var txHistories []*types.TxHistoryInfo
	for _, message := range messages {
		txHistories = append(txHistories, getTxHistoryInfo(message))
	}

	err := h.cacheTxsInfo(ctx, cacheKey, txHistories)
	if err != nil {
		log.Error("failed to cache txs info", "key", cacheKey, "err", err)
		return nil, 0, err
	}

	pagedTxs, total, isHit, err := h.getCachedTxsInfo(ctx, cacheKey, page, pageSize)
	if err != nil {
		log.Error("failed to get cached tx info", "cached key", cacheKey, "page", page, "page size", pageSize, "error", err)
		return nil, 0, err
	}

	if !isHit {
		log.Error("cache miss after write, expect hit", "cached key", cacheKey, "page", page, "page size", pageSize, "error", err)
		return nil, 0, err
	}
	return pagedTxs, total, nil
}
