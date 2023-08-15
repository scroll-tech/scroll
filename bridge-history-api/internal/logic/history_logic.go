package logic

import (
	"context"
	"strconv"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"
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
func getCrossTxClaimInfo(ctx context.Context, msgHash string, db *gorm.DB) *types.UserClaimInfo {
	l2SentMsgOrm := orm.NewL2SentMsg(db)
	rollupOrm := orm.NewRollupBatch(db)
	l2sentMsg, err := l2SentMsgOrm.GetL2SentMsgByHash(ctx, msgHash)
	if err != nil || l2sentMsg == nil {
		log.Debug("getCrossTxClaimInfo failed", "error", err)
		return &types.UserClaimInfo{}
	}
	batch, err := rollupOrm.GetRollupBatchByIndex(ctx, l2sentMsg.BatchIndex)
	if err != nil {
		log.Debug("getCrossTxClaimInfo failed", "error", err)
		return &types.UserClaimInfo{}
	}
	return &types.UserClaimInfo{
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

func updateCrossTxHash(ctx context.Context, msgHash string, txInfo *types.TxHistoryInfo, db *gorm.DB) {
	relayed := orm.NewRelayedMsg(db)
	relayed, err := relayed.GetRelayedMsgByHash(ctx, msgHash)
	if err != nil {
		log.Debug("updateCrossTxHash failed", "error", err)
		return
	}
	if relayed == nil {
		return
	}
	if relayed.Layer1Hash != "" {
		txInfo.FinalizeTx.Hash = relayed.Layer1Hash
		txInfo.FinalizeTx.BlockNumber = relayed.Height
		return
	}
	if relayed.Layer2Hash != "" {
		txInfo.FinalizeTx.Hash = relayed.Layer2Hash
		txInfo.FinalizeTx.BlockNumber = relayed.Height
		return
	}

}

// GetClaimableTxsByAddress get all claimable txs under given address
func (h *HistoryLogic) GetClaimableTxsByAddress(ctx context.Context, address common.Address, offset int, limit int) ([]*types.TxHistoryInfo, uint64, error) {
	var txHistories []*types.TxHistoryInfo
	l2SentMsgOrm := orm.NewL2SentMsg(h.db)
	l2CrossMsgOrm := orm.NewCrossMsg(h.db)
	total, err := l2SentMsgOrm.GetClaimableL2SentMsgByAddressTotalNum(ctx, address.Hex())
	if err != nil || total == 0 {
		return txHistories, 0, err
	}
	results, err := l2SentMsgOrm.GetClaimableL2SentMsgByAddressWithOffset(ctx, address.Hex(), offset, limit)
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
			IsL1:        false,
			BlockNumber: result.Height,
			FinalizeTx:  &types.Finalized{},
			ClaimInfo:   getCrossTxClaimInfo(ctx, result.MsgHash, h.db),
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
	return txHistories, total, err
}

// GetTxsByAddress get all txs under given address
func (h *HistoryLogic) GetTxsByAddress(ctx context.Context, address common.Address, offset int, limit int) ([]*types.TxHistoryInfo, uint64, error) {
	var txHistories []*types.TxHistoryInfo
	utilOrm := orm.NewCrossMsg(h.db)
	total, err := utilOrm.GetTotalCrossMsgCountByAddress(ctx, address.String())
	if err != nil || total == 0 {
		return txHistories, 0, err
	}
	result, err := utilOrm.GetCrossMsgsByAddressWithOffset(ctx, address.String(), offset, limit)

	if err != nil {
		return nil, 0, err
	}
	for _, msg := range result {
		txHistory := &types.TxHistoryInfo{
			Hash:           msg.Layer1Hash + msg.Layer2Hash,
			Amount:         msg.Amount,
			To:             msg.Target,
			L1Token:        msg.Layer1Token,
			L2Token:        msg.Layer2Token,
			IsL1:           msg.MsgType == int(orm.Layer1Msg),
			BlockNumber:    msg.Height,
			BlockTimestamp: msg.Timestamp,
			CreatedAt:      msg.CreatedAt,
			FinalizeTx: &types.Finalized{
				Hash: "",
			},
			ClaimInfo: getCrossTxClaimInfo(ctx, msg.MsgHash, h.db),
		}
		updateCrossTxHash(ctx, msg.MsgHash, txHistory, h.db)
		txHistories = append(txHistories, txHistory)
	}
	return txHistories, total, nil
}

// GetTxsByHashes get tx infos under given tx hashes
func (h *HistoryLogic) GetTxsByHashes(ctx context.Context, hashes []string) ([]*types.TxHistoryInfo, error) {
	txHistories := make([]*types.TxHistoryInfo, 0)
	CrossMsgOrm := orm.NewCrossMsg(h.db)
	for _, hash := range hashes {
		l1result, err := CrossMsgOrm.GetL1CrossMsgByHash(ctx, common.HexToHash(hash))
		if err != nil {
			return nil, err
		}
		if l1result != nil {
			txHistory := &types.TxHistoryInfo{
				Hash:           l1result.Layer1Hash,
				Amount:         l1result.Amount,
				To:             l1result.Target,
				IsL1:           true,
				L1Token:        l1result.Layer1Token,
				L2Token:        l1result.Layer2Token,
				BlockNumber:    l1result.Height,
				BlockTimestamp: l1result.Timestamp,
				CreatedAt:      l1result.CreatedAt,
				FinalizeTx: &types.Finalized{
					Hash: "",
				},
			}
			updateCrossTxHash(ctx, l1result.MsgHash, txHistory, h.db)
			txHistories = append(txHistories, txHistory)
			continue
		}
		l2result, err := CrossMsgOrm.GetL2CrossMsgByHash(ctx, common.HexToHash(hash))
		if err != nil {
			return nil, err
		}
		if l2result != nil {
			txHistory := &types.TxHistoryInfo{
				Hash:           l2result.Layer2Hash,
				Amount:         l2result.Amount,
				To:             l2result.Target,
				IsL1:           false,
				L1Token:        l2result.Layer1Token,
				L2Token:        l2result.Layer2Token,
				BlockNumber:    l2result.Height,
				BlockTimestamp: l2result.Timestamp,
				CreatedAt:      l2result.CreatedAt,
				FinalizeTx: &types.Finalized{
					Hash: "",
				},
				ClaimInfo: getCrossTxClaimInfo(ctx, l2result.MsgHash, h.db),
			}
			updateCrossTxHash(ctx, l2result.MsgHash, txHistory, h.db)
			txHistories = append(txHistories, txHistory)
			continue
		}
	}
	return txHistories, nil
}
