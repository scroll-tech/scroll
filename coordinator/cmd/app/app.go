package app

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/docker/docker/pkg/reexec"

	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rpc"
	"github.com/urfave/cli/v2"

	"scroll-tech/common/utils"
	"scroll-tech/common/version"
	"scroll-tech/database"

	rollers "scroll-tech/coordinator"
	"scroll-tech/coordinator/config"
)

var (
	// Set up Coordinator app info.
	app = cli.NewApp()
)

func init() {
	app.Action = action
	app.Name = "coordinator"
	app.Usage = "The Scroll L2 Coordinator"
	app.Version = version.Version
	app.Flags = append(app.Flags, utils.CommonFlags...)
	app.Flags = append(app.Flags, utils.DBFlags...)
	app.Flags = append(app.Flags, apiFlags...)

	app.Before = func(ctx *cli.Context) error {
		return utils.Setup(&utils.LogConfig{
			LogFile:       ctx.String(utils.LogFileFlag.Name),
			LogJSONFormat: ctx.Bool(utils.LogJSONFormat.Name),
			LogDebug:      ctx.Bool(utils.LogDebugFlag.Name),
			Verbosity:     ctx.Int(utils.VerbosityFlag.Name),
		})
	}

	// Run the app for integration-test
	reexec.Register("coordinator-test", func() {
		if err := app.Run(os.Args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	})
	// check if we have been reexec'd
	reexec.Init()
}

// RunCoordinator run coordinator.
func RunCoordinator() {
	// Run the coordinator.
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func applyConfig(ctx *cli.Context, cfg *config.Config) {
	if ctx.IsSet(wsPortFlag.Name) {
		cfg.RollerManagerConfig.Endpoint = fmt.Sprintf(":%d", ctx.Int(wsPortFlag.Name))
	}
	if ctx.IsSet(verifierFlag.Name) {
		cfg.RollerManagerConfig.VerifierEndpoint = ctx.String(verifierFlag.Name)
	}
	if ctx.IsSet(utils.DriverFlag.Name) {
		cfg.DBConfig.DriverName = ctx.String(utils.DriverFlag.Name)
	}
	if ctx.IsSet(utils.DSNFlag.Name) {
		cfg.DBConfig.DSN = ctx.String(utils.DSNFlag.Name)
	}
}

func action(ctx *cli.Context) error {
	// Load config file.
	cfgFile := ctx.String(utils.ConfigFileFlag.Name)
	cfg, err := config.NewConfig(cfgFile)
	if err != nil {
		log.Crit("failed to load config file", "config file", cfgFile, "error", err)
	}
	applyConfig(ctx, cfg)

	// init db connection
	var ormFactory database.OrmFactory
	if ormFactory, err = database.NewOrmFactory(cfg.DBConfig); err != nil {
		log.Crit("failed to init db connection", "err", err)
	}

	// Initialize all coordinator modules.
	rollerManager, err := rollers.New(ctx.Context, cfg.RollerManagerConfig, ormFactory)
	if err != nil {
		return err
	}
	defer func() {
		rollerManager.Stop()
		err = ormFactory.Close()
		if err != nil {
			log.Error("can not close ormFactory", err)
		}
	}()

	// Start all modules.
	if err = rollerManager.Start(); err != nil {
		log.Crit("couldn't start roller manager", "error", err)
	}

	// Register api and start rpc service.
	if ctx.Bool(httpEnabledFlag.Name) {
		srv := rpc.NewServer()
		apis := rollerManager.APIs()
		for _, api := range apis {
			if err = srv.RegisterName(api.Namespace, api.Service); err != nil {
				log.Crit("register namespace failed", "namespace", api.Namespace, "error", err)
			}
		}
		handler, addr, err := utils.StartHTTPEndpoint(
			fmt.Sprintf(
				"%s:%d",
				ctx.String(httpListenAddrFlag.Name),
				ctx.Int(httpPortFlag.Name)),
			rpc.DefaultHTTPTimeouts,
			srv)
		if err != nil {
			log.Crit("Could not start RPC api", "error", err)
		}
		defer func() {
			_ = handler.Shutdown(ctx.Context)
			log.Info("HTTP endpoint closed", "url", fmt.Sprintf("http://%v/", addr))
		}()
		log.Info("HTTP endpoint opened", "url", fmt.Sprintf("http://%v/", addr))
	}

	// Catch CTRL-C to ensure a graceful shutdown.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Wait until the interrupt signal is received from an OS signal.
	<-interrupt

	return nil
}
