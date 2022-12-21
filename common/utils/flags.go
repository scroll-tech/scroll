package utils

import (
	"fmt"
	"os"

	"github.com/docker/docker/pkg/reexec"
	"github.com/urfave/cli/v2"
)

var (
	// CommonFlags is used for app common flags in different modules
	CommonFlags = []cli.Flag{
		&ConfigFileFlag,
		&VerbosityFlag,
		&LogFileFlag,
		&LogJSONFormat,
		&LogDebugFlag,
	}
	// ConfigFileFlag load json type config file.
	ConfigFileFlag = cli.StringFlag{
		Name:  "config",
		Usage: "JSON configuration file",
		Value: "./config.json",
	}
	// VerbosityFlag log level.
	VerbosityFlag = cli.IntFlag{
		Name:  "verbosity",
		Usage: "Logging verbosity: 0=silent, 1=error, 2=warn, 3=info, 4=debug, 5=detail",
		Value: 3,
	}
	// LogFileFlag decides where the logger output is sent. If this flag is left
	// empty, it will log to stdout.
	LogFileFlag = cli.StringFlag{
		Name:  "log.file",
		Usage: "Tells the module where to write log entries",
	}
	// LogJSONFormat decides the log format is json or not
	LogJSONFormat = cli.BoolFlag{
		Name:  "log.json",
		Usage: "Tells the module whether log format is json or not",
		Value: true,
	}
	// LogDebugFlag make log messages with call-site location
	LogDebugFlag = cli.BoolFlag{
		Name:  "log.debug",
		Usage: "Prepends log messages with call-site location (file and line number)",
	}
)

// RegisterSimulation register initializer function for integration-test.
func RegisterSimulation(app *cli.App, name string) {
	// Run the app for integration-test
	reexec.Register(name, func() {
		if err := app.Run(os.Args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	})
	// check if we have been reexec'd
	if reexec.Init() {
		return
	}
}
