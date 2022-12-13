package main

import (
	"fmt"
	"os"

	"github.com/scroll-tech/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"scroll-tech/common/utils"
	"scroll-tech/common/version"

	"scroll-tech/roller/config"
	"scroll-tech/roller/core"
)

func main() {
	app := cli.NewApp()

	app.Action = action
	app.Name = "Roller"
	app.Usage = "The Scroll L2 Roller"
	app.Version = version.Version
	app.Flags = append(app.Flags, utils.CommonFlags...)

	app.Before = func(ctx *cli.Context) error {
		return utils.LogSetup(ctx)
	}

	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func action(ctx *cli.Context) error {
	// Load config file.
	cfgFile := ctx.String(utils.ConfigFileFlag.Name)
	cfg, err := config.NewConfig(cfgFile)
	if err != nil {
		log.Crit("failed to load config file", "config file", cfgFile, "error", err)
	}

	// Create roller
	r, err := core.NewRoller(cfg)
	if err != nil {
		return err
	}
	defer r.Close()
	log.Info("roller start successfully", "name", cfg.RollerName, "publickey", r.PublicKey(), "version", version.Version)

	return r.Run()
}
