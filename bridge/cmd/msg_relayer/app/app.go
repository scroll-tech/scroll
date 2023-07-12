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

	"scroll-tech/database"

	"scroll-tech/common/metrics"
	"scroll-tech/common/version"

	"scroll-tech/bridge/config"
	"scroll-tech/bridge/relayer"

	cutils "scroll-tech/common/utils"
)

var (
	app *cli.App
)

func init() {
	// Set up message-relayer app info.
	app = cli.NewApp()

	app.Action = action
	app.Name = "message-relayer"
	app.Usage = "The Scroll Message Relayer"
	app.Description = "Message Relayer contains two main service: 1) relay l1 message to l2. 2) relay l2 message to l1."
	app.Version = version.Version
	app.Flags = append(app.Flags, cutils.CommonFlags...)
	app.Commands = []*cli.Command{}

	app.Before = func(ctx *cli.Context) error {
		return cutils.LogSetup(ctx)
	}

	// Register `message-relayer-test` app for integration-test.
	cutils.RegisterSimulation(app, cutils.MessageRelayerApp)
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
	var ormFactory database.OrmFactory
	if ormFactory, err = database.NewOrmFactory(cfg.DBConfig); err != nil {
		log.Crit("failed to init db connection", "err", err)
	}

	defer func() {
		cancel()
		err = ormFactory.Close()
		if err != nil {
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

	l1relayer, err := relayer.NewLayer1Relayer(ctx.Context, ormFactory, cfg.L1Config.RelayerConfig)
	if err != nil {
		log.Error("failed to create new l1 relayer", "config file", cfgFile, "error", err)
		return err
	}
	l2relayer, err := relayer.NewLayer2Relayer(ctx.Context, l2client, ormFactory, cfg.L2Config.RelayerConfig)
	if err != nil {
		log.Error("failed to create new l2 relayer", "config file", cfgFile, "error", err)
		return err
	}

	// Start l1relayer process
	go cutils.Loop(subCtx, 10*time.Second, l1relayer.ProcessSavedEvents)

	// Start l2relayer process
	go cutils.Loop(subCtx, 2*time.Second, l2relayer.ProcessSavedEvents)

	// Finish start all message relayer functions
	log.Info("Start message-relayer successfully")

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
