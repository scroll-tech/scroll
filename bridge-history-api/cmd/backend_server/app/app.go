package app

import (
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/log"
	"github.com/gin-gonic/gin"
	"github.com/urfave/cli/v2"

	"bridge-history-api/config"
	"bridge-history-api/internal/controller"
	"bridge-history-api/internal/route"
	cutils "bridge-history-api/utils"

	"scroll-tech/common/database"
)

var (
	app *cli.App
)

func init() {
	app = cli.NewApp()

	app.Action = action
	app.Name = "Scroll Bridge History Web Service"
	app.Usage = "The Scroll Bridge History Web Service"
	app.Flags = append(app.Flags, cutils.CommonFlags...)
	app.Commands = []*cli.Command{}

	app.Before = func(ctx *cli.Context) error {
		return cutils.LogSetup(ctx)
	}
}

func action(ctx *cli.Context) error {
	// Load config file.
	cfgFile := ctx.String(cutils.ConfigFileFlag.Name)
	cfg, err := config.NewConfig(cfgFile)
	if err != nil {
		log.Crit("failed to load config file", "config file", cfgFile, "error", err)
	}
	dbCfg := &database.Config{
		DriverName: cfg.DB.DriverName,
		DSN:        cfg.DB.DSN,
		MaxOpenNum: cfg.DB.MaxOpenNum,
		MaxIdleNum: cfg.DB.MaxIdleNum,
	}
	db, err := database.InitDB(dbCfg)
	if err != nil {
		log.Crit("failed to init db", "err", err)
	}
	defer func() {
		if deferErr := database.CloseDB(db); deferErr != nil {
			log.Error("failed to close db", "err", err)
		}
	}()
	// init Prover Stats API
	port := ctx.String(cfg.Server.HostPort)

	router := gin.Default()
	controller.InitController(db)
	route.Route(router, cfg)

	go func() {
		if runServerErr := router.Run(fmt.Sprintf(":%s", port)); runServerErr != nil {
			log.Crit("run http server failure", "error", runServerErr)
		}
	}()

	return nil
}

// Run event watcher cmd instance.
func Run() {
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
