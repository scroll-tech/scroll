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
	ID           uint64     `json:"id" db:"id"`
	MsgHash      string     `json:"msg_hash" db:"msg_hash"`
	Height       uint64     `json:"height" db:"height"`
	Sender       string     `json:"sender" db:"sender"`
	Target       string     `json:"target" db:"target"`
	Amount       string     `json:"amount" db:"amount"`
	Layer1Hash   string     `json:"layer1_hash" db:"layer1_hash"`
	Layer2Hash   string     `json:"layer2_hash" db:"layer2_hash"`
	Layer1Token  string     `json:"layer1_token" db:"layer1_token"`
	Layer2Token  string     `json:"layer2_token" db:"layer2_token"`
	TokenIDs     string     `json:"token_ids" db:"token_ids"`
	TokenAmounts string     `json:"token_amounts" db:"token_amounts"`
	Asset        int        `json:"asset" db:"asset"`
	MsgType      int        `json:"msg_type" db:"msg_type"`
	Timestamp    *time.Time `json:"timestamp" db:"block_timestamp"`
	CreatedAt    *time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at" db:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at" db:"deleted_at"`
}

// L1CrossMsgOrm provides operations on l1_cross_message table
type L1CrossMsgOrm interface {
	GetL1CrossMsgByHash(l1Hash common.Hash) (*CrossMsg, error)
	GetL1CrossMsgsByAddress(sender common.Address) ([]*CrossMsg, error)
	BatchInsertL1CrossMsgDBTx(dbTx *sqlx.Tx, messages []*CrossMsg) error
	// UpdateL1CrossMsgHashDBTx invoked when SentMessage event is received
	UpdateL1CrossMsgHashDBTx(ctx context.Context, dbTx *sqlx.Tx, l1Hash, msgHash common.Hash) error
	UpdateL1CrossMsgHash(ctx context.Context, l1Hash, msgHash common.Hash) error
	GetLatestL1ProcessedHeight() (int64, error)
	DeleteL1CrossMsgAfterHeightDBTx(dbTx *sqlx.Tx, height int64) error
	UpdateL1BlockTimestamp(height uint64, timestamp time.Time) error
	GetL1EarliestNoBlockTimestampHeight() (uint64, error)
}

// L2CrossMsgOrm provides operations on cross_message table
type L2CrossMsgOrm interface {
	GetL2CrossMsgByMsgHash(msgHash string) (*CrossMsg, error)
	GetL2CrossMsgByHash(l2Hash common.Hash) (*CrossMsg, error)
	GetL2CrossMsgByAddress(sender common.Address) ([]*CrossMsg, error)
	BatchInsertL2CrossMsgDBTx(dbTx *sqlx.Tx, messages []*CrossMsg) error
	// UpdateL2CrossMsgHashDBTx invoked when SentMessage event is received
	UpdateL2CrossMsgHashDBTx(ctx context.Context, dbTx *sqlx.Tx, l2Hash, msgHash common.Hash) error
	UpdateL2CrossMsgHash(ctx context.Context, l2Hash, msgHash common.Hash) error
	GetLatestL2ProcessedHeight() (int64, error)
	DeleteL2CrossMsgFromHeightDBTx(dbTx *sqlx.Tx, height int64) error
	UpdateL2BlockTimestamp(height uint64, timestamp time.Time) error
	GetL2EarliestNoBlockTimestampHeight() (uint64, error)
	GetL2CrossMsgByMsgHashList(msgHashList []string) ([]*CrossMsg, error)
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
	GetLatestL2SentMsgBatchIndex() (int64, error)
	GetClaimableL2SentMsgByAddressWithOffset(address string, offset int64, limit int64) ([]*L2SentMsg, error)
	GetClaimableL2SentMsgByAddressTotalNum(address string) (uint64, error)
	DeleteL2SentMsgAfterHeightDBTx(dbTx *sqlx.Tx, height int64) error
}

type RollupBatchOrm interface {
	GetLatestRollupBatch() (*RollupBatch, error)
	GetRollupBatchByIndex(index uint64) (*RollupBatch, error)
	BatchInsertRollupBatchDBTx(dbTx *sqlx.Tx, messages []*RollupBatch) error
}
