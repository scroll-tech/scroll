package relayer

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/da-codec/encoding/codecv0"
	"github.com/scroll-tech/da-codec/encoding/codecv1"
	"github.com/scroll-tech/da-codec/encoding/codecv2"
	"github.com/scroll-tech/da-codec/encoding/codecv3"
	"github.com/scroll-tech/da-codec/encoding/codecv4"
	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/crypto/kzg4844"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/params"
	"gorm.io/gorm"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
	"scroll-tech/common/utils"

	bridgeAbi "scroll-tech/rollup/abi"
	"scroll-tech/rollup/internal/config"
	"scroll-tech/rollup/internal/controller/sender"
	"scroll-tech/rollup/internal/orm"
	rutils "scroll-tech/rollup/internal/utils"
)

// Layer2Relayer is responsible for:
// i. committing and finalizing L2 blocks on L1.
// ii. updating L2 gas price oracle contract on L1.
type Layer2Relayer struct {
	ctx context.Context

	l2Client *ethclient.Client

	db         *gorm.DB
	bundleOrm  *orm.Bundle
	batchOrm   *orm.Batch
	chunkOrm   *orm.Chunk
	l2BlockOrm *orm.L2Block

	cfg *config.RelayerConfig

	commitSender   *sender.Sender
	finalizeSender *sender.Sender
	l1RollupABI    *abi.ABI

	gasOracleSender *sender.Sender
	l2GasOracleABI  *abi.ABI

	lastGasPrice uint64
	minGasPrice  uint64
	gasPriceDiff uint64

	// Used to get batch status from chain_monitor api.
	chainMonitorClient *resty.Client

	metrics *l2RelayerMetrics

	chainCfg *params.ChainConfig
}

// NewLayer2Relayer will return a new instance of Layer2RelayerClient
func NewLayer2Relayer(ctx context.Context, l2Client *ethclient.Client, db *gorm.DB, cfg *config.RelayerConfig, chainCfg *params.ChainConfig, initGenesis bool, serviceType ServiceType, reg prometheus.Registerer) (*Layer2Relayer, error) {
	gasOracleSenderPrivateKey, commitSenderPrivateKey, finalizeSenderPrivateKey, err := parsePrivateKeys(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private keys provided by config, err: %v", err)
	}

	var gasOracleSender, commitSender, finalizeSender *sender.Sender
	switch serviceType {
	case ServiceTypeL2GasOracle:
		gasOracleSender, err = sender.NewSender(ctx, cfg.SenderConfig, gasOracleSenderPrivateKey, "l2_relayer", "gas_oracle_sender", types.SenderTypeL2GasOracle, db, reg)
		if err != nil {
			addr := crypto.PubkeyToAddress(gasOracleSenderPrivateKey.PublicKey)
			return nil, fmt.Errorf("new gas oracle sender failed for address %s, err: %w", addr.Hex(), err)
		}

		// Ensure test features aren't enabled on the ethereum mainnet.
		if gasOracleSender.GetChainID().Cmp(big.NewInt(1)) == 0 && cfg.EnableTestEnvBypassFeatures {
			return nil, errors.New("cannot enable test env features in mainnet")
		}

	case ServiceTypeL2RollupRelayer:
		commitSender, err = sender.NewSender(ctx, cfg.SenderConfig, commitSenderPrivateKey, "l2_relayer", "commit_sender", types.SenderTypeCommitBatch, db, reg)
		if err != nil {
			addr := crypto.PubkeyToAddress(commitSenderPrivateKey.PublicKey)
			return nil, fmt.Errorf("new commit sender failed for address %s, err: %w", addr.Hex(), err)
		}

		finalizeSender, err = sender.NewSender(ctx, cfg.SenderConfig, finalizeSenderPrivateKey, "l2_relayer", "finalize_sender", types.SenderTypeFinalizeBatch, db, reg)
		if err != nil {
			addr := crypto.PubkeyToAddress(finalizeSenderPrivateKey.PublicKey)
			return nil, fmt.Errorf("new finalize sender failed for address %s, err: %w", addr.Hex(), err)
		}

		// Ensure test features aren't enabled on the ethereum mainnet.
		if commitSender.GetChainID().Cmp(big.NewInt(1)) == 0 && cfg.EnableTestEnvBypassFeatures {
			return nil, errors.New("cannot enable test env features in mainnet")
		}

	default:
		return nil, fmt.Errorf("invalid service type for l2_relayer: %v", serviceType)
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

	layer2Relayer := &Layer2Relayer{
		ctx: ctx,
		db:  db,

		bundleOrm:  orm.NewBundle(db),
		batchOrm:   orm.NewBatch(db),
		l2BlockOrm: orm.NewL2Block(db),
		chunkOrm:   orm.NewChunk(db),

		l2Client: l2Client,

		commitSender:   commitSender,
		finalizeSender: finalizeSender,
		l1RollupABI:    bridgeAbi.ScrollChainABI,

		gasOracleSender: gasOracleSender,
		l2GasOracleABI:  bridgeAbi.L2GasPriceOracleABI,

		minGasPrice:  minGasPrice,
		gasPriceDiff: gasPriceDiff,

		cfg:      cfg,
		chainCfg: chainCfg,
	}

	// chain_monitor client
	if cfg.ChainMonitor.Enabled {
		layer2Relayer.chainMonitorClient = resty.New()
		layer2Relayer.chainMonitorClient.SetRetryCount(cfg.ChainMonitor.TryTimes)
		layer2Relayer.chainMonitorClient.SetTimeout(time.Duration(cfg.ChainMonitor.TimeOut) * time.Second)
	}

	// Initialize genesis before we do anything else
	if initGenesis {
		if err := layer2Relayer.initializeGenesis(); err != nil {
			return nil, fmt.Errorf("failed to initialize and commit genesis batch, err: %v", err)
		}
	}
	layer2Relayer.metrics = initL2RelayerMetrics(reg)

	switch serviceType {
	case ServiceTypeL2GasOracle:
		go layer2Relayer.handleL2GasOracleConfirmLoop(ctx)
	case ServiceTypeL2RollupRelayer:
		go layer2Relayer.handleL2RollupRelayerConfirmLoop(ctx)
	default:
		return nil, fmt.Errorf("invalid service type for l2_relayer: %v", serviceType)
	}

	return layer2Relayer, nil
}

func (r *Layer2Relayer) initializeGenesis() error {
	if count, err := r.batchOrm.GetBatchCount(r.ctx); err != nil {
		return fmt.Errorf("failed to get batch count: %v", err)
	} else if count > 0 {
		log.Info("genesis already imported", "batch count", count)
		return nil
	}

	genesis, err := r.l2Client.HeaderByNumber(r.ctx, big.NewInt(0))
	if err != nil {
		return fmt.Errorf("failed to retrieve L2 genesis header: %v", err)
	}

	log.Info("retrieved L2 genesis header", "hash", genesis.Hash().String())

	chunk := &encoding.Chunk{
		Blocks: []*encoding.Block{{
			Header:         genesis,
			Transactions:   nil,
			WithdrawRoot:   common.Hash{},
			RowConsumption: &gethTypes.RowConsumption{},
		}},
	}

	err = r.db.Transaction(func(dbTX *gorm.DB) error {
		var dbChunk *orm.Chunk
		dbChunk, err = r.chunkOrm.InsertChunk(r.ctx, chunk, rutils.CodecConfig{Version: encoding.CodecV0}, rutils.ChunkMetrics{}, dbTX)
		if err != nil {
			return fmt.Errorf("failed to insert chunk: %v", err)
		}

		if err = r.chunkOrm.UpdateProvingStatus(r.ctx, dbChunk.Hash, types.ProvingTaskVerified, dbTX); err != nil {
			return fmt.Errorf("failed to update genesis chunk proving status: %v", err)
		}

		batch := &encoding.Batch{
			Index:                      0,
			TotalL1MessagePoppedBefore: 0,
			ParentBatchHash:            common.Hash{},
			Chunks:                     []*encoding.Chunk{chunk},
		}

		var dbBatch *orm.Batch
		dbBatch, err = r.batchOrm.InsertBatch(r.ctx, batch, rutils.CodecConfig{Version: encoding.CodecV0}, rutils.BatchMetrics{}, dbTX)
		if err != nil {
			return fmt.Errorf("failed to insert batch: %v", err)
		}

		if err = r.chunkOrm.UpdateBatchHashInRange(r.ctx, 0, 0, dbBatch.Hash, dbTX); err != nil {
			return fmt.Errorf("failed to update batch hash for chunks: %v", err)
		}

		if err = r.batchOrm.UpdateProvingStatus(r.ctx, dbBatch.Hash, types.ProvingTaskVerified, dbTX); err != nil {
			return fmt.Errorf("failed to update genesis batch proving status: %v", err)
		}

		if err = r.batchOrm.UpdateRollupStatus(r.ctx, dbBatch.Hash, types.RollupFinalized, dbTX); err != nil {
			return fmt.Errorf("failed to update genesis batch rollup status: %v", err)
		}

		// commit genesis batch on L1
		// note: we do this inside the DB transaction so that we can revert all DB changes if this step fails
		return r.commitGenesisBatch(dbBatch.Hash, dbBatch.BatchHeader, common.HexToHash(dbBatch.StateRoot))
	})

	if err != nil {
		return fmt.Errorf("update genesis transaction failed: %v", err)
	}

	log.Info("successfully imported genesis chunk and batch")

	return nil
}

func (r *Layer2Relayer) commitGenesisBatch(batchHash string, batchHeader []byte, stateRoot common.Hash) error {
	// encode "importGenesisBatch" transaction calldata
	calldata, packErr := r.l1RollupABI.Pack("importGenesisBatch", batchHeader, stateRoot)
	if packErr != nil {
		return fmt.Errorf("failed to pack importGenesisBatch with batch header: %v and state root: %v. error: %v", common.Bytes2Hex(batchHeader), stateRoot, packErr)
	}

	// submit genesis batch to L1 rollup contract
	txHash, err := r.commitSender.SendTransaction(batchHash, &r.cfg.RollupContractAddress, calldata, nil, 0)
	if err != nil {
		return fmt.Errorf("failed to send import genesis batch tx to L1, error: %v", err)
	}
	log.Info("importGenesisBatch transaction sent", "contract", r.cfg.RollupContractAddress, "txHash", txHash.String(), "batchHash", batchHash)

	// wait for confirmation
	// we assume that no other transactions are sent before initializeGenesis completes
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		// print progress
		case <-ticker.C:
			log.Info("Waiting for confirmation")

		// timeout
		case <-time.After(5 * time.Minute):
			return fmt.Errorf("import genesis timeout after 5 minutes, original txHash: %v", txHash.String())

		// handle confirmation
		case confirmation := <-r.commitSender.ConfirmChan():
			if confirmation.ContextID != batchHash {
				return fmt.Errorf("unexpected import genesis confirmation id, expected: %v, got: %v", batchHash, confirmation.ContextID)
			}
			if !confirmation.IsSuccessful {
				return errors.New("import genesis batch tx failed")
			}
			log.Info("Successfully committed genesis batch on L1", "txHash", confirmation.TxHash.String())
			return nil
		}
	}
}

// ProcessGasPriceOracle imports gas price to layer1
func (r *Layer2Relayer) ProcessGasPriceOracle() {
	r.metrics.rollupL2RelayerGasPriceOraclerRunTotal.Inc()
	batch, err := r.batchOrm.GetLatestBatch(r.ctx)
	if err != nil {
		log.Error("Failed to GetLatestBatch", "err", err)
		return
	}

	if types.GasOracleStatus(batch.OracleStatus) == types.GasOraclePending {
		suggestGasPrice, err := r.l2Client.SuggestGasPrice(r.ctx)
		if err != nil {
			log.Error("Failed to fetch SuggestGasPrice from l2geth", "err", err)
			return
		}
		suggestGasPriceUint64 := uint64(suggestGasPrice.Int64())
		expectedDelta := r.lastGasPrice * r.gasPriceDiff / gasPriceDiffPrecision
		if r.lastGasPrice > 0 && expectedDelta == 0 {
			expectedDelta = 1
		}

		// last is undefine or (suggestGasPriceUint64 >= minGasPrice && exceed diff)
		if r.lastGasPrice == 0 || (suggestGasPriceUint64 >= r.minGasPrice && (suggestGasPriceUint64 >= r.lastGasPrice+expectedDelta || suggestGasPriceUint64+expectedDelta <= r.lastGasPrice)) {
			data, err := r.l2GasOracleABI.Pack("setL2BaseFee", suggestGasPrice)
			if err != nil {
				log.Error("Failed to pack setL2BaseFee", "batch.Hash", batch.Hash, "GasPrice", suggestGasPrice.Uint64(), "err", err)
				return
			}

			hash, err := r.gasOracleSender.SendTransaction(batch.Hash, &r.cfg.GasPriceOracleContractAddress, data, nil, 0)
			if err != nil {
				log.Error("Failed to send setL2BaseFee tx to layer2 ", "batch.Hash", batch.Hash, "err", err)
				return
			}

			err = r.batchOrm.UpdateL2GasOracleStatusAndOracleTxHash(r.ctx, batch.Hash, types.GasOracleImporting, hash.String())
			if err != nil {
				log.Error("UpdateGasOracleStatusAndOracleTxHash failed", "batch.Hash", batch.Hash, "err", err)
				return
			}
			r.lastGasPrice = suggestGasPriceUint64
			r.metrics.rollupL2RelayerLastGasPrice.Set(float64(r.lastGasPrice))
			log.Info("Update l2 gas price", "txHash", hash.String(), "GasPrice", suggestGasPrice)
		}
	}
}

// ProcessPendingBatches processes the pending batches by sending commitBatch transactions to layer 1.
func (r *Layer2Relayer) ProcessPendingBatches() {
	// get pending batches from database in ascending order by their index.
	dbBatches, err := r.batchOrm.GetFailedAndPendingBatches(r.ctx, 5)
	if err != nil {
		log.Error("Failed to fetch pending L2 batches", "err", err)
		return
	}
	for _, dbBatch := range dbBatches {
		r.metrics.rollupL2RelayerProcessPendingBatchTotal.Inc()

		dbChunks, err := r.chunkOrm.GetChunksInRange(r.ctx, dbBatch.StartChunkIndex, dbBatch.EndChunkIndex)
		if err != nil {
			log.Error("failed to get chunks in range", "err", err)
			return
		}

		chunks := make([]*encoding.Chunk, len(dbChunks))
		for i, c := range dbChunks {
			blocks, getErr := r.l2BlockOrm.GetL2BlocksInRange(r.ctx, c.StartBlockNumber, c.EndBlockNumber)
			if getErr != nil {
				log.Error("failed to get blocks in range", "err", getErr)
				return
			}
			chunks[i] = &encoding.Chunk{Blocks: blocks}
		}

		if dbBatch.Index == 0 {
			log.Error("invalid args: batch index is 0, should only happen in committing genesis batch")
			return
		}

		dbParentBatch, getErr := r.batchOrm.GetBatchByIndex(r.ctx, dbBatch.Index-1)
		if getErr != nil {
			log.Error("failed to get parent batch header", "err", getErr)
			return
		}

		var calldata []byte
		var blob *kzg4844.Blob
		if encoding.CodecVersion(dbBatch.CodecVersion) == encoding.CodecV0 {
			calldata, err = r.constructCommitBatchPayloadCodecV0(dbBatch, dbParentBatch, dbChunks, chunks)
			if err != nil {
				log.Error("failed to construct commitBatch payload codecv0", "index", dbBatch.Index, "err", err)
				return
			}
		} else if encoding.CodecVersion(dbBatch.CodecVersion) == encoding.CodecV1 {
			calldata, blob, err = r.constructCommitBatchPayloadCodecV1(dbBatch, dbParentBatch, dbChunks, chunks)
			if err != nil {
				log.Error("failed to construct commitBatch payload codecv1", "index", dbBatch.Index, "err", err)
				return
			}
		} else if encoding.CodecVersion(dbBatch.CodecVersion) == encoding.CodecV2 {
			calldata, blob, err = r.constructCommitBatchPayloadCodecV2(dbBatch, dbParentBatch, dbChunks, chunks)
			if err != nil {
				log.Error("failed to construct commitBatch payload codecv2", "index", dbBatch.Index, "err", err)
				return
			}
		} else if encoding.CodecVersion(dbBatch.CodecVersion) == encoding.CodecV3 {
			calldata, blob, err = r.constructCommitBatchPayloadCodecV3(dbBatch, dbParentBatch, dbChunks, chunks)
			if err != nil {
				log.Error("failed to construct commitBatchWithBlobProof payload codecv3", "index", dbBatch.Index, "err", err)
				return
			}
		} else if encoding.CodecVersion(dbBatch.CodecVersion) == encoding.CodecV4 {
			calldata, blob, err = r.constructCommitBatchPayloadCodecV4(dbBatch, dbParentBatch, dbChunks, chunks)
			if err != nil {
				log.Error("failed to construct commitBatchWithBlobProof payload codecv4", "index", dbBatch.Index, "err", err)
				return
			}
		}

		// fallbackGasLimit is non-zero only in sending non-blob transactions.
		fallbackGasLimit := uint64(float64(dbBatch.TotalL1CommitGas) * r.cfg.L1CommitGasLimitMultiplier)
		if types.RollupStatus(dbBatch.RollupStatus) == types.RollupCommitFailed {
			// use eth_estimateGas if this batch has been committed and failed at least once.
			fallbackGasLimit = 0
			log.Warn("Batch commit previously failed, using eth_estimateGas for the re-submission", "hash", dbBatch.Hash)
		}

		txHash, err := r.commitSender.SendTransaction(dbBatch.Hash, &r.cfg.RollupContractAddress, calldata, blob, fallbackGasLimit)
		if err != nil {
			if errors.Is(err, sender.ErrTooManyPendingBlobTxs) {
				r.metrics.rollupL2RelayerProcessPendingBatchErrTooManyPendingBlobTxsTotal.Inc()
				log.Debug(
					"Skipped sending commitBatch tx to L1: too many pending blob txs",
					"maxPending", r.cfg.SenderConfig.MaxPendingBlobTxs,
					"err", err,
				)
				return
			}
			log.Error(
				"Failed to send commitBatch tx to layer1",
				"index", dbBatch.Index,
				"hash", dbBatch.Hash,
				"RollupContractAddress", r.cfg.RollupContractAddress,
				"err", err,
				"calldata", common.Bytes2Hex(calldata),
			)
			return
		}

		err = r.batchOrm.UpdateCommitTxHashAndRollupStatus(r.ctx, dbBatch.Hash, txHash.String(), types.RollupCommitting)
		if err != nil {
			log.Error("UpdateCommitTxHashAndRollupStatus failed", "hash", dbBatch.Hash, "index", dbBatch.Index, "err", err)
			return
		}
		r.metrics.rollupL2RelayerProcessPendingBatchSuccessTotal.Inc()
		log.Info("Sent the commitBatch tx to layer1", "batch index", dbBatch.Index, "batch hash", dbBatch.Hash, "tx hash", txHash.String())
	}
}

// ProcessCommittedBatches submit proof to layer 1 rollup contract
func (r *Layer2Relayer) ProcessCommittedBatches() {
	// retrieves the earliest batch whose rollup status is 'committed'
	fields := map[string]interface{}{
		"rollup_status": types.RollupCommitted,
	}
	orderByList := []string{"index ASC"}
	limit := 1
	batches, err := r.batchOrm.GetBatches(r.ctx, fields, orderByList, limit)
	if err != nil {
		log.Error("Failed to fetch committed L2 batches", "err", err)
		return
	}
	if len(batches) != 1 {
		log.Warn("Unexpected result for GetBlockBatches", "number of batches", len(batches))
		return
	}

	r.metrics.rollupL2RelayerProcessCommittedBatchesTotal.Inc()

	batch := batches[0]
	status := types.ProvingStatus(batch.ProvingStatus)
	switch status {
	case types.ProvingTaskUnassigned, types.ProvingTaskAssigned:
		if batch.CommittedAt == nil {
			log.Error("batch.CommittedAt is nil", "index", batch.Index, "hash", batch.Hash)
			return
		}

		if r.cfg.EnableTestEnvBypassFeatures && utils.NowUTC().Sub(*batch.CommittedAt) > time.Duration(r.cfg.FinalizeBatchWithoutProofTimeoutSec)*time.Second {
			if err := r.finalizeBatch(batch, false); err != nil {
				log.Error("Failed to finalize timeout batch without proof", "index", batch.Index, "hash", batch.Hash, "err", err)
			}
		}

	case types.ProvingTaskVerified:
		r.metrics.rollupL2RelayerProcessCommittedBatchesFinalizedTotal.Inc()
		if err := r.finalizeBatch(batch, true); err != nil {
			log.Error("Failed to finalize batch with proof", "index", batch.Index, "hash", batch.Hash, "err", err)
		}

	case types.ProvingTaskFailed:
		// We were unable to prove this batch. There are two possibilities:
		// (a) Prover bug. In this case, we should fix and redeploy the prover.
		//     In the meantime, we continue to commit batches to L1 as well as
		//     proposing and proving chunks and batches.
		// (b) Unprovable batch, e.g. proof overflow. In this case we need to
		//     stop the ledger, fix the limit, revert all the violating blocks,
		//     chunks and batches and all subsequent ones, and resume, i.e. this
		//     case requires manual resolution.
		log.Error(
			"batch proving failed",
			"Index", batch.Index,
			"Hash", batch.Hash,
			"ProvedAt", batch.ProvedAt,
			"ProofTimeSec", batch.ProofTimeSec,
		)

	default:
		log.Error("encounter unreachable case in ProcessCommittedBatches", "proving status", status)
	}
}

// ProcessPendingBundles submits proof to layer 1 rollup contract
func (r *Layer2Relayer) ProcessPendingBundles() {
	r.metrics.rollupL2RelayerProcessPendingBundlesTotal.Inc()

	bundle, err := r.bundleOrm.GetFirstPendingBundle(r.ctx)
	if bundle == nil && err == nil {
		return
	}
	if err != nil {
		log.Error("Failed to fetch first pending L2 bundle", "err", err)
		return
	}

	status := types.ProvingStatus(bundle.ProvingStatus)
	switch status {
	case types.ProvingTaskUnassigned, types.ProvingTaskAssigned:
		if r.cfg.EnableTestEnvBypassFeatures && utils.NowUTC().Sub(bundle.CreatedAt) > time.Duration(r.cfg.FinalizeBundleWithoutProofTimeoutSec)*time.Second {
			if err := r.finalizeBundle(bundle, false); err != nil {
				log.Error("Failed to finalize timeout bundle without proof", "index", bundle.Index, "start batch index", bundle.StartBatchIndex, "end batch index", bundle.EndBatchIndex, "err", err)
			}
		}

	case types.ProvingTaskVerified:
		log.Info("Start to roll up zk proof", "bundle hash", bundle.Hash)
		r.metrics.rollupL2RelayerProcessPendingBundlesFinalizedTotal.Inc()
		if err := r.finalizeBundle(bundle, true); err != nil {
			log.Error("Failed to finalize bundle with proof", "index", bundle.Index, "start batch index", bundle.StartBatchIndex, "end batch index", bundle.EndBatchIndex, "err", err)
		}

	case types.ProvingTaskFailed:
		// We were unable to prove this bundle. There are two possibilities:
		// (a) Prover bug. In this case, we should fix and redeploy the prover.
		//     In the meantime, we continue to commit batches to L1 as well as
		//     proposing and proving chunks, batches and bundles.
		// (b) Unprovable bundle, e.g. proof overflow. In this case we need to
		//     stop the ledger, fix the limit, revert all the violating blocks,
		//     chunks, batches, bundles and all subsequent ones, and resume,
		//     i.e. this case requires manual resolution.
		log.Error("bundle proving failed", "index", bundle.Index, "hash", bundle.Hash, "proved at", bundle.ProvedAt, "proof time sec", bundle.ProofTimeSec)

	default:
		log.Error("encounter unreachable case in ProcessPendingBundles", "proving status", status)
	}
}

func (r *Layer2Relayer) finalizeBatch(dbBatch *orm.Batch, withProof bool) error {
	// Check batch status before sending `finalizeBatch` tx.
	if r.cfg.ChainMonitor.Enabled {
		var batchStatus bool
		batchStatus, err := r.getBatchStatusByIndex(dbBatch)
		if err != nil {
			r.metrics.rollupL2ChainMonitorLatestFailedCall.Inc()
			log.Warn("failed to get batch status, please check chain_monitor api server", "batch_index", dbBatch.Index, "err", err)
			return err
		}
		if !batchStatus {
			r.metrics.rollupL2ChainMonitorLatestFailedBatchStatus.Inc()
			log.Error("the batch status is false, stop finalize batch and check the reason", "batch_index", dbBatch.Index)
			return errors.New("the batch status is false")
		}
	}

	if dbBatch.Index == 0 {
		return errors.New("invalid args: batch index is 0, should only happen in finalizing genesis batch")
	}

	dbParentBatch, getErr := r.batchOrm.GetBatchByIndex(r.ctx, dbBatch.Index-1)
	if getErr != nil {
		return fmt.Errorf("failed to get batch, index: %d, err: %w", dbBatch.Index-1, getErr)
	}

	dbChunks, err := r.chunkOrm.GetChunksInRange(r.ctx, dbBatch.StartChunkIndex, dbBatch.EndChunkIndex)
	if err != nil {
		return fmt.Errorf("failed to fetch chunks: %w", err)
	}

	var aggProof *message.BatchProof
	if withProof {
		aggProof, getErr = r.batchOrm.GetVerifiedProofByHash(r.ctx, dbBatch.Hash)
		if getErr != nil {
			return fmt.Errorf("failed to get verified proof by hash, index: %d, err: %w", dbBatch.Index, getErr)
		}

		if err = aggProof.SanityCheck(); err != nil {
			return fmt.Errorf("failed to check agg_proof sanity, index: %d, err: %w", dbBatch.Index, err)
		}
	}

	var calldata []byte
	if !r.chainCfg.IsBernoulli(new(big.Int).SetUint64(dbChunks[0].StartBlockNumber)) { // codecv0
		log.Info("Start to roll up zk proof", "batch hash", dbBatch.Hash)

		calldata, err = r.constructFinalizeBatchPayloadCodecV0(dbBatch, dbParentBatch, aggProof)
		if err != nil {
			return fmt.Errorf("failed to construct finalizeBatch payload codecv0, index: %v, err: %w", dbBatch.Index, err)
		}
	} else if !r.chainCfg.IsCurie(new(big.Int).SetUint64(dbChunks[0].StartBlockNumber)) { // codecv1
		log.Info("Start to roll up zk proof", "batch hash", dbBatch.Hash)

		chunks := make([]*encoding.Chunk, len(dbChunks))
		for i, c := range dbChunks {
			blocks, dbErr := r.l2BlockOrm.GetL2BlocksInRange(r.ctx, c.StartBlockNumber, c.EndBlockNumber)
			if dbErr != nil {
				return fmt.Errorf("failed to fetch blocks: %w", dbErr)
			}
			chunks[i] = &encoding.Chunk{Blocks: blocks}
		}

		calldata, err = r.constructFinalizeBatchPayloadCodecV1(dbBatch, dbParentBatch, dbChunks, chunks, aggProof)
		if err != nil {
			return fmt.Errorf("failed to construct finalizeBatch payload codecv1, index: %v, err: %w", dbBatch.Index, err)
		}
	} else if !r.chainCfg.IsDarwin(dbChunks[0].StartBlockTime) { // codecv2
		log.Info("Start to roll up zk proof", "batch hash", dbBatch.Hash)

		chunks := make([]*encoding.Chunk, len(dbChunks))
		for i, c := range dbChunks {
			blocks, dbErr := r.l2BlockOrm.GetL2BlocksInRange(r.ctx, c.StartBlockNumber, c.EndBlockNumber)
			if dbErr != nil {
				return fmt.Errorf("failed to fetch blocks: %w", dbErr)
			}
			chunks[i] = &encoding.Chunk{Blocks: blocks}
		}

		calldata, err = r.constructFinalizeBatchPayloadCodecV2(dbBatch, dbParentBatch, dbChunks, chunks, aggProof)
		if err != nil {
			return fmt.Errorf("failed to construct finalizeBatch payload codecv2, index: %v, err: %w", dbBatch.Index, err)
		}
	} else { // codecv3
		log.Debug("encoding is codecv3, using finalizeBundle instead", "index", dbBatch.Index)
		return nil
	}

	txHash, err := r.finalizeSender.SendTransaction(dbBatch.Hash, &r.cfg.RollupContractAddress, calldata, nil, 0)
	if err != nil {
		log.Error(
			"finalizeBatch in layer1 failed",
			"with proof", withProof,
			"index", dbBatch.Index,
			"hash", dbBatch.Hash,
			"RollupContractAddress", r.cfg.RollupContractAddress,
			"err", err,
			"calldata", common.Bytes2Hex(calldata),
		)
		return err
	}

	log.Info("finalizeBatch in layer1", "with proof", withProof, "index", dbBatch.Index, "batch hash", dbBatch.Hash, "tx hash", txHash.String())

	// Updating rollup status in database.
	if err := r.batchOrm.UpdateFinalizeTxHashAndRollupStatus(r.ctx, dbBatch.Hash, txHash.String(), types.RollupFinalizing); err != nil {
		log.Error("UpdateFinalizeTxHashAndRollupStatus failed", "index", dbBatch.Index, "batch hash", dbBatch.Hash, "tx hash", txHash.String(), "err", err)
		return err
	}

	// Updating the proving status when finalizing without proof, thus the coordinator could omit this task.
	// it isn't a necessary step, so don't put in a transaction with UpdateFinalizeTxHashAndRollupStatus
	if !withProof {
		txErr := r.db.Transaction(func(dbTX *gorm.DB) error {
			if updateErr := r.batchOrm.UpdateProvingStatus(r.ctx, dbBatch.Hash, types.ProvingTaskVerified, dbTX); updateErr != nil {
				return updateErr
			}
			if updateErr := r.chunkOrm.UpdateProvingStatusByBatchHash(r.ctx, dbBatch.Hash, types.ProvingTaskVerified, dbTX); updateErr != nil {
				return updateErr
			}
			return nil
		})
		if txErr != nil {
			log.Error("Updating chunk and batch proving status when finalizing without proof failure", "batchHash", dbBatch.Hash, "err", txErr)
		}
	}

	r.metrics.rollupL2RelayerProcessCommittedBatchesFinalizedSuccessTotal.Inc()
	return nil
}

func (r *Layer2Relayer) finalizeBundle(bundle *orm.Bundle, withProof bool) error {
	// Check batch status before sending `finalizeBundle` tx.
	if r.cfg.ChainMonitor.Enabled {
		for batchIndex := bundle.StartBatchIndex; batchIndex <= bundle.EndBatchIndex; batchIndex++ {
			tmpBatch, getErr := r.batchOrm.GetBatchByIndex(r.ctx, batchIndex)
			if getErr != nil {
				log.Error("failed to get batch by index", "batch index", batchIndex, "error", getErr)
				return getErr
			}
			batchStatus, getErr := r.getBatchStatusByIndex(tmpBatch)
			if getErr != nil {
				r.metrics.rollupL2ChainMonitorLatestFailedCall.Inc()
				log.Error("failed to get batch status, please check chain_monitor api server", "batch_index", tmpBatch.Index, "err", getErr)
				return getErr
			}
			if !batchStatus {
				r.metrics.rollupL2ChainMonitorLatestFailedBatchStatus.Inc()
				log.Error("the batch status is false, stop finalize batch and check the reason", "batch_index", tmpBatch.Index)
				return errors.New("the batch status is false")
			}
		}
	}

	dbBatch, err := r.batchOrm.GetBatchByIndex(r.ctx, bundle.EndBatchIndex)
	if err != nil {
		log.Error("failed to get batch by index", "batch index", bundle.EndBatchIndex, "error", err)
		return err
	}

	var aggProof *message.BundleProof
	if withProof {
		aggProof, err = r.bundleOrm.GetVerifiedProofByHash(r.ctx, bundle.Hash)
		if err != nil {
			return fmt.Errorf("failed to get verified proof by bundle index: %d, err: %w", bundle.Index, err)
		}

		if err = aggProof.SanityCheck(); err != nil {
			return fmt.Errorf("failed to check agg_proof sanity, index: %d, err: %w", bundle.Index, err)
		}
	}

	calldata, err := r.constructFinalizeBundlePayloadCodecV3AndV4(dbBatch, aggProof)
	if err != nil {
		return fmt.Errorf("failed to construct finalizeBundle payload codecv3, index: %v, err: %w", dbBatch.Index, err)
	}

	txHash, err := r.finalizeSender.SendTransaction("finalizeBundle-"+bundle.Hash, &r.cfg.RollupContractAddress, calldata, nil, 0)
	if err != nil {
		log.Error("finalizeBundle in layer1 failed", "with proof", withProof, "index", bundle.Index,
			"start batch index", bundle.StartBatchIndex, "end batch index", bundle.EndBatchIndex,
			"RollupContractAddress", r.cfg.RollupContractAddress, "err", err, "calldata", common.Bytes2Hex(calldata))
		return err
	}

	log.Info("finalizeBundle in layer1", "with proof", withProof, "index", bundle.Index, "start batch index", bundle.StartBatchIndex, "end batch index", bundle.EndBatchIndex, "tx hash", txHash.String())

	// Updating rollup status in database.
	if err := r.bundleOrm.UpdateFinalizeTxHashAndRollupStatus(r.ctx, bundle.Hash, txHash.String(), types.RollupFinalizing); err != nil {
		log.Error("UpdateFinalizeTxHashAndRollupStatus failed", "index", bundle.Index, "bundle hash", bundle.Hash, "tx hash", txHash.String(), "err", err)
		return err
	}

	// Updating the proving status when finalizing without proof, thus the coordinator could omit this task.
	// it isn't a necessary step, so don't put in a transaction with UpdateFinalizeTxHashAndRollupStatus
	if !withProof {
		txErr := r.db.Transaction(func(dbTX *gorm.DB) error {
			if updateErr := r.bundleOrm.UpdateProvingStatus(r.ctx, bundle.Hash, types.ProvingTaskVerified, dbTX); updateErr != nil {
				return updateErr
			}
			if updateErr := r.batchOrm.UpdateProvingStatusByBundleHash(r.ctx, bundle.Hash, types.ProvingTaskVerified, dbTX); updateErr != nil {
				return updateErr
			}
			for batchIndex := bundle.StartBatchIndex; batchIndex <= bundle.EndBatchIndex; batchIndex++ {
				tmpBatch, getErr := r.batchOrm.GetBatchByIndex(r.ctx, batchIndex)
				if getErr != nil {
					return getErr
				}
				if updateErr := r.chunkOrm.UpdateProvingStatusByBatchHash(r.ctx, tmpBatch.Hash, types.ProvingTaskVerified, dbTX); updateErr != nil {
					return updateErr
				}
			}
			return nil
		})
		if txErr != nil {
			log.Error("Updating chunk and batch proving status when finalizing without proof failure", "bundleHash", bundle.Hash, "err", txErr)
		}
	}

	r.metrics.rollupL2RelayerProcessPendingBundlesFinalizedSuccessTotal.Inc()
	return nil
}

// batchStatusResponse the response schema
type batchStatusResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
	Data    bool   `json:"data"`
}

func (r *Layer2Relayer) getBatchStatusByIndex(batch *orm.Batch) (bool, error) {
	chunks, getChunkErr := r.chunkOrm.GetChunksInRange(r.ctx, batch.StartChunkIndex, batch.EndChunkIndex)
	if getChunkErr != nil {
		log.Error("Layer2Relayer.getBatchStatusByIndex get chunks range failed", "startChunkIndex", batch.StartChunkIndex, "endChunkIndex", batch.EndChunkIndex, "err", getChunkErr)
		return false, getChunkErr
	}
	if len(chunks) == 0 {
		log.Error("Layer2Relayer.getBatchStatusByIndex get empty chunks", "startChunkIndex", batch.StartChunkIndex, "endChunkIndex", batch.EndChunkIndex)
		return false, fmt.Errorf("startChunksIndex:%d endChunkIndex:%d get empty chunks", batch.StartChunkIndex, batch.EndChunkIndex)
	}

	sort.Slice(chunks, func(i, j int) bool {
		return chunks[i].StartBlockNumber < chunks[j].StartBlockNumber
	})

	startBlockNum := chunks[0].StartBlockNumber
	endBlockNum := chunks[len(chunks)-1].EndBlockNumber
	var response batchStatusResponse
	resp, err := r.chainMonitorClient.R().
		SetQueryParams(map[string]string{
			"batch_index":        fmt.Sprintf("%d", batch.Index),
			"start_block_number": fmt.Sprintf("%d", startBlockNum),
			"end_block_number":   fmt.Sprintf("%d", endBlockNum),
		}).
		SetResult(&response).
		Get(fmt.Sprintf("%s/v1/batch_status", r.cfg.ChainMonitor.BaseURL))
	if err != nil {
		return false, err
	}
	if resp.IsError() {
		return false, resp.Error().(error)
	}
	if response.ErrCode != 0 {
		return false, fmt.Errorf("failed to get batch status, errCode: %d, errMsg: %s", response.ErrCode, response.ErrMsg)
	}

	return response.Data, nil
}

func (r *Layer2Relayer) handleConfirmation(cfm *sender.Confirmation) {
	switch cfm.SenderType {
	case types.SenderTypeCommitBatch:
		var status types.RollupStatus
		if cfm.IsSuccessful {
			status = types.RollupCommitted
			r.metrics.rollupL2BatchesCommittedConfirmedTotal.Inc()
		} else {
			status = types.RollupCommitFailed
			r.metrics.rollupL2BatchesCommittedConfirmedFailedTotal.Inc()
			log.Warn("CommitBatchTxType transaction confirmed but failed in layer1", "confirmation", cfm)
		}

		err := r.batchOrm.UpdateCommitTxHashAndRollupStatus(r.ctx, cfm.ContextID, cfm.TxHash.String(), status)
		if err != nil {
			log.Warn("UpdateCommitTxHashAndRollupStatus failed", "confirmation", cfm, "err", err)
		}
	case types.SenderTypeFinalizeBatch:
		if strings.HasPrefix(cfm.ContextID, "finalizeBundle-") {
			bundleHash := strings.TrimPrefix(cfm.ContextID, "finalizeBundle-")
			var status types.RollupStatus
			if cfm.IsSuccessful {
				status = types.RollupFinalized
				r.metrics.rollupL2BundlesFinalizedConfirmedTotal.Inc()
			} else {
				status = types.RollupFinalizeFailed
				r.metrics.rollupL2BundlesFinalizedConfirmedFailedTotal.Inc()
				log.Warn("FinalizeBundleTxType transaction confirmed but failed in layer1", "confirmation", cfm)
			}

			err := r.db.Transaction(func(dbTX *gorm.DB) error {
				if err := r.batchOrm.UpdateFinalizeTxHashAndRollupStatusByBundleHash(r.ctx, bundleHash, cfm.TxHash.String(), status); err != nil {
					log.Warn("UpdateFinalizeTxHashAndRollupStatusByBundleHash failed", "confirmation", cfm, "err", err)
					return err
				}

				if err := r.bundleOrm.UpdateFinalizeTxHashAndRollupStatus(r.ctx, bundleHash, cfm.TxHash.String(), status); err != nil {
					log.Warn("UpdateFinalizeTxHashAndRollupStatus failed", "confirmation", cfm, "err", err)
					return err
				}
				return nil
			})
			if err != nil {
				log.Warn("failed to update rollup status of bundle and batches", "err", err)
			}
			return
		}

		var status types.RollupStatus
		if cfm.IsSuccessful {
			status = types.RollupFinalized
			r.metrics.rollupL2BatchesFinalizedConfirmedTotal.Inc()
		} else {
			status = types.RollupFinalizeFailed
			r.metrics.rollupL2BatchesFinalizedConfirmedFailedTotal.Inc()
			log.Warn("FinalizeBatchTxType transaction confirmed but failed in layer1", "confirmation", cfm)
		}

		err := r.batchOrm.UpdateFinalizeTxHashAndRollupStatus(r.ctx, cfm.ContextID, cfm.TxHash.String(), status)
		if err != nil {
			log.Warn("UpdateFinalizeTxHashAndRollupStatus failed", "confirmation", cfm, "err", err)
		}
	case types.SenderTypeL2GasOracle:
		batchHash := cfm.ContextID
		var status types.GasOracleStatus
		if cfm.IsSuccessful {
			status = types.GasOracleImported
			r.metrics.rollupL2UpdateGasOracleConfirmedTotal.Inc()
		} else {
			status = types.GasOracleImportedFailed
			r.metrics.rollupL2UpdateGasOracleConfirmedFailedTotal.Inc()
			log.Warn("UpdateGasOracleTxType transaction confirmed but failed in layer1", "confirmation", cfm)
		}

		err := r.batchOrm.UpdateL2GasOracleStatusAndOracleTxHash(r.ctx, batchHash, status, cfm.TxHash.String())
		if err != nil {
			log.Warn("UpdateL2GasOracleStatusAndOracleTxHash failed", "confirmation", cfm, "err", err)
		}
	default:
		log.Warn("Unknown transaction type", "confirmation", cfm)
	}

	log.Info("Transaction confirmed in layer1", "confirmation", cfm)
}

func (r *Layer2Relayer) handleL2GasOracleConfirmLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case cfm := <-r.gasOracleSender.ConfirmChan():
			r.handleConfirmation(cfm)
		}
	}
}

func (r *Layer2Relayer) handleL2RollupRelayerConfirmLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case cfm := <-r.commitSender.ConfirmChan():
			r.handleConfirmation(cfm)
		case cfm := <-r.finalizeSender.ConfirmChan():
			r.handleConfirmation(cfm)
		}
	}
}

func (r *Layer2Relayer) constructCommitBatchPayloadCodecV0(dbBatch *orm.Batch, dbParentBatch *orm.Batch, dbChunks []*orm.Chunk, chunks []*encoding.Chunk) ([]byte, error) {
	daBatch, err := codecv0.NewDABatchFromBytes(dbBatch.BatchHeader)
	if err != nil {
		return nil, fmt.Errorf("failed to create DA batch from bytes: %w", err)
	}

	encodedChunks := make([][]byte, len(dbChunks))
	for i, c := range dbChunks {
		daChunk, createErr := codecv0.NewDAChunk(chunks[i], c.TotalL1MessagesPoppedBefore)
		if createErr != nil {
			return nil, fmt.Errorf("failed to create DA chunk: %w", createErr)
		}
		daChunkBytes, encodeErr := daChunk.Encode()
		if encodeErr != nil {
			return nil, fmt.Errorf("failed to encode DA chunk: %w", encodeErr)
		}
		encodedChunks[i] = daChunkBytes
	}

	calldata, packErr := r.l1RollupABI.Pack("commitBatch", daBatch.Version, dbParentBatch.BatchHeader, encodedChunks, daBatch.SkippedL1MessageBitmap)
	if packErr != nil {
		return nil, fmt.Errorf("failed to pack commitBatch: %w", packErr)
	}
	return calldata, nil
}

func (r *Layer2Relayer) constructCommitBatchPayloadCodecV1(dbBatch *orm.Batch, dbParentBatch *orm.Batch, dbChunks []*orm.Chunk, chunks []*encoding.Chunk) ([]byte, *kzg4844.Blob, error) {
	batch := &encoding.Batch{
		Index:                      dbBatch.Index,
		TotalL1MessagePoppedBefore: dbChunks[0].TotalL1MessagesPoppedBefore,
		ParentBatchHash:            common.HexToHash(dbParentBatch.Hash),
		Chunks:                     chunks,
	}

	daBatch, createErr := codecv1.NewDABatch(batch)
	if createErr != nil {
		return nil, nil, fmt.Errorf("failed to create DA batch: %w", createErr)
	}

	encodedChunks := make([][]byte, len(dbChunks))
	for i, c := range dbChunks {
		daChunk, createErr := codecv1.NewDAChunk(chunks[i], c.TotalL1MessagesPoppedBefore)
		if createErr != nil {
			return nil, nil, fmt.Errorf("failed to create DA chunk: %w", createErr)
		}
		encodedChunks[i] = daChunk.Encode()
	}

	calldata, packErr := r.l1RollupABI.Pack("commitBatch", daBatch.Version, dbParentBatch.BatchHeader, encodedChunks, daBatch.SkippedL1MessageBitmap)
	if packErr != nil {
		return nil, nil, fmt.Errorf("failed to pack commitBatch: %w", packErr)
	}
	return calldata, daBatch.Blob(), nil
}

func (r *Layer2Relayer) constructCommitBatchPayloadCodecV2(dbBatch *orm.Batch, dbParentBatch *orm.Batch, dbChunks []*orm.Chunk, chunks []*encoding.Chunk) ([]byte, *kzg4844.Blob, error) {
	batch := &encoding.Batch{
		Index:                      dbBatch.Index,
		TotalL1MessagePoppedBefore: dbChunks[0].TotalL1MessagesPoppedBefore,
		ParentBatchHash:            common.HexToHash(dbParentBatch.Hash),
		Chunks:                     chunks,
	}

	daBatch, createErr := codecv2.NewDABatch(batch)
	if createErr != nil {
		return nil, nil, fmt.Errorf("failed to create DA batch: %w", createErr)
	}

	encodedChunks := make([][]byte, len(dbChunks))
	for i, c := range dbChunks {
		daChunk, createErr := codecv2.NewDAChunk(chunks[i], c.TotalL1MessagesPoppedBefore)
		if createErr != nil {
			return nil, nil, fmt.Errorf("failed to create DA chunk: %w", createErr)
		}
		encodedChunks[i] = daChunk.Encode()
	}

	calldata, packErr := r.l1RollupABI.Pack("commitBatch", daBatch.Version, dbParentBatch.BatchHeader, encodedChunks, daBatch.SkippedL1MessageBitmap)
	if packErr != nil {
		return nil, nil, fmt.Errorf("failed to pack commitBatch: %w", packErr)
	}
	return calldata, daBatch.Blob(), nil
}

func (r *Layer2Relayer) constructCommitBatchPayloadCodecV3(dbBatch *orm.Batch, dbParentBatch *orm.Batch, dbChunks []*orm.Chunk, chunks []*encoding.Chunk) ([]byte, *kzg4844.Blob, error) {
	batch := &encoding.Batch{
		Index:                      dbBatch.Index,
		TotalL1MessagePoppedBefore: dbChunks[0].TotalL1MessagesPoppedBefore,
		ParentBatchHash:            common.HexToHash(dbParentBatch.Hash),
		Chunks:                     chunks,
	}

	daBatch, createErr := codecv3.NewDABatch(batch)
	if createErr != nil {
		return nil, nil, fmt.Errorf("failed to create DA batch: %w", createErr)
	}

	encodedChunks := make([][]byte, len(dbChunks))
	for i, c := range dbChunks {
		daChunk, createErr := codecv3.NewDAChunk(chunks[i], c.TotalL1MessagesPoppedBefore)
		if createErr != nil {
			return nil, nil, fmt.Errorf("failed to create DA chunk: %w", createErr)
		}
		encodedChunks[i] = daChunk.Encode()
	}

	blobDataProof, err := daBatch.BlobDataProofForPointEvaluation()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get blob data proof for point evaluation: %w", err)
	}

	skippedL1MessageBitmap, _, err := encoding.ConstructSkippedBitmap(batch.Index, batch.Chunks, batch.TotalL1MessagePoppedBefore)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to construct skipped L1 message bitmap: %w", err)
	}

	calldata, packErr := r.l1RollupABI.Pack("commitBatchWithBlobProof", daBatch.Version, dbParentBatch.BatchHeader, encodedChunks, skippedL1MessageBitmap, blobDataProof)
	if packErr != nil {
		return nil, nil, fmt.Errorf("failed to pack commitBatchWithBlobProof: %w", packErr)
	}
	return calldata, daBatch.Blob(), nil
}

func (r *Layer2Relayer) constructCommitBatchPayloadCodecV4(dbBatch *orm.Batch, dbParentBatch *orm.Batch, dbChunks []*orm.Chunk, chunks []*encoding.Chunk) ([]byte, *kzg4844.Blob, error) {
	batch := &encoding.Batch{
		Index:                      dbBatch.Index,
		TotalL1MessagePoppedBefore: dbChunks[0].TotalL1MessagesPoppedBefore,
		ParentBatchHash:            common.HexToHash(dbParentBatch.Hash),
		Chunks:                     chunks,
	}

	daBatch, createErr := codecv4.NewDABatch(batch, dbBatch.EnableCompress)
	if createErr != nil {
		return nil, nil, fmt.Errorf("failed to create DA batch: %w", createErr)
	}

	encodedChunks := make([][]byte, len(dbChunks))
	for i, c := range dbChunks {
		daChunk, createErr := codecv4.NewDAChunk(chunks[i], c.TotalL1MessagesPoppedBefore)
		if createErr != nil {
			return nil, nil, fmt.Errorf("failed to create DA chunk: %w", createErr)
		}
		encodedChunks[i] = daChunk.Encode()
	}

	blobDataProof, err := daBatch.BlobDataProofForPointEvaluation()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get blob data proof for point evaluation: %w", err)
	}

	skippedL1MessageBitmap, _, err := encoding.ConstructSkippedBitmap(batch.Index, batch.Chunks, batch.TotalL1MessagePoppedBefore)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to construct skipped L1 message bitmap: %w", err)
	}

	calldata, packErr := r.l1RollupABI.Pack("commitBatchWithBlobProof", daBatch.Version, dbParentBatch.BatchHeader, encodedChunks, skippedL1MessageBitmap, blobDataProof)
	if packErr != nil {
		return nil, nil, fmt.Errorf("failed to pack commitBatchWithBlobProof: %w", packErr)
	}
	return calldata, daBatch.Blob(), nil
}

func (r *Layer2Relayer) constructFinalizeBatchPayloadCodecV0(dbBatch *orm.Batch, dbParentBatch *orm.Batch, aggProof *message.BatchProof) ([]byte, error) {
	if aggProof != nil { // finalizeBatch with proof.
		calldata, packErr := r.l1RollupABI.Pack(
			"finalizeBatchWithProof",
			dbBatch.BatchHeader,
			common.HexToHash(dbParentBatch.StateRoot),
			common.HexToHash(dbBatch.StateRoot),
			common.HexToHash(dbBatch.WithdrawRoot),
			aggProof.Proof,
		)
		if packErr != nil {
			return nil, fmt.Errorf("failed to pack finalizeBatchWithProof: %w", packErr)
		}
		return calldata, nil
	}

	// finalizeBatch without proof.
	calldata, packErr := r.l1RollupABI.Pack(
		"finalizeBatch",
		dbBatch.BatchHeader,
		common.HexToHash(dbParentBatch.StateRoot),
		common.HexToHash(dbBatch.StateRoot),
		common.HexToHash(dbBatch.WithdrawRoot),
	)
	if packErr != nil {
		return nil, fmt.Errorf("failed to pack finalizeBatch: %w", packErr)
	}
	return calldata, nil
}

func (r *Layer2Relayer) constructFinalizeBatchPayloadCodecV1(dbBatch *orm.Batch, dbParentBatch *orm.Batch, dbChunks []*orm.Chunk, chunks []*encoding.Chunk, aggProof *message.BatchProof) ([]byte, error) {
	batch := &encoding.Batch{
		Index:                      dbBatch.Index,
		TotalL1MessagePoppedBefore: dbChunks[0].TotalL1MessagesPoppedBefore,
		ParentBatchHash:            common.HexToHash(dbParentBatch.Hash),
		Chunks:                     chunks,
	}

	daBatch, createErr := codecv1.NewDABatch(batch)
	if createErr != nil {
		return nil, fmt.Errorf("failed to create DA batch: %w", createErr)
	}

	blobDataProof, getErr := daBatch.BlobDataProof()
	if getErr != nil {
		return nil, fmt.Errorf("failed to get blob data proof: %w", getErr)
	}

	if aggProof != nil { // finalizeBatch4844 with proof.
		calldata, packErr := r.l1RollupABI.Pack(
			"finalizeBatchWithProof4844",
			dbBatch.BatchHeader,
			common.HexToHash(dbParentBatch.StateRoot),
			common.HexToHash(dbBatch.StateRoot),
			common.HexToHash(dbBatch.WithdrawRoot),
			blobDataProof,
			aggProof.Proof,
		)
		if packErr != nil {
			return nil, fmt.Errorf("failed to pack finalizeBatchWithProof4844: %w", packErr)
		}
		return calldata, nil
	}

	// finalizeBatch4844 without proof.
	calldata, packErr := r.l1RollupABI.Pack(
		"finalizeBatch4844",
		dbBatch.BatchHeader,
		common.HexToHash(dbParentBatch.StateRoot),
		common.HexToHash(dbBatch.StateRoot),
		common.HexToHash(dbBatch.WithdrawRoot),
		blobDataProof,
	)
	if packErr != nil {
		return nil, fmt.Errorf("failed to pack finalizeBatch4844: %w", packErr)
	}
	return calldata, nil
}

func (r *Layer2Relayer) constructFinalizeBatchPayloadCodecV2(dbBatch *orm.Batch, dbParentBatch *orm.Batch, dbChunks []*orm.Chunk, chunks []*encoding.Chunk, aggProof *message.BatchProof) ([]byte, error) {
	batch := &encoding.Batch{
		Index:                      dbBatch.Index,
		TotalL1MessagePoppedBefore: dbChunks[0].TotalL1MessagesPoppedBefore,
		ParentBatchHash:            common.HexToHash(dbParentBatch.Hash),
		Chunks:                     chunks,
	}

	daBatch, createErr := codecv2.NewDABatch(batch)
	if createErr != nil {
		return nil, fmt.Errorf("failed to create DA batch: %w", createErr)
	}

	blobDataProof, getErr := daBatch.BlobDataProof()
	if getErr != nil {
		return nil, fmt.Errorf("failed to get blob data proof: %w", getErr)
	}

	if aggProof != nil { // finalizeBatch4844 with proof.
		calldata, packErr := r.l1RollupABI.Pack(
			"finalizeBatchWithProof4844",
			dbBatch.BatchHeader,
			common.HexToHash(dbParentBatch.StateRoot),
			common.HexToHash(dbBatch.StateRoot),
			common.HexToHash(dbBatch.WithdrawRoot),
			blobDataProof,
			aggProof.Proof,
		)
		if packErr != nil {
			return nil, fmt.Errorf("failed to pack finalizeBatchWithProof4844: %w", packErr)
		}
		return calldata, nil
	}

	// finalizeBatch4844 without proof.
	calldata, packErr := r.l1RollupABI.Pack(
		"finalizeBatch4844",
		dbBatch.BatchHeader,
		common.HexToHash(dbParentBatch.StateRoot),
		common.HexToHash(dbBatch.StateRoot),
		common.HexToHash(dbBatch.WithdrawRoot),
		blobDataProof,
	)
	if packErr != nil {
		return nil, fmt.Errorf("failed to pack finalizeBatch4844: %w", packErr)
	}
	return calldata, nil
}

func (r *Layer2Relayer) constructFinalizeBundlePayloadCodecV3AndV4(dbBatch *orm.Batch, aggProof *message.BundleProof) ([]byte, error) {
	if aggProof != nil { // finalizeBundle with proof.
		calldata, packErr := r.l1RollupABI.Pack(
			"finalizeBundleWithProof",
			dbBatch.BatchHeader,
			common.HexToHash(dbBatch.StateRoot),
			common.HexToHash(dbBatch.WithdrawRoot),
			aggProof.Proof,
		)
		if packErr != nil {
			return nil, fmt.Errorf("failed to pack finalizeBundleWithProof: %w", packErr)
		}
		return calldata, nil
	}

	// finalizeBundle without proof.
	calldata, packErr := r.l1RollupABI.Pack(
		"finalizeBundle",
		dbBatch.BatchHeader,
		common.HexToHash(dbBatch.StateRoot),
		common.HexToHash(dbBatch.WithdrawRoot),
	)
	if packErr != nil {
		return nil, fmt.Errorf("failed to pack finalizeBundle: %w", packErr)
	}
	return calldata, nil
}

// StopSenders stops the senders of the rollup-relayer to prevent querying the removed pending_transaction table in unit tests.
// for unit test
func (r *Layer2Relayer) StopSenders() {
	if r.gasOracleSender != nil {
		r.gasOracleSender.Stop()
	}

	if r.commitSender != nil {
		r.commitSender.Stop()
	}

	if r.finalizeSender != nil {
		r.finalizeSender.Stop()
	}
}

func parsePrivateKeys(cfg *config.RelayerConfig) (*ecdsa.PrivateKey, *ecdsa.PrivateKey, *ecdsa.PrivateKey, error) {
	parseKey := func(hexKey string) (*ecdsa.PrivateKey, error) {
		return crypto.ToECDSA(common.FromHex(hexKey))
	}

	gasOracleKey, err := parseKey(cfg.GasOracleSenderPrivateKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("parse gas oracle sender private key failed: %w", err)
	}

	commitKey, err := parseKey(cfg.CommitSenderPrivateKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("parse commit sender private key failed: %w", err)
	}

	finalizeKey, err := parseKey(cfg.FinalizeSenderPrivateKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("parse finalize sender private key failed: %w", err)
	}

	// Check if all three private keys are different
	addrGasOracle := crypto.PubkeyToAddress(gasOracleKey.PublicKey)
	addrCommit := crypto.PubkeyToAddress(commitKey.PublicKey)
	addrFinalize := crypto.PubkeyToAddress(finalizeKey.PublicKey)

	if addrGasOracle == addrCommit || addrGasOracle == addrFinalize || addrCommit == addrFinalize {
		return nil, nil, nil, fmt.Errorf("gas oracle, commit, and finalize sender addresses must be different. Got: Gas Oracle=%s, Commit=%s, Finalize=%s",
			addrGasOracle.Hex(), addrCommit.Hex(), addrFinalize.Hex())
	}

	return gasOracleKey, commitKey, finalizeKey, nil
}
