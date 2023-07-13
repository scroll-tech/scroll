package app

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/scroll-tech/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"scroll-tech/prover-stats-api/internal/config"
	"scroll-tech/prover-stats-api/internal/controller"
	"scroll-tech/prover-stats-api/internal/logic"
	"scroll-tech/prover-stats-api/internal/orm"

	"scroll-tech/common/database"
	"scroll-tech/common/utils"
	"scroll-tech/common/version"

	_ "scroll-tech/prover-stats-api/cmd/docs"

	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
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
	RunMinerAPIs(db, port)

	// Catch CTRL-C to ensure a graceful shutdown.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Wait until the interrupt signal is received from an OS signal.
	<-interrupt

	return nil
}

func RunMinerAPIs(db *gorm.DB, port string) {
	ptdb := orm.NewProverTask(db)
	taskService := logic.NewProverTaskService(ptdb)

	r := gin.Default()
	r.GET("swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
	router := r.Group("/api/v1")

	c := controller.NewProverTaskController(router, taskService)
	c.Route()

	go func() {
		r.Run(port)
	}()
}

// Run run prover-stats-api.
func Run() {
	// RunApp the prover-stats-api.
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
