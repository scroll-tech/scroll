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

	"scroll-tech/bridge/config"
	"scroll-tech/bridge/relayer"
	"scroll-tech/bridge/utils"
	"scroll-tech/common/types"
	cutil "scroll-tech/common/utils"
	"scroll-tech/common/version"
	"scroll-tech/database"
)

var (
	app *cli.App
)

func init() {
	// Set up Bridge app info.
	app = cli.NewApp()

	app.Action = action
	app.Name = "message-relayer"
	app.Usage = "The Scroll Message Relayer"
	app.Description = "Message Relayer contains two main service: 1) relay l1 message to l2. 2) relay l2 message to l1."
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
	masterCtx := context.Background()
	subCtx, cancel := context.WithCancel(masterCtx)

	// Init l2geth connection
	l2client, err := ethclient.Dial(cfg.L1Config.Endpoint)
	if err != nil {
		log.Crit("failed to connect l2 geth", "config file", cfgFile, "error", err)
	}

	// Init db connection
	var ormFactory database.OrmFactory
	if ormFactory, err = database.NewOrmFactory(cfg.DBConfig); err != nil {
		log.Crit("failed to init db connection", "err", err)
	}

	l1relayer, err := relayer.NewLayer1Relayer(ctx.Context, ormFactory, cfg.L1Config.RelayerConfig)
	if err != nil {
		log.Crit("failed to create new l1 relayer", "config file", cfgFile, "error", err)
	}
	l2relayer, err := relayer.NewLayer2Relayer(ctx.Context, l2client, ormFactory, cfg.L2Config.RelayerConfig)
	if err != nil {
		log.Crit("failed to create new l2 relayer", "config file", cfgFile, "error", err)
	}

	// Start l1relayer process
	go utils.Loop(subCtx, time.NewTicker(2*time.Second), l1relayer.ProcessSavedEvents)
	go utils.Loop(subCtx, time.NewTicker(2*time.Second), l1relayer.ProcessGasPriceOracle)

	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			case cfm := <-l1relayer.GetMsgChanel():
				if !cfm.IsSuccessful {
					log.Warn("transaction confirmed but failed in layer2", "confirmation", cfm)
				} else {
					// @todo handle db error
					err := ormFactory.UpdateLayer1StatusAndLayer2Hash(ctx, cfm.ID, types.MsgConfirmed, cfm.TxHash.String())
					if err != nil {
						log.Warn("UpdateLayer1StatusAndLayer2Hash failed", "err", err)
					}
					log.Info("transaction confirmed in layer2", "confirmation", cfm)
				}
			case cfm := <-l1relayer.GetGasOracleChanel():
				if !cfm.IsSuccessful {
					// @discuss: maybe make it pending again?
					err := ormFactory.UpdateL1GasOracleStatusAndOracleTxHash(ctx, cfm.ID, types.GasOracleFailed, cfm.TxHash.String())
					if err != nil {
						log.Warn("UpdateL1GasOracleStatusAndOracleTxHash failed", "err", err)
					}
					log.Warn("transaction confirmed but failed in layer2", "confirmation", cfm)
				} else {
					// @todo handle db error
					err := ormFactory.UpdateL1GasOracleStatusAndOracleTxHash(ctx, cfm.ID, types.GasOracleImported, cfm.TxHash.String())
					if err != nil {
						log.Warn("UpdateGasOracleStatusAndOracleTxHash failed", "err", err)
					}
					log.Info("transaction confirmed in layer2", "confirmation", cfm)
				}
			}
		}
	}(subCtx)

	// Start l2relayer process
	go utils.Loop(subCtx, time.NewTicker(time.Second), l2relayer.ProcessSavedEvents)
	go utils.Loop(subCtx, time.NewTicker(time.Second), l2relayer.ProcessGasPriceOracle)

	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			case confirmation := <-l2relayer.GetMsgChanel():
				l2relayer.HandleConfirmation(confirmation)
			case cfm := <-l2relayer.GetGasOracleChanel():
				if !cfm.IsSuccessful {
					// @discuss: maybe make it pending again?
					err := ormFactory.UpdateL2GasOracleStatusAndOracleTxHash(ctx, cfm.ID, types.GasOracleFailed, cfm.TxHash.String())
					if err != nil {
						log.Warn("UpdateL2GasOracleStatusAndOracleTxHash failed", "err", err)
					}
					log.Warn("transaction confirmed but failed in layer1", "confirmation", cfm)
				} else {
					// @todo handle db error
					err := ormFactory.UpdateL2GasOracleStatusAndOracleTxHash(ctx, cfm.ID, types.GasOracleImported, cfm.TxHash.String())
					if err != nil {
						log.Warn("UpdateL2GasOracleStatusAndOracleTxHash failed", "err", err)
					}
					log.Info("transaction confirmed in layer1", "confirmation", cfm)
				}
			}
		}
	}(subCtx)

	// Finish start all message relayer functions
	log.Info("Start message_relayer successfully")

	defer func() {
		cancel()
		err = ormFactory.Close()
		if err != nil {
			log.Error("can not close ormFactory", "error", err)
		}
	}()

	// Catch CTRL-C to ensure a graceful shutdown.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Wait until the interrupt signal is received from an OS signal.
	<-interrupt

	return nil
}

// Run run message_relayer cmd instance.
func Run() {
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
