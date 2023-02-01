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
	"scroll-tech/bridge/l1"
	"scroll-tech/bridge/l2"
)

var (
	app *cli.App
)

func init() {
	// Set up Bridge app info.
	app = cli.NewApp()

	app.Action = action
	app.Name = "message-replayer"
	app.Usage = "The Scroll Message Relayer"
	app.Description = "Message Relayer contains two main service: 1) relay l1 message to l2. 2) relay l2 message to l1."
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
		l1relayer *l1.Layer1Relayer
		l2relayer *l2.Layer2Relayer
	)
	l1watcher, err = l1.NewLayer1Relayer(ctx.Context, l1client, int64(cfg.L1Config.Confirmations), ormFactory, cfg.L1Config.RelayerConfig)
	if err != nil {
		log.Crit("failed to create new l1 relayer", "config file", cfgFile, "error", err)
	}
	l2watcher, err = l2.NewLayer2Relayer(ctx.Context, ormFactory, cfg.L2Config.RelayerConfig)
	if err != nil {
		log.Crit("failed to creatw new l2 relayer", "config file", cfgFile, "error", err)
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
