package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"scroll-tech/common/database"
	"scroll-tech/common/observability"
	"scroll-tech/common/utils"

	"scroll-tech/bridge-history-api/internal/config"
	"scroll-tech/bridge-history-api/internal/controller/fetcher"
)

var app *cli.App

func init() {
	app = cli.NewApp()
	app.Name = "Scroll Bridge History API Message Fetcher"
	app.Usage = "The Scroll Bridge History API Message Fetcher"
	app.Flags = append(app.Flags, utils.CommonFlags...)
	app.Before = func(ctx *cli.Context) error {
		return utils.LogSetup(ctx)
	}
}

func action(ctx *cli.Context) error {
	cfgFile := ctx.String(utils.ConfigFileFlag.Name)
	cfg, err := config.NewConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config file %s: %w", cfgFile, err)
	}

	subCtx, cancel := context.WithCancel(ctx.Context)
	defer cancel()

	l1Client, err := ethclient.Dial(cfg.L1.Endpoint)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 geth endpoint %s: %w", cfg.L1.Endpoint, err)
	}

	l2Client, err := ethclient.Dial(cfg.L2.Endpoint)
	if err != nil {
		return fmt.Errorf("failed to connect to L2 geth endpoint %s: %w", cfg.L2.Endpoint, err)
	}

	db, err := database.InitDB(cfg.DB)
	if err != nil {
		return fmt.Errorf("failed to init db: %w", err)
	}
	defer func() {
		if deferErr := database.CloseDB(db); deferErr != nil {
			log.Error("failed to close db", "err", deferErr)
		}
	}()

	observability.Server(ctx, db)

	l1MessageFetcher := fetcher.NewL1MessageFetcher(subCtx, cfg.L1, db, l1Client)
	go l1MessageFetcher.Start()

	l2MessageFetcher := fetcher.NewL2MessageFetcher(subCtx, cfg.L2, db, l2Client)
	go l2MessageFetcher.Start()

	// Catch CTRL-C to ensure a graceful shutdown.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Wait until the interrupt signal is received from an OS signal.
	<-interrupt

	return nil
}

// Run event watcher cmd instance.
func Run() {
	app.Action = action
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
