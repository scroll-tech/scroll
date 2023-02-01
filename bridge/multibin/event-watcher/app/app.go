package app

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"scroll-tech/database"

	"scroll-tech/common/utils"
	"scroll-tech/common/version"

	"scroll-tech/bridge/config"
	eventwatcher "scroll-tech/bridge/multibin/event-watcher"
)

var (
	app *cli.App
)

func init() {
	// Set up Bridge app info.
	app = cli.NewApp()

	app.Action = action
	app.Name = "event-watcher"
	app.Usage = "The Scroll Event Watcher"
	app.Version = version.Version
	app.Flags = append(app.Flags, utils.CommonFlags...)
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

	// init db connection
	var ormFactory database.OrmFactory
	if ormFactory, err = database.NewOrmFactory(cfg.DBConfig); err != nil {
		log.Crit("failed to init db connection", "err", err)
	}

	l1client, err := ethclient.Dial(cfg.L1Config.Endpoint)
	if err != nil {
		log.Crit("failed to connect l1 geth", "config file", cfgFile, "error", err)
	}

	l2client, err := ethclient.Dial(cfg.L1Config.Endpoint)
	if err != nil {
		log.Crit("failed to connect l1 geth", "config file", cfgFile, "error", err)
	}
	var (
		l1watcher *eventwatcher.L1EventWatcher
		l2watcher *eventwatcher.L2EventWatcher
	)
	l1watcher = eventwatcher.NewL1EventWatcher(ctx.Context, l1client, cfg.L1Config, ormFactory)
	l2watcher, err = eventwatcher.NewL2EventWatcher(ctx.Context, l2client, cfg.L2Config, ormFactory)
	if err != nil {
		return err
	}
	defer func() {
		l1watcher.Stop()
		l2watcher.Stop()
		err = ormFactory.Close()
		if err != nil {
			log.Error("can not close ormFactory", "error", err)
		}
	}()

	// Start all modules.
	l1watcher.Start()
	l2watcher.Start()
	log.Info("Start event-watcher successfully")

	// Catch CTRL-C to ensure a graceful shutdown.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Wait until the interrupt signal is received from an OS signal.
	<-interrupt

	return nil
}

// Run run bridge cmd instance.
func Run() {
	// Run the bridge.
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
