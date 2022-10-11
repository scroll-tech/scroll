package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rpc"
	"github.com/urfave/cli/v2"

	"scroll-tech/store"
	"scroll-tech/utils"

	rollers "scroll-tech/coordinator"
	"scroll-tech/coordinator/config"
)

func main() {
	// Set up Coordinator app info.
	app := cli.NewApp()

	app.Action = action
	app.Name = "coordinator"
	app.Usage = "The Scroll L2 Coordinator"
	app.Version = "v0.0.1"
	app.Flags = append(app.Flags, utils.CommonFlags...)
	app.Flags = append(app.Flags, utils.APIFlags...)
	app.Flags = append(app.Flags, utils.DBflags...)

	app.Before = func(ctx *cli.Context) error {
		return utils.Setup(ctx)
	}
	app.Commands = []*cli.Command{
		{
			Name:   "reset",
			Usage:  "Clean and reset database.",
			Action: utils.ResetDB,
			Flags: []cli.Flag{
				&utils.ConfigFileFlag,
			},
		},
		{
			Name:   "status",
			Usage:  "Check migration status.",
			Action: utils.CheckDBStatus,
			Flags: []cli.Flag{
				&utils.ConfigFileFlag,
			},
		},
		{
			Name:   "version",
			Usage:  "Display the current database version.",
			Action: utils.DBVersion,
			Flags: []cli.Flag{
				&utils.ConfigFileFlag,
			},
		},
		{
			Name:   "migrate",
			Usage:  "Migrate the database to the latest version.",
			Action: utils.MigrateDB,
			Flags: []cli.Flag{
				&utils.ConfigFileFlag,
			},
		},
		{
			Name:   "rollback",
			Usage:  "Roll back the database to a previous <version>. Rolls back a single migration if no version specified.",
			Action: utils.RollbackDB,
			Flags: []cli.Flag{
				&utils.ConfigFileFlag,
				&cli.IntFlag{
					Name:  "version",
					Usage: "Rollback to the specified version.",
					Value: 0,
				},
			},
		},
	}

	// Run the coordinator.
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func applyConfig(ctx *cli.Context, cfg *config.Config) {
	if ctx.IsSet(utils.WSPortFlag.Name) {
		cfg.RollerManagerConfig.Endpoint = fmt.Sprintf(":%d", ctx.Int(utils.WSPortFlag.Name))
	}
	if ctx.IsSet(utils.VerifierFlag.Name) {
		cfg.RollerManagerConfig.VerifierEndpoint = ctx.String(utils.VerifierFlag.Name)
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
	var ormFactory store.OrmFactory
	if ormFactory, err = store.NewOrmFactory(cfg.DBConfig); err != nil {
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
	if ctx.Bool(utils.HTTPEnabledFlag.Name) {
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
				ctx.String(utils.HTTPListenAddrFlag.Name),
				ctx.Int(utils.HTTPPortFlag.Name)),
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
