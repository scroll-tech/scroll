package relayer

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
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

// RelaxType enumerates the relaxation functions we support when
// turning a baseline fee into a “target” fee.
type RelaxType int

const (
	// NoRelaxation means “don’t touch the baseline” (i.e. fallback/default).
	NoRelaxation RelaxType = iota
	Exponential
	Sigmoid
)

const secondsPerBlock = 12

// BaselineType enumerates the baseline types we support when
// turning a baseline fee into a “target” fee.
type BaselineType int

const (
	// PctMin means “take the minimum of the last N blocks’ fees, then
	// take the PCT of that”.
	PctMin BaselineType = iota
	// EWMA means “take the exponentially‐weighted moving average of
	// the last N blocks’ fees”.
	EWMA
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
	l1BlockOrm *orm.L1Block

	cfg *config.RelayerConfig

	commitSender   *sender.Sender
	finalizeSender *sender.Sender
	l1RollupABI    *abi.ABI

	l2GasOracleABI *abi.ABI

	// Used to get batch status from chain_monitor api.
	chainMonitorClient *resty.Client

	metrics *l2RelayerMetrics

	chainCfg *params.ChainConfig

	lastFetchedBlock uint64     // highest block number ever pulled
	feeHistory       []*big.Int // sliding window of blob fees
	batchStrategy    StrategyParams
}

// StrategyParams holds the per‐window fee‐submission rules.
type StrategyParams struct {
	BaselineType  BaselineType // "pct_min" or "ewma"
	BaselineParam float64      // percentile (0–1) or α for EWMA
	Gamma         float64      // relaxation γ
	Beta          float64      // relaxation β
	RelaxType     RelaxType    // Exponential or Sigmoid
}

// bestParams maps your 2h/5h/12h windows to their best rules.
// Timeouts are in seconds, 2, 5 and 12 hours (and same + 20 mins to account for
// time to create batch currently roughly, as time is measured from block creation)
var bestParams = map[uint64]StrategyParams{
	7200:  {BaselineType: PctMin, BaselineParam: 0.10, Gamma: 0.4, Beta: 8, RelaxType: Exponential},
	8400:  {BaselineType: PctMin, BaselineParam: 0.10, Gamma: 0.4, Beta: 8, RelaxType: Exponential},
	18000: {BaselineType: PctMin, BaselineParam: 0.30, Gamma: 0.6, Beta: 20, RelaxType: Sigmoid},
	19200: {BaselineType: PctMin, BaselineParam: 0.30, Gamma: 0.6, Beta: 20, RelaxType: Sigmoid},
	42800: {BaselineType: PctMin, BaselineParam: 0.50, Gamma: 0.5, Beta: 20, RelaxType: Sigmoid},
	44400: {BaselineType: PctMin, BaselineParam: 0.50, Gamma: 0.5, Beta: 20, RelaxType: Sigmoid},
}

// NewLayer2Relayer will return a new instance of Layer2RelayerClient
func NewLayer2Relayer(ctx context.Context, l2Client *ethclient.Client, db *gorm.DB, cfg *config.RelayerConfig, chainCfg *params.ChainConfig, serviceType ServiceType, reg prometheus.Registerer) (*Layer2Relayer, error) {
	var commitSender, finalizeSender *sender.Sender

	switch serviceType {
	case ServiceTypeL2RollupRelayer:
		commitSenderAddr, err := addrFromSignerConfig(cfg.CommitSenderSignerConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to parse addr from commit sender config, err: %v", err)
		}
		finalizeSenderAddr, err := addrFromSignerConfig(cfg.FinalizeSenderSignerConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to parse addr from finalize sender config, err: %v", err)
		}
		if commitSenderAddr == finalizeSenderAddr {
			return nil, fmt.Errorf("commit and finalize sender addresses must be different. Got: Commit=%s, Finalize=%s", commitSenderAddr.Hex(), finalizeSenderAddr.Hex())
		}

		commitSender, err = sender.NewSender(ctx, cfg.SenderConfig, cfg.CommitSenderSignerConfig, "l2_relayer", "commit_sender", types.SenderTypeCommitBatch, db, reg)
		if err != nil {
			return nil, fmt.Errorf("new commit sender failed, err: %w", err)
		}

		finalizeSender, err = sender.NewSender(ctx, cfg.SenderConfig, cfg.FinalizeSenderSignerConfig, "l2_relayer", "finalize_sender", types.SenderTypeFinalizeBatch, db, reg)
		if err != nil {
			return nil, fmt.Errorf("new finalize sender failed, err: %w", err)
		}

		// Ensure test features aren't enabled on the ethereum mainnet.
		if commitSender.GetChainID().Cmp(big.NewInt(1)) == 0 && cfg.EnableTestEnvBypassFeatures {
			return nil, errors.New("cannot enable test env features in mainnet")
		}

	default:
		return nil, fmt.Errorf("invalid service type for l2_relayer: %v", serviceType)
	}

	strategy, ok := bestParams[uint64(cfg.BatchSubmission.TimeoutSec)]
	if !ok {
		return nil, fmt.Errorf("invalid timeout for batch submission: %v", cfg.BatchSubmission.TimeoutSec)
	}

	layer2Relayer := &Layer2Relayer{
		ctx: ctx,
		db:  db,

		bundleOrm:  orm.NewBundle(db),
		batchOrm:   orm.NewBatch(db),
		l1BlockOrm: orm.NewL1Block(db),
		l2BlockOrm: orm.NewL2Block(db),
		chunkOrm:   orm.NewChunk(db),

		l2Client: l2Client,

		commitSender:   commitSender,
		finalizeSender: finalizeSender,
		l1RollupABI:    bridgeAbi.ScrollChainABI,

		l2GasOracleABI: bridgeAbi.L2GasPriceOracleABI,
		batchStrategy:  strategy,
		cfg:            cfg,
		chainCfg:       chainCfg,
	}

	// chain_monitor client
	if serviceType == ServiceTypeL2RollupRelayer && cfg.ChainMonitor.Enabled {
		layer2Relayer.chainMonitorClient = resty.New()
		layer2Relayer.chainMonitorClient.SetRetryCount(cfg.ChainMonitor.TryTimes)
		layer2Relayer.chainMonitorClient.SetTimeout(time.Duration(cfg.ChainMonitor.TimeOut) * time.Second)
	}

	// Initialize genesis before we do anything else
	if err := layer2Relayer.initializeGenesis(); err != nil {
		return nil, fmt.Errorf("failed to initialize and commit genesis batch, err: %v", err)
	}
	layer2Relayer.metrics = initL2RelayerMetrics(reg)

	switch serviceType {
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

	chunk := &encoding.Chunk{Blocks: []*encoding.Block{{Header: genesis}}}

	err = r.db.Transaction(func(dbTX *gorm.DB) error {
		if err = r.l2BlockOrm.InsertL2Blocks(r.ctx, chunk.Blocks, dbTX); err != nil {
			return fmt.Errorf("failed to insert genesis block: %v", err)
		}

		var dbChunk *orm.Chunk
		dbChunk, err = r.chunkOrm.InsertChunk(r.ctx, chunk, encoding.CodecV0, rutils.ChunkMetrics{}, dbTX)
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
		dbBatch, err = r.batchOrm.InsertBatch(r.ctx, batch, encoding.CodecV0, rutils.BatchMetrics{}, dbTX)
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
	txHash, _, err := r.commitSender.SendTransaction(batchHash, &r.cfg.RollupContractAddress, calldata, nil)
	if err != nil {
		return fmt.Errorf("failed to send import genesis batch tx to L1, error: %v", err)
	}
	log.Info("importGenesisBatch transaction sent", "contract", r.cfg.RollupContractAddress, "txHash", txHash, "batchHash", batchHash)

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

// ProcessPendingBatches processes the pending batches by sending commitBatch transactions to layer 1.
// Pending batchess are submitted if one of the following conditions is met:
// - the first batch is too old -> forceSubmit
// - backlogCount > r.cfg.BatchSubmission.BacklogMax -> forceSubmit
// - we have at least minBatches AND price hits a desired target price
func (r *Layer2Relayer) ProcessPendingBatches() {
	// get pending batches from database in ascending order by their index.
	dbBatches, err := r.batchOrm.GetFailedAndPendingBatches(r.ctx, r.cfg.BatchSubmission.MaxBatches)
	if err != nil {
		log.Error("Failed to fetch pending L2 batches", "err", err)
		return
	}

	// nothing to do if we don't have any pending batches
	if len(dbBatches) == 0 {
		return
	}

	// if backlog outgrow max size, force‐submit enough oldest batches
	backlogCount, err := r.batchOrm.GetFailedAndPendingBatchesCount(r.ctx)
	r.metrics.rollupL2RelayerBacklogCounts.Set(float64(backlogCount))

	if err != nil {
		log.Error("Failed to fetch pending L2 batches", "err", err)
		return
	}

	var forceSubmit bool

	startChunk, err := r.chunkOrm.GetChunkByIndex(r.ctx, dbBatches[0].StartChunkIndex)
	if err != nil {
		log.Error("failed to get first chunk", "err", err, "batch index", dbBatches[0].Index, "chunk index", dbBatches[0].StartChunkIndex)
		return
	}
	oldestBlockTimestamp := time.Unix(int64(startChunk.StartBlockTime), 0)

	// if the batch with the oldest index is too old, we force submit all batches that we have so far in the next step
	if r.cfg.BatchSubmission.TimeoutSec > 0 && time.Since(oldestBlockTimestamp) > time.Duration(r.cfg.BatchSubmission.TimeoutSec)*time.Second {
		forceSubmit = true
	}

	// force submit if backlog is too big
	if backlogCount > r.cfg.BatchSubmission.BacklogMax {
		forceSubmit = true
	}

	if !forceSubmit {
		// check if we should skip submitting the batch based on the fee target
		skip, err := r.skipSubmitByFee(oldestBlockTimestamp, r.metrics)
		// return if not hitting target price
		if skip {
			log.Debug("Skipping batch submission", "first batch index", dbBatches[0].Index, "backlog count", backlogCount, "reason", err)
			log.Debug("first batch index", dbBatches[0].Index)
			log.Debug("backlog count", backlogCount)
			return
		}
		if err != nil {
			log.Warn("Failed to check if we should skip batch submission, fallback to immediate submission", "err", err)
		}
	}

	var batchesToSubmit []*dbBatchWithChunks
	for i, dbBatch := range dbBatches {
		var dbChunks []*orm.Chunk
		var dbParentBatch *orm.Batch

		// Verify batches compatibility
		dbChunks, err = r.chunkOrm.GetChunksInRange(r.ctx, dbBatch.StartChunkIndex, dbBatch.EndChunkIndex)
		if err != nil {
			log.Error("failed to get chunks in range", "err", err)
			return
		}

		// check codec version
		for _, dbChunk := range dbChunks {
			if dbBatch.CodecVersion != dbChunk.CodecVersion {
				log.Error("batch codec version is different from chunk codec version", "batch index", dbBatch.Index, "chunk index", dbChunk.Index, "batch codec version", dbBatch.CodecVersion, "chunk codec version", dbChunk.CodecVersion)
				return
			}
		}

		if dbBatch.Index == 0 {
			log.Error("invalid args: batch index is 0, should only happen in committing genesis batch")
			return
		}

		// get parent batch
		if i == 0 {
			dbParentBatch, err = r.batchOrm.GetBatchByIndex(r.ctx, dbBatch.Index-1)
			if err != nil {
				log.Error("failed to get parent batch header", "err", err)
				return
			}
		} else {
			dbParentBatch = dbBatches[i-1]
		}

		// make sure batch index is continuous
		if dbParentBatch.Index != dbBatch.Index-1 {
			log.Error("parent batch index is not equal to current batch index - 1", "index", dbBatch.Index, "parent index", dbParentBatch.Index)
			return
		}

		if dbParentBatch.CodecVersion > dbBatch.CodecVersion {
			log.Error("parent batch codec version is greater than current batch codec version", "index", dbBatch.Index, "hash", dbBatch.Hash, "parent codec version", dbParentBatch.CodecVersion, "current codec version", dbBatch.CodecVersion)
			return
		}

		// make sure we commit batches of the same codec version together.
		// If we encounter a batch with a different codec version, we stop here and will commit the batches we have so far.
		// The next call of ProcessPendingBatches will then start with the batch with the different codec version.
		batchesToSubmitLen := len(batchesToSubmit)
		if batchesToSubmitLen > 0 && batchesToSubmit[batchesToSubmitLen-1].Batch.CodecVersion != dbBatch.CodecVersion {
			break
		}

		if batchesToSubmitLen < r.cfg.BatchSubmission.MaxBatches {
			batchesToSubmit = append(batchesToSubmit, &dbBatchWithChunks{
				Batch:  dbBatch,
				Chunks: dbChunks,
			})
		}

		if len(batchesToSubmit) >= r.cfg.BatchSubmission.MaxBatches {
			break
		}
	}

	// we only submit batches if we have a timeout or if we have enough batches to submit
	if !forceSubmit && len(batchesToSubmit) < r.cfg.BatchSubmission.MinBatches {
		log.Debug("Not enough batches to submit", "count", len(batchesToSubmit), "minBatches", r.cfg.BatchSubmission.MinBatches, "maxBatches", r.cfg.BatchSubmission.MaxBatches)
		return
	}

	if forceSubmit {
		log.Info("Forcing submission of batches due to timeout", "batch index", batchesToSubmit[0].Batch.Index, "first block created at", oldestBlockTimestamp)
	}

	// We have at least 1 batch to commit
	firstBatch := batchesToSubmit[0].Batch
	lastBatch := batchesToSubmit[len(batchesToSubmit)-1].Batch

	var calldata []byte
	var blobs []*kzg4844.Blob
	var maxBlockHeight uint64
	var totalGasUsed uint64

	codecVersion := encoding.CodecVersion(firstBatch.CodecVersion)
	switch codecVersion {
	case encoding.CodecV7, encoding.CodecV8:
		calldata, blobs, maxBlockHeight, totalGasUsed, err = r.constructCommitBatchPayloadCodecV7(batchesToSubmit, firstBatch, lastBatch)
		if err != nil {
			log.Error("failed to construct constructCommitBatchPayloadCodecV7 payload for V7", "codecVersion", codecVersion, "start index", firstBatch.Index, "end index", lastBatch.Index, "err", err)
			return
		}
	default:
		log.Error("unsupported codec version in ProcessPendingBatches", "codecVersion", codecVersion, "start index", firstBatch, "end index", lastBatch.Index)
		return
	}

	txHash, blobBaseFee, err := r.commitSender.SendTransaction(r.contextIDFromBatches(codecVersion, batchesToSubmit), &r.cfg.RollupContractAddress, calldata, blobs)
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
			"start index", firstBatch.Index,
			"start hash", firstBatch.Hash,
			"end index", lastBatch.Index,
			"end hash", lastBatch.Hash,
			"RollupContractAddress", r.cfg.RollupContractAddress,
			"err", err,
			"calldata", common.Bytes2Hex(calldata),
		)
		return
	}

	if err = r.db.Transaction(func(dbTX *gorm.DB) error {
		for _, dbBatch := range batchesToSubmit {
			if err = r.batchOrm.UpdateCommitTxHashAndRollupStatus(r.ctx, dbBatch.Batch.Hash, txHash.String(), types.RollupCommitting, dbTX); err != nil {
				return fmt.Errorf("UpdateCommitTxHashAndRollupStatus failed for batch %d: %s, err %v", dbBatch.Batch.Index, dbBatch.Batch.Hash, err)
			}
		}

		return nil
	}); err != nil {
		log.Error("failed to update status for batches to RollupCommitting", "err", err)
	}

	r.metrics.rollupL2RelayerCommitBlockHeight.Set(float64(maxBlockHeight))
	r.metrics.rollupL2RelayerCommitThroughput.Add(float64(totalGasUsed))
	r.metrics.rollupL2RelayerProcessPendingBatchSuccessTotal.Add(float64(len(batchesToSubmit)))
	r.metrics.rollupL2RelayerProcessBatchesPerTxCount.Set(float64(len(batchesToSubmit)))
	r.metrics.rollupL2RelayerCommitLatency.Set(time.Since(oldestBlockTimestamp).Seconds())
	r.metrics.rollupL2RelayerCommitPrice.Set(float64(blobBaseFee))

	log.Info("Sent the commitBatches tx to layer1", "batches count", len(batchesToSubmit), "start index", firstBatch.Index, "start hash", firstBatch.Hash, "end index", lastBatch.Index, "end hash", lastBatch.Hash, "tx hash", txHash.String())
}

func (r *Layer2Relayer) contextIDFromBatches(codecVersion encoding.CodecVersion, batches []*dbBatchWithChunks) string {
	contextIDs := []string{fmt.Sprintf("v%d", codecVersion)}
	for _, batch := range batches {
		contextIDs = append(contextIDs, batch.Batch.Hash)
	}
	return strings.Join(contextIDs, "-")
}

func (r *Layer2Relayer) batchHashesFromContextID(contextID string) []string {
	parts := strings.SplitN(contextID, "-", 2)
	if len(parts) == 2 && strings.HasPrefix(parts[0], "v") {
		return strings.Split(parts[1], "-")
	}
	return []string{contextID}
}

type dbBatchWithChunks struct {
	Batch  *orm.Batch
	Chunks []*orm.Chunk
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
			// check if last batch is finalized, because in fake finalize bundle mode, the contract does not verify if the previous bundle or batch is finalized.
			if bundle.StartBatchIndex == 0 {
				log.Error("invalid args: start batch index of bundle is 0", "bundle index", bundle.Index, "start batch index", bundle.StartBatchIndex, "end batch index", bundle.EndBatchIndex)
				return
			}

			lastBatch, err := r.batchOrm.GetBatchByIndex(r.ctx, bundle.StartBatchIndex-1)
			if err != nil {
				log.Error("failed to get last batch", "batch index", bundle.StartBatchIndex-1, "err", err)
				return
			}

			if types.RollupStatus(lastBatch.RollupStatus) != types.RollupFinalized {
				log.Error("previous bundle or batch is not finalized", "batch index", lastBatch.Index, "batch hash", lastBatch.Hash, "rollup status", types.RollupStatus(lastBatch.RollupStatus))
				return
			}

			if err := r.finalizeBundle(bundle, false); err != nil {
				log.Error("failed to finalize timeout bundle without proof", "bundle index", bundle.Index, "start batch index", bundle.StartBatchIndex, "end batch index", bundle.EndBatchIndex, "err", err)
				return
			}
		}

	case types.ProvingTaskVerified:
		log.Info("Start to roll up zk proof", "index", bundle.Index, "bundle hash", bundle.Hash)
		r.metrics.rollupL2RelayerProcessPendingBundlesFinalizedTotal.Inc()
		if err := r.finalizeBundle(bundle, true); err != nil {
			log.Error("failed to finalize bundle with proof", "bundle index", bundle.Index, "start batch index", bundle.StartBatchIndex, "end batch index", bundle.EndBatchIndex, "err", err)
			return
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
		log.Error("bundle proving failed", "bundle index", bundle.Index, "bundle hash", bundle.Hash, "proved at", bundle.ProvedAt, "proof time sec", bundle.ProofTimeSec)

	default:
		log.Error("encounter unreachable case in ProcessPendingBundles", "proving status", status)
	}
}

func (r *Layer2Relayer) finalizeBundle(bundle *orm.Bundle, withProof bool) error {
	// Check if current bundle codec version is not less than the preceding one
	if bundle.StartBatchIndex > 0 {
		prevBatch, err := r.batchOrm.GetBatchByIndex(r.ctx, bundle.StartBatchIndex-1)
		if err != nil {
			log.Error("failed to get previous batch",
				"current bundle index", bundle.Index,
				"start batch index", bundle.StartBatchIndex,
				"error", err)
			return err
		}
		if bundle.CodecVersion < prevBatch.CodecVersion {
			log.Error("current bundle codec version is less than the preceding batch",
				"current bundle index", bundle.Index,
				"current codec version", bundle.CodecVersion,
				"prev batch index", prevBatch.Index,
				"prev codec version", prevBatch.CodecVersion)
			return errors.New("current bundle codec version cannot be less than the preceding batch")
		}
	}

	// Check batch status before sending `finalizeBundle` tx.
	for batchIndex := bundle.StartBatchIndex; batchIndex <= bundle.EndBatchIndex; batchIndex++ {
		tmpBatch, getErr := r.batchOrm.GetBatchByIndex(r.ctx, batchIndex)
		if getErr != nil {
			log.Error("failed to get batch by index", "batch index", batchIndex, "error", getErr)
			return getErr
		}

		// check codec version
		if tmpBatch.CodecVersion != bundle.CodecVersion {
			log.Error("bundle codec version is different from batch codec version", "bundle index", bundle.Index, "batch index", tmpBatch.Index, "bundle codec version", bundle.CodecVersion, "batch codec version", tmpBatch.CodecVersion)
			return errors.New("bundle codec version is different from batch codec version")
		}

		if r.cfg.ChainMonitor.Enabled {
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

	firstChunk, err := r.chunkOrm.GetChunkByIndex(r.ctx, dbBatch.StartChunkIndex)
	if err != nil || firstChunk == nil {
		log.Error("failed to get first chunk of batch", "chunk index", dbBatch.StartChunkIndex, "error", err)
		return fmt.Errorf("failed to get first chunk of batch: %w", err)
	}

	endChunk, err := r.chunkOrm.GetChunkByIndex(r.ctx, dbBatch.EndChunkIndex)
	if err != nil || endChunk == nil {
		log.Error("failed to get end chunk of batch", "chunk index", dbBatch.EndChunkIndex, "error", err)
		return fmt.Errorf("failed to get end chunk of batch: %w", err)
	}

	var aggProof *message.OpenVMBundleProof
	if withProof {
		aggProof, err = r.bundleOrm.GetVerifiedProofByHash(r.ctx, bundle.Hash)
		if err != nil {
			return fmt.Errorf("failed to get verified proof by bundle index: %d, err: %w", bundle.Index, err)
		}

		if err = aggProof.SanityCheck(); err != nil {
			return fmt.Errorf("failed to check agg_proof sanity, index: %d, err: %w", bundle.Index, err)
		}
	}

	var calldata []byte
	switch encoding.CodecVersion(bundle.CodecVersion) {
	case encoding.CodecV7, encoding.CodecV8:
		calldata, err = r.constructFinalizeBundlePayloadCodecV7(dbBatch, endChunk, aggProof)
		if err != nil {
			return fmt.Errorf("failed to construct finalizeBundle payload codecv7, bundle index: %v, last batch index: %v, err: %w", bundle.Index, dbBatch.Index, err)
		}
	default:
		return fmt.Errorf("unsupported codec version in finalizeBundle, bundle index: %v, version: %d", bundle.Index, bundle.CodecVersion)
	}

	txHash, _, err := r.finalizeSender.SendTransaction("finalizeBundle-"+bundle.Hash, &r.cfg.RollupContractAddress, calldata, nil)
	if err != nil {
		log.Error("finalizeBundle in layer1 failed", "with proof", withProof, "index", bundle.Index,
			"start batch index", bundle.StartBatchIndex, "end batch index", bundle.EndBatchIndex,
			"RollupContractAddress", r.cfg.RollupContractAddress, "err", err, "calldata", common.Bytes2Hex(calldata))
		return err
	}

	log.Info("finalizeBundle in layer1", "with proof", withProof, "index", bundle.Index, "start batch index", bundle.StartBatchIndex, "end batch index", bundle.EndBatchIndex, "tx hash", txHash.String())

	// Updating rollup status in database.
	err = r.db.Transaction(func(dbTX *gorm.DB) error {
		if err = r.batchOrm.UpdateFinalizeTxHashAndRollupStatusByBundleHash(r.ctx, bundle.Hash, txHash.String(), types.RollupFinalizing, dbTX); err != nil {
			log.Warn("UpdateFinalizeTxHashAndRollupStatusByBundleHash failed", "index", bundle.Index, "bundle hash", bundle.Hash, "tx hash", txHash.String(), "err", err)
			return err
		}

		if err = r.bundleOrm.UpdateFinalizeTxHashAndRollupStatus(r.ctx, bundle.Hash, txHash.String(), types.RollupFinalizing, dbTX); err != nil {
			log.Warn("UpdateFinalizeTxHashAndRollupStatus failed", "index", bundle.Index, "bundle hash", bundle.Hash, "tx hash", txHash.String(), "err", err)
			return err
		}

		return nil
	})

	if err != nil {
		log.Warn("failed to update rollup status of bundle and batches", "err", err)
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

		batchHashes := r.batchHashesFromContextID(cfm.ContextID)
		if err := r.db.Transaction(func(dbTX *gorm.DB) error {
			for _, batchHash := range batchHashes {
				if err := r.batchOrm.UpdateCommitTxHashAndRollupStatus(r.ctx, batchHash, cfm.TxHash.String(), status, dbTX); err != nil {
					return fmt.Errorf("UpdateCommitTxHashAndRollupStatus failed, batchHash: %s, txHash: %s, status: %s, err: %w", batchHash, cfm.TxHash.String(), status, err)
				}
			}

			return nil
		}); err != nil {
			log.Warn("failed to update confirmation status for batches", "confirmation", cfm, "err", err)
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
				if err := r.batchOrm.UpdateFinalizeTxHashAndRollupStatusByBundleHash(r.ctx, bundleHash, cfm.TxHash.String(), status, dbTX); err != nil {
					log.Warn("UpdateFinalizeTxHashAndRollupStatusByBundleHash failed", "confirmation", cfm, "err", err)
					return err
				}

				if err := r.bundleOrm.UpdateFinalizeTxHashAndRollupStatus(r.ctx, bundleHash, cfm.TxHash.String(), status, dbTX); err != nil {
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
	default:
		log.Warn("Unknown transaction type", "confirmation", cfm)
	}

	log.Info("Transaction confirmed in layer1", "confirmation", cfm)
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

func (r *Layer2Relayer) constructCommitBatchPayloadCodecV7(batchesToSubmit []*dbBatchWithChunks, firstBatch, lastBatch *orm.Batch) ([]byte, []*kzg4844.Blob, uint64, uint64, error) {
	var maxBlockHeight uint64
	var totalGasUsed uint64
	blobs := make([]*kzg4844.Blob, 0, len(batchesToSubmit))

	version := encoding.CodecVersion(batchesToSubmit[0].Batch.CodecVersion)
	// construct blobs
	for _, b := range batchesToSubmit {
		// double check that all batches have the same version
		batchVersion := encoding.CodecVersion(b.Batch.CodecVersion)
		if batchVersion != version {
			return nil, nil, 0, 0, fmt.Errorf("codec version mismatch, expected: %d, got: %d for batches %d and %d", version, batchVersion, batchesToSubmit[0].Batch.Index, b.Batch.Index)
		}

		var batchBlocks []*encoding.Block
		for _, c := range b.Chunks {
			blocks, err := r.l2BlockOrm.GetL2BlocksInRange(r.ctx, c.StartBlockNumber, c.EndBlockNumber)
			if err != nil {
				return nil, nil, 0, 0, fmt.Errorf("failed to get blocks in range for batch %d: %w", b.Batch.Index, err)
			}

			batchBlocks = append(batchBlocks, blocks...)

			if c.EndBlockNumber > maxBlockHeight {
				maxBlockHeight = c.EndBlockNumber
			}
			totalGasUsed += c.TotalL2TxGas
		}

		encodingBatch := &encoding.Batch{
			Index:                  b.Batch.Index,
			ParentBatchHash:        common.HexToHash(b.Batch.ParentBatchHash),
			PrevL1MessageQueueHash: common.HexToHash(b.Batch.PrevL1MessageQueueHash),
			PostL1MessageQueueHash: common.HexToHash(b.Batch.PostL1MessageQueueHash),
			Blocks:                 batchBlocks,
		}

		codec, err := encoding.CodecFromVersion(version)
		if err != nil {
			return nil, nil, 0, 0, fmt.Errorf("failed to get codec from version %d, err: %w", b.Batch.CodecVersion, err)
		}

		daBatch, err := codec.NewDABatch(encodingBatch)
		if err != nil {
			return nil, nil, 0, 0, fmt.Errorf("failed to create DA batch: %w", err)
		}

		blobs = append(blobs, daBatch.Blob())
	}

	calldata, err := r.l1RollupABI.Pack("commitBatches", version, common.HexToHash(firstBatch.ParentBatchHash), common.HexToHash(lastBatch.Hash))
	if err != nil {
		return nil, nil, 0, 0, fmt.Errorf("failed to pack commitBatches: %w", err)
	}
	return calldata, blobs, maxBlockHeight, totalGasUsed, nil
}

func (r *Layer2Relayer) constructFinalizeBundlePayloadCodecV7(dbBatch *orm.Batch, endChunk *orm.Chunk, aggProof *message.OpenVMBundleProof) ([]byte, error) {
	if aggProof != nil { // finalizeBundle with proof.
		calldata, packErr := r.l1RollupABI.Pack(
			"finalizeBundlePostEuclidV2",
			dbBatch.BatchHeader,
			new(big.Int).SetUint64(endChunk.TotalL1MessagesPoppedBefore+endChunk.TotalL1MessagesPoppedInChunk),
			common.HexToHash(dbBatch.StateRoot),
			common.HexToHash(dbBatch.WithdrawRoot),
			aggProof.Proof(),
		)
		if packErr != nil {
			return nil, fmt.Errorf("failed to pack finalizeBundlePostEuclidV2 with proof: %w", packErr)
		}
		return calldata, nil
	}

	fmt.Println("packing finalizeBundlePostEuclidV2NoProof", len(dbBatch.BatchHeader), dbBatch.CodecVersion, dbBatch.BatchHeader, new(big.Int).SetUint64(endChunk.TotalL1MessagesPoppedBefore+endChunk.TotalL1MessagesPoppedInChunk), common.HexToHash(dbBatch.StateRoot), common.HexToHash(dbBatch.WithdrawRoot))
	// finalizeBundle without proof.
	calldata, packErr := r.l1RollupABI.Pack(
		"finalizeBundlePostEuclidV2NoProof",
		dbBatch.BatchHeader,
		new(big.Int).SetUint64(endChunk.TotalL1MessagesPoppedBefore+endChunk.TotalL1MessagesPoppedInChunk),
		common.HexToHash(dbBatch.StateRoot),
		common.HexToHash(dbBatch.WithdrawRoot),
	)
	if packErr != nil {
		return nil, fmt.Errorf("failed to pack finalizeBundlePostEuclidV2NoProof: %w", packErr)
	}
	return calldata, nil
}

// StopSenders stops the senders of the rollup-relayer to prevent querying the removed pending_transaction table in unit tests.
// for unit test
func (r *Layer2Relayer) StopSenders() {
	if r.commitSender != nil {
		r.commitSender.Stop()
	}

	if r.finalizeSender != nil {
		r.finalizeSender.Stop()
	}
}

// fetchBlobFeeHistory returns the last WindowSec seconds of blob‐fee samples,
// by reading L1Block table’s BlobBaseFee column.
func (r *Layer2Relayer) fetchBlobFeeHistory(windowSec uint64) ([]*big.Int, error) {
	latest, err := r.l1BlockOrm.GetLatestL1BlockHeight(r.ctx)
	if err != nil {
		return nil, fmt.Errorf("GetLatestL1BlockHeight: %w", err)
	}
	// bootstrap on first call
	if r.lastFetchedBlock == 0 {
		// start window
		r.lastFetchedBlock = latest - windowSec/secondsPerBlock
	}
	from := r.lastFetchedBlock + 1
	//if new blocks
	if from <= latest {
		raw, err := r.l1BlockOrm.GetBlobFeesInRange(r.ctx, from, latest)
		if err != nil {
			return nil, fmt.Errorf("GetBlobFeesInRange: %w", err)
		}
		// append them
		for _, v := range raw {
			r.feeHistory = append(r.feeHistory, new(big.Int).SetUint64(v))
			r.lastFetchedBlock++
		}
	}

	maxLen := int(windowSec / secondsPerBlock)
	if len(r.feeHistory) > maxLen {
		r.feeHistory = r.feeHistory[len(r.feeHistory)-maxLen:]
	}

	return r.feeHistory, nil
}

// calculateTargetPrice applies pct_min/ewma + relaxation to get a BigInt target
func calculateTargetPrice(windowSec uint64, strategy StrategyParams, firstTime time.Time, history []*big.Int) *big.Int {
	var baseline float64 // baseline in Gwei (converting to float, small loss of precision)
	n := len(history)
	if n == 0 {
		return big.NewInt(0)
	}
	switch strategy.BaselineType {
	case PctMin:
		// make a copy, sort by big.Int.Cmp, then pick the percentile element
		sorted := make([]*big.Int, n)
		copy(sorted, history)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Cmp(sorted[j]) < 0
		})
		idx := int(strategy.BaselineParam * float64(n-1))
		if idx < 0 {
			idx = 0
		}
		baseline, _ = new(big.Float).
			Quo(new(big.Float).SetInt(sorted[idx]), big.NewFloat(1e9)).
			Float64()

	case EWMA:
		one := big.NewFloat(1)
		alpha := big.NewFloat(strategy.BaselineParam)
		oneMinusAlpha := new(big.Float).Sub(one, alpha)

		// start from first history point
		ewma := new(big.Float).
			Quo(new(big.Float).SetInt(history[0]), big.NewFloat(1e9))

		for i := 1; i < n; i++ {
			curr := new(big.Float).
				Quo(new(big.Float).SetInt(history[i]), big.NewFloat(1e9))
			term1 := new(big.Float).Mul(alpha, curr)
			term2 := new(big.Float).Mul(oneMinusAlpha, ewma)
			ewma = new(big.Float).Add(term1, term2)
		}
		baseline, _ = ewma.Float64()

	default:
		// fallback to last element
		baseline, _ = new(big.Float).
			Quo(new(big.Float).SetInt(history[n-1]), big.NewFloat(1e9)).
			Float64()
	} // now baseline holds our baseline in float64 Gwei

	// relaxation
	age := time.Since(firstTime).Seconds()
	frac := age / float64(windowSec)
	var adjusted float64
	switch strategy.RelaxType {
	case Exponential:
		adjusted = baseline * (1 + strategy.Gamma*math.Exp(strategy.Beta*(frac-1)))
	case Sigmoid:
		adjusted = baseline * (1 + strategy.Gamma/(1+math.Exp(-strategy.Beta*(frac-0.5))))
	default:
		adjusted = baseline
	}
	// back to wei
	f := new(big.Float).Mul(big.NewFloat(adjusted), big.NewFloat(1e9))
	out, _ := f.Int(nil)
	return out
}

// skipSubmitByFee returns (true, nil) when submission should be skipped right now
// because the blob‐fee is above target and the timeout window hasn’t yet elapsed.
// Otherwise returns (false, err)
func (r *Layer2Relayer) skipSubmitByFee(oldest time.Time, metrics *l2RelayerMetrics) (bool, error) {
	windowSec := uint64(r.cfg.BatchSubmission.TimeoutSec)

	hist, err := r.fetchBlobFeeHistory(windowSec)
	if err != nil || len(hist) == 0 {
		return false, fmt.Errorf(
			"blob-fee history unavailable or empty: %w (history_length=%d)",
			err, len(hist),
		)
	}

	// calculate target & get current (in wei)
	target := calculateTargetPrice(windowSec, r.batchStrategy, oldest, hist)
	current := hist[len(hist)-1]

	currentFloat, _ := current.Float64()
	targetFloat, _ := target.Float64()
	metrics.rollupL2RelayerCurrentBlobPrice.Set(currentFloat)
	metrics.rollupL2RelayerTargetBlobPrice.Set(targetFloat)

	// if current fee > target and still inside the timeout window, skip
	if current.Cmp(target) > 0 && time.Since(oldest) < time.Duration(windowSec)*time.Second {
		return true, fmt.Errorf(
			"blob-fee above target & window not yet passed; current=%s target=%s age=%s",
			current.String(), target.String(), time.Since(oldest),
		)
	}

	// otherwise proceed with submission
	return false, nil
}

func addrFromSignerConfig(config *config.SignerConfig) (common.Address, error) {
	switch config.SignerType {
	case sender.PrivateKeySignerType:
		privKey, err := crypto.ToECDSA(common.FromHex(config.PrivateKeySignerConfig.PrivateKey))
		if err != nil {
			return common.Address{}, fmt.Errorf("parse sender private key failed: %w", err)
		}
		return crypto.PubkeyToAddress(privKey.PublicKey), nil
	case sender.RemoteSignerType:
		if config.RemoteSignerConfig.SignerAddress == "" {
			return common.Address{}, fmt.Errorf("signer address is empty")
		}
		return common.HexToAddress(config.RemoteSignerConfig.SignerAddress), nil
	default:
		return common.Address{}, fmt.Errorf("failed to determine signer address, unknown signer type: %v", config.SignerType)
	}
}
