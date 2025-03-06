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

	app.Action = action
	app.Name = "Scroll Bridge History API Message Fetcher"
	app.Usage = "The Scroll Bridge History API Message Fetcher"
	app.Flags = append(app.Flags, utils.CommonFlags...)
	app.Commands = []*cli.Command{}

	app.Before = func(ctx *cli.Context) error {
		return utils.LogSetup(ctx)
	}
}

func action(ctx *cli.Context) error {
	cfgFile := ctx.String(utils.ConfigFileFlag.Name)
	cfg, err := config.NewConfig(cfgFile)
	if err != nil {
		log.Crit("failed to load config file", "config file", cfgFile, "error", err)
	}
	subCtx, cancel := context.WithCancel(ctx.Context)
	defer cancel()

	l1Client, err := ethclient.Dial(cfg.L1.Endpoint)
	if err != nil {
		log.Crit("failed to connect to L1 geth", "endpoint", cfg.L1.Endpoint, "err", err)
	}

	l2Client, err := ethclient.Dial(cfg.L2.Endpoint)
	if err != nil {
		log.Crit("failed to connect to L2 geth", "endpoint", cfg.L2.Endpoint, "err", err)
	}

	db, err := database.InitDB(cfg.DB)
	if err != nil {
		log.Crit("failed to init db", "err", err)
	}
	defer func() {
		if deferErr := database.CloseDB(db); deferErr != nil {
			log.Error("failed to close db", "err", err)
		}
	}()
	if err != nil {
		log.Crit("failed to connect to db", "config file", cfgFile, "error", err)
	}

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

// Run bridge-history-backend fetcher cmd instance.
func Run() {
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
