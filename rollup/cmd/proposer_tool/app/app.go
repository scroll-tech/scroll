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
	"scroll-tech/common/types"
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

	genesisHeader, err := l2Client.HeaderByNumber(subCtx, big.NewInt(0))
	if err != nil {
		return fmt.Errorf("failed to retrieve L2 genesis header: %v", err)
	}

	chunk := &encoding.Chunk{
		Blocks: []*encoding.Block{{
			Header:         genesisHeader,
			Transactions:   nil,
			WithdrawRoot:   common.Hash{},
			RowConsumption: &gethTypes.RowConsumption{},
		}},
	}

	var dbChunk *orm.Chunk
	dbChunk, err = orm.NewChunk(db).InsertChunk(subCtx, chunk, encoding.CodecV0, rutils.ChunkMetrics{})
	if err != nil {
		log.Crit("failed to insert chunk", "error", err)
	}

	if err = orm.NewChunk(db).UpdateProvingStatus(subCtx, dbChunk.Hash, types.ProvingTaskVerified); err != nil {
		log.Crit("failed to update genesis chunk proving status", "error", err)
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
	chunkProposer := watcher.NewChunkProposer(subCtx, cfg.L2Config.ChunkProposerConfig, minCodecVersion, genesis.Config, dbForReplay, db, registry)
	batchProposer := watcher.NewBatchProposer(subCtx, cfg.L2Config.BatchProposerConfig, minCodecVersion, genesis.Config, dbForReplay, db, registry)
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
