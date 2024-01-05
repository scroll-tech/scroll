package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rpc"
	"github.com/urfave/cli/v2"

	"scroll-tech/common/database"
	"scroll-tech/common/observability"
	"scroll-tech/common/utils"
	"scroll-tech/common/version"

	"scroll-tech/rollup/internal/config"
	"scroll-tech/rollup/internal/controller/relayer"
	"scroll-tech/rollup/internal/controller/watcher"
	butils "scroll-tech/rollup/internal/utils"
)

var app *cli.App

func init() {
	// Set up gas-oracle app info.
	app = cli.NewApp()
	app.Action = action
	app.Name = "gas-oracle"
	app.Usage = "The Scroll Gas Oracle"
	app.Description = "Scroll Gas Oracle."
	app.Version = version.Version
	app.Flags = append(app.Flags, utils.CommonFlags...)
	app.Commands = []*cli.Command{}
	app.Before = func(ctx *cli.Context) error {
		return utils.LogSetup(ctx)
	}
	// Register `gas-oracle-test` app for integration-test.
	utils.RegisterSimulation(app, utils.GasOracleApp)
}

func action(ctx *cli.Context) error {
	// Load config file.
	cfgFile := ctx.String(utils.ConfigFileFlag.Name)
	cfg, err := config.NewConfig(cfgFile)
	if err != nil {
		log.Crit("failed to load config file", "config file", cfgFile, "error", err)
	}
	instanceCtx, instanceCancel := context.WithCancel(ctx.Context)
	loopCtx, loopCancel := context.WithCancel(ctx.Context)
	// Init db connection
	db, err := database.InitDB(cfg.DBConfig)
	if err != nil {
		log.Crit("failed to init db connection", "err", err)
	}

	registry := prometheus.DefaultRegisterer
	observability.Server(ctx, db)

	l1client, err := ethclient.Dial(cfg.L1Config.Endpoint)
	if err != nil {
		log.Crit("failed to connect l1 geth", "config file", cfgFile, "error", err)
	}

	// Init l2geth connection
	l2client, err := ethclient.Dial(cfg.L2Config.Endpoint)
	if err != nil {
		log.Crit("failed to connect l2 geth", "config file", cfgFile, "error", err)
	}

	l1watcher := watcher.NewL1WatcherClient(instanceCtx, l1client, cfg.L1Config.StartHeight, cfg.L1Config.Confirmations, cfg.L1Config.L1MessageQueueAddress, cfg.L1Config.ScrollChainContractAddress, db, registry)

	l1relayer, err := relayer.NewLayer1Relayer(instanceCtx, db, cfg.L1Config.RelayerConfig, registry)
	if err != nil {
		log.Crit("failed to create new l1 relayer", "config file", cfgFile, "error", err)
	}
	l2relayer, err := relayer.NewLayer2Relayer(instanceCtx, l2client, db, cfg.L2Config.RelayerConfig, false /* initGenesis */, registry)
	if err != nil {
		log.Crit("failed to create new l2 relayer", "config file", cfgFile, "error", err)
	}
	// Start l1 watcher process
	go utils.LoopWithContext(loopCtx, 10*time.Second, func(ctx context.Context) {
		// Fetch the latest block number to decrease the delay when fetching gas prices
		// Use latest block number - 1 to prevent frequent reorg
		number, loopErr := butils.GetLatestConfirmedBlockNumber(ctx, l1client, rpc.LatestBlockNumber)
		if loopErr != nil {
			log.Error("failed to get block number", "err", loopErr)
			return
		}

		if loopErr = l1watcher.FetchBlockHeader(number - 1); loopErr != nil {
			log.Error("Failed to fetch L1 block header", "lastest", number-1, "err", loopErr)
		}
	})

	// Goroutines that may send transactions periodically.
	go utils.Loop(loopCtx, 10*time.Second, l1relayer.ProcessGasPriceOracle)
	go utils.Loop(loopCtx, 2*time.Second, l2relayer.ProcessGasPriceOracle)

	defer func() {
		log.Info("Graceful shutdown initiated")

		// Prevent new transactions by canceling the loop context.
		loopCancel()

		// Close relayers to ensure all pending transactions are processed.
		// This includes any in-flight transactions that have not yet been confirmed.
		l1relayer.Close()
		l2relayer.Close()

		// Halt confirmation signal handling by canceling the instance context.
		instanceCancel()

		// Close the database connection.
		if err = database.CloseDB(db); err != nil {
			log.Error("Failed to close database connection", "error", err)
		}

		log.Info("Graceful shutdown done")
	}()

	// Finish start all message relayer functions
	log.Info("Start gas-oracle successfully")

	// Catch CTRL-C to ensure a graceful shutdown.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Wait until the interrupt signal is received from an OS signal.
	<-interrupt

	return nil
}

// Run message_relayer cmd instance.
func Run() {
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
