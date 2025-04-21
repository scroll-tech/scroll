package app

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"os/signal"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/common"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"scroll-tech/common/database"
	"scroll-tech/common/utils"
	"scroll-tech/common/version"
	"scroll-tech/database/migrate"

	"scroll-tech/rollup/internal/config"
	"scroll-tech/rollup/internal/controller/watcher"
	"scroll-tech/rollup/internal/orm"
	rutils "scroll-tech/rollup/internal/utils"
)

var app *cli.App

func init() {
	// Set up proposer-tool app info.
	app = cli.NewApp()
	app.Action = action
	app.Name = "proposer-tool"
	app.Usage = "The Scroll Proposer Tool"
	app.Version = version.Version
	app.Flags = append(app.Flags, utils.CommonFlags...)
	app.Flags = append(app.Flags, utils.RollupRelayerFlags...)
	app.Flags = append(app.Flags, utils.ProposerToolFlags...)
	app.Commands = []*cli.Command{}
	app.Before = func(ctx *cli.Context) error {
		return utils.LogSetup(ctx)
	}
}

func action(ctx *cli.Context) error {
	// Load config file.
	cfgFile := ctx.String(utils.ConfigFileFlag.Name)
	cfg, err := config.NewConfigForReplay(cfgFile)
	if err != nil {
		log.Crit("failed to load config file", "config file", cfgFile, "error", err)
	}

	subCtx, cancel := context.WithCancel(ctx.Context)
	// Init db connection
	db, err := database.InitDB(cfg.DBConfig)
	if err != nil {
		log.Crit("failed to init db connection", "err", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Crit("failed to get db connection", "error", err)
	}
	if err = migrate.ResetDB(sqlDB); err != nil {
		log.Crit("failed to reset db", "error", err)
	}
	log.Info("successfully reset db")
	defer func() {
		cancel()
		if err = database.CloseDB(db); err != nil {
			log.Crit("failed to close db connection", "error", err)
		}
	}()

	// Init dbForReplay connection
	dbForReplay, err := database.InitDB(cfg.DBConfigForReplay)
	if err != nil {
		log.Crit("failed to init dbForReplay connection", "err", err)
	}
	defer func() {
		cancel()
		if err = database.CloseDB(dbForReplay); err != nil {
			log.Crit("failed to close dbForReplay connection", "error", err)
		}
	}()

	// Init l2geth connection
	l2Client, err := ethclient.Dial(cfg.L2Config.Endpoint)
	if err != nil {
		log.Crit("failed to connect l2 geth", "config file", cfgFile, "error", err)
	}

	startL2BlockHeight := ctx.Uint64(utils.StartL2BlockFlag.Name)

	prevChunk, err := orm.NewChunk(dbForReplay).GetParentChunkByBlockNumber(subCtx, startL2BlockHeight)
	if err != nil {
		log.Crit("failed to get previous chunk", "error", err)
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
		block, err := l2Client.BlockByNumber(context.Background(), big.NewInt(int64(blockNum)))
		if err != nil {
			log.Crit("failed to get block", "block number", blockNum, "error", err)
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
	_, err = orm.NewChunk(db).InsertTestChunkForProposerTool(subCtx, chunk, encoding.CodecV0, startQueueIndex)
	if err != nil {
		log.Crit("failed to insert chunk", "error", err)
	}

	batch := &encoding.Batch{
		Index:                      0,
		TotalL1MessagePoppedBefore: 0,
		ParentBatchHash:            common.Hash{},
		Chunks:                     []*encoding.Chunk{chunk},
	}

	var dbBatch *orm.Batch
	dbBatch, err = orm.NewBatch(db).InsertBatch(subCtx, batch, encoding.CodecV0, rutils.BatchMetrics{})
	if err != nil {
		log.Crit("failed to insert batch", "error", err)
	}

	if err = orm.NewChunk(db).UpdateBatchHashInRange(subCtx, 0, 0, dbBatch.Hash); err != nil {
		log.Crit("failed to update batch hash for chunks", "error", err)
	}

	registry := prometheus.DefaultRegisterer

	genesisPath := ctx.String(utils.Genesis.Name)
	genesis, err := utils.ReadGenesis(genesisPath)
	if err != nil {
		log.Crit("failed to read genesis", "genesis file", genesisPath, "error", err)
	}

	// sanity check config
	if cfg.L2Config.BatchProposerConfig.MaxChunksPerBatch <= 0 {
		log.Crit("cfg.L2Config.BatchProposerConfig.MaxChunksPerBatch must be greater than 0")
	}
	if cfg.L2Config.ChunkProposerConfig.MaxL2GasPerChunk <= 0 {
		log.Crit("cfg.L2Config.ChunkProposerConfig.MaxL2GasPerChunk must be greater than 0")
	}

	minCodecVersion := encoding.CodecVersion(ctx.Uint(utils.MinCodecVersionFlag.Name))
	chunkProposer := watcher.NewChunkProposer(subCtx, cfg.L2Config.ChunkProposerConfig, minCodecVersion, genesis.Config, db, registry)
	chunkProposer.SetReplayDB(dbForReplay)
	batchProposer := watcher.NewBatchProposer(subCtx, cfg.L2Config.BatchProposerConfig, minCodecVersion, genesis.Config, db, registry)
	batchProposer.SetReplayDB(dbForReplay)
	bundleProposer := watcher.NewBundleProposer(subCtx, cfg.L2Config.BundleProposerConfig, minCodecVersion, genesis.Config, db, registry)

	go utils.Loop(subCtx, 100*time.Millisecond, chunkProposer.TryProposeChunk)
	go utils.Loop(subCtx, 100*time.Millisecond, batchProposer.TryProposeBatch)
	go utils.Loop(subCtx, 100*time.Millisecond, bundleProposer.TryProposeBundle)

	// Finish start all proposer tool functions.
	log.Info("Start proposer-tool successfully", "version", version.Version)

	// Catch CTRL-C to ensure a graceful shutdown.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Wait until the interrupt signal is received from an OS signal.
	<-interrupt

	return nil
}

// Run proposer tool cmd instance.
func Run() {
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
