package orm

import (
	"context"
	"database/sql"
	"fmt"

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

	// MsgFailed represents the from_layer message status is failed
	MsgFailed

	// MsgExpired represents the from_layer message status is expired
	MsgExpired
)

// L1Message is structure of stored layer1 bridge message
type L1Message struct {
	QueueIndex uint64    `json:"queue_index" db:"queue_index"`
	MsgHash    string    `json:"msg_hash" db:"msg_hash"`
	Height     uint64    `json:"height" db:"height"`
	Sender     string    `json:"sender" db:"sender"`
	Value      string    `json:"value" db:"value"`
	Target     string    `json:"target" db:"target"`
	Calldata   string    `json:"calldata" db:"calldata"`
	GasLimit   uint64    `json:"gas_limit" db:"gas_limit"`
	Layer1Hash string    `json:"layer1_hash" db:"layer1_hash"`
	Status     MsgStatus `json:"status" db:"status"`
}

// L2Message is structure of stored layer2 bridge message
type L2Message struct {
	Nonce      uint64    `json:"nonce" db:"nonce"`
	MsgHash    string    `json:"msg_hash" db:"msg_hash"`
	Height     uint64    `json:"height" db:"height"`
	Sender     string    `json:"sender" db:"sender"`
	Value      string    `json:"value" db:"value"`
	Target     string    `json:"target" db:"target"`
	Calldata   string    `json:"calldata" db:"calldata"`
	Layer2Hash string    `json:"layer2_hash" db:"layer2_hash"`
	Status     MsgStatus `json:"status" db:"status"`
}

// BlockInfo is structure of stored `block_trace` without `trace`
type BlockInfo struct {
	Number         uint64         `json:"number" db:"number"`
	Hash           string         `json:"hash" db:"hash"`
	ParentHash     string         `json:"parent_hash" db:"parent_hash"`
	BatchID        sql.NullString `json:"batch_id" db:"batch_id"`
	TxNum          uint64         `json:"tx_num" db:"tx_num"`
	GasUsed        uint64         `json:"gas_used" db:"gas_used"`
	BlockTimestamp uint64         `json:"block_timestamp" db:"block_timestamp"`
}

// RollerProveStatus is the roller prove status of a block batch (session)
type RollerProveStatus int32

const (
	// RollerAssigned indicates roller assigned but has not submitted proof
	RollerAssigned RollerProveStatus = iota
	// RollerProofValid indicates roller has submitted valid proof
	RollerProofValid
	// RollerProofInvalid indicates roller has submitted invalid proof
	RollerProofInvalid
)

func (s RollerProveStatus) String() string {
	switch s {
	case RollerAssigned:
		return "RollerAssigned"
	case RollerProofValid:
		return "RollerProofValid"
	case RollerProofInvalid:
		return "RollerProofInvalid"
	default:
		return fmt.Sprintf("Bad Value: %d", int32(s))
	}
}

// RollerStatus is the roller name and roller prove status
type RollerStatus struct {
	PublicKey string            `json:"public_key"`
	Name      string            `json:"name"`
	Status    RollerProveStatus `json:"status"`
}

// SessionInfo is assigned rollers info of a block batch (session)
type SessionInfo struct {
	ID             string                   `json:"id"`
	Rollers        map[string]*RollerStatus `json:"rollers"`
	StartTimestamp int64                    `json:"start_timestamp"`
}

// BlockTraceOrm block_trace operation interface
type BlockTraceOrm interface {
	Exist(number uint64) (bool, error)
	GetBlockTracesLatestHeight() (int64, error)
	GetBlockTraces(fields map[string]interface{}, args ...string) ([]*types.BlockTrace, error)
	GetBlockInfos(fields map[string]interface{}, args ...string) ([]*BlockInfo, error)
	// GetUnbatchedBlocks add `GetUnbatchedBlocks` because `GetBlockInfos` cannot support query "batch_id is NULL"
	GetUnbatchedBlocks(fields map[string]interface{}, args ...string) ([]*BlockInfo, error)
	GetHashByNumber(number uint64) (*common.Hash, error)
	DeleteTracesByBatchID(batchID string) error
	InsertBlockTraces(blockTraces []*types.BlockTrace) error
	SetBatchIDForBlocksInDBTx(dbTx *sqlx.Tx, numbers []uint64, batchID string) error
}

// SessionInfoOrm sessions info operation inte
type SessionInfoOrm interface {
	GetSessionInfosByIDs(ids []string) ([]*SessionInfo, error)
	SetSessionInfo(rollersInfo *SessionInfo) error
}

// BlockBatchOrm block_batch operation interface
type BlockBatchOrm interface {
	GetBlockBatches(fields map[string]interface{}, args ...string) ([]*BlockBatch, error)
	GetProvingStatusByID(id string) (ProvingStatus, error)
	GetVerifiedProofAndInstanceByID(id string) ([]byte, []byte, error)
	UpdateProofByID(ctx context.Context, id string, proof, instanceCommitments []byte, proofTimeSec uint64) error
	UpdateProvingStatus(id string, status ProvingStatus) error
	ResetProvingStatusFor(before ProvingStatus) error
	NewBatchInDBTx(dbTx *sqlx.Tx, startBlock *BlockInfo, endBlock *BlockInfo, parentHash string, totalTxNum uint64, gasUsed uint64) (string, error)
	BatchRecordExist(id string) (bool, error)
	GetPendingBatches(limit uint64) ([]string, error)
	GetCommittedBatches(limit uint64) ([]string, error)
	GetRollupStatus(id string) (RollupStatus, error)
	GetRollupStatusByIDList(ids []string) ([]RollupStatus, error)
	GetLatestBatch() (*BlockBatch, error)
	GetLatestCommittedBatch() (*BlockBatch, error)
	GetLatestFinalizedBatch() (*BlockBatch, error)
	UpdateRollupStatus(ctx context.Context, id string, status RollupStatus) error
	UpdateCommitTxHashAndRollupStatus(ctx context.Context, id string, commitTxHash string, status RollupStatus) error
	UpdateFinalizeTxHashAndRollupStatus(ctx context.Context, id string, finalizeTxHash string, status RollupStatus) error
	GetAssignedBatchIDs() ([]string, error)
	UpdateSkippedBatches() (int64, error)

	GetCommitTxHash(id string) (sql.NullString, error)   // for unit tests only
	GetFinalizeTxHash(id string) (sql.NullString, error) // for unit tests only
}

// L1MessageOrm is layer1 message db interface
type L1MessageOrm interface {
	GetL1MessageByNonce(nonce uint64) (*L1Message, error)
	GetL1MessageByMsgHash(msgHash string) (*L1Message, error)
	GetL1MessagesByStatus(status MsgStatus, limit uint64) ([]*L1Message, error)
	GetL1ProcessedNonce() (int64, error)
	SaveL1Messages(ctx context.Context, messages []*L1Message) error
	UpdateLayer2Hash(ctx context.Context, msgHash string, layer2Hash string) error
	UpdateLayer1Status(ctx context.Context, msgHash string, status MsgStatus) error
	UpdateLayer1StatusAndLayer2Hash(ctx context.Context, msgHash string, status MsgStatus, layer2Hash string) error
	GetLayer1LatestWatchedHeight() (int64, error)

	GetRelayL1MessageTxHash(nonce uint64) (sql.NullString, error) // for unit tests only
}

// L2MessageOrm is layer2 message db interface
type L2MessageOrm interface {
	GetL2MessageByNonce(nonce uint64) (*L2Message, error)
	GetL2MessageByMsgHash(msgHash string) (*L2Message, error)
	MessageProofExist(nonce uint64) (bool, error)
	GetMessageProofByNonce(nonce uint64) (string, error)
	GetL2Messages(fields map[string]interface{}, args ...string) ([]*L2Message, error)
	GetL2ProcessedNonce() (int64, error)
	SaveL2Messages(ctx context.Context, messages []*L2Message) error
	UpdateLayer1Hash(ctx context.Context, msgHash string, layer1Hash string) error
	UpdateLayer2Status(ctx context.Context, msgHash string, status MsgStatus) error
	UpdateLayer2StatusAndLayer1Hash(ctx context.Context, msgHash string, status MsgStatus, layer1Hash string) error
	UpdateMessageProof(ctx context.Context, nonce uint64, proof string) error
	GetLayer2LatestWatchedHeight() (int64, error)

	GetRelayL2MessageTxHash(nonce uint64) (sql.NullString, error) // for unit tests only
}
