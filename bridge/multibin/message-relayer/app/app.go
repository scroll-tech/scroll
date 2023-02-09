package app

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/scroll-tech/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"scroll-tech/database"

	"scroll-tech/common/utils"
	"scroll-tech/common/version"

	"scroll-tech/bridge/config"
	messagerelayer "scroll-tech/bridge/multibin/message-relayer"
)

var (
	app *cli.App
)

func init() {
	// Set up Bridge app info.
	app = cli.NewApp()

	app.Action = action
	app.Name = "message-relayer"
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

	var (
		l1relayer *messagerelayer.L1MsgRelayer
		l2relayer *messagerelayer.L2MsgRelayer
	)
	l1relayer, err = messagerelayer.NewL1MsgRelayer(ctx.Context, int64(cfg.L1Config.Confirmations), ormFactory, cfg.L1Config.RelayerConfig)
	if err != nil {
		log.Crit("failed to create new l1 relayer", "config file", cfgFile, "error", err)
	}
	l2relayer, err = messagerelayer.NewL2MsgRelayer(ctx.Context, ormFactory, cfg.L2Config.RelayerConfig)
	if err != nil {
		log.Crit("failed to create new l2 relayer", "config file", cfgFile, "error", err)
	}
	defer func() {
		l1relayer.Stop()
		l2relayer.Stop()
		err = ormFactory.Close()
		if err != nil {
			log.Error("can not close ormFactory", "error", err)
		}
	}()

	// Start all modules.
	l1relayer.Start()
	l2relayer.Start()
	log.Info("Start message_relayer successfully")

	// Catch CTRL-C to ensure a graceful shutdown.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Wait until the interrupt signal is received from an OS signal.
	<-interrupt

	return nil
}

// Run run message_relayer cmd instance.
func Run() {
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
