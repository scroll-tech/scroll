package orm

import (
	"context"

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

// TODO: define prove_task structure
// ProveTask is structure of stored prove_task
type ProveTask struct {
	ID uint64 `json:"id" db:"id"`
}

// RollupResult is structure of stored rollup result
type RollupResult struct {
	// TODO:
	Number         int          `json:"number" db:"number"`
	Status         RollupStatus `json:"status" db:"status"`
	RollupTxHash   string       `json:"rollup_tx_hash" db:"rollup_tx_hash"`
	FinalizeTxHash string       `json:"finalize_tx_hash" db:"finalize_tx_hash"`
}

// ProveTaskOrm prove_task operation interface
type ProveTaskOrm interface {
	GetProveTasks(fields map[string]interface{}, args ...string) ([]*ProveTask, error)
	GetTaskStatusByID(id uint64) (TaskStatus, error)
	GetVerifiedProofAndInstanceByID(id uint64) ([]byte, []byte, error)
	// TODO: fix this
	UpdateProofByID(ctx context.Context, id uint64, proof, instance_commitments []byte, proofTimeSec uint64) error
	UpdateTaskStatus(id uint64, status TaskStatus) error
}

// BlockResultOrm blockResult operation interface
type BlockResultOrm interface {
	Exist(number uint64) (bool, error)
	GetBlockResultsLatestHeight() (int64, error)
	GetBlockResultsOldestHeight() (int64, error)
	GetBlockResults(fields map[string]interface{}, args ...string) ([]*types.BlockResult, error)
	GetHashByNumber(number uint64) (*common.Hash, error)
	DeleteTraceByNumber(number uint64) error
	InsertBlockResults(ctx context.Context, blockResults []*types.BlockResult) error
	NumberOfBlocksInLastHour() (uint64, error)
}

// RollupResultOrm rollupResult operation interface
type RollupResultOrm interface {
	RollupRecordExist(number uint64) (bool, error)
	GetPendingBlocks() ([]uint64, error)
	GetCommittedBlocks() ([]uint64, error)
	GetRollupStatus(number uint64) (RollupStatus, error)
	GetLatestFinalizedBlock() (uint64, error)
	InsertPendingBlocks(ctx context.Context, blocks []uint64) error
	UpdateRollupStatus(ctx context.Context, number uint64, status RollupStatus) error
	UpdateRollupTxHash(ctx context.Context, number uint64, rollup_tx_hash string) error
	UpdateFinalizeTxHash(ctx context.Context, number uint64, finalize_tx_hash string) error
	UpdateRollupTxHashAndStatus(ctx context.Context, number uint64, rollup_tx_hash string, status RollupStatus) error
	UpdateFinalizeTxHashAndStatus(ctx context.Context, number uint64, finalize_tx_hash string, status RollupStatus) error
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
