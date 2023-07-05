package service

import (
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"bridge-history-api/db"
	"bridge-history-api/db/orm"
)

type Finalized struct {
	Hash           string     `json:"hash"`
	Amount         string     `json:"amount"`
	To             string     `json:"to"` // useless
	IsL1           bool       `json:"isL1"`
	BlockNumber    uint64     `json:"blockNumber"`
	BlockTimestamp *time.Time `json:"blockTimestamp"` // uselesss
}

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
	GetTxsByAddress(address common.Address, offset int64, limit int64) ([]*TxHistoryInfo, uint64, error)
	GetTxsByHashes(hashes []string) ([]*TxHistoryInfo, error)
}

// NewHistoryService returns a service backed with a "db"
func NewHistoryService(db db.OrmFactory) HistoryService {
	service := &historyBackend{db: db, prefix: "Scroll-Bridge-History-Server"}
	return service
}

type historyBackend struct {
	prefix string
	db     db.OrmFactory
}

func GetCrossTxClaimInfo(msgHash string, db db.OrmFactory) *UserClaimInfo {
	l2sentMsg, err := db.GetL2SentMsgByHash(msgHash)
	if err != nil {
		log.Debug("GetCrossTxClaimInfo failed", "error", err)
		return &UserClaimInfo{}
	}
	batch, err := db.GetRollupBatchByIndex(l2sentMsg.BatchIndex)
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

func updateCrossTxHash(msgHash string, txInfo *TxHistoryInfo, db db.OrmFactory) {
	relayed, err := db.GetRelayedMsgByHash(msgHash)
	if err != nil {
		log.Error("updateCrossTxHash failed", "error", err)
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

func (h *historyBackend) GetTxsByAddress(address common.Address, offset int64, limit int64) ([]*TxHistoryInfo, uint64, error) {
	var txHistories []*TxHistoryInfo
	total, err := h.db.GetTotalCrossMsgCountByAddress(address.String())
	if err != nil || total == 0 {
		return txHistories, 0, err
	}
	result, err := h.db.GetCrossMsgsByAddressWithOffset(address.String(), offset, limit)

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
			ClaimInfo: GetCrossTxClaimInfo(msg.MsgHash, h.db),
		}
		updateCrossTxHash(msg.MsgHash, txHistory, h.db)
		txHistories = append(txHistories, txHistory)
	}
	return txHistories, total, nil
}

func (h *historyBackend) GetTxsByHashes(hashes []string) ([]*TxHistoryInfo, error) {
	txHistories := make([]*TxHistoryInfo, 0)
	for _, hash := range hashes {
		l1result, err := h.db.GetL1CrossMsgByHash(common.HexToHash(hash))
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
			updateCrossTxHash(l1result.MsgHash, txHistory, h.db)
			txHistories = append(txHistories, txHistory)
			continue
		}
		l2result, err := h.db.GetL2CrossMsgByHash(common.HexToHash(hash))
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
				ClaimInfo: GetCrossTxClaimInfo(l2result.MsgHash, h.db),
			}
			updateCrossTxHash(l2result.MsgHash, txHistory, h.db)
			txHistories = append(txHistories, txHistory)
			continue
		}
	}
	return txHistories, nil
}
