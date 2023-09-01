package app

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/gin-gonic/gin"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"scroll-tech/common/database"
	"scroll-tech/common/utils"
	"scroll-tech/common/version"

	"scroll-tech/prover-stats-api/internal/config"
	"scroll-tech/prover-stats-api/internal/controller"
	"scroll-tech/prover-stats-api/internal/route"
)

var app *cli.App

func init() {
	// Set up prover-stats-api info.
	app = cli.NewApp()
	app.Action = action
	app.Name = "Prover Stats API"
	app.Usage = "The Scroll L2 ZK Prover Stats API"
	app.Version = version.Version
	app.Flags = append(app.Flags, utils.CommonFlags...)
	app.Flags = append(app.Flags, apiFlags...)
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

	// init db handler
	db, err := database.InitDB(cfg.DBConfig)
	if err != nil {
		log.Crit("failed to init db connection", "err", err)
	}
	defer func() {
		if err = database.CloseDB(db); err != nil {
			log.Error("can not close ormFactory", "error", err)
		}
	}()

	// init Prover Stats API
	port := ctx.String(httpPortFlag.Name)

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

	// Wait until the interrupt signal is received from an OS signal.
	<-interrupt

	return nil
}

// Run run prover-stats-api.
func Run() {
	// RunApp the prover-stats-api.
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
