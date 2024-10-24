package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rollup/l1"
	"github.com/urfave/cli/v2"
	"gorm.io/gorm"

	"scroll-tech/common/database"
	"scroll-tech/common/observability"
	"scroll-tech/common/utils"
	"scroll-tech/common/version"
	"scroll-tech/database/migrate"
	"scroll-tech/rollup/internal/config"
	"scroll-tech/rollup/internal/controller/watcher"
	"scroll-tech/rollup/internal/orm"
)

var app *cli.App

func init() {
	// Set up rollup-relayer app info.
	app = cli.NewApp()
	app.Action = action
	app.Name = "permissionless-batches"
	app.Usage = "The Scroll Rollup Relayer for permissionless batch production"
	app.Version = version.Version
	app.Flags = append(app.Flags, utils.CommonFlags...)
	app.Flags = append(app.Flags, utils.RollupRelayerFlags...)
	app.Commands = []*cli.Command{}
	app.Before = func(ctx *cli.Context) error {
		return utils.LogSetup(ctx)
	}
}

func action(ctx *cli.Context) error {
	// Load config file.
	cfgFile := ctx.String(utils.ConfigFileFlag.Name)
	cfg, err := config.NewConfig(cfgFile)
	if err != nil {
		log.Crit("failed to load config file", "config file", cfgFile, "error", err)
	}

	subCtx, cancel := context.WithCancel(ctx.Context)
	defer cancel()

	db, err := initDB(cfg)
	if err != nil {
		return fmt.Errorf("failed to init db: %w", err)
	}
	defer func() {
		if err = database.CloseDB(db); err != nil {
			log.Crit("failed to close db connection", "error", err)
		}
	}()

	registry := prometheus.DefaultRegisterer
	observability.Server(ctx, db)

	genesisPath := ctx.String(utils.Genesis.Name)
	genesis, err := utils.ReadGenesis(genesisPath)
	if err != nil {
		log.Crit("failed to read genesis", "genesis file", genesisPath, "error", err)
	}

	chunkProposer := watcher.NewChunkProposer(subCtx, cfg.L2Config.ChunkProposerConfig, genesis.Config, db, registry)
	batchProposer := watcher.NewBatchProposer(subCtx, cfg.L2Config.BatchProposerConfig, genesis.Config, db, registry)
	//bundleProposer := watcher.NewBundleProposer(subCtx, cfg.L2Config.BundleProposerConfig, genesis.Config, db, registry)

	fmt.Println(cfg.L1Config)
	fmt.Println(cfg.L2Config)
	fmt.Println(cfg.DBConfig)
	fmt.Println(cfg.RecoveryConfig)

	// Restore minimal previous state required to be able to create new chunks, batches and bundles.
	latestFinalizedChunk, latestFinalizedBatch, err := restoreMinimalPreviousState(cfg, chunkProposer, batchProposer)
	if err != nil {
		return fmt.Errorf("failed to restore minimal previous state: %w", err)
	}

	// Fetch and insert the missing blocks from the last block in the latestFinalizedBatch to the latest L2 block.
	fromBlock := latestFinalizedChunk.EndBlockNumber + 1
	toBlock, err := fetchL2Blocks(subCtx, cfg, genesis, db, registry, fromBlock, cfg.RecoveryConfig.L2BlockHeightLimit)
	if err != nil {
		return fmt.Errorf("failed to fetch L2 blocks: %w", err)
	}

	fmt.Println(latestFinalizedChunk.Index, latestFinalizedBatch.Index, fromBlock, toBlock)

	// Create chunks for L2 blocks.
	log.Info("Creating chunks for L2 blocks", "from", fromBlock, "to", toBlock)

	var latestChunk *orm.Chunk
	var count int
	for {
		if err = chunkProposer.ProposeChunk(); err != nil {
			return fmt.Errorf("failed to propose chunk: %w", err)
		}
		count++

		latestChunk, err = chunkProposer.ChunkORM().GetLatestChunk(subCtx)
		if err != nil {
			return fmt.Errorf("failed to get latest latestFinalizedChunk: %w", err)
		}

		log.Info("Chunk created", "index", latestChunk.Index, "hash", latestChunk.Hash, "StartBlockNumber", latestChunk.StartBlockNumber, "EndBlockNumber", latestChunk.EndBlockNumber, "TotalL1MessagesPoppedBefore", latestChunk.TotalL1MessagesPoppedBefore)

		// We have created chunks for all available L2 blocks.
		if latestChunk.EndBlockNumber >= toBlock {
			break
		}
	}

	log.Info("Chunks created", "count", count, "latest latestFinalizedChunk", latestChunk.Index, "hash", latestChunk.Hash, "StartBlockNumber", latestChunk.StartBlockNumber, "EndBlockNumber", latestChunk.EndBlockNumber, "TotalL1MessagesPoppedBefore", latestChunk.TotalL1MessagesPoppedBefore)

	// Create batch for the created chunks. We only allow 1 batch it needs to be submitted (and finalized) with a proof in a single step.
	log.Info("Creating batch for chunks", "from", latestFinalizedChunk.Index+1, "to", latestChunk.Index)

	batchProposer.TryProposeBatch()
	latestBatch, err := batchProposer.BatchORM().GetLatestBatch(subCtx)
	if err != nil {
		return fmt.Errorf("failed to get latest latestFinalizedBatch: %w", err)
	}

	if latestBatch.EndChunkIndex != latestChunk.Index {
		return fmt.Errorf("latest chunk in produced batch %d != %d, too many L2 blocks - specify less L2 blocks and retry again", latestBatch.EndChunkIndex, latestChunk.Index)
	}

	log.Info("Batch created", "index", latestBatch.Index, "hash", latestBatch.Hash, "StartChunkIndex", latestBatch.StartChunkIndex, "EndChunkIndex", latestBatch.EndChunkIndex)

	// Catch CTRL-C to ensure a graceful shutdown.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Wait until the interrupt signal is received from an OS signal.
	<-interrupt

	return nil
}

// Run rollup relayer cmd instance.
func Run() {
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func initDB(cfg *config.Config) (*gorm.DB, error) {
	// init db connection
	db, err := database.InitDB(cfg.DBConfig)
	if err != nil {
		log.Crit("failed to init db connection", "err", err)
	}

	// make sure we are starting from a fresh DB
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed ")
	}

	// reset and init DB
	var v int64
	err = migrate.Rollback(sqlDB, &v)
	if err != nil {
		return nil, fmt.Errorf("failed to rollback db: %w", err)
	}

	err = migrate.Migrate(sqlDB)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate db: %w", err)
	}

	return db, nil
}

func fetchL2Blocks(ctx context.Context, cfg *config.Config, genesis *core.Genesis, db *gorm.DB, registry prometheus.Registerer, fromBlock uint64, l2BlockHeightLimit uint64) (uint64, error) {
	if l2BlockHeightLimit > 0 && fromBlock > l2BlockHeightLimit {
		return 0, fmt.Errorf("fromBlock (latest finalized L2 block) is higher than specified L2BlockHeightLimit: %d > %d", fromBlock, l2BlockHeightLimit)
	}

	log.Info("Fetching L2 blocks with", "fromBlock", fromBlock, "l2BlockHeightLimit", l2BlockHeightLimit)

	// Init l2geth connection
	l2client, err := ethclient.Dial(cfg.L2Config.Endpoint)
	if err != nil {
		return 0, fmt.Errorf("failed to connect to L2geth at RPC=%s: %w", cfg.L2Config.Endpoint, err)
	}

	l2Watcher := watcher.NewL2WatcherClient(ctx, l2client, cfg.L2Config.Confirmations, cfg.L2Config.L2MessageQueueAddress, cfg.L2Config.WithdrawTrieRootSlot, genesis.Config, db, registry)

	// Fetch and insert the missing blocks from the last block in the batch to the latest L2 block.
	latestL2Block, err := l2Watcher.Client.BlockNumber(context.Background())
	if err != nil {
		return 0, fmt.Errorf("failed to get latest L2 block number: %w", err)
	}

	log.Info("Latest L2 block number", "latest L2 block", latestL2Block)

	if l2BlockHeightLimit > latestL2Block {
		return 0, fmt.Errorf("l2BlockHeightLimit is higher than the latest L2 block number, not all blocks are available in L2geth: %d > %d", l2BlockHeightLimit, latestL2Block)
	}

	toBlock := latestL2Block
	if l2BlockHeightLimit > 0 {
		toBlock = l2BlockHeightLimit
	}

	err = l2Watcher.GetAndStoreBlocks(context.Background(), fromBlock, toBlock)
	if err != nil {
		return 0, fmt.Errorf("failed to get and store blocks: %w", err)
	}

	log.Info("Fetched L2 blocks from", "fromBlock", fromBlock, "toBlock", toBlock)

	return toBlock, nil
}

// restoreMinimalPreviousState restores the minimal previous state required to be able to create new chunks, batches and bundles.
func restoreMinimalPreviousState(cfg *config.Config, chunkProposer *watcher.ChunkProposer, batchProposer *watcher.BatchProposer) (*orm.Chunk, *orm.Batch, error) {
	log.Info("Restoring previous state with", "L1 block height", cfg.RecoveryConfig.L1BlockHeight, "latest finalized batch", cfg.RecoveryConfig.LatestFinalizedBatch)

	// TODO: make these parameters -> part of genesis config?
	scrollChainAddress := common.HexToAddress("0x2D567EcE699Eabe5afCd141eDB7A4f2D0D6ce8a0")
	l1MessageQueueAddress := common.HexToAddress("0xF0B2293F5D834eAe920c6974D50957A1732de763")

	l1Client, err := ethclient.Dial(cfg.L1Config.Endpoint)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to L1 client: %w", err)
	}
	reader, err := l1.NewReader(context.Background(), l1.Config{
		ScrollChainAddress:    scrollChainAddress,
		L1MessageQueueAddress: l1MessageQueueAddress,
	}, l1Client)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create L1 reader: %w", err)
	}

	// 1. Sanity check user input: Make sure that the user's L1 block height is not higher than the latest finalized block number.
	latestFinalizedL1Block, err := reader.GetLatestFinalizedBlockNumber()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get latest finalized L1 block number: %w", err)
	}
	if cfg.RecoveryConfig.L1BlockHeight > latestFinalizedL1Block {
		return nil, nil, fmt.Errorf("specified L1 block height is higher than the latest finalized block number: %d > %d", cfg.RecoveryConfig.L1BlockHeight, latestFinalizedL1Block)
	}

	log.Info("Latest finalized L1 block number", "latest finalized L1 block", latestFinalizedL1Block)

	// 2. Make sure that the specified batch is indeed finalized on the L1 rollup contract and is the latest finalized batch.
	// TODO: enable check
	//latestFinalizedBatch, err := reader.LatestFinalizedBatch(latestFinalizedL1Block)
	//if cfg.RecoveryConfig.LatestFinalizedBatch != latestFinalizedBatch {
	//	return nil, nil, fmt.Errorf("batch %d is not the latest finalized batch: %d", cfg.RecoveryConfig.LatestFinalizedBatch, latestFinalizedBatch)
	//}

	var batchCommitEvent *l1.CommitBatchEvent
	err = reader.FetchRollupEventsInRangeWithCallback(cfg.RecoveryConfig.L1BlockHeight, latestFinalizedL1Block, func(event l1.RollupEvent) bool {
		if event.Type() == l1.CommitEventType && event.BatchIndex().Uint64() == cfg.RecoveryConfig.LatestFinalizedBatch {
			batchCommitEvent = event.(*l1.CommitBatchEvent)
			return false
		}

		return true
	})
	if batchCommitEvent == nil {
		return nil, nil, fmt.Errorf("commit event not found for batch %d", cfg.RecoveryConfig.LatestFinalizedBatch)
	}

	log.Info("Found commit event for batch", "batch", batchCommitEvent.BatchIndex(), "hash", batchCommitEvent.BatchHash(), "L1 block height", batchCommitEvent.BlockNumber(), "L1 tx hash", batchCommitEvent.TxHash())

	// 3. Fetch commit tx data for latest finalized batch.
	args, err := reader.FetchCommitTxData(batchCommitEvent)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch commit tx data: %w", err)
	}

	codec, err := encoding.CodecFromVersion(encoding.CodecVersion(args.Version))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get codec: %w", err)
	}

	daChunksRawTxs, err := codec.DecodeDAChunksRawTx(args.Chunks)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode DA chunks: %w", err)
	}
	lastChunk := daChunksRawTxs[len(daChunksRawTxs)-1]
	lastBlockInBatch := lastChunk.Blocks[len(lastChunk.Blocks)-1].Number()

	log.Info("Last L2 block in batch", "batch", batchCommitEvent.BatchIndex(), "L2 block", lastBlockInBatch)

	// 4. Get the L1 messages count after the latest finalized batch.
	l1MessagesCount, err := reader.FinalizedL1MessageQueueIndex(latestFinalizedL1Block)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get L1 messages count: %w", err)
	}
	// TODO: remove this. only for testing
	l1MessagesCount = 220853

	log.Info("L1 messages count after latest finalized batch", "batch", batchCommitEvent.BatchIndex(), "count", l1MessagesCount)

	// 5. Insert minimal state to DB.
	chunk, err := chunkProposer.ChunkORM().InsertChunkRaw(context.Background(), codec.Version(), lastChunk, l1MessagesCount)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to insert chunk raw: %w", err)
	}

	log.Info("Inserted last finalized chunk to DB", "chunk", chunk.Index, "hash", chunk.Hash, "StartBlockNumber", chunk.StartBlockNumber, "EndBlockNumber", chunk.EndBlockNumber, "TotalL1MessagesPoppedBefore", chunk.TotalL1MessagesPoppedBefore)

	batch, err := batchProposer.BatchORM().InsertBatchRaw(context.Background(), batchCommitEvent.BatchIndex(), batchCommitEvent.BatchHash(), codec.Version(), chunk)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to insert batch raw: %w", err)
	}

	log.Info("Inserted last finalized batch to DB", "batch", batch.Index, "hash", batch.Hash)

	return chunk, batch, nil
}

//docker run --rm -it \
// -e POSTGRES_HOST_AUTH_METHOD=trust \
// -e POSTGRES_DB=scroll \
// -v ${PWD}/db_data:/var/lib/postgresql/data \
// -p 5432:5432 \
// postgres
