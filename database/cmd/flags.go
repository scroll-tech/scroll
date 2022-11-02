package main

import "github.com/urfave/cli/v2"

var (
	commonFlags = []cli.Flag{
		&configFileFlag,
		&verbosityFlag,
		&logFileFlag,
		&logJSONFormat,
		&logDebugFlag,
	}
	// configFileFlag load json type config file.
	configFileFlag = cli.StringFlag{
		Name:  "config",
		Usage: "JSON configuration file",
		Value: "./config.json",
	}
	// verbosityFlag log level.
	verbosityFlag = cli.IntFlag{
		Name:  "verbosity",
		Usage: "Logging verbosity: 0=silent, 1=error, 2=warn, 3=info, 4=debug, 5=detail",
		Value: 3,
	}
	// logFileFlag decides where the logger output is sent. If this flag is left
	// empty, it will log to stdout.
	logFileFlag = cli.StringFlag{
		Name:  "log.file",
		Usage: "Tells the database where to write log entries",
	}
	logJSONFormat = cli.BoolFlag{
		Name:  "log.json",
		Usage: "Tells the database whether log format is json or not",
		Value: true,
	}
	logDebugFlag = cli.BoolFlag{
		Name:  "log.debug",
		Usage: "Prepends log messages with call-site location (file and line number)",
	}
)
