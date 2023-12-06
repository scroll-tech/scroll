package app

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"scroll-tech/bridge-history-api/internal/config"
	"scroll-tech/bridge-history-api/internal/controller/api"
	"scroll-tech/bridge-history-api/internal/route"
	"scroll-tech/common/database"
	"scroll-tech/common/observability"
	"scroll-tech/common/utils"
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
	db, err := database.InitDB(cfg.DB)
	if err != nil {
		log.Crit("failed to init db", "err", err)
	}
	defer func() {
		if deferErr := database.CloseDB(db); deferErr != nil {
			log.Error("failed to close db", "err", err)
		}
	}()
	redis := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Address,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	api.InitController(db, redis)

	router := gin.Default()
	registry := prometheus.DefaultRegisterer
	route.Route(router, cfg, registry)

	go func() {
		port := cfg.Server.HostPort
		if runServerErr := router.Run(fmt.Sprintf(":%s", port)); runServerErr != nil {
			log.Crit("run http server failure", "error", runServerErr)
		}
	}()

	observability.Server(ctx, db)

	// Catch CTRL-C to ensure a graceful shutdown.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

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
