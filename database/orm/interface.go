package orm

import (
	"context"
	"database/sql"

	"scroll-tech/common/types"

	"github.com/jmoiron/sqlx"
	"github.com/scroll-tech/go-ethereum/common"
	eth_types "github.com/scroll-tech/go-ethereum/core/types"
)

// L1BlockOrm l1_block operation interface
type L1BlockOrm interface {
	GetL1BlockInfos(fields map[string]interface{}, args ...string) ([]*types.L1BlockInfo, error)
	InsertL1Blocks(ctx context.Context, blocks []*types.L1BlockInfo) error
	DeleteHeaderRLPByBlockHash(ctx context.Context, blockHash string) error
	UpdateImportTxHash(ctx context.Context, blockHash, txHash string) error
	UpdateL1BlockStatus(ctx context.Context, blockHash string, status types.L1BlockStatus) error
	UpdateL1BlockStatusAndImportTxHash(ctx context.Context, blockHash string, status types.L1BlockStatus, txHash string) error
	UpdateL1OracleTxHash(ctx context.Context, blockHash, txHash string) error
	UpdateL1GasOracleStatus(ctx context.Context, blockHash string, status types.GasOracleStatus) error
	UpdateL1GasOracleStatusAndOracleTxHash(ctx context.Context, blockHash string, status types.GasOracleStatus, txHash string) error
	GetLatestL1BlockHeight() (uint64, error)
	GetLatestImportedL1Block() (*types.L1BlockInfo, error)
}

// BlockTraceOrm block_trace operation interface
type BlockTraceOrm interface {
	IsL2BlockExists(number uint64) (bool, error)
	GetL2BlockTracesLatestHeight() (int64, error)
	GetL2BlockTraces(fields map[string]interface{}, args ...string) ([]*eth_types.BlockTrace, error)
	GetL2BlockInfos(fields map[string]interface{}, args ...string) ([]*types.BlockInfo, error)
	// GetUnbatchedBlocks add `GetUnbatchedBlocks` because `GetBlockInfos` cannot support query "batch_hash is NULL"
	GetUnbatchedL2Blocks(fields map[string]interface{}, args ...string) ([]*types.BlockInfo, error)
	GetL2BlockHashByNumber(number uint64) (*common.Hash, error)
	DeleteTracesByBatchHash(batchHash string) error
	InsertL2BlockTraces(blockTraces []*eth_types.BlockTrace) error
	SetBatchHashForL2BlocksInDBTx(dbTx *sqlx.Tx, numbers []uint64, batchHash string) error
}

// SessionInfoOrm sessions info operation inte
type SessionInfoOrm interface {
	GetSessionInfosByHashes(hashes []string) ([]*types.SessionInfo, error)
	SetSessionInfo(rollersInfo *types.SessionInfo) error
}

// BlockBatchOrm block_batch operation interface
type BlockBatchOrm interface {
	GetBlockBatches(fields map[string]interface{}, args ...string) ([]*types.BlockBatch, error)
	GetProvingStatusByHash(hash string) (types.ProvingStatus, error)
	GetVerifiedProofAndInstanceByHash(hash string) ([]byte, []byte, error)
	UpdateProofByHash(ctx context.Context, hash string, proof, instanceCommitments []byte, proofTimeSec uint64) error
	UpdateProvingStatus(hash string, status types.ProvingStatus) error
	ResetProvingStatusFor(before types.ProvingStatus) error
	NewBatchInDBTx(dbTx *sqlx.Tx, batchData *types.BatchData) error
	BatchRecordExist(hash string) (bool, error)
	GetPendingBatches(limit uint64) ([]string, error)
	GetCommittedBatches(limit uint64) ([]string, error)
	GetRollupStatus(hash string) (types.RollupStatus, error)
	GetRollupStatusByHashList(hashes []string) ([]types.RollupStatus, error)
	GetLatestBatch() (*types.BlockBatch, error)
	GetLatestCommittedBatch() (*types.BlockBatch, error)
	GetLatestFinalizedBatch() (*types.BlockBatch, error)
	GetLatestFinalizingOrFinalizedBatch() (*types.BlockBatch, error)
	UpdateRollupStatus(ctx context.Context, hash string, status types.RollupStatus) error
	UpdateCommitTxHashAndRollupStatus(ctx context.Context, hash string, commitTxHash string, status types.RollupStatus) error
	UpdateFinalizeTxHashAndRollupStatus(ctx context.Context, hash string, finalizeTxHash string, status types.RollupStatus) error
	GetAssignedBatchHashes() ([]string, error)
	UpdateSkippedBatches() (int64, error)
	GetBatchCount() (int64, error)

	UpdateL2OracleTxHash(ctx context.Context, hash, txHash string) error
	UpdateL2GasOracleStatus(ctx context.Context, hash string, status types.GasOracleStatus) error
	UpdateL2GasOracleStatusAndOracleTxHash(ctx context.Context, hash string, status types.GasOracleStatus, txHash string) error

	GetCommitTxHash(hash string) (sql.NullString, error)   // for unit tests only
	GetFinalizeTxHash(hash string) (sql.NullString, error) // for unit tests only
}

// L1MessageOrm is layer1 message db interface
type L1MessageOrm interface {
	GetL1MessageByQueueIndex(queueIndex uint64) (*types.L1Message, error)
	GetL1MessageByMsgHash(msgHash string) (*types.L1Message, error)
	GetL1MessagesByStatus(status types.MsgStatus, limit uint64) ([]*types.L1Message, error)
	GetL1ProcessedQueueIndex() (int64, error)
	SaveL1Messages(ctx context.Context, messages []*types.L1Message) error
	UpdateLayer2Hash(ctx context.Context, msgHash string, layer2Hash string) error
	UpdateLayer1Status(ctx context.Context, msgHash string, status types.MsgStatus) error
	UpdateLayer1StatusAndLayer2Hash(ctx context.Context, msgHash string, status types.MsgStatus, layer2Hash string) error
	GetLayer1LatestWatchedHeight() (int64, error)

	GetRelayL1MessageTxHash(queueIndex uint64) (sql.NullString, error) // for unit tests only
}

// L2MessageOrm is layer2 message db interface
type L2MessageOrm interface {
	GetL2MessageByNonce(nonce uint64) (*types.L2Message, error)
	GetL2MessageByMsgHash(msgHash string) (*types.L2Message, error)
	GetL2MessageProofByNonce(nonce uint64) (sql.NullString, error)
	GetL2Messages(fields map[string]interface{}, args ...string) ([]*types.L2Message, error)
	GetL2ProcessedNonce() (int64, error)
	SaveL2Messages(ctx context.Context, messages []*types.L2Message) error
	GetLastL2MessageNonceLEHeight(ctx context.Context, height uint64) (sql.NullInt64, error)
	GetL2MessagesBetween(ctx context.Context, startHeight, finishHeight uint64) ([]*types.L2Message, error)
	UpdateLayer1Hash(ctx context.Context, msgHash string, layer1Hash string) error
	UpdateLayer2Status(ctx context.Context, msgHash string, status types.MsgStatus) error
	UpdateLayer2StatusAndLayer1Hash(ctx context.Context, msgHash string, status types.MsgStatus, layer1Hash string) error
	UpdateL2MessageProof(ctx context.Context, nonce uint64, proof string) error
	UpdateL2MessageProofInDbTx(ctx context.Context, dbTx *sqlx.Tx, msgHash, proof string) error
	GetLayer2LatestWatchedHeight() (int64, error)

	GetRelayL2MessageTxHash(nonce uint64) (sql.NullString, error) // for unit tests only
}
