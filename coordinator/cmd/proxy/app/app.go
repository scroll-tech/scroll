package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"scroll-tech/common/utils"
	"scroll-tech/common/version"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/controller/proxy"
	"scroll-tech/coordinator/internal/route"
)

var app *cli.App

func init() {
	// Set up coordinator app info.
	app = cli.NewApp()
	app.Action = action
	app.Name = "coordinator proxy"
	app.Usage = "Proxy for multiple Scroll L2 Coordinators"
	app.Version = version.Version
	app.Flags = append(app.Flags, utils.CommonFlags...)
	app.Flags = append(app.Flags, apiFlags...)
	app.Before = func(ctx *cli.Context) error {
		return utils.LogSetup(ctx)
	}
	// Register `coordinator-test` app for integration-test.
	utils.RegisterSimulation(app, utils.CoordinatorAPIApp)
}

func action(ctx *cli.Context) error {
	cfgFile := ctx.String(utils.ConfigFileFlag.Name)
	cfg, err := config.NewProxyConfig(cfgFile)
	if err != nil {
		log.Crit("failed to load config file", "config file", cfgFile, "error", err)
	}

	//observability.Server(ctx, db)
	registry := prometheus.DefaultRegisterer

	apiSrv := server(ctx, cfg, registry)

	log.Info(
		"Start coordinator api successfully.",
		"version", version.Version,
	)

	// Catch CTRL-C to ensure a graceful shutdown.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Wait until the interrupt signal is received from an OS signal.
	<-interrupt
	log.Info("start shutdown coordinator proxy server ...")

	closeCtx, cancelExit := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelExit()
	if err = apiSrv.Shutdown(closeCtx); err != nil {
		log.Warn("shutdown coordinator proxy server failure", "error", err)
		return nil
	}

	<-closeCtx.Done()
	log.Info("coordinator proxy server exiting success")
	return nil
}

func server(ctx *cli.Context, cfg *config.ProxyConfig, reg prometheus.Registerer) *http.Server {
	router := gin.New()
	proxy.InitController(cfg, reg)
	route.ProxyRoute(router, cfg, reg)
	port := ctx.String(httpPortFlag.Name)
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%s", port),
		Handler:           router,
		ReadHeaderTimeout: time.Minute,
	}

	go func() {
		if runServerErr := srv.ListenAndServe(); runServerErr != nil && !errors.Is(runServerErr, http.ErrServerClosed) {
			log.Crit("run coordinator proxy http server failure", "error", runServerErr)
		}
	}()
	return srv
}

// Run coordinator.
func Run() {
	// RunApp the coordinator.
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
