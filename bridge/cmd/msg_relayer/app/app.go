package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/scroll-tech/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"scroll-tech/common/metrics"
	cutils "scroll-tech/common/utils"
	"scroll-tech/common/version"

	"scroll-tech/bridge/internal/config"
	"scroll-tech/bridge/internal/controller/relayer"
	"scroll-tech/bridge/internal/utils"
)

var app *cli.App
var logger log.Logger

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
		var err error
		logger, err = cutils.LogSetup(ctx)
		return err
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

	l1relayer, err := relayer.NewLayer1Relayer(ctx.Context, db, cfg.L1Config.RelayerConfig)
	if err != nil {
		log.Error("failed to create new l1 relayer", "config file", cfgFile, "error", err)
		return err
	}

	// Start l1relayer process
	go cutils.Loop(subCtx, 10*time.Second, l1relayer.ProcessSavedEvents)

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
