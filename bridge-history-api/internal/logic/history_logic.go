package logic

import (
	"context"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"gorm.io/gorm"

	"bridge-history-api/internal/types"
	"bridge-history-api/orm"
)

// HistoryLogic example service.
type HistoryLogic struct {
	db *gorm.DB
}

// NewHistoryLogic returns services backed with a "db"
func NewHistoryLogic(db *gorm.DB) *HistoryLogic {
	logic := &HistoryLogic{db: db}
	return logic
}

// getCrossTxClaimInfo get UserClaimInfos by address
//func getCrossTxClaimInfo(ctx context.Context, msgHash string, db *gorm.DB) *types.UserClaimInfo {
//	l2SentMsgOrm := orm.NewL2SentMsg(db)
//	rollupOrm := orm.NewRollupBatch(db)
//	l2sentMsg, err := l2SentMsgOrm.GetL2SentMsgByHash(ctx, msgHash)
//	if err != nil || l2sentMsg == nil {
//		log.Debug("getCrossTxClaimInfo failed", "error", err)
//		return &types.UserClaimInfo{}
//	}
//	batch, err := rollupOrm.GetRollupBatchByIndex(ctx, l2sentMsg.BatchIndex)
//	if err != nil {
//		log.Debug("getCrossTxClaimInfo failed", "error", err)
//		return &types.UserClaimInfo{}
//	}
//	return &types.UserClaimInfo{
//		From:       l2sentMsg.Sender,
//		To:         l2sentMsg.Target,
//		Value:      l2sentMsg.Value,
//		Nonce:      strconv.FormatUint(l2sentMsg.Nonce, 10),
//		Message:    l2sentMsg.MsgData,
//		Proof:      "0x" + l2sentMsg.MsgProof,
//		BatchHash:  batch.BatchHash,
//		BatchIndex: strconv.FormatUint(l2sentMsg.BatchIndex, 10),
//	}
//}

// updateL2TxClaimInfo updates UserClaimInfos for each transaction history.
func updateL2TxClaimInfo(ctx context.Context, txHistories []*types.TxHistoryInfo, db *gorm.DB) {
	l2SentMsgOrm := orm.NewL2SentMsg(db)
	rollupOrm := orm.NewRollupBatch(db)

	var l2MsgHashes []string
	for _, txHistory := range txHistories {
		if !txHistory.IsL1 {
			l2MsgHashes = append(l2MsgHashes, txHistory.MsgHash)
		}
	}

	l2sentMsgs, err := l2SentMsgOrm.GetL2SentMsgsByHashes(ctx, l2MsgHashes)
	if err != nil {
		log.Debug("GetL2SentMsgsByHashes failed", "error", err)
		return
	}

	var batchIndexes []uint64
	for _, l2sentMsg := range l2sentMsgs {
		batchIndexes = append(batchIndexes, l2sentMsg.BatchIndex)
	}

	batches, err := rollupOrm.GetRollupBatchesByIndexes(ctx, batchIndexes)
	if err != nil {
		log.Debug("GetRollupBatchesByIndexes failed", "error", err)
		return
	}

	for _, txHistory := range txHistories {
		if txHistory.IsL1 {
			continue
		}

		var l2sentMsg *orm.L2SentMsg
		var batch *orm.RollupBatch
		for _, msg := range l2sentMsgs {
			if msg.MsgHash == txHistory.MsgHash {
				l2sentMsg = msg
				break
			}
		}

		if l2sentMsg != nil {
			for _, b := range batches {
				if b.BatchIndex == l2sentMsg.BatchIndex {
					batch = b
					break
				}
			}
		}

		if l2sentMsg != nil && batch != nil {
			txHistory.ClaimInfo = &types.UserClaimInfo{
				From:       l2sentMsg.Sender,
				To:         l2sentMsg.Target,
				Value:      l2sentMsg.Value,
				Nonce:      strconv.FormatUint(l2sentMsg.Nonce, 10),
				Message:    l2sentMsg.MsgData,
				Proof:      "0x" + l2sentMsg.MsgProof,
				BatchHash:  batch.BatchHash,
				BatchIndex: strconv.FormatUint(l2sentMsg.BatchIndex, 10),
			}
		}
	}
}

func updateCrossTxHashes(ctx context.Context, txHistories []*types.TxHistoryInfo, db *gorm.DB) {
	msgHashes := make([]string, len(txHistories))
	for i, txHistory := range txHistories {
		msgHashes[i] = txHistory.MsgHash
	}

	relayed := orm.NewRelayedMsg(db)
	relayedMsgs, err := relayed.GetRelayedMsgsByHashes(ctx, msgHashes)
	if err != nil {
		log.Debug("GetRelayedMsgsByHashes failed", "msg hashes", msgHashes, "error", err)
		return
	}

	for _, relayedMsg := range relayedMsgs {
		for _, txHistory := range txHistories {
			if txHistory.MsgHash == relayedMsg.MsgHash {
				if relayedMsg.Layer1Hash != "" {
					txHistory.FinalizeTx.Hash = relayedMsg.Layer1Hash
					txHistory.FinalizeTx.BlockNumber = relayedMsg.Height
				}
				if relayedMsg.Layer2Hash != "" {
					txHistory.FinalizeTx.Hash = relayedMsg.Layer2Hash
					txHistory.FinalizeTx.BlockNumber = relayedMsg.Height
				}
			}
		}
	}
}

func updateCrossTxHashesAndL2TxClaimInfo(ctx context.Context, txHistories []*types.TxHistoryInfo, db *gorm.DB) {
	updateCrossTxHashes(ctx, txHistories, db)
	updateL2TxClaimInfo(ctx, txHistories, db)
}

// GetClaimableTxsByAddress get all claimable txs under given address
func (h *HistoryLogic) GetClaimableTxsByAddress(ctx context.Context, address common.Address) ([]*types.TxHistoryInfo, uint64, error) {
	var txHistories []*types.TxHistoryInfo
	l2SentMsgOrm := orm.NewL2SentMsg(h.db)
	l2CrossMsgOrm := orm.NewCrossMsg(h.db)
	results, err := l2SentMsgOrm.GetClaimableL2SentMsgByAddress(ctx, address.Hex())
	if err != nil || len(results) == 0 {
		return txHistories, 0, err
	}
	var msgHashList []string
	for _, result := range results {
		msgHashList = append(msgHashList, result.MsgHash)
	}
	crossMsgs, err := l2CrossMsgOrm.GetL2CrossMsgByMsgHashList(ctx, msgHashList)
	// crossMsgs can be empty, because they can be emitted by user directly call contract
	if err != nil {
		return txHistories, 0, err
	}
	crossMsgMap := make(map[string]*orm.CrossMsg)
	for _, crossMsg := range crossMsgs {
		crossMsgMap[crossMsg.MsgHash] = crossMsg
	}
	for _, result := range results {
		txInfo := &types.TxHistoryInfo{
			Hash:        result.TxHash,
			MsgHash:     result.MsgHash,
			IsL1:        false,
			BlockNumber: result.Height,
			FinalizeTx:  &types.Finalized{},
		}
		if crossMsg, exist := crossMsgMap[result.MsgHash]; exist {
			txInfo.Amount = crossMsg.Amount
			txInfo.To = crossMsg.Target
			txInfo.BlockTimestamp = crossMsg.Timestamp
			txInfo.CreatedAt = crossMsg.CreatedAt
			txInfo.L1Token = crossMsg.Layer1Token
			txInfo.L2Token = crossMsg.Layer2Token
		}
		txHistories = append(txHistories, txInfo)
	}
	updateL2TxClaimInfo(ctx, txHistories, h.db)
	return txHistories, uint64(len(results)), err
}

// GetTxsByAddress get all txs under given address
//func (h *HistoryLogic) GetTxsByAddress(ctx context.Context, address common.Address, offset int, limit int) ([]*types.TxHistoryInfo, uint64, error) {
// 	var txHistories []*types.TxHistoryInfo
//	utilOrm := orm.NewCrossMsg(h.db)
//	total, err := utilOrm.GetTotalCrossMsgCountByAddress(ctx, address.String())
//	if err != nil || total == 0 {
//		return txHistories, 0, err
//	}
//	result, err := utilOrm.GetCrossMsgsByAddressWithOffset(ctx, address.String(), offset, limit)
//
//	if err != nil {
//		return nil, 0, err
//	}
//	for _, msg := range result {
//		txHistory := &types.TxHistoryInfo{
//			Hash:           msg.Layer1Hash + msg.Layer2Hash,
//			Amount:         msg.Amount,
//			To:             msg.Target,
//			L1Token:        msg.Layer1Token,
//			L2Token:        msg.Layer2Token,
//			IsL1:           msg.MsgType == int(orm.Layer1Msg),
//			BlockNumber:    msg.Height,
//			BlockTimestamp: msg.Timestamp,
//			CreatedAt:      msg.CreatedAt,
//			FinalizeTx: &types.Finalized{
//				Hash: "",
//			},
//			ClaimInfo: getCrossTxClaimInfo(ctx, msg.MsgHash, h.db),
//		}
//		updateCrossTxHash(ctx, msg.MsgHash, txHistory, h.db)
//		txHistories = append(txHistories, txHistory)
//	}
//	return txHistories, total, nil
//}

// GetTxsByHashes get tx infos under given tx hashes
func (h *HistoryLogic) GetTxsByHashes(ctx context.Context, hashes []string) ([]*types.TxHistoryInfo, error) {
	CrossMsgOrm := orm.NewCrossMsg(h.db)

	results, err := CrossMsgOrm.GetCrossMsgsByHashes(ctx, hashes)
	if err != nil {
		return nil, err
	}

	var txHistories []*types.TxHistoryInfo
	for _, result := range results {
		txHistory := &types.TxHistoryInfo{
			Hash:           result.Layer1Hash + result.Layer2Hash,
			MsgHash:        result.MsgHash,
			Amount:         result.Amount,
			To:             result.Target,
			IsL1:           orm.MsgType(result.MsgType) == orm.Layer1Msg,
			L1Token:        result.Layer1Token,
			L2Token:        result.Layer2Token,
			BlockNumber:    result.Height,
			BlockTimestamp: result.Timestamp,
			CreatedAt:      result.CreatedAt,
			FinalizeTx:     &types.Finalized{Hash: ""},
		}
		txHistories = append(txHistories, txHistory)
	}

	updateCrossTxHashesAndL2TxClaimInfo(ctx, txHistories, h.db)
	return txHistories, nil
}
