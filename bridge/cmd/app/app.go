package app

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/docker/docker/pkg/reexec"
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
	app.Name = "bridge"
	app.Usage = "The Scroll Bridge"
	app.Version = version.Version
	app.Flags = append(app.Flags, utils.CommonFlags...)
	app.Flags = append(app.Flags, apiFlags...)

	app.Before = func(ctx *cli.Context) error {
		return utils.LogSetup(ctx)
	}

	// Run the app for integration-test
	reexec.Register("bridge-test", func() {
		if err := app.Run(os.Args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	})
	// check if we have been reexec'd
	reexec.Init()
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
		l1Backend *l1.Backend
		l2Backend *l2.Backend
	)
	// @todo change nil to actual client after https://scroll-tech/bridge/pull/40 merged
	l1Backend, err = l1.New(ctx.Context, cfg.L1Config, ormFactory)
	if err != nil {
		return err
	}
	l2Backend, err = l2.New(ctx.Context, cfg.L2Config, ormFactory)
	if err != nil {
		return err
	}
	defer func() {
		l1Backend.Stop()
		l2Backend.Stop()
		err = ormFactory.Close()
		if err != nil {
			log.Error("can not close ormFactory", err)
		}
	}()

	// Start all modules.
	if err = l1Backend.Start(); err != nil {
		log.Crit("couldn't start l1 backend", "error", err)
	}
	if err = l2Backend.Start(); err != nil {
		log.Crit("couldn't start l2 backend", "error", err)
	}

	// Register api and start rpc service.
	if ctx.Bool(httpEnabledFlag.Name) {
		handler, addr, err := utils.StartHTTPEndpoint(
			fmt.Sprintf(
				"%s:%d",
				ctx.String(httpListenAddrFlag.Name),
				ctx.Int(httpPortFlag.Name)),
			l2Backend.APIs())
		if err != nil {
			log.Crit("Could not start RPC api", "error", err)
		}
		defer func() {
			_ = handler.Shutdown(ctx.Context)
			log.Info("HTTP endpoint closed", "url", fmt.Sprintf("http://%v/", addr))
		}()
		log.Info("HTTP endpoint opened", "url", fmt.Sprintf("http://%v/", addr))
	}

	log.Info("Start bridge successfully")

	// Catch CTRL-C to ensure a graceful shutdown.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Wait until the interrupt signal is received from an OS signal.
	<-interrupt

	return nil
}

// RunBridge run bridge cmd instance.
func RunBridge() {
	// Run the sequencer.
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
