package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	"scroll-tech/common/utils"
	"scroll-tech/common/version"
)

func main() {
	// Set up database app info.
	app := cli.NewApp()
	app.Name = "db_cli"
	app.Usage = "The Scroll Database CLI"
	app.Version = version.Version
	app.Flags = append(app.Flags, utils.CommonFlags...)

	app.Before = func(ctx *cli.Context) error {
		return utils.LogSetup(ctx)
	}

	app.Commands = []*cli.Command{
		{
			Name:   "reset",
			Usage:  "Clean and reset database.",
			Action: resetDB,
			Flags: []cli.Flag{
				&utils.ConfigFileFlag,
			},
		},
		{
			Name:   "status",
			Usage:  "Check migration status.",
			Action: checkDBStatus,
			Flags: []cli.Flag{
				&utils.ConfigFileFlag,
			},
		},
		{
			Name:   "version",
			Usage:  "Display the current database version.",
			Action: dbVersion,
			Flags: []cli.Flag{
				&utils.ConfigFileFlag,
			},
		},
		{
			Name:   "migrate",
			Usage:  "Migrate the database to the latest version.",
			Action: migrateDB,
			Flags: []cli.Flag{
				&utils.ConfigFileFlag,
			},
		},
		{
			Name:   "rollback",
			Usage:  "Roll back the database to a previous <version>. Rolls back a single migration if no version specified.",
			Action: rollbackDB,
			Flags: []cli.Flag{
				&utils.ConfigFileFlag,
				&cli.IntFlag{
					Name:  "version",
					Usage: "Rollback to the specified version.",
					Value: 0,
				},
			},
		},
	}

	// Run the database.
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
