package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/scroll-tech/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"scroll-tech/prover"

	"scroll-tech/common/utils"
	"scroll-tech/common/version"

	"scroll-tech/prover/config"
)

var app *cli.App

func init() {
	app = cli.NewApp()
	app.Action = action
	app.Name = "prover"
	app.Usage = "The Scroll L2 Prover"
	app.Version = version.Version
	app.Flags = append(app.Flags, utils.CommonFlags...)
	app.Before = func(ctx *cli.Context) error {
		return utils.LogSetup(ctx)
	}

	// Register `prover-test` app for integration-test.
	utils.RegisterSimulation(app, utils.ChunkProverApp)
	utils.RegisterSimulation(app, utils.BatchProverApp)
}

func action(ctx *cli.Context) error {
	// Load config file.
	cfgFile := ctx.String(utils.ConfigFileFlag.Name)
	cfg, err := config.NewConfig(cfgFile)
	if err != nil {
		log.Crit("failed to load config file", "config file", cfgFile, "error", err)
	}

	// Create prover
	r, err := prover.NewProver(context.Background(), cfg)
	if err != nil {
		return err
	}
	// Start prover.
	r.Start()

	defer r.Stop()
	log.Info(
		"prover start successfully",
		"name", cfg.ProverName, "type", cfg.Core.ProofType,
		"publickey", r.PublicKey(), "version", version.Version,
	)

	// Catch CTRL-C to ensure a graceful shutdown.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Wait until the interrupt signal is received from an OS signal.
	<-interrupt

	return nil
}

// Run the prover cmd func.
func Run() {
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
