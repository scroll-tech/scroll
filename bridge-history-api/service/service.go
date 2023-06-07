package service

import (
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

type ERC20TxHistoryInfo struct {
	Hash           string     `json:"hash"`
	Amount         string     `json:"amount"`
	To             string     `json:"to"` // useless
	IsL1           bool       `json:"isL1"`
	BlockNumber    uint64     `json:"blockNumber"`
	BlockTimestamp *time.Time `json:"blockTimestamp"` // useless
	FinalizeTx     *Finalized `json:"finalizeTx"`
	CreatedAt      *time.Time `json:"createdTime"`
}

type ResponseData struct {
	Results []interface{} `json:"results"`
}

type NFTTxHistoryInfo struct {
	Hash           string     `json:"hash"`
	TokenType      string     `json:"tokenType"`
	Amounts        []int64    `json:"amount"`
	TokenIds       []int64    `json:"tokenIds"`
	To             string     `json:"to"` // useless
	IsL1           bool       `json:"isL1"`
	BlockNumber    uint64     `json:"blockNumber"`
	BlockTimestamp *time.Time `json:"blockTimestamp"` // useless
	FinalizeTx     *Finalized `json:"finalizeTx"`
	CreatedAt      *time.Time `json:"createdTime"`
}

// HistoryService example service.
type HistoryService interface {
	GetERC20TxsByAddress(address common.Address, offset int64, limit int64) (*ResponseData, error)
	GetNFTTxsByAddress(address common.Address, offset int64, limit int64, asset orm.AssetType) (*ResponseData, error)
	GetTxsByHashes(hashes []string) (*ResponseData, error)
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

func GetCrossTxHashFinalizedInfo(msgHash string, db db.OrmFactory) (*Finalized, error) {
	relayed, err := db.GetRelayedMsgByHash(msgHash)
	var result *Finalized
	if err != nil {
		log.Error("GetCrossTxHashFinalizedInfo failed", "error", err)
		return nil, err
	}
	if relayed == nil {
		return result, nil
	}
	if relayed.Layer1Hash != "" {
		result.Hash = relayed.Layer1Hash
		result.BlockNumber = relayed.Height
		return result, nil
	}
	if relayed.Layer2Hash != "" {
		result.Hash = relayed.Layer2Hash
		result.BlockNumber = relayed.Height
		return result, nil
	}
	return nil, nil
}

func (h *historyBackend) GetNFTTxsByAddress(address common.Address, offset int64, limit int64, nftType orm.AssetType) (*ResponseData, error) {
	response := make([]interface{}, 0)
	result, err := h.db.GetCrossMsgsByAddressWithOffset(address.String(), offset, limit, nftType)
	if err != nil {
		return nil, err
	}
	for _, msg := range result {
		txHistory := &NFTTxHistoryInfo{
			Hash:        msg.MsgHash,
			TokenIds:    msg.TokenIDs,
			Amounts:     msg.TokenAmounts,
			To:          msg.Target,
			IsL1:        msg.MsgType == int(orm.Layer1Msg),
			BlockNumber: msg.Height,
			CreatedAt:   msg.CreatedAt,
			FinalizeTx: &Finalized{
				Hash: "",
			},
		}
		finalizedData, err := GetCrossTxHashFinalizedInfo(msg.MsgHash, h.db)
		if err != nil {
			return nil, err
		}
		if finalizedData != nil {
			txHistory.FinalizeTx = finalizedData
		}
		response = append(response, txHistory)
	}
	return &ResponseData{Results: response}, nil
}

func (h *historyBackend) GetERC20TxsByAddress(address common.Address, offset int64, limit int64) (*ResponseData, error) {
	response := make([]interface{}, 0)
	result, err := h.db.GetCrossMsgsByAddressWithOffset(address.String(), offset, limit, orm.ERC20)
	if err != nil {
		return nil, err
	}
	for _, msg := range result {
		txHistory := &ERC20TxHistoryInfo{
			Hash:        msg.MsgHash,
			Amount:      msg.Amount,
			To:          msg.Target,
			IsL1:        msg.MsgType == int(orm.Layer1Msg),
			BlockNumber: msg.Height,
			CreatedAt:   msg.CreatedAt,
			FinalizeTx: &Finalized{
				Hash: "",
			},
		}
		finalizedData, err := GetCrossTxHashFinalizedInfo(msg.MsgHash, h.db)
		if err != nil {
			return nil, err
		}
		if finalizedData != nil {
			txHistory.FinalizeTx = finalizedData
		}
		response = append(response, txHistory)
	}
	return &ResponseData{Results: response}, nil
}

func (h *historyBackend) GetTxsByHashes(hashes []string) (*ResponseData, error) {
	txHistories := make([]interface{}, 0)
	for _, hash := range hashes {
		l1result, err := h.db.GetL1CrossMsgByHash(common.HexToHash(hash))
		if err != nil {
			return nil, err
		}
		var txHistory interface{}
		if l1result != nil {
			finalizedData, err := GetCrossTxHashFinalizedInfo(l1result.MsgHash, h.db)
			if err != nil {
				return nil, err
			}
			switch l1result.Asset {
			case int(orm.ERC20):
				txHistory = &ERC20TxHistoryInfo{
					Hash:        l1result.Layer1Hash,
					Amount:      l1result.Amount,
					To:          l1result.Target,
					IsL1:        true,
					BlockNumber: l1result.Height,
					CreatedAt:   l1result.CreatedAt,
					FinalizeTx:  finalizedData,
				}
			case int(orm.ERC721):
				txHistory = &NFTTxHistoryInfo{
					Hash:        l1result.Layer1Hash,
					TokenType:   orm.ERC721.String(),
					TokenIds:    l1result.TokenIDs,
					Amounts:     l1result.TokenAmounts,
					To:          l1result.Target,
					IsL1:        true,
					BlockNumber: l1result.Height,
					CreatedAt:   l1result.CreatedAt,
					FinalizeTx:  finalizedData,
				}
			case int(orm.ERC1155):
				txHistory = &NFTTxHistoryInfo{
					Hash:        l1result.Layer1Hash,
					TokenType:   orm.ERC1155.String(),
					TokenIds:    l1result.TokenIDs,
					Amounts:     l1result.TokenAmounts,
					To:          l1result.Target,
					IsL1:        true,
					BlockNumber: l1result.Height,
					CreatedAt:   l1result.CreatedAt,
					FinalizeTx:  finalizedData,
				}
			default:
				continue
			}
			txHistories = append(txHistories, txHistory)
			continue
		}
		l2result, err := h.db.GetL2CrossMsgByHash(common.HexToHash(hash))
		if err != nil {
			return nil, err
		}
		if l2result != nil {
			finalizedData, err := GetCrossTxHashFinalizedInfo(l2result.MsgHash, h.db)
			if err != nil {
				return nil, err
			}
			switch l2result.Asset {
			case int(orm.ERC20):
				txHistory = &ERC20TxHistoryInfo{
					Hash:        l2result.Layer2Hash,
					Amount:      l2result.Amount,
					To:          l2result.Target,
					IsL1:        true,
					BlockNumber: l2result.Height,
					CreatedAt:   l2result.CreatedAt,
					FinalizeTx:  finalizedData,
				}
			case int(orm.ERC721):
				txHistory = &NFTTxHistoryInfo{
					Hash:        l2result.Layer2Hash,
					TokenType:   orm.ERC721.String(),
					TokenIds:    l2result.TokenIDs,
					Amounts:     l2result.TokenAmounts,
					To:          l2result.Target,
					IsL1:        true,
					BlockNumber: l2result.Height,
					CreatedAt:   l2result.CreatedAt,
					FinalizeTx:  finalizedData,
				}
			case int(orm.ERC1155):
				txHistory = &NFTTxHistoryInfo{
					Hash:        l2result.Layer2Hash,
					TokenType:   orm.ERC1155.String(),
					TokenIds:    l2result.TokenIDs,
					Amounts:     l2result.TokenAmounts,
					To:          l2result.Target,
					IsL1:        true,
					BlockNumber: l2result.Height,
					CreatedAt:   l2result.CreatedAt,
					FinalizeTx:  finalizedData,
				}
			default:
				continue
			}
			txHistories = append(txHistories, txHistory)
			continue
		}
	}
	return &ResponseData{Results: txHistories}, nil
}
