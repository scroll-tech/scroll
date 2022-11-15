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

	// logFileFlag decides where the logger output is sent. If this flag is left
	// empty, it will log to stdout.
	logFileFlag = cli.StringFlag{
		Name:  "logfile",
		Usage: "Tells the sequencer where to write log entries",
	}

	// verbosityFlag log level.
	verbosityFlag = cli.IntFlag{
		Name:  "verbosity",
		Usage: "Logging verbosity: 0=silent, 1=error, 2=warn, 3=info, 4=debug, 5=detail",
		Value: 3,
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
		&logFileFlag,
		&verbosityFlag,
	}...)
	app.Before = func(ctx *cli.Context) error {
		return utils.Setup(&utils.LogConfig{
			LogFile:   ctx.String(logFileFlag.Name),
			Verbosity: ctx.Int(verbosityFlag.Name),
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

	// Create roller
	r, err := core.NewRoller(cfg)
	if err != nil {
		return err
	}
	defer r.Close()
	log.Info("roller start successfully", "name", cfg.RollerName)

	return r.Run()
}
