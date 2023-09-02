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
	subCtx, cancel := context.WithCancel(ctx.Context)
	// Init db connection
	db, err := database.InitDB(cfg.DBConfig)
	if err != nil {
		log.Crit("failed to init db connection", "err", err)
	}
	defer func() {
		cancel()
		if err = database.CloseDB(db); err != nil {
			log.Error("can not close ormFactory", "error", err)
		}
	}()

	registry := prometheus.DefaultRegisterer
	metrics.Server(ctx, registry.(*prometheus.Registry))

	l1client, err := ethclient.Dial(cfg.L1Config.Endpoint)
	if err != nil {
		log.Error("failed to connect l1 geth", "config file", cfgFile, "error", err)
		return err
	}

	// Init l2geth connection
	l2client, err := ethclient.Dial(cfg.L2Config.Endpoint)
	if err != nil {
		log.Error("failed to connect l2 geth", "config file", cfgFile, "error", err)
		return err
	}

	l1watcher := watcher.NewL1WatcherClient(ctx.Context, l1client, cfg.L1Config.StartHeight, cfg.L1Config.Confirmations, cfg.L1Config.L1MessageQueueAddress, cfg.L1Config.ScrollChainContractAddress, db, registry)

	l1relayer, err := relayer.NewLayer1Relayer(ctx.Context, db, cfg.L1Config.RelayerConfig, registry)
	if err != nil {
		log.Error("failed to create new l1 relayer", "config file", cfgFile, "error", err)
		return err
	}
	l2relayer, err := relayer.NewLayer2Relayer(ctx.Context, l2client, db, cfg.L2Config.RelayerConfig, false /* initGenesis */, registry)
	if err != nil {
		log.Error("failed to create new l2 relayer", "config file", cfgFile, "error", err)
		return err
	}
	// Start l1 watcher process
	go utils.LoopWithContext(subCtx, 10*time.Second, func(ctx context.Context) {
		number, loopErr := butils.GetLatestConfirmedBlockNumber(ctx, l1client, cfg.L1Config.Confirmations)
		if loopErr != nil {
			log.Error("failed to get block number", "err", loopErr)
			return
		}

		if loopErr = l1watcher.FetchBlockHeader(number); loopErr != nil {
			log.Error("Failed to fetch L1 block header", "lastest", number, "err", loopErr)
		}
	})

	// Start l1relayer process
	go utils.Loop(subCtx, 10*time.Second, l1relayer.ProcessGasPriceOracle)
	go utils.Loop(subCtx, 2*time.Second, l2relayer.ProcessGasPriceOracle)

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
