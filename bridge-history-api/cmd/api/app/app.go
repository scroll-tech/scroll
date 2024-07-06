package app

import (
	"crypto/tls"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"scroll-tech/common/database"
	"scroll-tech/common/observability"
	"scroll-tech/common/utils"

	"scroll-tech/bridge-history-api/internal/config"
	"scroll-tech/bridge-history-api/internal/controller/api"
	"scroll-tech/bridge-history-api/internal/route"
)

var app *cli.App

func init() {
	app = cli.NewApp()

	app.Action = action
	app.Name = "Scroll Bridge History API Web Service"
	app.Usage = "The Scroll Bridge History API Web Service"
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
	opts := &redis.Options{
		Addr:         cfg.Redis.Address,
		Username:     cfg.Redis.Username,
		Password:     cfg.Redis.Password,
		MinIdleConns: cfg.Redis.MinIdleConns,
		ReadTimeout:  time.Duration(cfg.Redis.ReadTimeoutMs * int(time.Millisecond)),
	}
	// Production Redis service has enabled transit_encryption.
	if !cfg.Redis.Local {
		opts.TLSConfig = &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: true, //nolint:gosec
		}
	}
	log.Info("init redis client", "addr", opts.Addr, "user name", opts.Username, "is local", cfg.Redis.Local,
		"min idle connections", opts.MinIdleConns, "read timeout", opts.ReadTimeout)
	redisClient := redis.NewClient(opts)
	api.InitController(db, redisClient)

	router := gin.Default()
	registry := prometheus.DefaultRegisterer
	route.Route(router, cfg, registry)

	go func() {
		port := ctx.Int(utils.ServicePortFlag.Name)
		if runServerErr := router.Run(fmt.Sprintf(":%d", port)); runServerErr != nil {
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

// Run bridge-history-backend api cmd instance.
func Run() {
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
