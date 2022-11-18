package app

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rpc"
	"github.com/urfave/cli/v2"

	"scroll-tech/common/utils"
	"scroll-tech/database"

	"scroll-tech/bridge/config"
	"scroll-tech/bridge/l1"
	"scroll-tech/bridge/l2"
)

func applyConfig(ctx *cli.Context, cfg *config.Config) {
	if ctx.IsSet(l1UrlFlag.Name) {
		url := ctx.String(l1UrlFlag.Name)
		cfg.L1Config.Endpoint = url
		cfg.L2Config.RelayerConfig.SenderConfig.Endpoint = url
	}
	if ctx.IsSet(l2UrlFlag.Name) {
		url := ctx.String(l2UrlFlag.Name)
		cfg.L2Config.Endpoint = url
		cfg.L1Config.RelayerConfig.SenderConfig.Endpoint = url
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

// RunBridge run bridge cmd instance.
func RunBridge() {
	// Run the sequencer.
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
