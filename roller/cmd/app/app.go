package app

import (
	"fmt"
	"os"

	"github.com/docker/docker/pkg/reexec"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"scroll-tech/common/utils"

	"scroll-tech/roller/config"
	"scroll-tech/roller/core"
)

var (
	// cfgFileFlag load json type config file.
	cfgFileFlag = cli.StringFlag{
		Name:  "config",
		Usage: "TOML configuration file",
		Value: "./config.toml",
	}
	urlFlag = cli.StringFlag{
		Name:  "url",
		Usage: "coordinator ws url",
		Value: "ws://localhost:9000",
	}

	app = cli.NewApp()
)

func init() {
	app.Action = action
	app.Name = "Roller"
	app.Usage = "The Scroll L2 Roller"
	app.Version = core.Version
	app.Flags = append(app.Flags, []cli.Flag{
		&cfgFileFlag,
		&urlFlag,
		&utils.LogFileFlag,
		&utils.LogDebugFlag,
		&utils.VerbosityFlag,
	}...)
	app.Before = func(ctx *cli.Context) error {
		return utils.Setup(&utils.LogConfig{
			LogFile:   ctx.String(utils.LogFileFlag.Name),
			Verbosity: ctx.Int(utils.VerbosityFlag.Name),
		})
	}

	// Run the app for integration-test
	reexec.Register("roller-test", func() {
		if err := app.Run(os.Args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	})
	// check if we have been reexec'd
	reexec.Init()
}

// RunRoller the roller cmd func.
func RunRoller() {
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func action(ctx *cli.Context) error {
	// Get config
	cfg, err := config.InitConfig(ctx.String(cfgFileFlag.Name))
	if err != nil {
		return err
	}
	if ctx.IsSet(urlFlag.Name) {
		cfg.ScrollURL = ctx.String(urlFlag.Name)
	}

	// Create roller
	r, err := core.NewRoller(cfg)
	if err != nil {
		return err
	}
	defer r.Close()
	log.Info("roller start successfully", "name", cfg.RollerName)

	return r.Run()
}
