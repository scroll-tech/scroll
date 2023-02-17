package orm

import (
	"context"
	"database/sql"

	"scroll-tech/common/types"

	"github.com/jmoiron/sqlx"
	"github.com/scroll-tech/go-ethereum/common"
	eth_types "github.com/scroll-tech/go-ethereum/core/types"
)

// BlockTraceOrm block_trace operation interface
type BlockTraceOrm interface {
	Exist(number uint64) (bool, error)
	GetBlockTracesLatestHeight() (int64, error)
	GetBlockTraces(fields map[string]interface{}, args ...string) ([]*eth_types.BlockTrace, error)
	GetBlockInfos(fields map[string]interface{}, args ...string) ([]*types.BlockInfo, error)
	// GetUnbatchedBlocks add `GetUnbatchedBlocks` because `GetBlockInfos` cannot support query "batch_id is NULL"
	GetUnbatchedBlocks(fields map[string]interface{}, args ...string) ([]*types.BlockInfo, error)
	GetHashByNumber(number uint64) (*common.Hash, error)
	DeleteTracesByBatchID(batchID string) error
	InsertBlockTraces(blockTraces []*eth_types.BlockTrace) error
	SetBatchIDForBlocksInDBTx(dbTx *sqlx.Tx, numbers []uint64, batchID string) error
}

// SessionInfoOrm sessions info operation inte
type SessionInfoOrm interface {
	GetSessionInfosByIDs(ids []string) ([]*types.SessionInfo, error)
	SetSessionInfo(rollersInfo *types.SessionInfo) error
}

// BlockBatchOrm block_batch operation interface
type BlockBatchOrm interface {
	GetBlockBatches(fields map[string]interface{}, args ...string) ([]*types.BlockBatch, error)
	GetProvingStatusByID(id string) (types.ProvingStatus, error)
	GetVerifiedProofAndInstanceByID(id string) ([]byte, []byte, error)
	UpdateProofByID(ctx context.Context, id string, proof, instanceCommitments []byte, proofTimeSec uint64) error
	UpdateProvingStatus(id string, status types.ProvingStatus) error
	ResetProvingStatusFor(before types.ProvingStatus) error
	NewBatchInDBTx(dbTx *sqlx.Tx, batchData *types.BatchData) error
	BatchRecordExist(id string) (bool, error)
	GetPendingBatches(limit uint64) ([]string, error)
	GetCommittedBatches(limit uint64) ([]string, error)
	GetRollupStatus(id string) (types.RollupStatus, error)
	GetRollupStatusByIDList(ids []string) ([]types.RollupStatus, error)
	GetLatestBatch() (*types.BlockBatch, error)
	GetLatestCommittedBatch() (*types.BlockBatch, error)
	GetLatestFinalizedBatch() (*types.BlockBatch, error)
	UpdateRollupStatus(ctx context.Context, id string, status types.RollupStatus) error
	UpdateCommitTxHashAndRollupStatus(ctx context.Context, id string, commitTxHash string, status types.RollupStatus) error
	UpdateFinalizeTxHashAndRollupStatus(ctx context.Context, id string, finalizeTxHash string, status types.RollupStatus) error
	GetAssignedBatchIDs() ([]string, error)
	UpdateSkippedBatches() (int64, error)

	GetCommitTxHash(id string) (sql.NullString, error)   // for unit tests only
	GetFinalizeTxHash(id string) (sql.NullString, error) // for unit tests only
}

// L1MessageOrm is layer1 message db interface
type L1MessageOrm interface {
	GetL1MessageByNonce(nonce uint64) (*types.L1Message, error)
	GetL1MessageByMsgHash(msgHash string) (*types.L1Message, error)
	GetL1MessagesByStatus(status types.MsgStatus, limit uint64) ([]*types.L1Message, error)
	GetL1ProcessedNonce() (int64, error)
	SaveL1Messages(ctx context.Context, messages []*types.L1Message) error
	UpdateLayer2Hash(ctx context.Context, msgHash string, layer2Hash string) error
	UpdateLayer1Status(ctx context.Context, msgHash string, status types.MsgStatus) error
	UpdateLayer1StatusAndLayer2Hash(ctx context.Context, msgHash string, status types.MsgStatus, layer2Hash string) error
	GetLayer1LatestWatchedHeight() (int64, error)

	GetRelayL1MessageTxHash(nonce uint64) (sql.NullString, error) // for unit tests only
}

// L2MessageOrm is layer2 message db interface
type L2MessageOrm interface {
	GetL2MessageByNonce(nonce uint64) (*types.L2Message, error)
	GetL2MessageByMsgHash(msgHash string) (*types.L2Message, error)
	MessageProofExist(nonce uint64) (bool, error)
	GetMessageProofByNonce(nonce uint64) (string, error)
	GetL2Messages(fields map[string]interface{}, args ...string) ([]*types.L2Message, error)
	GetL2ProcessedNonce() (int64, error)
	SaveL2Messages(ctx context.Context, messages []*types.L2Message) error
	UpdateLayer1Hash(ctx context.Context, msgHash string, layer1Hash string) error
	UpdateLayer2Status(ctx context.Context, msgHash string, status types.MsgStatus) error
	UpdateLayer2StatusAndLayer1Hash(ctx context.Context, msgHash string, status types.MsgStatus, layer1Hash string) error
	UpdateMessageProof(ctx context.Context, nonce uint64, proof string) error
	GetLayer2LatestWatchedHeight() (int64, error)

	GetRelayL2MessageTxHash(nonce uint64) (sql.NullString, error) // for unit tests only
}
