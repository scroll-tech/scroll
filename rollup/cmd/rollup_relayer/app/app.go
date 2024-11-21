package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"scroll-tech/common/database"
	"scroll-tech/common/observability"
	"scroll-tech/common/utils"
	"scroll-tech/common/version"

	"scroll-tech/rollup/internal/config"
	"scroll-tech/rollup/internal/controller/relayer"
	"scroll-tech/rollup/internal/controller/watcher"
	rutils "scroll-tech/rollup/internal/utils"
)

var app *cli.App

func init() {
	// Set up rollup-relayer app info.
	app = cli.NewApp()
	app.Action = action
	app.Name = "rollup-relayer"
	app.Usage = "The Scroll Rollup Relayer"
	app.Version = version.Version
	app.Flags = append(app.Flags, utils.CommonFlags...)
	app.Flags = append(app.Flags, utils.RollupRelayerFlags...)
	app.Commands = []*cli.Command{}
	app.Before = func(ctx *cli.Context) error {
		return utils.LogSetup(ctx)
	}
	// Register `rollup-relayer-test` app for integration-test.
	utils.RegisterSimulation(app, utils.RollupRelayerApp)
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
	defer func() {
		cancel()
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

	initGenesis := ctx.Bool(utils.ImportGenesisFlag.Name)
	l2relayer, err := relayer.NewLayer2Relayer(ctx.Context, l2client, db, cfg.L2Config.RelayerConfig, genesis.Config, initGenesis, relayer.ServiceTypeL2RollupRelayer, registry)
	if err != nil {
		log.Crit("failed to create l2 relayer", "config file", cfgFile, "error", err)
	}

	minCodecVersion := encoding.CodecVersion(ctx.Uint(utils.MinCodecVersionFlag.Name))
	chunkProposer := watcher.NewChunkProposer(subCtx, cfg.L2Config.ChunkProposerConfig, minCodecVersion, genesis.Config, db, registry)
	batchProposer := watcher.NewBatchProposer(subCtx, cfg.L2Config.BatchProposerConfig, minCodecVersion, genesis.Config, db, registry)
	bundleProposer := watcher.NewBundleProposer(subCtx, cfg.L2Config.BundleProposerConfig, minCodecVersion, genesis.Config, db, registry)

	l2watcher := watcher.NewL2WatcherClient(subCtx, l2client, cfg.L2Config.Confirmations, cfg.L2Config.L2MessageQueueAddress, cfg.L2Config.WithdrawTrieRootSlot, genesis.Config, db, registry)

	// Watcher loop to fetch missing blocks
	go utils.LoopWithContext(subCtx, 2*time.Second, func(ctx context.Context) {
		number, loopErr := rutils.GetLatestConfirmedBlockNumber(ctx, l2client, cfg.L2Config.Confirmations)
		if loopErr != nil {
			log.Error("failed to get block number", "err", loopErr)
			return
		}
		l2watcher.TryFetchRunningMissingBlocks(number)
	})

	go utils.Loop(subCtx, time.Duration(cfg.L2Config.ChunkProposerConfig.ProposeIntervalMilliseconds)*time.Millisecond, chunkProposer.TryProposeChunk)

	go utils.Loop(subCtx, time.Duration(cfg.L2Config.BatchProposerConfig.ProposeIntervalMilliseconds)*time.Millisecond, batchProposer.TryProposeBatch)

	go utils.Loop(subCtx, 10*time.Second, bundleProposer.TryProposeBundle)

	go utils.Loop(subCtx, 2*time.Second, l2relayer.ProcessPendingBatches)

	go utils.Loop(subCtx, 15*time.Second, l2relayer.ProcessPendingBundles)

	// Finish start all rollup relayer functions.
	log.Info("Start rollup-relayer successfully", "version", version.Version)

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
