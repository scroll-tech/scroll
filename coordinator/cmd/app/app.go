package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/scroll-tech/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"scroll-tech/common/metrics"
	cutils "scroll-tech/common/utils"
	"scroll-tech/common/version"

	"scroll-tech/coordinator"
	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/utils"
)

var app *cli.App

func init() {
	// Set up coordinator app info.
	app = cli.NewApp()
	app.Action = action
	app.Name = "coordinator"
	app.Usage = "The Scroll L2 Coordinator"
	app.Version = version.Version
	app.Flags = append(app.Flags, cutils.CommonFlags...)
	app.Flags = append(app.Flags, apiFlags...)
	app.Before = func(ctx *cli.Context) error {
		return cutils.LogSetup(ctx)
	}
	// Register `coordinator-test` app for integration-test.
	cutils.RegisterSimulation(app, cutils.CoordinatorApp)
}

func action(ctx *cli.Context) error {
	// Load config file.
	cfgFile := ctx.String(cutils.ConfigFileFlag.Name)
	cfg, err := config.NewConfig(cfgFile)
	if err != nil {
		log.Crit("failed to load config file", "config file", cfgFile, "error", err)
	}

	subCtx, cancel := context.WithCancel(ctx.Context)
	// init db connection
	db, err := utils.InitDB(cfg.DBConfig)
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

	// Initialize all coordinator modules.
	rollerManager, err := coordinator.New(subCtx, cfg.RollerManagerConfig, db)
	if err != nil {
		return err
	}

	// Start all modules.
	if err = rollerManager.Start(); err != nil {
		log.Crit("couldn't start roller manager", "error", err)
	}

	apis := rollerManager.APIs()
	// Register api and start rpc service.
	if ctx.Bool(httpEnabledFlag.Name) {
		handler, addr, err := cutils.StartHTTPEndpoint(
			fmt.Sprintf(
				"%s:%d",
				ctx.String(httpListenAddrFlag.Name),
				ctx.Int(httpPortFlag.Name)),
			apis)
		if err != nil {
			log.Crit("Could not start RPC api", "error", err)
		}
		defer func() {
			_ = handler.Shutdown(ctx.Context)
			log.Info("HTTP endpoint closed", "url", fmt.Sprintf("http://%v/", addr))
		}()
		log.Info("HTTP endpoint opened", "url", fmt.Sprintf("http://%v/", addr))
	}
	// Register api and start ws service.
	if ctx.Bool(wsEnabledFlag.Name) {
		handler, addr, err := cutils.StartWSEndpoint(
			fmt.Sprintf(
				"%s:%d",
				ctx.String(wsListenAddrFlag.Name),
				ctx.Int(wsPortFlag.Name)),
			apis, cfg.RollerManagerConfig.CompressionLevel)
		if err != nil {
			log.Crit("Could not start WS api", "error", err)
		}
		defer func() {
			_ = handler.Shutdown(ctx.Context)
			log.Info("WS endpoint closed", "url", fmt.Sprintf("ws://%v/", addr))
		}()
		log.Info("WS endpoint opened", "url", fmt.Sprintf("ws://%v/", addr))
	}

	// Catch CTRL-C to ensure a graceful shutdown.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Wait until the interrupt signal is received from an OS signal.
	<-interrupt

	return nil
}

// Run run coordinator.
func Run() {
	// RunApp the coordinator.
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
