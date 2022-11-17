package app

import (
	"fmt"
	"os"

	"github.com/docker/docker/pkg/reexec"
	"github.com/urfave/cli/v2"

	"scroll-tech/common/utils"
	"scroll-tech/common/version"
)

var (
	commonFlags = []cli.Flag{
		&verbosityFlag,
		&logFileFlag,
		&logJSONFormat,
		&logDebugFlag,
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
	dbFlags = []cli.Flag{
		&configFileFlag,
		&driverFlag,
		&dsnFlag,
	}
	// configFileFlag load json type config file.
	configFileFlag = cli.StringFlag{
		Name:  "config",
		Usage: "JSON configuration file",
		Value: "./config.json",
	}
	driverFlag = cli.StringFlag{
		Name:  "db.driver",
		Usage: "db driver name",
		Value: "postgres",
	}
	dsnFlag = cli.StringFlag{
		Name:  "db.dsn",
		Usage: "data source name",
		Value: "postgres://postgres:@localhost/postgres?sslmode=disable",
	}
)

var (
	// Set up database app info.
	app = cli.NewApp()
)

func init() {
	// Set up database app info.
	app.Name = "database"
	app.Usage = "The Scroll Database CLI"
	app.Version = version.Version
	app.Flags = append(app.Flags, commonFlags...)

	app.Before = func(ctx *cli.Context) error {
		return utils.Setup(&utils.LogConfig{
			LogFile:       ctx.String(logFileFlag.Name),
			LogJSONFormat: ctx.Bool(logJSONFormat.Name),
			LogDebug:      ctx.Bool(logDebugFlag.Name),
			Verbosity:     ctx.Int(verbosityFlag.Name),
		})
	}

	app.Commands = []*cli.Command{
		{
			Name:   "reset",
			Usage:  "Clean and reset database.",
			Action: resetDB,
			Flags:  dbFlags,
		},
		{
			Name:   "status",
			Usage:  "Check migration status.",
			Action: checkDBStatus,
			Flags:  dbFlags,
		},
		{
			Name:   "version",
			Usage:  "Display the current database version.",
			Action: dbVersion,
			Flags:  dbFlags,
		},
		{
			Name:   "migrate",
			Usage:  "Migrate the database to the latest version.",
			Action: migrateDB,
			Flags:  dbFlags,
		},
		{
			Name:   "rollback",
			Usage:  "Roll back the database to a previous <version>. Rolls back a single migration if no version specified.",
			Action: rollbackDB,
			Flags: append(dbFlags,
				&cli.IntFlag{
					Name:  "version",
					Usage: "Rollback to the specified version.",
					Value: 0,
				}),
		},
	}

	// Run the app for integration-test
	reexec.Register("database-test", func() {
		if err := app.Run(os.Args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	})
	// check if we have been reexec'd
	reexec.Init()
}
