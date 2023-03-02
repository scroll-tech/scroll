package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"scroll-tech/database"

	cutil "scroll-tech/common/utils"
	"scroll-tech/common/version"

	"scroll-tech/bridge/utils"

	"scroll-tech/bridge/config"
	"scroll-tech/bridge/relayer"
	"scroll-tech/bridge/watcher"
)

var (
	app *cli.App
)

func init() {
	// Set up Bridge app info.
	app = cli.NewApp()

	app.Action = action
	app.Name = "rollup-relayer"
	app.Usage = "The Scroll Rollup Relayer"
	app.Version = version.Version
	app.Flags = append(app.Flags, cutil.CommonFlags...)
	app.Commands = []*cli.Command{}

	app.Before = func(ctx *cli.Context) error {
		return cutil.LogSetup(ctx)
	}
}

func action(ctx *cli.Context) error {
	// Load config file.
	cfgFile := ctx.String(cutil.ConfigFileFlag.Name)
	cfg, err := config.NewConfig(cfgFile)
	if err != nil {
		log.Crit("failed to load config file", "config file", cfgFile, "error", err)
	}

	// Init db connection
	var ormFactory database.OrmFactory
	if ormFactory, err = database.NewOrmFactory(cfg.DBConfig); err != nil {
		log.Crit("failed to init db connection", "err", err)
	}
	subCtx, cancel := context.WithCancel(ctx.Context)
	defer cancel()

	l2client, err := ethclient.Dial(cfg.L1Config.Endpoint)
	if err != nil {
		log.Crit("failed to connect l2 geth", "config file", cfgFile, "error", err)
	}
	l2relayer, err := relayer.NewLayer2Relayer(subCtx, l2client, ormFactory, cfg.L2Config.RelayerConfig)
	if err != nil {
		log.Crit("failed to create l2 relayer", "config file", cfgFile, "error", err)
	}
	batchProposer := watcher.NewBatchProposer(subCtx, cfg.L2Config.BatchProposerConfig, l2relayer, ormFactory)
	if err != nil {
		log.Crit("failed to create batchProposer", "config file", cfgFile, "error", err)
	}
	watcher := watcher.NewL2WatcherClient(subCtx, l2client, cfg.L2Config.Confirmations, cfg.L2Config.L2MessengerAddress, cfg.L2Config.L2MessageQueueAddress, ormFactory)

	// Watcher loop to fetch missing blocks
	go utils.LoopWithContext(subCtx, 3*time.Second, func(ctx context.Context) {
		number, loopErr := utils.GetLatestConfirmedBlockNumber(ctx, l2client, cfg.L2Config.Confirmations)
		if loopErr != nil {
			log.Error("failed to get block number", "err", loopErr)
			return
		}
		watcher.TryFetchRunningMissingBlocks(ctx, number)
	})

	// Batch proposer loop
	go utils.Loop(subCtx, 3*time.Second, func() {
		batchProposer.TryProposeBatch()
		batchProposer.TryCommitBatches()
	})

	// Finish start main functions of batch proposer.
	log.Info("Start batch_proposer successfully")

	defer func() {
		err = ormFactory.Close()
		if err != nil {
			log.Error("can not close ormFactory", "error", err)
		}
	}()

	// Catch CTRL-C to ensure a graceful xsshutdown.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Wait until the interrupt signal is received from an OS signal.
	<-interrupt

	return nil
}

// Run run batch_proposer cmd instance.
func Run() {
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
