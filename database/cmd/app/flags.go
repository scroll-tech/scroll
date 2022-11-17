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
	// Set up database app info.
	app = cli.NewApp()
)

var (
	dbFlags = append(utils.DBFlags, &utils.ConfigFileFlag)
)

func init() {
	// Set up database app info.
	app.Name = "database"
	app.Usage = "The Scroll Database CLI"
	app.Version = version.Version
	app.Flags = append(app.Flags, utils.CommonFlags...)
	app.Flags = append(app.Flags, utils.DBFlags...)

	app.Before = func(ctx *cli.Context) error {
		return utils.Setup(&utils.LogConfig{
			LogFile:       ctx.String(utils.LogFileFlag.Name),
			LogJSONFormat: ctx.Bool(utils.LogJSONFormat.Name),
			LogDebug:      ctx.Bool(utils.LogDebugFlag.Name),
			Verbosity:     ctx.Int(utils.VerbosityFlag.Name),
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
			Flags:  utils.DBFlags,
		},
		{
			Name:   "version",
			Usage:  "Display the current database version.",
			Action: dbVersion,
			Flags:  utils.DBFlags,
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
			Flags: append(utils.DBFlags,
				&cli.IntFlag{
					Name:  "version",
					Usage: "Rollback to the specified version.",
					Value: 0,
				}),
		},
	}

	// Run the app for integration-test
	reexec.Register("db_cli-test", func() {
		if err := app.Run(os.Args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	})
	// check if we have been reexec'd
	reexec.Init()
}
