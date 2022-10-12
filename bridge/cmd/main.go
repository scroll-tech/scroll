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

	"scroll-tech/bridge/config"
	"scroll-tech/bridge/l1"
	"scroll-tech/bridge/l2"
)

func main() {
	// Set up Bridge app info.
	app := cli.NewApp()

	app.Action = action
	app.Name = "bridge"
	app.Usage = "The Scroll Bridge"
	app.Version = "v0.0.1"
	app.Flags = append(app.Flags, utils.CommonFlags...)
	app.Flags = append(app.Flags, utils.APIFlags...)
	app.Flags = append(app.Flags, utils.L1Flags...)
	app.Flags = append(app.Flags, utils.L2Flags...)
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

	// Run the sequencer.
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func applyConfig(ctx *cli.Context, cfg *config.Config) {
	if ctx.IsSet(utils.L1ChainIDFlag.Name) {
		cfg.L1Config.ChainID = ctx.Int64(utils.L1ChainIDFlag.Name)
	}
	if ctx.IsSet(utils.L1UrlFlag.Name) {
		cfg.L1Config.Endpoint = ctx.String(utils.L1UrlFlag.Name)
	}
	if ctx.IsSet(utils.L2ChainIDFlag.Name) {
		cfg.L2Config.ChainID = ctx.Int64(utils.L2ChainIDFlag.Name)
	}
	if ctx.IsSet(utils.L2UrlFlag.Name) {
		cfg.L2Config.Endpoint = ctx.String(utils.L2UrlFlag.Name)
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
	if ctx.Bool(utils.HTTPEnabledFlag.Name) {
		srv := rpc.NewServer()
		apis := l2Backend.APIs()
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
