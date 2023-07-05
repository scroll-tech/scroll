package app

import (
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/log"
	"github.com/iris-contrib/middleware/cors"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/mvc"
	"github.com/urfave/cli/v2"

	"bridge-history-api/config"
	"bridge-history-api/controller"
	"bridge-history-api/db"
	"bridge-history-api/service"
	cutils "bridge-history-api/utils"
)

var (
	app *cli.App
)

var database db.OrmFactory

func pong(ctx iris.Context) {
	ctx.WriteString("pong")
}

func setupQueryByAddressHandler(backend_app *mvc.Application) {
	// Register Dependencies.
	backend_app.Register(
		database,
		service.NewHistoryService,
	)

	// Register Controllers.
	backend_app.Handle(new(controller.QueryAddressController))
}

func setupQueryByHashHandler(backend_app *mvc.Application) {
	backend_app.Register(
		database,
		service.NewHistoryService,
	)
	backend_app.Handle(new(controller.QueryHashController))
}

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
	corsOptions := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
		AllowCredentials: true,
	})
	// Load config file.
	cfgFile := ctx.String(cutils.ConfigFileFlag.Name)
	cfg, err := config.NewConfig(cfgFile)
	if err != nil {
		log.Crit("failed to load config file", "config file", cfgFile, "error", err)
	}
	database, err = db.NewOrmFactory(cfg)
	if err != nil {
		log.Crit("can not connect to database", "err", err)
	}
	defer database.Close()
	bridgeApp := iris.New()
	bridgeApp.UseRouter(corsOptions)
	bridgeApp.Get("/ping", pong).Describe("healthcheck")

	mvc.Configure(bridgeApp.Party("/api/txs"), setupQueryByAddressHandler)
	mvc.Configure(bridgeApp.Party("/api/txsbyhashes"), setupQueryByHashHandler)

	// TODO: make debug mode configurable
	err = bridgeApp.Listen(cfg.Server.HostPort, iris.WithLogLevel("debug"))
	if err != nil {
		log.Crit("can not start server", "err", err)
	}

	return nil
}

// Run event watcher cmd instance.
func Run() {
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
