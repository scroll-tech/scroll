package app

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"os"
	"os/signal"
	"scroll-tech/miner-api/controller"
	"scroll-tech/miner-api/internal/config"
	"scroll-tech/miner-api/internal/orm"
	"scroll-tech/miner-api/service"

	"github.com/scroll-tech/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"scroll-tech/common/database"
	"scroll-tech/common/utils"
	"scroll-tech/common/version"

	_ "scroll-tech/miner-api/cmd/docs"

	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

var app *cli.App

func init() {
	// Set up miner-api info.
	app = cli.NewApp()
	app.Action = action
	app.Name = "Miner API"
	app.Usage = "The Scroll L2 ZK Miner API"
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

	// init miner api
	ptdb := orm.NewProverTask(db)
	taskService := service.NewProverTaskService(ptdb)

	r := gin.Default()
	r.GET("swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
	router := r.Group("/api/v1")

	c := controller.NewProverTaskController(router, taskService)
	c.Route()

	go func() {
		r.Run(ctx.String(httpPortFlag.Name))
	}()

	// Catch CTRL-C to ensure a graceful shutdown.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Wait until the interrupt signal is received from an OS signal.
	<-interrupt

	return nil
}

// Run run miner-api.
func Run() {
	// RunApp the miner-api.
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
