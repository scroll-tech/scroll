package app

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/ethereum/go-ethereum/log"
	"github.com/gin-gonic/gin"
	"github.com/urfave/cli/v2"

	"bridge-history-api/config"
	"bridge-history-api/internal/controller"
	"bridge-history-api/internal/route"
	"bridge-history-api/observability"
	"bridge-history-api/utils"
)

var (
	app *cli.App
)

func init() {
	app = cli.NewApp()

	app.Action = action
	app.Name = "Scroll Bridge History Web Service"
	app.Usage = "The Scroll Bridge History Web Service"
	app.Flags = append(app.Flags, utils.CommonFlags...)
	app.Commands = []*cli.Command{}

	app.Before = func(ctx *cli.Context) error {
		return utils.LogSetup(ctx)
	}
}

func action(ctx *cli.Context) error {
	// Load config file.
	cfgFile := ctx.String(utils.ConfigFileFlag.Name)
	cfg, err := config.NewConfig(cfgFile)
	if err != nil {
		log.Crit("failed to load config file", "config file", cfgFile, "error", err)
	}
	db, err := utils.InitDB(cfg.DB)
	if err != nil {
		log.Crit("failed to init db", "err", err)
	}
	defer func() {
		if deferErr := utils.CloseDB(db); deferErr != nil {
			log.Error("failed to close db", "err", err)
		}
	}()
	// init Prover Stats API
	port := cfg.Server.HostPort

	router := gin.Default()
	controller.InitController(db)
	route.Route(router, cfg)

	go func() {
		if runServerErr := router.Run(fmt.Sprintf(":%s", port)); runServerErr != nil {
			log.Crit("run http server failure", "error", runServerErr)
		}
	}()
	// Catch CTRL-C to ensure a graceful shutdown.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	observability.Server(ctx, db)

	// Wait until the interrupt signal is received from an OS signal.
	<-interrupt

	return nil
}

// Run event watcher cmd instance.
func Run() {
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
