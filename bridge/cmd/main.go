package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rpc"
	"github.com/urfave/cli/v2"

	"scroll-tech/common/utils"
	"scroll-tech/common/version"
	"scroll-tech/database"

	"scroll-tech/bridge/config"
	"scroll-tech/bridge/l1"
	"scroll-tech/bridge/l2"
)

var (
	// Set up Bridge app info.
	app = cli.NewApp()
)

func init() {
	app.Action = action
	app.Name = "bridge"
	app.Usage = "The Scroll Bridge"
	app.Version = version.Version
	app.Flags = append(app.Flags, commonFlags...)
	app.Flags = append(app.Flags, apiFlags...)
	app.Flags = append(app.Flags, l1Flags...)
	app.Flags = append(app.Flags, l2Flags...)

	app.Before = func(ctx *cli.Context) error {
		return utils.Setup(&utils.LogConfig{
			LogFile:       ctx.String(logFileFlag.Name),
			LogJSONFormat: ctx.Bool(logJSONFormat.Name),
			LogDebug:      ctx.Bool(logDebugFlag.Name),
			Verbosity:     ctx.Int(verbosityFlag.Name),
		})
	}
}

func main() {
	// Run the sequencer.
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func applyConfig(ctx *cli.Context) (*config.Config, error) {
	// Load config file.
	cfgFile := ctx.String(configFileFlag.Name)
	cfg, err := config.NewConfig(cfgFile)
	if err != nil {
		return nil, err
	}
	if ctx.IsSet(l1UrlFlag.Name) {
		cfg.L1Config.Endpoint = ctx.String(l1UrlFlag.Name)
	}
	if ctx.IsSet(l2UrlFlag.Name) {
		cfg.L2Config.Endpoint = ctx.String(l2UrlFlag.Name)
	}
	return cfg, nil
}

func action(ctx *cli.Context) error {
	// Load config file.
	cfg, err := applyConfig(ctx)
	if err != nil {
		log.Crit("failed to load config file", "error", err)
	}

	// init db connection
	var ormFactory database.OrmFactory
	if ormFactory, err = database.NewOrmFactory(cfg.DBConfig); err != nil {
		log.Crit("failed to init db connection", "err", err)
	}
	defer func() {
		err = ormFactory.Close()
		if err != nil {
			log.Error("can not close ormFactory", err)
		}
	}()

	var (
		l1Backend *l1.Backend
		l2Backend *l2.Backend
	)
	// @todo change nil to actual client after https://scroll-tech/bridge/pull/40 merged
	l1Backend, err = l1.New(ctx.Context, cfg.L1Config, ormFactory, l2Backend)
	if err != nil {
		return err
	}
	l2Backend, err = l2.New(ctx.Context, cfg.L2Config, ormFactory, l1Backend)
	if err != nil {
		return err
	}
	defer func() {
		l1Backend.Stop()
		l2Backend.Stop()
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
