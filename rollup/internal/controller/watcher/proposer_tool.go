package watcher

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/common"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/params"
	"gorm.io/gorm"

	"scroll-tech/database/migrate"

	"scroll-tech/common/database"
	"scroll-tech/common/utils"

	"scroll-tech/rollup/internal/config"
	"scroll-tech/rollup/internal/orm"
	rutils "scroll-tech/rollup/internal/utils"
)

// ProposerTool is a tool for proposing chunks and bundles to the L1 chain.
type ProposerTool struct {
	ctx    context.Context
	cancel context.CancelFunc

	db          *gorm.DB
	dbForReplay *gorm.DB
	client      *ethclient.Client

	chunkProposer  *ChunkProposer
	batchProposer  *BatchProposer
	bundleProposer *BundleProposer
}

// NewProposerTool creates a new ProposerTool instance.
func NewProposerTool(ctx context.Context, cancel context.CancelFunc, cfg *config.ConfigForReplay, startL2BlockHeight uint64, minCodecVersion encoding.CodecVersion, chainCfg *params.ChainConfig) (*ProposerTool, error) {
	// Init db connection
	db, err := database.InitDB(cfg.DBConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to init db connection: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get db connection: %w", err)
	}
	if err = migrate.ResetDB(sqlDB); err != nil {
		return nil, fmt.Errorf("failed to reset db: %w", err)
	}
	log.Info("successfully reset db")

	// Init dbForReplay connection
	dbForReplay, err := database.InitDB(cfg.DBConfigForReplay)
	if err != nil {
		return nil, fmt.Errorf("failed to init dbForReplay connection: %w", err)
	}

	client, err := ethclient.Dial(cfg.L2Config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to L2 geth, endpoint: %s, err: %w", cfg.L2Config.Endpoint, err)
	}

	prevChunk, err := orm.NewChunk(dbForReplay).GetParentChunkByBlockNumber(ctx, startL2BlockHeight)
	if err != nil {
		return nil, fmt.Errorf("failed to get previous chunk: %w", err)
	}

	var startQueueIndex uint64
	if prevChunk != nil {
		startQueueIndex = prevChunk.TotalL1MessagesPoppedBefore + prevChunk.TotalL1MessagesPoppedInChunk
	}

	startBlock := uint64(0)
	if prevChunk != nil {
		startBlock = prevChunk.EndBlockNumber + 1
	}

	var chunk *encoding.Chunk
	for blockNum := startBlock; blockNum <= startL2BlockHeight; blockNum++ {
		block, err := client.BlockByNumber(ctx, new(big.Int).SetUint64(blockNum))
		if err != nil {
			return nil, fmt.Errorf("failed to get block %d: %w", blockNum, err)
		}

		for _, tx := range block.Transactions() {
			if tx.Type() == gethTypes.L1MessageTxType {
				startQueueIndex++
			}
		}

		if blockNum == startL2BlockHeight {
			chunk = &encoding.Chunk{Blocks: []*encoding.Block{{Header: block.Header()}}}
		}
	}

	// Setting empty hash as the post_l1_message_queue_hash of the first chunk,
	// i.e., treating the first L1 message after this chunk as the first L1 message in message queue v2.
	// Though this setting is different from mainnet, it's simple yet sufficient for data analysis usage.
	_, err = orm.NewChunk(db).InsertTestChunkForProposerTool(ctx, chunk, minCodecVersion, startQueueIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to insert chunk, minCodecVersion: %d, startQueueIndex: %d, err: %w", minCodecVersion, startQueueIndex, err)
	}

	batch := &encoding.Batch{
		Index:                      0,
		TotalL1MessagePoppedBefore: 0,
		ParentBatchHash:            common.Hash{},
		Chunks:                     []*encoding.Chunk{chunk},
	}

	var dbBatch *orm.Batch
	dbBatch, err = orm.NewBatch(db).InsertBatch(ctx, batch, encoding.CodecV0, rutils.BatchMetrics{})
	if err != nil {
		return nil, fmt.Errorf("failed to insert batch: %w", err)
	}

	if err = orm.NewChunk(db).UpdateBatchHashInRange(ctx, 0, 0, dbBatch.Hash); err != nil {
		return nil, fmt.Errorf("failed to update batch hash for chunks: %w", err)
	}

	chunkProposer := NewChunkProposer(ctx, cfg.L2Config.ChunkProposerConfig, minCodecVersion, chainCfg, db, nil)
	chunkProposer.SetReplayDB(dbForReplay)
	batchProposer := NewBatchProposer(ctx, cfg.L2Config.BatchProposerConfig, minCodecVersion, chainCfg, db, nil)
	batchProposer.SetReplayDB(dbForReplay)
	bundleProposer := NewBundleProposer(ctx, cfg.L2Config.BundleProposerConfig, minCodecVersion, chainCfg, db, nil)

	return &ProposerTool{
		ctx:    ctx,
		cancel: cancel,

		db:          db,
		dbForReplay: dbForReplay,
		client:      client,

		chunkProposer:  chunkProposer,
		batchProposer:  batchProposer,
		bundleProposer: bundleProposer,
	}, nil
}

func (p *ProposerTool) Start() {
	go utils.Loop(p.ctx, 100*time.Millisecond, p.chunkProposer.TryProposeChunk)
	go utils.Loop(p.ctx, 100*time.Millisecond, p.batchProposer.TryProposeBatch)
	go utils.Loop(p.ctx, 100*time.Millisecond, p.bundleProposer.TryProposeBundle)
}

func (p *ProposerTool) Stop() {
	p.cancel()
	if err := database.CloseDB(p.db); err != nil {
		log.Error("failed to close db connection", "error", err)
	}
	if err := database.CloseDB(p.dbForReplay); err != nil {
		log.Error("failed to close dbForReplay connection", "error", err)
	}
	p.client.Close()
}
