package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"bridge-history-api/config"
	"bridge-history-api/crossmsg"
	"bridge-history-api/crossmsg/messageproof"
	"bridge-history-api/db"
	cutils "bridge-history-api/utils"
)

var (
	app *cli.App
)

func init() {
	app = cli.NewApp()

	app.Action = action
	app.Name = "Scroll Bridge History API"
	app.Usage = "The Scroll Bridge Web Backend"
	app.Flags = append(app.Flags, cutils.CommonFlags...)
	app.Commands = []*cli.Command{}

	app.Before = func(ctx *cli.Context) error {
		return cutils.LogSetup(ctx)
	}
}

func action(ctx *cli.Context) error {
	// Load config file.
	cfgFile := ctx.String(cutils.ConfigFileFlag.Name)
	cfg, err := config.NewConfig(cfgFile)
	if err != nil {
		log.Crit("failed to load config file", "config file", cfgFile, "error", err)
	}
	subCtx, cancel := context.WithCancel(ctx.Context)
	defer cancel()

	l1client, err := ethclient.Dial(cfg.L1.Endpoint)
	if err != nil {
		log.Crit("failed to connect l1 geth", "config file", cfgFile, "error", err)
	}
	l2client, err := ethclient.Dial(cfg.L2.Endpoint)
	if err != nil {
		log.Crit("failed to connect l2 geth", "config file", cfgFile, "error", err)
	}
	db, err := db.NewOrmFactory(cfg)

	defer func() {
		err = db.Close()
		if err != nil {
			log.Error("failed to close db", "err", err)
		}
	}()

	if err != nil {
		log.Crit("failed to connect to db", "config file", cfgFile, "error", err)
	}

	l1worker := &crossmsg.FetchEventWorker{F: crossmsg.L1FetchAndSaveEvents, G: crossmsg.GetLatestL1ProcessedHeight, Name: "L1 events fetch Worker"}

	l2worker := &crossmsg.FetchEventWorker{F: crossmsg.L2FetchAndSaveEvents, G: crossmsg.GetLatestL2ProcessedHeight, Name: "L2 events fetch Worker"}

	l1AddressList := []common.Address{
		common.HexToAddress(cfg.L1.CustomERC20GatewayAddr),
		common.HexToAddress(cfg.L1.ERC721GatewayAddr),
		common.HexToAddress(cfg.L1.ERC1155GatewayAddr),
		common.HexToAddress(cfg.L1.MessengerAddr),
		common.HexToAddress(cfg.L1.ETHGatewayAddr),
		common.HexToAddress(cfg.L1.StandardERC20Gateway),
		common.HexToAddress(cfg.L1.WETHGatewayAddr),
	}

	l2AddressList := []common.Address{
		common.HexToAddress(cfg.L2.CustomERC20GatewayAddr),
		common.HexToAddress(cfg.L2.ERC721GatewayAddr),
		common.HexToAddress(cfg.L2.ERC1155GatewayAddr),
		common.HexToAddress(cfg.L2.MessengerAddr),
		common.HexToAddress(cfg.L2.ETHGatewayAddr),
		common.HexToAddress(cfg.L2.StandardERC20Gateway),
		common.HexToAddress(cfg.L2.WETHGatewayAddr),
	}

	l1crossMsgFetcher, err := crossmsg.NewMsgFetcher(subCtx, cfg.L1, db, l1client, l1worker, l1AddressList, crossmsg.L1ReorgHandling)
	if err != nil {
		log.Crit("failed to create l1 cross message fetcher", "error", err)
	}

	go l1crossMsgFetcher.Start()
	defer l1crossMsgFetcher.Stop()

	l2crossMsgFetcher, err := crossmsg.NewMsgFetcher(subCtx, cfg.L2, db, l2client, l2worker, l2AddressList, crossmsg.L2ReorgHandling)
	if err != nil {
		log.Crit("failed to create l2 cross message fetcher", "error", err)
	}

	go l2crossMsgFetcher.Start()
	defer l2crossMsgFetcher.Stop()

	// BlockTimestamp fetcher for l1 and l2
	l1BlockTimeFetcher := crossmsg.NewBlockTimestampFetcher(subCtx, cfg.L1.Confirmation, int(cfg.L1.BlockTime), l1client, db.UpdateL1BlockTimestamp, db.GetL1EarliestNoBlockTimestampHeight)
	go l1BlockTimeFetcher.Start()
	defer l1BlockTimeFetcher.Stop()

	l2BlockTimeFetcher := crossmsg.NewBlockTimestampFetcher(subCtx, cfg.L2.Confirmation, int(cfg.L2.BlockTime), l2client, db.UpdateL2BlockTimestamp, db.GetL2EarliestNoBlockTimestampHeight)
	go l2BlockTimeFetcher.Start()
	defer l2BlockTimeFetcher.Stop()

	// Proof updater and batch fetcher
	l2msgProofUpdater := messageproof.NewMsgProofUpdater(subCtx, cfg.L1.Confirmation, cfg.BatchInfoFetcher.BatchIndexStartBlock, db)
	batchFetcher := crossmsg.NewBatchInfoFetcher(subCtx, common.HexToAddress(cfg.BatchInfoFetcher.ScrollChainAddr), cfg.BatchInfoFetcher.BatchIndexStartBlock, cfg.L1.Confirmation, int(cfg.L1.BlockTime), l1client, db, l2msgProofUpdater)
	go batchFetcher.Start()
	defer batchFetcher.Stop()

	// Catch CTRL-C to ensure a graceful shutdown.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Wait until the interrupt signal is received from an OS signal.
	<-interrupt

	return nil
}

// Run event watcher cmd instance.
func Run() {
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
