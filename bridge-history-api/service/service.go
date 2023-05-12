package service

import (
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"bridge-history-api/db"
)

type Finalized struct {
	Hash           string     `json:"hash"`
	Amount         string     `json:"amount"`
	To             string     `json:"to"` // useless
	IsL1           bool       `json:"isL1"`
	BlockNumber    uint64     `json:"blockNumber"`
	BlockTimestamp *time.Time `json:"blockTimestamp"` // uselesss
}

type TxHistoryInfo struct {
	Hash           string     `json:"hash"`
	Amount         string     `json:"amount"`
	To             string     `json:"to"` // useless
	IsL1           bool       `json:"isL1"`
	BlockNumber    uint64     `json:"blockNumber"`
	BlockTimestamp *time.Time `json:"blockTimestamp"` // useless
	FinalizeTx     *Finalized `json:"finalizeTx"`
	CreatedTime    *time.Time `json:"createdTime"`
}

// HistoryService example service.
type HistoryService interface {
	GetTxsByAddress(address common.Address, offset int64, limit int64) ([]*TxHistoryInfo, error)
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

func (h *historyBackend) GetTxsByAddress(address common.Address, offset int64, limit int64) ([]*TxHistoryInfo, error) {
	txHistories := make([]*TxHistoryInfo, 0)
	result, err := h.db.GetL1CrossMsgsByAddressWithOffset(address, offset, limit)
	if err != nil {
		return nil, err
	}
	if len(result) > 0 {
		for _, r := range result {
			txHistory := &TxHistoryInfo{
				Hash:        r.Layer1Hash,
				Amount:      r.Amount,
				To:          r.Target,
				IsL1:        true,
				BlockNumber: r.Height,
				CreatedTime: r.CreatedTime,
				FinalizeTx: &Finalized{
					Hash: "",
				},
			}
			// update relayed info into results
			updateCrossTxHash(r.MsgHash, txHistory, h.db)
			txHistories = append(txHistories, txHistory)
		}
	}
	l2Result, err := h.db.GetL2CrossMsgsByAddressWithOffset(address, offset, limit)
	if err != nil {
		return nil, err
	}
	if len(l2Result) > 0 {
		for _, r := range l2Result {
			txHistory := &TxHistoryInfo{
				Hash:        r.Layer2Hash,
				Amount:      r.Amount,
				To:          r.Target,
				IsL1:        false,
				BlockNumber: r.Height,
				CreatedTime: r.CreatedTime,
				FinalizeTx: &Finalized{
					Hash: "",
				},
			}
			updateCrossTxHash(r.MsgHash, txHistory, h.db)
			txHistories = append(txHistories, txHistory)
		}
	}
	if len(txHistories) > 0 {
		sort.Slice(txHistories, func(i, j int) bool {
			return txHistories[i].CreatedTime.Second() > txHistories[j].CreatedTime.Second()
		})
	}
	return txHistories, nil
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
				Hash:        l1result.Layer1Hash,
				Amount:      l1result.Amount,
				To:          l1result.Target,
				IsL1:        true,
				BlockNumber: l1result.Height,
				CreatedTime: l1result.CreatedTime,
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
				Hash:        l2result.Layer2Hash,
				Amount:      l2result.Amount,
				To:          l2result.Target,
				IsL1:        false,
				BlockNumber: l2result.Height,
				CreatedTime: l2result.CreatedTime,
				FinalizeTx: &Finalized{
					Hash: "",
				},
			}
			updateCrossTxHash(l2result.MsgHash, txHistory, h.db)
			txHistories = append(txHistories, txHistory)
			continue
		}
	}
	return txHistories, nil
}
