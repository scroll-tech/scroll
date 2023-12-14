package app

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	"scroll-tech/common/utils"
)

// Set up database app info.
var app *cli.App

func init() {
	app = cli.NewApp()
	// Set up database app info.
	app.Name = "db_cli"
	app.Usage = "The Scroll Bridge-history-api DB CLI"
	app.Flags = append(app.Flags, utils.CommonFlags...)

	app.Before = func(ctx *cli.Context) error {
		return utils.LogSetup(ctx)
	}

	app.Commands = []*cli.Command{
		{
			Name:   "reset",
			Usage:  "Clean and reset database.",
			Action: resetDB,
			Flags:  []cli.Flag{&utils.ConfigFileFlag},
		},
		{
			Name:   "status",
			Usage:  "Check migration status.",
			Action: checkDBStatus,
			Flags:  []cli.Flag{&utils.ConfigFileFlag},
		},
		{
			Name:   "version",
			Usage:  "Display the current database version.",
			Action: dbVersion,
			Flags:  []cli.Flag{&utils.ConfigFileFlag},
		},
		{
			Name:   "migrate",
			Usage:  "Migrate the database to the latest version.",
			Action: migrateDB,
			Flags:  []cli.Flag{&utils.ConfigFileFlag},
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
}

// Run database cmd instance.
func Run() {
	// RunApp the db_cli.
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
