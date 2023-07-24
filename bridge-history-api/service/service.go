package service

import (
	"context"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"gorm.io/gorm"

	"bridge-history-api/orm"
)

// Finalized the schema of tx finalized infos
type Finalized struct {
	Hash           string     `json:"hash"`
	Amount         string     `json:"amount"`
	To             string     `json:"to"` // useless
	IsL1           bool       `json:"isL1"`
	BlockNumber    uint64     `json:"blockNumber"`
	BlockTimestamp *time.Time `json:"blockTimestamp"` // uselesss
}

// UserClaimInfo the schema of tx claim infos
type UserClaimInfo struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Value      string `json:"value"`
	Nonce      string `json:"nonce"`
	BatchHash  string `json:"batch_hash"`
	Message    string `json:"message"`
	Proof      string `json:"proof"`
	BatchIndex string `json:"batch_index"`
}

// TxHistoryInfo the schema of tx history infos
type TxHistoryInfo struct {
	Hash           string         `json:"hash"`
	Amount         string         `json:"amount"`
	To             string         `json:"to"` // useless
	IsL1           bool           `json:"isL1"`
	BlockNumber    uint64         `json:"blockNumber"`
	BlockTimestamp *time.Time     `json:"blockTimestamp"` // useless
	FinalizeTx     *Finalized     `json:"finalizeTx"`
	ClaimInfo      *UserClaimInfo `json:"claimInfo"`
	CreatedAt      *time.Time     `json:"createdTime"`
}

// HistoryService example service.
type HistoryService interface {
	GetTxsByAddress(address common.Address, offset int, limit int) ([]*TxHistoryInfo, uint64, error)
	GetTxsByHashes(hashes []string) ([]*TxHistoryInfo, error)
	GetClaimableTxsByAddress(address common.Address, offset int, limit int) ([]*TxHistoryInfo, uint64, error)
}

// NewHistoryService returns a service backed with a "db"
func NewHistoryService(ctx context.Context, db *gorm.DB) HistoryService {
	service := &historyBackend{ctx: ctx, db: db, prefix: "Scroll-Bridge-History-Server"}
	return service
}

type historyBackend struct {
	prefix string
	ctx    context.Context
	db     *gorm.DB
}

// GetCrossTxClaimInfo get UserClaimInfos by address
func GetCrossTxClaimInfo(ctx context.Context, msgHash string, db *gorm.DB) *UserClaimInfo {
	l2SentMsgOrm := orm.NewL2SentMsg(db)
	rollupOrm := orm.NewRollupBatch(db)
	l2sentMsg, err := l2SentMsgOrm.GetL2SentMsgByHash(ctx, msgHash)
	if err != nil || l2sentMsg == nil {
		log.Debug("GetCrossTxClaimInfo failed", "error", err)
		return &UserClaimInfo{}
	}
	batch, err := rollupOrm.GetRollupBatchByIndex(ctx, l2sentMsg.BatchIndex)
	if err != nil {
		log.Debug("GetCrossTxClaimInfo failed", "error", err)
		return &UserClaimInfo{}
	}
	return &UserClaimInfo{
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

func updateCrossTxHash(ctx context.Context, msgHash string, txInfo *TxHistoryInfo, db *gorm.DB) {
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
func (h *historyBackend) GetClaimableTxsByAddress(address common.Address, offset int, limit int) ([]*TxHistoryInfo, uint64, error) {
	var txHistories []*TxHistoryInfo
	l2SentMsgOrm := orm.NewL2SentMsg(h.db)
	l2CrossMsgOrm := orm.NewCrossMsg(h.db)
	total, err := l2SentMsgOrm.GetClaimableL2SentMsgByAddressTotalNum(h.ctx, address.Hex())
	if err != nil || total == 0 {
		return txHistories, 0, err
	}
	results, err := l2SentMsgOrm.GetClaimableL2SentMsgByAddressWithOffset(h.ctx, address.Hex(), offset, limit)
	if err != nil || len(results) == 0 {
		return txHistories, 0, err
	}
	var msgHashList []string
	for _, result := range results {
		msgHashList = append(msgHashList, result.MsgHash)
	}
	crossMsgs, err := l2CrossMsgOrm.GetL2CrossMsgByMsgHashList(h.ctx, msgHashList)
	// crossMsgs can be empty, because they can be emitted by user directly call contract
	if err != nil {
		return txHistories, 0, err
	}
	crossMsgMap := make(map[string]*orm.CrossMsg)
	for _, crossMsg := range crossMsgs {
		crossMsgMap[crossMsg.MsgHash] = crossMsg
	}
	for _, result := range results {
		txInfo := &TxHistoryInfo{
			Hash:        result.TxHash,
			IsL1:        false,
			BlockNumber: result.Height,
			FinalizeTx:  &Finalized{},
			ClaimInfo:   GetCrossTxClaimInfo(h.ctx, result.MsgHash, h.db),
		}
		if crossMsg, exist := crossMsgMap[result.MsgHash]; exist {
			txInfo.Amount = crossMsg.Amount
			txInfo.To = crossMsg.Target
			txInfo.BlockTimestamp = crossMsg.Timestamp
			txInfo.CreatedAt = crossMsg.CreatedAt
		}
		txHistories = append(txHistories, txInfo)
	}
	return txHistories, total, err
}

// GetTxsByAddress get all txs under given address
func (h *historyBackend) GetTxsByAddress(address common.Address, offset int, limit int) ([]*TxHistoryInfo, uint64, error) {
	var txHistories []*TxHistoryInfo
	utilOrm := orm.NewCrossMsg(h.db)
	total, err := utilOrm.GetTotalCrossMsgCountByAddress(h.ctx, address.String())
	if err != nil || total == 0 {
		return txHistories, 0, err
	}
	result, err := utilOrm.GetCrossMsgsByAddressWithOffset(h.ctx, address.String(), offset, limit)

	if err != nil {
		return nil, 0, err
	}
	for _, msg := range result {
		txHistory := &TxHistoryInfo{
			Hash:           msg.Layer1Hash + msg.Layer2Hash,
			Amount:         msg.Amount,
			To:             msg.Target,
			IsL1:           msg.MsgType == int(orm.Layer1Msg),
			BlockNumber:    msg.Height,
			BlockTimestamp: msg.Timestamp,
			CreatedAt:      msg.CreatedAt,
			FinalizeTx: &Finalized{
				Hash: "",
			},
			ClaimInfo: GetCrossTxClaimInfo(h.ctx, msg.MsgHash, h.db),
		}
		updateCrossTxHash(h.ctx, msg.MsgHash, txHistory, h.db)
		txHistories = append(txHistories, txHistory)
	}
	return txHistories, total, nil
}

// GetTxsByHashes get tx infos under given tx hashes
func (h *historyBackend) GetTxsByHashes(hashes []string) ([]*TxHistoryInfo, error) {
	txHistories := make([]*TxHistoryInfo, 0)
	CrossMsgOrm := orm.NewCrossMsg(h.db)
	for _, hash := range hashes {
		l1result, err := CrossMsgOrm.GetL1CrossMsgByHash(h.ctx, common.HexToHash(hash))
		if err != nil {
			return nil, err
		}
		if l1result != nil {
			txHistory := &TxHistoryInfo{
				Hash:           l1result.Layer1Hash,
				Amount:         l1result.Amount,
				To:             l1result.Target,
				IsL1:           true,
				BlockNumber:    l1result.Height,
				BlockTimestamp: l1result.Timestamp,
				CreatedAt:      l1result.CreatedAt,
				FinalizeTx: &Finalized{
					Hash: "",
				},
			}
			updateCrossTxHash(h.ctx, l1result.MsgHash, txHistory, h.db)
			txHistories = append(txHistories, txHistory)
			continue
		}
		l2result, err := CrossMsgOrm.GetL2CrossMsgByHash(h.ctx, common.HexToHash(hash))
		if err != nil {
			return nil, err
		}
		if l2result != nil {
			txHistory := &TxHistoryInfo{
				Hash:           l2result.Layer2Hash,
				Amount:         l2result.Amount,
				To:             l2result.Target,
				IsL1:           false,
				BlockNumber:    l2result.Height,
				BlockTimestamp: l2result.Timestamp,
				CreatedAt:      l2result.CreatedAt,
				FinalizeTx: &Finalized{
					Hash: "",
				},
				ClaimInfo: GetCrossTxClaimInfo(h.ctx, l2result.MsgHash, h.db),
			}
			updateCrossTxHash(h.ctx, l2result.MsgHash, txHistory, h.db)
			txHistories = append(txHistories, txHistory)
			continue
		}
	}
	return txHistories, nil
}
