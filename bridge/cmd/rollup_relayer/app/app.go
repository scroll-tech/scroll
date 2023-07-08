package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"scroll-tech/common/database"
	"scroll-tech/common/metrics"
	"scroll-tech/common/utils"
	"scroll-tech/common/version"

	"scroll-tech/bridge/internal/config"
	"scroll-tech/bridge/internal/controller/relayer"
	"scroll-tech/bridge/internal/controller/watcher"
	butils "scroll-tech/bridge/internal/utils"
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
	dbHandler, err := database.InitDB(cfg.DBConfig)
	if err != nil {
		log.Crit("failed to init db connection", "err", err)
	}
	defer func() {
		cancel()
		if err = database.CloseDB(dbHandler); err != nil {
			log.Error("can not close ormFactory", "error", err)
		}
	}()

	// Start metrics server.
	metrics.Serve(subCtx, ctx)

	// Init l2geth connection
	l2client, err := ethclient.Dial(cfg.L2Config.Endpoint)
	if err != nil {
		log.Error("failed to connect l2 geth", "config file", cfgFile, "error", err)
		return err
	}

	initGenesis := ctx.Bool(utils.ImportGenesisFlag.Name)
	l2relayer, err := relayer.NewLayer2Relayer(ctx.Context, l2client, dbHandler, cfg.L2Config.RelayerConfig, initGenesis)
	if err != nil {
		log.Error("failed to create l2 relayer", "config file", cfgFile, "error", err)
		return err
	}

	chunkProposer := watcher.NewChunkProposer(subCtx, cfg.L2Config.ChunkProposerConfig, dbHandler)
	if err != nil {
		log.Error("failed to create chunkProposer", "config file", cfgFile, "error", err)
		return err
	}

	batchProposer := watcher.NewBatchProposer(subCtx, cfg.L2Config.BatchProposerConfig, dbHandler)
	if err != nil {
		log.Error("failed to create batchProposer", "config file", cfgFile, "error", err)
		return err
	}

	l2watcher := watcher.NewL2WatcherClient(subCtx, l2client, cfg.L2Config.Confirmations, cfg.L2Config.L2MessengerAddress,
		cfg.L2Config.L2MessageQueueAddress, cfg.L2Config.WithdrawTrieRootSlot, dbHandler)

	// Watcher loop to fetch missing blocks
	go utils.LoopWithContext(subCtx, 2*time.Second, func(ctx context.Context) {
		number, loopErr := butils.GetLatestConfirmedBlockNumber(ctx, l2client, cfg.L2Config.Confirmations)
		if loopErr != nil {
			log.Error("failed to get block number", "err", loopErr)
			return
		}
		l2watcher.TryFetchRunningMissingBlocks(number)
	})

	go utils.Loop(subCtx, 2*time.Second, chunkProposer.TryProposeChunk)

	go utils.Loop(subCtx, 2*time.Second, batchProposer.TryProposeBatch)

	go utils.Loop(subCtx, 2*time.Second, l2relayer.ProcessPendingBatches)

	go utils.Loop(subCtx, 2*time.Second, l2relayer.ProcessCommittedBatches)

	// Finish start all rollup relayer functions.
	log.Info("Start rollup-relayer successfully")

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
