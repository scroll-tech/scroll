package logic

import (
	"context"
	"strconv"

	"github.com/scroll-tech/go-ethereum/common"
	"gorm.io/gorm"

	"scroll-tech/bridge-history-api/internal/orm"
	"scroll-tech/bridge-history-api/internal/types"
)

// HistoryLogic services.
type HistoryLogic struct {
	crossMessageOrm *orm.CrossMessage
	batchEventOrm   *orm.BatchEvent
}

// NewHistoryLogic returns bridge history services.
func NewHistoryLogic(db *gorm.DB) *HistoryLogic {
	logic := &HistoryLogic{
		crossMessageOrm: orm.NewCrossMessage(db),
		batchEventOrm:   orm.NewBatchEvent(db),
	}
	return logic
}

// GetL2ClaimableWithdrawalsByAddress gets all claimable withdrawal txs under given address.
func (h *HistoryLogic) GetL2ClaimableWithdrawalsByAddress(ctx context.Context, address string) ([]*types.TxHistoryInfo, error) {
	messages, err := h.crossMessageOrm.GetL2ClaimableWithdrawalsByAddress(ctx, address)
	if err != nil {
		return nil, err
	}
	var txHistories []*types.TxHistoryInfo
	for _, message := range messages {
		txHistories = append(txHistories, getTxHistoryInfo(message))
	}
	return txHistories, err
}

// GetL2WithdrawalsByAddress gets all withdrawal txs under given address.
func (h *HistoryLogic) GetL2WithdrawalsByAddress(ctx context.Context, address string) ([]*types.TxHistoryInfo, error) {
	messages, err := h.crossMessageOrm.GetL2WithdrawalsByAddress(ctx, address)
	if err != nil {
		return nil, err
	}
	var txHistories []*types.TxHistoryInfo
	for _, message := range messages {
		txHistories = append(txHistories, getTxHistoryInfo(message))
	}
	return txHistories, err
}

// GetTxsByAddress gets tx infos under given address.
func (h *HistoryLogic) GetTxsByAddress(ctx context.Context, address string) ([]*types.TxHistoryInfo, error) {
	messages, err := h.crossMessageOrm.GetTxsByAddress(ctx, address)
	if err != nil {
		return nil, err
	}
	var txHistories []*types.TxHistoryInfo
	for _, message := range messages {
		txHistories = append(txHistories, getTxHistoryInfo(message))
	}
	return txHistories, err
}

// GetTxsByHashes gets tx infos under given tx hashes.
func (h *HistoryLogic) GetTxsByHashes(ctx context.Context, txHashes []string) ([]*types.TxHistoryInfo, error) {
	messages, err := h.crossMessageOrm.GetMessagesByTxHashes(ctx, txHashes)
	if err != nil {
		return nil, err
	}
	var txHistories []*types.TxHistoryInfo
	for _, message := range messages {
		txHistories = append(txHistories, getTxHistoryInfo(message))
	}
	return txHistories, nil
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
			}
		}
	}
	return txHistory
}
