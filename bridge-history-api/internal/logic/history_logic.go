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
	btypes "scroll-tech/bridge-history-api/internal/types"
	"scroll-tech/bridge-history-api/internal/utils"
)

const (
	// cacheKeyPrefixBridgeHistory serves as a specific namespace for all Redis cache keys
	// associated with the 'bridge-history' user. This prefix is used to enforce access controls
	// in Redis, allowing permissions to be set such that only users with the appropriate
	// access rights can read or write to keys starting with "bridge-history".
	cacheKeyPrefixBridgeHistory = "bridge-history-"

	cacheKeyPrefixL2ClaimableWithdrawalsByAddr = cacheKeyPrefixBridgeHistory + "l2ClaimableWithdrawalsByAddr:"
	cacheKeyPrefixL2WithdrawalsByAddr          = cacheKeyPrefixBridgeHistory + "l2WithdrawalsByAddr:"
	cacheKeyPrefixTxsByAddr                    = cacheKeyPrefixBridgeHistory + "txsByAddr:"
	cacheKeyPrefixQueryTxsByHashes             = cacheKeyPrefixBridgeHistory + "queryTxsByHashes:"
	cacheKeyExpiredTime                        = 1 * time.Minute
)

// HistoryLogic services.
type HistoryLogic struct {
	crossMessageOrm       *orm.CrossMessage
	batchEventOrm         *orm.BatchEvent
	bridgeBatchDepositOrm *orm.BridgeBatchDepositEvent

	redis        *redis.Client
	singleFlight singleflight.Group
	cacheMetrics *cacheMetrics
}

// NewHistoryLogic returns bridge history services.
func NewHistoryLogic(db *gorm.DB, redis *redis.Client) *HistoryLogic {
	logic := &HistoryLogic{
		crossMessageOrm:       orm.NewCrossMessage(db),
		batchEventOrm:         orm.NewBatchEvent(db),
		bridgeBatchDepositOrm: orm.NewBridgeBatchDepositEvent(db),
		redis:                 redis,
		cacheMetrics:          initCacheMetrics(),
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
		var txHistoryInfos []*types.TxHistoryInfo
		crossMessages, getErr := h.crossMessageOrm.GetL2UnclaimedWithdrawalsByAddress(ctx, address)
		if getErr != nil {
			return nil, getErr
		}
		for _, message := range crossMessages {
			txHistoryInfos = append(txHistoryInfos, getTxHistoryInfoFromCrossMessage(message))
		}
		return txHistoryInfos, nil
	})
	if err != nil {
		log.Error("failed to get L2 claimable withdrawals by address", "address", address, "error", err)
		return nil, 0, err
	}

	txHistoryInfos, ok := result.([]*types.TxHistoryInfo)
	if !ok {
		log.Error("unexpected type", "expected", "[]*types.TxHistoryInfo", "got", reflect.TypeOf(result), "address", address)
		return nil, 0, errors.New("unexpected error")
	}

	return h.processAndCacheTxHistoryInfo(ctx, cacheKey, txHistoryInfos, page, pageSize)
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
		var txHistoryInfos []*types.TxHistoryInfo
		crossMessages, getErr := h.crossMessageOrm.GetL2WithdrawalsByAddress(ctx, address)
		if getErr != nil {
			return nil, getErr
		}
		for _, message := range crossMessages {
			txHistoryInfos = append(txHistoryInfos, getTxHistoryInfoFromCrossMessage(message))
		}
		return txHistoryInfos, nil
	})
	if err != nil {
		log.Error("failed to get L2 withdrawals by address", "address", address, "error", err)
		return nil, 0, err
	}

	txHistoryInfos, ok := result.([]*types.TxHistoryInfo)
	if !ok {
		log.Error("unexpected type", "expected", "[]*types.TxHistoryInfo", "got", reflect.TypeOf(result), "address", address)
		return nil, 0, errors.New("unexpected error")
	}

	return h.processAndCacheTxHistoryInfo(ctx, cacheKey, txHistoryInfos, page, pageSize)
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
		var txHistoryInfos []*types.TxHistoryInfo
		crossMessages, getErr := h.crossMessageOrm.GetTxsByAddress(ctx, address)
		if getErr != nil {
			return nil, getErr
		}
		for _, message := range crossMessages {
			txHistoryInfos = append(txHistoryInfos, getTxHistoryInfoFromCrossMessage(message))
		}

		batchDepositMessages, getErr := h.bridgeBatchDepositOrm.GetTxsByAddress(ctx, address)
		if getErr != nil {
			return nil, getErr
		}
		for _, message := range batchDepositMessages {
			txHistoryInfos = append(txHistoryInfos, getTxHistoryInfoFromBridgeBatchDepositMessage(message))
		}
		return txHistoryInfos, nil
	})
	if err != nil {
		log.Error("failed to get txs by address", "address", address, "error", err)
		return nil, 0, err
	}

	txHistoryInfos, ok := result.([]*types.TxHistoryInfo)
	if !ok {
		log.Error("unexpected type", "expected", "[]*types.TxHistoryInfo", "got", reflect.TypeOf(result), "address", address)
		return nil, 0, errors.New("unexpected error")
	}

	return h.processAndCacheTxHistoryInfo(ctx, cacheKey, txHistoryInfos, page, pageSize)
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
		if err != nil && errors.Is(err, redis.Nil) {
			h.cacheMetrics.cacheMisses.WithLabelValues("PostQueryTxsByHashes").Inc()
			log.Info("cache miss", "cache key", cacheKey)
			uncachedHashes = append(uncachedHashes, hash)
			continue
		}

		if err != nil {
			log.Error("failed to get data from Redis", "error", err)
			uncachedHashes = append(uncachedHashes, hash)
			continue
		}

		h.cacheMetrics.cacheHits.WithLabelValues("PostQueryTxsByHashes").Inc()
		log.Info("cache hit", "cache key", cacheKey)

		if len(cachedData) == 0 {
			continue
		}

		var txInfo types.TxHistoryInfo
		if unmarshalErr := json.Unmarshal(cachedData, &txInfo); unmarshalErr != nil {
			log.Error("failed to unmarshal cached data", "error", unmarshalErr)
			uncachedHashes = append(uncachedHashes, hash)
			continue
		}
		results = append(results, &txInfo)
	}

	if len(uncachedHashes) > 0 {
		var txHistories []*types.TxHistoryInfo

		crossMessages, err := h.crossMessageOrm.GetMessagesByTxHashes(ctx, uncachedHashes)
		if err != nil {
			log.Error("failed to get cross messages by tx hashes", "hashes", uncachedHashes)
			return nil, err
		}
		for _, message := range crossMessages {
			txHistories = append(txHistories, getTxHistoryInfoFromCrossMessage(message))
		}

		batchDepositMessages, err := h.bridgeBatchDepositOrm.GetMessagesByTxHashes(ctx, uncachedHashes)
		if err != nil {
			log.Error("failed to get batch deposit messages by tx hashes", "hashes", uncachedHashes)
			return nil, err
		}
		for _, message := range batchDepositMessages {
			txHistories = append(txHistories, getTxHistoryInfoFromBridgeBatchDepositMessage(message))
		}

		resultMap := make(map[string]*types.TxHistoryInfo)
		for _, result := range txHistories {
			results = append(results, result)
			resultMap[result.Hash] = result
		}

		for _, hash := range uncachedHashes {
			cacheKey := cacheKeyPrefixQueryTxsByHashes + hash
			result, found := resultMap[hash]
			if !found {
				// tx hash not found, which is also a valid result, cache empty string.
				if cacheErr := h.redis.Set(ctx, cacheKey, "", cacheKeyExpiredTime).Err(); cacheErr != nil {
					log.Error("failed to set data to Redis", "error", cacheErr)
				}
				continue
			}

			jsonData, unmarshalErr := json.Marshal(result)
			if unmarshalErr != nil {
				log.Error("failed to marshal data", "error", unmarshalErr)
				continue
			}

			if cacheErr := h.redis.Set(ctx, cacheKey, jsonData, cacheKeyExpiredTime).Err(); cacheErr != nil {
				log.Error("failed to set data to Redis", "error", cacheErr)
			}
		}
	}
	return results, nil
}

func getTxHistoryInfoFromCrossMessage(message *orm.CrossMessage) *types.TxHistoryInfo {
	txHistory := &types.TxHistoryInfo{
		MessageHash:    message.MessageHash,
		TokenType:      btypes.TokenType(message.TokenType),
		TokenIDs:       utils.ConvertStringToStringArray(message.TokenIDs),
		TokenAmounts:   utils.ConvertStringToStringArray(message.TokenAmounts),
		L1TokenAddress: message.L1TokenAddress,
		L2TokenAddress: message.L2TokenAddress,
		MessageType:    btypes.MessageType(message.MessageType),
		TxStatus:       btypes.TxStatusType(message.TxStatus),
		BlockTimestamp: message.BlockTimestamp,
	}
	if txHistory.MessageType == btypes.MessageTypeL1SentMessage {
		txHistory.Hash = message.L1TxHash
		txHistory.ReplayTxHash = message.L1ReplayTxHash
		txHistory.RefundTxHash = message.L1RefundTxHash
		txHistory.BlockNumber = message.L1BlockNumber
		txHistory.CounterpartChainTx = &types.CounterpartChainTx{
			Hash:        message.L2TxHash,
			BlockNumber: message.L2BlockNumber,
		}
	} else {
		txHistory.Hash = message.L2TxHash
		txHistory.BlockNumber = message.L2BlockNumber
		txHistory.CounterpartChainTx = &types.CounterpartChainTx{
			Hash:        message.L1TxHash,
			BlockNumber: message.L1BlockNumber,
		}
		if btypes.RollupStatusType(message.RollupStatus) == btypes.RollupStatusTypeFinalized {
			txHistory.ClaimInfo = &types.ClaimInfo{
				From:    message.MessageFrom,
				To:      message.MessageTo,
				Value:   message.MessageValue,
				Nonce:   strconv.FormatUint(message.MessageNonce, 10),
				Message: message.MessageData,
				Proof: types.L2MessageProof{
					BatchIndex:  strconv.FormatUint(message.BatchIndex, 10),
					MerkleProof: "0x" + common.Bytes2Hex(message.MerkleProof),
				},
				Claimable: true,
			}
		}
	}
	return txHistory
}

func getTxHistoryInfoFromBridgeBatchDepositMessage(message *orm.BridgeBatchDepositEvent) *types.TxHistoryInfo {
	txHistory := &types.TxHistoryInfo{
		Hash:         message.L1TxHash,
		TokenType:    btypes.TokenType(message.TokenType),
		TokenAmounts: utils.ConvertStringToStringArray(message.TokenAmount),
		BlockNumber:  message.L1BlockNumber,
		MessageType:  btypes.MessageTypeL1BatchDeposit,
		TxStatus:     btypes.TxStatusType(message.TxStatus),
		CounterpartChainTx: &types.CounterpartChainTx{
			Hash:        message.L2TxHash,
			BlockNumber: message.L2BlockNumber,
		},
		BlockTimestamp:  message.BlockTimestamp,
		BatchDepositFee: message.Fee,
	}
	if txHistory.TokenType != btypes.TokenTypeETH {
		txHistory.L1TokenAddress = message.L1TokenAddress
		txHistory.L2TokenAddress = message.L2TokenAddress
	}
	return txHistory
}

func (h *HistoryLogic) getCachedTxsInfo(ctx context.Context, cacheKey string, pageNum, pageSize uint64) ([]*types.TxHistoryInfo, uint64, bool, error) {
	start := int64((pageNum - 1) * pageSize)
	end := start + int64(pageSize) - 1

	total, err := h.redis.ZCard(ctx, cacheKey).Result()
	if err != nil {
		log.Error("failed to get zcard result", "error", err)
		return nil, 0, false, err
	}

	if total == 0 {
		return nil, 0, false, nil
	}

	values, err := h.redis.ZRevRange(ctx, cacheKey, start, end).Result()
	if err != nil {
		log.Error("failed to get zrange result", "error", err)
		return nil, 0, false, err
	}

	if len(values) == 0 {
		return nil, 0, false, nil
	}

	// check if it's empty placeholder.
	if len(values) == 1 && values[0] == "empty_page" {
		return nil, 0, true, nil
	}

	var pagedTxs []*types.TxHistoryInfo
	for _, v := range values {
		var tx types.TxHistoryInfo
		if unmarshalErr := json.Unmarshal([]byte(v), &tx); unmarshalErr != nil {
			log.Error("failed to unmarshal transaction data", "error", unmarshalErr)
			return nil, 0, false, unmarshalErr
		}
		pagedTxs = append(pagedTxs, &tx)
	}
	return pagedTxs, uint64(total), true, nil
}

func (h *HistoryLogic) cacheTxsInfo(ctx context.Context, cacheKey string, txs []*types.TxHistoryInfo) error {
	_, err := h.redis.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		if len(txs) == 0 {
			if err := pipe.ZAdd(ctx, cacheKey, &redis.Z{Score: 0, Member: "empty_page"}).Err(); err != nil {
				log.Error("failed to add empty page indicator to sorted set", "error", err)
				return err
			}
		} else {
			// The transactions are sorted, thus we set the score as their index.
			for _, tx := range txs {
				txBytes, err := json.Marshal(tx)
				if err != nil {
					log.Error("failed to marshal transaction to json", "error", err)
					return err
				}
				if err := pipe.ZAdd(ctx, cacheKey, &redis.Z{Score: float64(tx.BlockTimestamp), Member: txBytes}).Err(); err != nil {
					log.Error("failed to add transaction to sorted set", "error", err)
					return err
				}
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

func (h *HistoryLogic) processAndCacheTxHistoryInfo(ctx context.Context, cacheKey string, txHistories []*types.TxHistoryInfo, page, pageSize uint64) ([]*types.TxHistoryInfo, uint64, error) {
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
