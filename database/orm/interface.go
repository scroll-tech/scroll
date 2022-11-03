package orm

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
)

// MsgStatus represents current layer1 transaction processing status
type MsgStatus int

const (
	// MsgUndefined : undefined msg status
	MsgUndefined MsgStatus = iota

	// MsgPending represents the from_layer message status is pending
	MsgPending

	// MsgSubmitted represents the from_layer message status is submitted
	MsgSubmitted

	// MsgConfirmed represents the from_layer message status is confirmed
	MsgConfirmed
)

// Layer1Message is structure of stored layer1 bridge message
type Layer1Message struct {
	Nonce      uint64    `json:"nonce" db:"nonce"`
	Height     uint64    `json:"height" db:"height"`
	Sender     string    `json:"sender" db:"sender"`
	Value      string    `json:"value" db:"value"`
	Fee        string    `json:"fee" db:"fee"`
	GasLimit   uint64    `json:"gas_limit" db:"gas_limit"`
	Deadline   uint64    `json:"deadline" db:"deadline"`
	Target     string    `json:"target" db:"target"`
	Calldata   string    `json:"calldata" db:"calldata"`
	Layer1Hash string    `json:"layer1_hash" db:"layer1_hash"`
	Status     MsgStatus `json:"status" db:"status"`
}

// Layer2Message is structure of stored layer2 bridge message
type Layer2Message struct {
	Nonce      uint64    `json:"nonce" db:"nonce"`
	Height     uint64    `json:"height" db:"height"`
	Sender     string    `json:"sender" db:"sender"`
	Value      string    `json:"value" db:"value"`
	Fee        string    `json:"fee" db:"fee"`
	GasLimit   uint64    `json:"gas_limit" db:"gas_limit"`
	Deadline   uint64    `json:"deadline" db:"deadline"`
	Target     string    `json:"target" db:"target"`
	Calldata   string    `json:"calldata" db:"calldata"`
	Layer2Hash string    `json:"layer2_hash" db:"layer2_hash"`
	Status     MsgStatus `json:"status" db:"status"`
}

// BlockInfo is structure of stored `block_result` without `content`
type BlockInfo struct {
	Number         uint64 `json:"number" db:"number"`
	Hash           string `json:"hash" db:"hash"`
	BatchID        uint64 `json:"batch_id" db:"batch_id"`
	TxNum          string `json:"tx_num" db:"tx_num"`
	GasUsed        uint64 `json:"gas_used" db:"gas_used"`
	BlockTimestamp uint64 `json:"block_timestamp" db:"block_timestamp"`
}

// BlockResultOrm blockResult operation interface
type BlockResultOrm interface {
	Exist(number uint64) (bool, error)
	GetBlockResultsLatestHeight() (int64, error)
	GetBlockResultsOldestHeight() (int64, error)
	GetBlockResults(fields map[string]interface{}, args ...string) ([]*types.BlockResult, error)
	GetBlockInfos(fields map[string]interface{}, args ...string) ([]*BlockInfo, error)
	GetHashByNumber(number uint64) (*common.Hash, error)
	DeleteTracesByBatchID(batch_id uint64) error
	InsertBlockResults(ctx context.Context, blockResults []*types.BlockResult) error
	SetBatchIDForBlocksInDBTx(dbTx *sqlx.Tx, blocks []uint64, batchID uint64) error
}

// BlockBatchOrm block_batch operation interface
type BlockBatchOrm interface {
	GetBlockBatches(fields map[string]interface{}, args ...string) ([]*BlockBatch, error)
	GetProvingStatusByID(id uint64) (ProvingStatus, error)
	GetVerifiedProofAndInstanceByID(id uint64) ([]byte, []byte, error)
	UpdateProofByID(ctx context.Context, id uint64, proof, instance_commitments []byte, proofTimeSec uint64) error
	UpdateProvingStatus(id uint64, status ProvingStatus) error
	NewBatchInDBTx(dbTx *sqlx.Tx, gasUsed uint64) (uint64, error)
	BatchRecordExist(number uint64) (bool, error)
	GetPendingBatches() ([]uint64, error)
	GetCommittedBatches() ([]uint64, error)
	GetRollupStatus(number uint64) (RollupStatus, error)
	GetLatestFinalizedBatch() (uint64, error)
	UpdateRollupStatus(ctx context.Context, number uint64, status RollupStatus) error
	UpdateRollupTxHashAndRollupStatus(ctx context.Context, number uint64, rollup_tx_hash string, status RollupStatus) error
	UpdateFinalizeTxHashAndRollupStatus(ctx context.Context, number uint64, finalize_tx_hash string, status RollupStatus) error
}

// Layer1MessageOrm is layer1 message db interface
type Layer1MessageOrm interface {
	GetLayer1MessageByNonce(nonce uint64) (*Layer1Message, error)
	GetL1UnprocessedMessages() ([]*Layer1Message, error)
	GetL1ProcessedNonce() (int64, error)
	SaveLayer1Messages(ctx context.Context, messages []*Layer1Message) error
	UpdateLayer2Hash(ctx context.Context, layer1Hash string, layer2Hash string) error
	UpdateLayer1Status(ctx context.Context, layer1Hash string, status MsgStatus) error
	UpdateLayer1StatusAndLayer2Hash(ctx context.Context, layer1Hash, layer2Hash string, status MsgStatus) error
	GetLayer1LatestWatchedHeight() (int64, error)
	GetLayer1MessageByLayer1Hash(layer1Hash string) (*Layer1Message, error)
}

// Layer2MessageOrm is layer2 message db interface
type Layer2MessageOrm interface {
	GetLayer2MessageByNonce(nonce uint64) (*Layer2Message, error)
	MessageProofExist(nonce uint64) (bool, error)
	GetMessageProofByNonce(nonce uint64) (string, error)
	GetL2UnprocessedMessages() ([]*Layer2Message, error)
	GetL2ProcessedNonce() (int64, error)
	SaveLayer2Messages(ctx context.Context, messages []*Layer2Message) error
	UpdateLayer1Hash(ctx context.Context, layer2Hash string, layer1Hash string) error
	UpdateLayer2Status(ctx context.Context, layer2Hash string, status MsgStatus) error
	GetLayer2MessageByLayer2Hash(layer2Hash string) (*Layer2Message, error)
	UpdateMessageProof(ctx context.Context, layer2Hash, proof string) error
	GetLayer2LatestWatchedHeight() (int64, error)
	GetMessageProofByLayer2Hash(layer2Hash string) (string, error)
	MessageProofExistByLayer2Hash(layer2Hash string) (bool, error)
	UpdateLayer2StatusAndLayer1Hash(ctx context.Context, layer2Hash string, layer1Hash string, status MsgStatus) error
}
