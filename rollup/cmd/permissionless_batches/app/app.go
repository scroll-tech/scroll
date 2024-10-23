package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/common"
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

	// Init l2geth connection
	l2client, err := ethclient.Dial(cfg.L2Config.Endpoint)
	if err != nil {
		log.Crit("failed to connect l2 geth", "config file", cfgFile, "error", err)
	}

	genesisPath := ctx.String(utils.Genesis.Name)
	genesis, err := utils.ReadGenesis(genesisPath)
	if err != nil {
		log.Crit("failed to read genesis", "genesis file", genesisPath, "error", err)
	}

	chunkProposer := watcher.NewChunkProposer(subCtx, cfg.L2Config.ChunkProposerConfig, genesis.Config, db, registry)
	///batchProposer := watcher.NewBatchProposer(subCtx, cfg.L2Config.BatchProposerConfig, genesis.Config, db, registry)
	//bundleProposer := watcher.NewBundleProposer(subCtx, cfg.L2Config.BundleProposerConfig, genesis.Config, db, registry)

	l2watcher := watcher.NewL2WatcherClient(subCtx, l2client, cfg.L2Config.Confirmations, cfg.L2Config.L2MessageQueueAddress, cfg.L2Config.WithdrawTrieRootSlot, genesis.Config, db, registry)

	// Finish start all rollup relayer functions.
	log.Info("Start rollup-relayer successfully", "version", version.Version)

	fmt.Println(cfg.L1Config)
	fmt.Println(cfg.L2Config)
	fmt.Println(cfg.DBConfig)

	if err = restorePreviousState(cfg, db, l2watcher, chunkProposer); err != nil {
		log.Crit("failed to recover relayer", "error", err)
	}

	// TODO: fetch L2 blocks that will be used to propose the next chunks, batches and bundles.
	// x. Get and insert the missing blocks from the last block in the batch to the latest L2 block.
	//latestL2Block, err := l2Watcher.Client.BlockNumber(context.Background())
	//if err != nil {
	//	return fmt.Errorf("failed to get latest L2 block number: %w", err)
	//}

	//err = l2Watcher.GetAndStoreBlocks(context.Background(), lastBlockInBatch, latestL2Block)
	//if err != nil {
	//	return fmt.Errorf("failed to get and store blocks: %w", err)
	//}

	// TODO: maybe start new goroutine for chunk and batch proposers like usual and stop them once max L2 blocks reached

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

// restorePreviousState restores the minimal previous state required to be able to create new chunks, batches and bundles.
func restorePreviousState(cfg *config.Config, db *gorm.DB, l2Watcher *watcher.L2WatcherClient, chunkProposer *watcher.ChunkProposer) error {
	// TODO: make these parameters
	scrollChainAddress := common.HexToAddress("0x2D567EcE699Eabe5afCd141eDB7A4f2D0D6ce8a0")
	l1MessageQueueAddress := common.HexToAddress("0xF0B2293F5D834eAe920c6974D50957A1732de763")
	userl1BlockHeight := uint64(4141928)
	userBatch := uint64(9705)

	l1Client, err := ethclient.Dial(cfg.L1Config.Endpoint)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 client: %w", err)
	}
	reader, err := l1.NewReader(context.Background(), l1.Config{
		ScrollChainAddress:    scrollChainAddress,
		L1MessageQueueAddress: l1MessageQueueAddress,
	}, l1Client)
	if err != nil {
		return fmt.Errorf("failed to create L1 reader: %w", err)
	}

	// 1. Sanity check user input: Make sure that the user's L1 block height is not higher than the latest finalized block number.
	latestFinalizedBlock, err := reader.GetLatestFinalizedBlockNumber()
	if err != nil {
		return fmt.Errorf("failed to get latest finalized block number: %w", err)
	}
	if userl1BlockHeight > latestFinalizedBlock {
		return fmt.Errorf("user's L1 block height is higher than the latest finalized block number: %d > %d", userl1BlockHeight, latestFinalizedBlock)
	}

	fmt.Println("latestFinalizedBlock", latestFinalizedBlock)
	fmt.Println("userl1BlockHeight", userl1BlockHeight)

	// 2. Make sure that the specified batch is indeed finalized on the L1 rollup contract and is the latest finalized batch.
	// ---
	//events, err := reader.FetchRollupEventsInRange(userl1BlockHeight, latestFinalizedBlock)
	//if err != nil {
	//	return fmt.Errorf("failed to fetch rollup events: %w", err)
	//}
	//var foundFinalizeEvent bool
	//var latestFinalizedBatch uint64
	//var userCommitEvent *l1.CommitBatchEvent
	//
	//for _, event := range events {
	//	switch event.Type() {
	//	case l1.CommitEventType:
	//		if event.BatchIndex().Uint64() == userBatch {
	//			userCommitEvent = event.(*l1.CommitBatchEvent)
	//		}
	//
	//	case l1.FinalizeEventType:
	//		if event.BatchIndex().Uint64() == userBatch {
	//			foundFinalizeEvent = true
	//		}
	//		if event.BatchIndex().Uint64() > latestFinalizedBatch {
	//			latestFinalizedBatch = event.BatchIndex().Uint64()
	//		}
	//
	//	default:
	//		// ignore revert events
	//	}
	//}
	//if userCommitEvent == nil {
	//	return fmt.Errorf("commit event not found for batch %d", userBatch)
	//}
	//if !foundFinalizeEvent {
	//	return fmt.Errorf("finalize event not found for batch %d", userBatch)
	//}
	//if userBatch != latestFinalizedBatch {
	//	return fmt.Errorf("batch %d is not the latest finalized batch: %d", userBatch, latestFinalizedBatch)
	//}
	var foundFinalizeEvent bool
	var latestFinalizedBatch uint64
	var userCommitEvent *l1.CommitBatchEvent
	{
		err = reader.FetchRollupEventsInRangeWithCallback(userl1BlockHeight, latestFinalizedBlock, func(event l1.RollupEvent) bool {
			switch event.Type() {
			case l1.CommitEventType:
				if event.BatchIndex().Uint64() == userBatch {
					userCommitEvent = event.(*l1.CommitBatchEvent)
				}

			case l1.FinalizeEventType:
				if event.BatchIndex().Uint64() == userBatch {
					foundFinalizeEvent = true
				}
				if event.BatchIndex().Uint64() > latestFinalizedBatch {
					latestFinalizedBatch = event.BatchIndex().Uint64()
				}

			default:
				// ignore revert events
			}

			if foundFinalizeEvent && userCommitEvent != nil {
				return false
			}

			return true
		})
	}

	// 3. Fetch the commit tx data for latest finalized batch.
	args, err := reader.FetchCommitTxData(userCommitEvent)
	if err != nil {
		return fmt.Errorf("failed to fetch commit tx data: %w", err)
	}

	codec, err := encoding.CodecFromVersion(encoding.CodecVersion(args.Version))
	if err != nil {
		return fmt.Errorf("failed to get codec: %w", err)
	}

	daChunksRawTxs, err := codec.DecodeDAChunksRawTx(args.Chunks)
	if err != nil {
		return fmt.Errorf("failed to decode DA chunks: %w", err)
	}
	lastChunk := daChunksRawTxs[len(daChunksRawTxs)-1]
	lastBlockInBatch := lastChunk.Blocks[len(lastChunk.Blocks)-1].Number()

	// 4. Get the L1 messages count after the latest finalized batch.
	l1MessagesCount, err := reader.FinalizedL1MessageQueueIndex(latestFinalizedBlock)
	if err != nil {
		return fmt.Errorf("failed to get L1 messages count: %w", err)
	}

	for i, chunk := range daChunksRawTxs {
		fmt.Println("chunk", i)
		for j, block := range chunk.Blocks {
			fmt.Println("block", j)
			fmt.Println("block.Number", block.Number())
		}
	}

	// 5. Insert minimal state to DB.
	if err = chunkProposer.ChunkORM().InsertChunkRaw(context.Background(), codec.Version(), lastChunk, l1MessagesCount); err != nil {
		return fmt.Errorf("failed to insert chunk raw: %w", err)
	}

	fmt.Println("l1MessagesCount", l1MessagesCount)
	fmt.Println("lastBlockInBatch", lastBlockInBatch)
	//fmt.Println("latestL2Block", latestL2Block)

	// TODO:
	//  - reconstruct latest finalized batch
	//  - reconstruct latest finalized chunk
	//  - try to chunkProposer.TryProposeChunk() and batchProposer.TryProposeBatch()
	//  - instead of proposing bundle, we need to call a special method to propose the bundle that contains only produced batch

	// next steps: chunk size of 1 matches -> now try to get the correct batch hash for the next batch (compare with DB)
	// batch.Index = dbParentBatch.Index + 1
	//	batch.ParentBatchHash = common.HexToHash(dbParentBatch.Hash)
	//	batch.TotalL1MessagePoppedBefore = firstUnbatchedChunk.TotalL1MessagesPoppedBefore
	// insert this batch to DB -> refactor to be cleaner and based on latest da-codec
	// -> come up with a way to test this deterministically
	return nil
}

//docker run --rm -it \
// -e POSTGRES_HOST_AUTH_METHOD=trust \
// -e POSTGRES_DB=scroll \
// -v ${PWD}/db_data:/var/lib/postgresql/data \
// -p 5432:5432 \
// postgres
