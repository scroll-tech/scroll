package orm

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
)

type AssetType int
type MsgType int

func (a AssetType) String() string {
	switch a {
	case ETH:
		return "ETH"
	case ERC20:
		return "ERC20"
	case ERC1155:
		return "ERC1155"
	case ERC721:
		return "ERC721"
	}
	return "Unknown Asset Type"
}

const (
	ETH AssetType = iota
	ERC20
	ERC721
	ERC1155
)

const (
	UnknownMsg MsgType = iota
	Layer1Msg
	Layer2Msg
)

// CrossMsg represents a cross message from layer 1 to layer 2
type CrossMsg struct {
	ID          uint64     `json:"id" db:"id"`
	MsgHash     string     `json:"msg_hash" db:"msg_hash"`
	Height      uint64     `json:"height" db:"height"`
	Sender      string     `json:"sender" db:"sender"`
	Target      string     `json:"target" db:"target"`
	Amount      string     `json:"amount" db:"amount"`
	Layer1Hash  string     `json:"layer1_hash" db:"layer1_hash"`
	Layer2Hash  string     `json:"layer2_hash" db:"layer2_hash"`
	Layer1Token string     `json:"layer1_token" db:"layer1_token"`
	Layer2Token string     `json:"layer2_token" db:"layer2_token"`
	TokenID     uint64     `json:"token_id" db:"token_id"`
	Asset       int        `json:"asset" db:"asset"`
	MsgType     int        `json:"msg_type" db:"msg_type"`
	IsDeleted   bool       `json:"is_deleted" db:"is_deleted"`
	Timestamp   *time.Time `json:"timestamp" db:"block_timestamp"`
	CreatedAt   *time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at" db:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at" db:"deleted_at"`
}

type RelayedMsg struct {
	MsgHash    string `json:"msg_hash" db:"msg_hash"`
	Height     uint64 `json:"height" db:"height"`
	Layer1Hash string `json:"layer1_hash" db:"layer1_hash"`
	Layer2Hash string `json:"layer2_hash" db:"layer2_hash"`
}

// L1CrossMsgOrm provides operations on l1_cross_message table
type L1CrossMsgOrm interface {
	GetL1CrossMsgByHash(l1Hash common.Hash) (*CrossMsg, error)
	GetL1CrossMsgsByAddress(sender common.Address) ([]*CrossMsg, error)
	BatchInsertL1CrossMsgDBTx(dbTx *sqlx.Tx, messages []*CrossMsg) error
	// UpdateL1CrossMsgHash invoked when SentMessage event is received
	UpdateL1CrossMsgHashDBTx(ctx context.Context, dbTx *sqlx.Tx, l1Hash, msgHash common.Hash) error
	UpdateL1CrossMsgHash(ctx context.Context, l1Hash, msgHash common.Hash) error
	GetLatestL1ProcessedHeight() (int64, error)
	DeleteL1CrossMsgAfterHeightDBTx(dbTx *sqlx.Tx, height int64) error
	UpdateL1Blocktimestamp(height uint64, timestamp time.Time) error
	GetL1EarliestNoBlocktimestampHeight() (uint64, error)
}

// L2CrossMsgOrm provides operations on cross_message table
type L2CrossMsgOrm interface {
	GetL2CrossMsgByHash(l2Hash common.Hash) (*CrossMsg, error)
	GetL2CrossMsgByAddress(sender common.Address) ([]*CrossMsg, error)
	BatchInsertL2CrossMsgDBTx(dbTx *sqlx.Tx, messages []*CrossMsg) error
	// UpdateL2CrossMsgHash invoked when SentMessage event is received
	UpdateL2CrossMsgHashDBTx(ctx context.Context, dbTx *sqlx.Tx, l2Hash, msgHash common.Hash) error
	UpdateL2CrossMsgHash(ctx context.Context, l2Hash, msgHash common.Hash) error
	GetLatestL2ProcessedHeight() (int64, error)
	DeleteL2CrossMsgFromHeightDBTx(dbTx *sqlx.Tx, height int64) error
	UpdateL2Blocktimestamp(height uint64, timestamp time.Time) error
	GetL2EarliestNoBlocktimestampHeight() (uint64, error)
}

type RelayedMsgOrm interface {
	BatchInsertRelayedMsgDBTx(dbTx *sqlx.Tx, messages []*RelayedMsg) error
	GetRelayedMsgByHash(msg_hash string) (*RelayedMsg, error)
	GetLatestRelayedHeightOnL1() (int64, error)
	GetLatestRelayedHeightOnL2() (int64, error)
	DeleteL1RelayedHashAfterHeightDBTx(dbTx *sqlx.Tx, height int64) error
	DeleteL2RelayedHashAfterHeightDBTx(dbTx *sqlx.Tx, height int64) error
}

type L2SentMsgOrm interface {
	BatchInsertL2SentMsgDBTx(dbTx *sqlx.Tx, messages []*L2SentMsg) error
	GetL2SentMsgByHash(l2Hash string) (*L2SentMsg, error)
	GetLatestSentMsgHeightOnL2() (int64, error)
	GetL2SentMessageByNonce(nonce uint64) (*L2SentMsg, error)
	GetLatestL2SentMsgLEHeight(endBlockNumber uint64) (*L2SentMsg, error)
	GetL2SentMsgMsgHashByHeightRange(startHeight, endHeight uint64) ([]*L2SentMsg, error)
	UpdateL2MessageProofInDBTx(ctx context.Context, dbTx *sqlx.Tx, msgHash string, proof string, batch_index uint64) error
	GetLatestL2SentMsgBactchIndex() (int64, error)
	DeleteL2SentMsgAfterHeightDBTx(dbTx *sqlx.Tx, height int64) error
	ResetL2SentMsgL1HashAfterHeightDBTx(dbTx *sqlx.Tx, height int64) error
}

type BridgeBatchOrm interface {
	GetLatestBridgeBatch() (*RollupBatch, error)
	GetBridgeBatchByIndex(index uint64) (*RollupBatch, error)
	BatchInsertBridgeBatchDBTx(dbTx *sqlx.Tx, messages []*RollupBatch) error
}
