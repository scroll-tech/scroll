package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"scroll-tech/common/utils"
	"scroll-tech/common/version"

	"scroll-tech/rollup/internal/config"
	"scroll-tech/rollup/internal/controller/watcher"
)

var app *cli.App

func init() {
	// Set up proposer-tool app info.
	app = cli.NewApp()
	app.Action = action
	app.Name = "proposer-tool"
	app.Usage = "The Scroll Proposer Tool"
	app.Version = version.Version
	app.Flags = append(app.Flags, utils.CommonFlags...)
	app.Flags = append(app.Flags, utils.RollupRelayerFlags...)
	app.Flags = append(app.Flags, utils.ProposerToolFlags...)
	app.Commands = []*cli.Command{}
	app.Before = func(ctx *cli.Context) error {
		return utils.LogSetup(ctx)
	}
}

func action(ctx *cli.Context) error {
	// Load config file.
	cfgFile := ctx.String(utils.ConfigFileFlag.Name)
	cfg, err := config.NewConfigForReplay(cfgFile)
	if err != nil {
		log.Crit("failed to load config file", "config file", cfgFile, "error", err)
	}

	subCtx, cancel := context.WithCancel(ctx.Context)

	startL2BlockHeight := ctx.Uint64(utils.StartL2BlockFlag.Name)

	genesisPath := ctx.String(utils.Genesis.Name)
	genesis, err := utils.ReadGenesis(genesisPath)
	if err != nil {
		log.Crit("failed to read genesis", "genesis file", genesisPath, "error", err)
	}

	minCodecVersion := encoding.CodecVersion(ctx.Uint(utils.MinCodecVersionFlag.Name))
	if minCodecVersion < encoding.CodecV7 {
		log.Crit("min codec version must be greater than or equal to CodecV7", "minCodecVersion", minCodecVersion)
	}

	// sanity check config
	if cfg.L2Config.BatchProposerConfig.MaxChunksPerBatch <= 0 {
		log.Crit("cfg.L2Config.BatchProposerConfig.MaxChunksPerBatch must be greater than 0")
	}
	if cfg.L2Config.ChunkProposerConfig.MaxL2GasPerChunk <= 0 {
		log.Crit("cfg.L2Config.ChunkProposerConfig.MaxL2GasPerChunk must be greater than 0")
	}

	proposerTool, err := watcher.NewProposerTool(subCtx, cancel, cfg, startL2BlockHeight, minCodecVersion, genesis.Config)
	if err != nil {
		log.Crit("failed to create proposer tool", "startL2BlockHeight", startL2BlockHeight, "minCodecVersion", minCodecVersion, "error", err)
	}
	proposerTool.Start()

	log.Info("Start proposer-tool successfully", "version", version.Version)

	// Catch CTRL-C to ensure a graceful shutdown.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Wait until the interrupt signal is received from an OS signal.
	<-interrupt

	cancel()
	proposerTool.Stop()

	return nil
}

// Run proposer tool cmd instance.
func Run() {
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
