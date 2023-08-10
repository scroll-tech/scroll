package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	// enable the pprof
	_ "net/http/pprof"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"scroll-tech/common/database"
	"scroll-tech/common/metrics"
	"scroll-tech/common/utils"
	"scroll-tech/common/version"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/controller/api"
	"scroll-tech/coordinator/internal/controller/cron"
	"scroll-tech/coordinator/internal/route"
)

var app *cli.App

func init() {
	// Set up coordinator app info.
	app = cli.NewApp()
	app.Action = action
	app.Name = "coordinator"
	app.Usage = "The Scroll L2 Coordinator"
	app.Version = version.Version
	app.Flags = append(app.Flags, utils.CommonFlags...)
	app.Flags = append(app.Flags, apiFlags...)
	app.Before = func(ctx *cli.Context) error {
		return utils.LogSetup(ctx)
	}
	// Register `coordinator-test` app for integration-test.
	utils.RegisterSimulation(app, utils.CoordinatorApp)
}

func action(ctx *cli.Context) error {
	cfgFile := ctx.String(utils.ConfigFileFlag.Name)
	cfg, err := config.NewConfig(cfgFile)
	if err != nil {
		log.Crit("failed to load config file", "config file", cfgFile, "error", err)
	}

	subCtx, cancel := context.WithCancel(ctx.Context)
	db, err := database.InitDB(cfg.DB)
	if err != nil {
		log.Crit("failed to init db connection", "err", err)
	}

	proofCollector := cron.NewCollector(subCtx, db, cfg, prometheus.DefaultRegisterer)
	defer func() {
		proofCollector.Stop()
		cancel()
		if err = database.CloseDB(db); err != nil {
			log.Error("can not close db connection", "error", err)
		}
	}()

	router := gin.Default()
	api.InitController(cfg, db)
	route.Route(router, cfg)
	port := ctx.String(httpPortFlag.Name)
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%s", port),
		Handler:           router,
		ReadHeaderTimeout: time.Minute,
	}

	// Start metrics server.
	metrics.Serve(subCtx, ctx)

	go func() {
		if runServerErr := srv.ListenAndServe(); err != nil && !errors.Is(runServerErr, http.ErrServerClosed) {
			log.Crit("run coordinator http server failure", "error", runServerErr)
		}
	}()

	// Catch CTRL-C to ensure a graceful shutdown.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Wait until the interrupt signal is received from an OS signal.
	<-interrupt
	log.Info("start shutdown coordinator server ...")

	closeCtx, cancelExit := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelExit()
	if err = srv.Shutdown(closeCtx); err != nil {
		log.Warn("shutdown coordinator server failure", "error", err)
		return nil
	}

	<-closeCtx.Done()
	log.Info("coordinator server exiting success")
	return nil
}

// Run run coordinator.
func Run() {
	// RunApp the coordinator.
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
