package l2

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	// not sure if this will make problems when relay with l1geth

	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/common/utils"

	"scroll-tech/common/types"
	"scroll-tech/database"

	bridge_abi "scroll-tech/bridge/abi"
	"scroll-tech/bridge/config"
	"scroll-tech/bridge/sender"
)

const (
	gasPriceDiffPrecision = 1000000

	defaultGasPriceDiff = 50000 // 5%
)

type batchInterface interface {
	GenerateBatchData(parentBatch *types.BlockBatch, blocks []*types.BlockInfo) (*types.BatchData, error)
}

// Layer2Relayer is responsible for
//  1. Committing and finalizing L2 blocks on L1
//  2. Relaying messages from L2 to L1
//
// Actions are triggered by new head from layer 1 geth node.
// @todo It's better to be triggered by watcher.
type Layer2Relayer struct {
	ctx context.Context

	l2Client *ethclient.Client

	db  database.OrmFactory
	cfg *config.RelayerConfig

	messageSender  *sender.Sender
	messageCh      <-chan *sender.Confirmation
	l1MessengerABI *abi.ABI

	rollupSender *sender.Sender
	rollupCh     <-chan *sender.Confirmation
	l1RollupABI  *abi.ABI

	gasOracleSender *sender.Sender
	gasOracleCh     <-chan *sender.Confirmation
	l2GasOracleABI  *abi.ABI

	lastGasPrice uint64
	minGasPrice  uint64
	gasPriceDiff uint64

	// A list of processing message.
	// key(string): confirmation ID, value(string): layer2 hash.
	processingMessage sync.Map

	// A list of processing batches commitment.
	// key(string): confirmation ID, value([]string): batch hashes.
	processingBatchesCommitment sync.Map

	// A list of processing batch finalization.
	// key(string): confirmation ID, value(string): batch hash.
	processingFinalization sync.Map

	// Use batch_proposer's GenerateBatchData interface.
	batchInterface

	stopCh chan struct{}
}

// NewLayer2Relayer will return a new instance of Layer2RelayerClient
func NewLayer2Relayer(ctx context.Context, l2Client *ethclient.Client, db database.OrmFactory, cfg *config.RelayerConfig) (*Layer2Relayer, error) {
	// @todo use different sender for relayer, block commit and proof finalize
	messageSender, err := sender.NewSender(ctx, cfg.SenderConfig, cfg.MessageSenderPrivateKeys)
	if err != nil {
		log.Error("Failed to create messenger sender", "err", err)
		return nil, err
	}

	rollupSender, err := sender.NewSender(ctx, cfg.SenderConfig, cfg.RollupSenderPrivateKeys)
	if err != nil {
		log.Error("Failed to create rollup sender", "err", err)
		return nil, err
	}

	gasOracleSender, err := sender.NewSender(ctx, cfg.SenderConfig, cfg.GasOracleSenderPrivateKeys)
	if err != nil {
		log.Error("Failed to create gas oracle sender", "err", err)
		return nil, err
	}

	var minGasPrice uint64
	var gasPriceDiff uint64
	if cfg.GasOracleConfig != nil {
		minGasPrice = cfg.GasOracleConfig.MinGasPrice
		gasPriceDiff = cfg.GasOracleConfig.GasPriceDiff
	} else {
		minGasPrice = 0
		gasPriceDiff = defaultGasPriceDiff
	}

	relayer := &Layer2Relayer{
		ctx: ctx,
		db:  db,

		l2Client: l2Client,

		messageSender:  messageSender,
		messageCh:      messageSender.ConfirmChan(),
		l1MessengerABI: bridge_abi.L1ScrollMessengerABI,

		rollupSender: rollupSender,
		rollupCh:     rollupSender.ConfirmChan(),
		l1RollupABI:  bridge_abi.ScrollChainABI,

		gasOracleSender: gasOracleSender,
		gasOracleCh:     gasOracleSender.ConfirmChan(),
		l2GasOracleABI:  bridge_abi.L2GasPriceOracleABI,

		minGasPrice:  minGasPrice,
		gasPriceDiff: gasPriceDiff,

		cfg:                         cfg,
		processingMessage:           sync.Map{},
		processingBatchesCommitment: sync.Map{},
		processingFinalization:      sync.Map{},
		stopCh:                      make(chan struct{}),
	}
	go relayer.confirmLoop(ctx)

	return relayer, nil
}

// SetBatchProposer set interface from batch_proposer.
func (r *Layer2Relayer) SetBatchProposer(proposer batchInterface) {
	r.batchInterface = proposer
}

// Prepare operate layer2's unconfirmed txs.
func (r *Layer2Relayer) Prepare() error {
	var eg errgroup.Group
	eg.Go(func() error {
		if err := r.checkFinalizingBatches(); err != nil {
			log.Error("failed to init layer2 finalizing batches", "err", err)
			return err
		}
		utils.TryTimes(-1, func() bool {
			return r.rollupSender.PendingCount() == 0
		})
		return nil
	})
	eg.Go(func() error {
		if err := r.checkSubmittedMessages(); err != nil {
			log.Error("failed to init layer2 submitted messages", "err", err)
			return err
		}
		utils.TryTimes(-1, func() bool {
			return r.messageSender.PendingCount() == 0
		})
		return nil
	})
	eg.Go(func() error {
		if err := r.checkRollupBatches(); err != nil {
			log.Error("failed to init layer2 rollupCommitting messages", "err", err)
			return err
		}
		utils.TryTimes(-1, func() bool {
			return r.rollupSender.PendingCount() == 0
		})
		return nil
	})

	return eg.Wait()
}

// Start the relayer process
func (r *Layer2Relayer) Start() {
	loop := func(ctx context.Context, f func()) {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				f()
			}
		}
	}

	go func() {
		ctx, cancel := context.WithCancel(r.ctx)

		go loop(ctx, r.ProcessSavedEvents)
		go loop(ctx, r.ProcessCommittedBatches)
		go loop(ctx, r.ProcessGasPriceOracle)

		<-r.stopCh
		cancel()
	}()
}

// Stop the relayer module, for a graceful shutdown.
func (r *Layer2Relayer) Stop() {
	close(r.stopCh)
}

func (r *Layer2Relayer) confirmLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case confirmation := <-r.messageCh:
			r.handleConfirmation(confirmation)
		case confirmation := <-r.rollupCh:
			r.handleConfirmation(confirmation)
		case cfm := <-r.gasOracleCh:
			if !cfm.IsSuccessful {
				// @discuss: maybe make it pending again?
				err := r.db.UpdateL2GasOracleStatusAndOracleTxHash(r.ctx, cfm.ID, types.GasOracleFailed, cfm.TxHash.String())
				if err != nil {
					log.Warn("UpdateL2GasOracleStatusAndOracleTxHash failed", "err", err)
				}
				log.Warn("transaction confirmed but failed in layer1", "confirmation", cfm)
			} else {
				// @todo handle db error
				err := r.db.UpdateL2GasOracleStatusAndOracleTxHash(r.ctx, cfm.ID, types.GasOracleImported, cfm.TxHash.String())
				if err != nil {
					log.Warn("UpdateL2GasOracleStatusAndOracleTxHash failed", "err", err)
				}
				log.Info("transaction confirmed in layer1", "confirmation", cfm)
			}
		}
	}
}

func (r *Layer2Relayer) handleConfirmation(confirmation *sender.Confirmation) {
	if !confirmation.IsSuccessful {
		log.Warn("transaction confirmed but failed in layer1", "confirmation", confirmation)
		return
	}

	transactionType := "Unknown"
	// check whether it is message relay transaction
	if msgHash, ok := r.processingMessage.Load(confirmation.ID); ok {
		transactionType = "MessageRelay"
		// @todo handle db error
		err := r.db.UpdateLayer2StatusAndLayer1Hash(r.ctx, msgHash.(string), types.MsgConfirmed, confirmation.TxHash.String())
		if err != nil {
			log.Warn("UpdateLayer2StatusAndLayer1Hash failed", "msgHash", msgHash.(string), "err", err)
		}
		r.processingMessage.Delete(confirmation.ID)
	}

	// check whether it is CommitBatches transaction
	if batchBatches, ok := r.processingBatchesCommitment.Load(confirmation.ID); ok {
		transactionType = "BatchesCommitment"
		for _, batchHash := range batchBatches.([]string) {
			// @todo handle db error
			err := r.db.UpdateCommitTxHashAndRollupStatus(r.ctx, batchHash, confirmation.TxHash.String(), types.RollupCommitted)
			if err != nil {
				log.Warn("UpdateCommitTxHashAndRollupStatus failed", "batch_hash", batchHash, "err", err)
			}
		}
		r.processingBatchesCommitment.Delete(confirmation.ID)
	}

	// check whether it is proof finalization transaction
	if batchHash, ok := r.processingFinalization.Load(confirmation.ID); ok {
		transactionType = "ProofFinalization"
		// @todo handle db error
		err := r.db.UpdateFinalizeTxHashAndRollupStatus(r.ctx, batchHash.(string), confirmation.TxHash.String(), types.RollupFinalized)
		if err != nil {
			log.Warn("UpdateFinalizeTxHashAndRollupStatus failed", "batch_hash", batchHash.(string), "err", err)
		}
		r.processingFinalization.Delete(confirmation.ID)
	}
	log.Info("transaction confirmed in layer1", "type", transactionType, "confirmation", confirmation)
}
