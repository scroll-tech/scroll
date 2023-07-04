package app

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/scroll-tech/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"scroll-tech/roller"

	"scroll-tech/common/utils"
	"scroll-tech/common/version"

	"scroll-tech/roller/config"
)

var app *cli.App

func init() {
	app = cli.NewApp()
	app.Action = action
	app.Name = "roller"
	app.Usage = "The Scroll L2 Roller"
	app.Version = version.Version
	app.Flags = append(app.Flags, utils.CommonFlags...)
	app.Before = func(ctx *cli.Context) error {
		_, err := utils.LogSetup(ctx)
		return err
	}

	// Register `roller-test` app for integration-test.
	utils.RegisterSimulation(app, utils.RollerApp)
}

func action(ctx *cli.Context) error {
	// Load config file.
	cfgFile := ctx.String(utils.ConfigFileFlag.Name)
	cfg, err := config.NewConfig(cfgFile)
	if err != nil {
		log.Crit("failed to load config file", "config file", cfgFile, "error", err)
	}

	// Create roller
	r, err := roller.NewRoller(cfg)
	if err != nil {
		return err
	}
	// Start roller.
	r.Start()

	defer r.Stop()
	log.Info("roller start successfully", "name", cfg.RollerName, "publickey", r.PublicKey(), "version", version.Version)

	// Catch CTRL-C to ensure a graceful shutdown.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Wait until the interrupt signal is received from an OS signal.
	<-interrupt

	return nil
}

// Run the roller cmd func.
func Run() {
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
