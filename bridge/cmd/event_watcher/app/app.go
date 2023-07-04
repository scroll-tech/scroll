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

	"scroll-tech/common/metrics"
	cutils "scroll-tech/common/utils"
	"scroll-tech/common/version"

	"scroll-tech/bridge/internal/config"
	"scroll-tech/bridge/internal/controller/watcher"
	"scroll-tech/bridge/internal/utils"
)

var (
	app    *cli.App
	logger log.Logger
)

func init() {
	// Set up event-watcher app info.
	app = cli.NewApp()
	app.Action = action
	app.Name = "event-watcher"
	app.Usage = "The Scroll Event Watcher"
	app.Version = version.Version
	app.Flags = append(app.Flags, cutils.CommonFlags...)
	app.Commands = []*cli.Command{}
	app.Before = func(ctx *cli.Context) error {
		var err error
		logger, err = cutils.LogSetup(ctx)
		return err
	}
	// Register `event-watcher-test` app for integration-test.
	cutils.RegisterSimulation(app, cutils.EventWatcherApp)
}

func action(ctx *cli.Context) error {
	// Load config file.
	cfgFile := ctx.String(cutils.ConfigFileFlag.Name)
	cfg, err := config.NewConfig(cfgFile)
	if err != nil {
		log.Crit("failed to load config file", "config file", cfgFile, "error", err)
	}

	subCtx, cancel := context.WithCancel(ctx.Context)
	// Init db connection
	db, err := utils.InitDB(cfg.DBConfig, logger)
	if err != nil {
		log.Crit("failed to init db connection", "err", err)
	}
	defer func() {
		cancel()
		if err = utils.CloseDB(db); err != nil {
			log.Error("can not close ormFactory", "error", err)
		}
	}()

	// Start metrics server.
	metrics.Serve(subCtx, ctx)
	l1client, err := ethclient.Dial(cfg.L1Config.Endpoint)
	if err != nil {
		log.Error("failed to connect l1 geth", "config file", cfgFile, "error", err)
		return err
	}

	l2client, err := ethclient.Dial(cfg.L2Config.Endpoint)
	if err != nil {
		log.Error("failed to connect l2 geth", "config file", cfgFile, "error", err)
		return err
	}
	l1watcher := watcher.NewL1WatcherClient(ctx.Context, l1client, cfg.L1Config.StartHeight, cfg.L1Config.Confirmations, cfg.L1Config.L1MessengerAddress, cfg.L1Config.L1MessageQueueAddress, cfg.L1Config.ScrollChainContractAddress, db)
	l2watcher := watcher.NewL2WatcherClient(ctx.Context, l2client, cfg.L2Config.Confirmations, cfg.L2Config.L2MessengerAddress, cfg.L2Config.L2MessageQueueAddress, cfg.L2Config.WithdrawTrieRootSlot, db)

	go cutils.Loop(subCtx, 10*time.Second, func() {
		if loopErr := l1watcher.FetchContractEvent(); loopErr != nil {
			log.Error("Failed to fetch bridge contract", "err", loopErr)
		}
	})

	// Start l2 watcher process
	go cutils.Loop(subCtx, 2*time.Second, l2watcher.FetchContractEvent)
	// Finish start all l2 functions
	log.Info("Start event-watcher successfully")

	// Catch CTRL-C to ensure a graceful shutdown.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Wait until the interrupt signal is received from an OS signal.
	<-interrupt

	return nil
}

// Run event watcher cmd instance.
func Run() {
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
