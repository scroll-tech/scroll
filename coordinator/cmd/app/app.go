package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/scroll-tech/go-ethereum/ethclient"

	"github.com/scroll-tech/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"scroll-tech/database"

	"scroll-tech/common/metrics"
	"scroll-tech/common/utils"
	"scroll-tech/common/version"

	"scroll-tech/coordinator"
	"scroll-tech/coordinator/config"
)

var (
	// Set up Coordinator app info.
	app *cli.App
)

func init() {
	app = cli.NewApp()
	app.Action = action
	app.Name = "coordinator"
	app.Usage = "The Scroll L2 Coordinator"
	app.Version = version.Version
	app.Flags = append(app.Flags, utils.CommonFlags...)
	app.Flags = append(app.Flags, apiFlags...)

	app.Before = func(ctx *cli.Context) error {
		_, err := utils.LogSetup(ctx)
		return err
	}

	// Register `coordinator-test` app for integration-test.
	utils.RegisterSimulation(app, utils.CoordinatorApp)
}

func action(ctx *cli.Context) error {
	// Load config file.
	cfgFile := ctx.String(utils.ConfigFileFlag.Name)
	cfg, err := config.NewConfig(cfgFile)
	if err != nil {
		log.Crit("failed to load config file", "config file", cfgFile, "error", err)
	}

	// Start metrics server.
	metrics.Serve(context.Background(), ctx)

	// init db connection
	var ormFactory database.OrmFactory
	if ormFactory, err = database.NewOrmFactory(cfg.DBConfig); err != nil {
		log.Crit("failed to init db connection", "err", err)
	}

	client, err := ethclient.Dial(cfg.L2Config.Endpoint)
	if err != nil {
		return err
	}

	// Initialize all coordinator modules.
	rollerManager, err := coordinator.New(ctx.Context, cfg.RollerManagerConfig, ormFactory, client)
	if err != nil {
		return err
	}
	defer func() {
		rollerManager.Stop()
		err = ormFactory.Close()
		if err != nil {
			log.Error("can not close ormFactory", "error", err)
		}
	}()

	// Start all modules.
	if err = rollerManager.Start(); err != nil {
		log.Crit("couldn't start roller manager", "error", err)
	}

	apis := rollerManager.APIs()
	// Register api and start rpc service.
	if ctx.Bool(httpEnabledFlag.Name) {
		handler, addr, err := utils.StartHTTPEndpoint(
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
		handler, addr, err := utils.StartWSEndpoint(
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
