package relayer

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/common/types"
	"scroll-tech/common/utils"

	bridgeAbi "scroll-tech/rollup/abi"
	"scroll-tech/rollup/internal/config"
	"scroll-tech/rollup/internal/controller/sender"
	"scroll-tech/rollup/internal/orm"
)

// Layer2Relayer is responsible for
//  1. Committing and finalizing L2 blocks on L1
//  2. Relaying messages from L2 to L1
//
// Actions are triggered by new head from layer 1 geth node.
// @todo It's better to be triggered by watcher.
type Layer2Relayer struct {
	ctx context.Context

	l2Client *ethclient.Client

	db         *gorm.DB
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

	// A list of processing batches commitment.
	// key(string): confirmation ID, value(string): batch hash.
	processingCommitment sync.Map

	// A list of processing batch finalization.
	// key(string): confirmation ID, value(string): batch hash.
	processingFinalization sync.Map

	metrics *l2RelayerMetrics
}

// NewLayer2Relayer will return a new instance of Layer2RelayerClient
func NewLayer2Relayer(ctx context.Context, l2Client *ethclient.Client, db *gorm.DB, cfg *config.RelayerConfig, initGenesis bool, reg prometheus.Registerer) (*Layer2Relayer, error) {
	commitSender, err := sender.NewSender(ctx, cfg.SenderConfig, cfg.CommitSenderPrivateKey, "l2_relayer", "commit_sender", reg)
	if err != nil {
		addr := crypto.PubkeyToAddress(cfg.CommitSenderPrivateKey.PublicKey)
		return nil, fmt.Errorf("new commit sender failed for address %s, err: %w", addr.Hex(), err)
	}
	finalizeSender, err := sender.NewSender(ctx, cfg.SenderConfig, cfg.FinalizeSenderPrivateKey, "l2_relayer", "finalize_sender", reg)
	if err != nil {
		addr := crypto.PubkeyToAddress(cfg.FinalizeSenderPrivateKey.PublicKey)
		return nil, fmt.Errorf("new finalize sender failed for address %s, err: %w", addr.Hex(), err)
	}

	gasOracleSender, err := sender.NewSender(ctx, cfg.SenderConfig, cfg.GasOracleSenderPrivateKey, "l2_relayer", "gas_oracle_sender", reg)
	if err != nil {
		addr := crypto.PubkeyToAddress(cfg.GasOracleSenderPrivateKey.PublicKey)
		return nil, fmt.Errorf("new gas oracle sender failed for address %s, err: %w", addr.Hex(), err)
	}

	// Ensure test features aren't enabled on the mainnet.
	if commitSender.GetChainID() == big.NewInt(1) && cfg.EnableTestEnvBypassFeatures {
		return nil, fmt.Errorf("cannot enable test env features in mainnet")
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

		cfg:                    cfg,
		processingCommitment:   sync.Map{},
		processingFinalization: sync.Map{},
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

	go layer2Relayer.handleConfirmLoop(ctx)
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

	chunk := &types.Chunk{
		Blocks: []*types.WrappedBlock{{
			Header:         genesis,
			Transactions:   nil,
			WithdrawRoot:   common.Hash{},
			RowConsumption: &gethTypes.RowConsumption{},
		}},
	}

	err = r.db.Transaction(func(dbTX *gorm.DB) error {
		var dbChunk *orm.Chunk
		dbChunk, err = r.chunkOrm.InsertChunk(r.ctx, chunk, dbTX)
		if err != nil {
			return fmt.Errorf("failed to insert chunk: %v", err)
		}

		if err = r.chunkOrm.UpdateProvingStatus(r.ctx, dbChunk.Hash, types.ProvingTaskVerified, dbTX); err != nil {
			return fmt.Errorf("failed to update genesis chunk proving status: %v", err)
		}

		batchMeta := &types.BatchMeta{
			StartChunkIndex: 0,
			StartChunkHash:  dbChunk.Hash,
			EndChunkIndex:   0,
			EndChunkHash:    dbChunk.Hash,
		}
		var batch *orm.Batch
		batch, err = r.batchOrm.InsertBatch(r.ctx, []*types.Chunk{chunk}, batchMeta, dbTX)
		if err != nil {
			return fmt.Errorf("failed to insert batch: %v", err)
		}

		if err = r.chunkOrm.UpdateBatchHashInRange(r.ctx, 0, 0, batch.Hash, dbTX); err != nil {
			return fmt.Errorf("failed to update batch hash for chunks: %v", err)
		}

		if err = r.batchOrm.UpdateProvingStatus(r.ctx, batch.Hash, types.ProvingTaskVerified, dbTX); err != nil {
			return fmt.Errorf("failed to update genesis batch proving status: %v", err)
		}

		if err = r.batchOrm.UpdateRollupStatus(r.ctx, batch.Hash, types.RollupFinalized, dbTX); err != nil {
			return fmt.Errorf("failed to update genesis batch rollup status: %v", err)
		}

		// commit genesis batch on L1
		// note: we do this inside the DB transaction so that we can revert all DB changes if this step fails
		return r.commitGenesisBatch(batch.Hash, batch.BatchHeader, common.HexToHash(batch.StateRoot))
	})

	if err != nil {
		return fmt.Errorf("update genesis transaction failed: %v", err)
	}

	log.Info("successfully imported genesis chunk and batch")

	return nil
}

func (r *Layer2Relayer) commitGenesisBatch(batchHash string, batchHeader []byte, stateRoot common.Hash) error {
	// encode "importGenesisBatch" transaction calldata
	calldata, err := r.l1RollupABI.Pack("importGenesisBatch", batchHeader, stateRoot)
	if err != nil {
		return fmt.Errorf("failed to pack importGenesisBatch with batch header: %v and state root: %v. error: %v", common.Bytes2Hex(batchHeader), stateRoot, err)
	}

	// submit genesis batch to L1 rollup contract
	txHash, err := r.commitSender.SendTransaction(batchHash, &r.cfg.RollupContractAddress, big.NewInt(0), calldata, 0)
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
			if confirmation.ID != batchHash {
				return fmt.Errorf("unexpected import genesis confirmation id, expected: %v, got: %v", batchHash, confirmation.ID)
			}
			if !confirmation.IsSuccessful {
				return fmt.Errorf("import genesis batch tx failed")
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
	if batch == nil || err != nil {
		log.Error("Failed to GetLatestBatch", "batch", batch, "err", err)
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

		// last is undefine or (suggestGasPriceUint64 >= minGasPrice && exceed diff)
		if r.lastGasPrice == 0 || (suggestGasPriceUint64 >= r.minGasPrice && (suggestGasPriceUint64 >= r.lastGasPrice+expectedDelta || suggestGasPriceUint64 <= r.lastGasPrice-expectedDelta)) {
			data, err := r.l2GasOracleABI.Pack("setL2BaseFee", suggestGasPrice)
			if err != nil {
				log.Error("Failed to pack setL2BaseFee", "batch.Hash", batch.Hash, "GasPrice", suggestGasPrice.Uint64(), "err", err)
				return
			}

			hash, err := r.gasOracleSender.SendTransaction(batch.Hash, &r.cfg.GasPriceOracleContractAddress, big.NewInt(0), data, 0)
			if err != nil {
				if !errors.Is(err, sender.ErrNoAvailableAccount) && !errors.Is(err, sender.ErrFullPending) {
					log.Error("Failed to send setL2BaseFee tx to layer2 ", "batch.Hash", batch.Hash, "err", err)
				}
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
	batches, err := r.batchOrm.GetFailedAndPendingBatches(r.ctx, 5)
	if err != nil {
		log.Error("Failed to fetch pending L2 batches", "err", err)
		return
	}
	for _, batch := range batches {
		r.metrics.rollupL2RelayerProcessPendingBatchTotal.Inc()
		// get current header and parent header.
		currentBatchHeader, err := types.DecodeBatchHeader(batch.BatchHeader)
		if err != nil {
			log.Error("Failed to decode batch header", "index", batch.Index, "error", err)
			return
		}
		parentBatch := &orm.Batch{}
		if batch.Index > 0 {
			parentBatch, err = r.batchOrm.GetBatchByIndex(r.ctx, batch.Index-1)
			if err != nil {
				log.Error("Failed to get parent batch header", "index", batch.Index-1, "error", err)
				return
			}

			if types.RollupStatus(parentBatch.RollupStatus) == types.RollupCommitFailed {
				log.Error("Previous batch commit failed, halting further committing",
					"index", parentBatch.Index, "tx hash", parentBatch.CommitTxHash)
				return
			}
		}

		// get the chunks for the batch
		startChunkIndex := batch.StartChunkIndex
		endChunkIndex := batch.EndChunkIndex
		dbChunks, err := r.chunkOrm.GetChunksInRange(r.ctx, startChunkIndex, endChunkIndex)
		if err != nil {
			log.Error("Failed to fetch chunks",
				"start index", startChunkIndex,
				"end index", endChunkIndex, "error", err)
			return
		}

		encodedChunks := make([][]byte, len(dbChunks))
		for i, c := range dbChunks {
			var wrappedBlocks []*types.WrappedBlock
			wrappedBlocks, err = r.l2BlockOrm.GetL2BlocksInRange(r.ctx, c.StartBlockNumber, c.EndBlockNumber)
			if err != nil {
				log.Error("Failed to fetch wrapped blocks",
					"start number", c.StartBlockNumber,
					"end number", c.EndBlockNumber, "error", err)
				return
			}
			chunk := &types.Chunk{
				Blocks: wrappedBlocks,
			}
			var chunkBytes []byte
			chunkBytes, err = chunk.Encode(c.TotalL1MessagesPoppedBefore)
			if err != nil {
				log.Error("Failed to encode chunk", "error", err)
				return
			}
			encodedChunks[i] = chunkBytes
		}

		calldata, err := r.l1RollupABI.Pack("commitBatch", currentBatchHeader.Version(), parentBatch.BatchHeader, encodedChunks, currentBatchHeader.SkippedL1MessageBitmap())
		if err != nil {
			log.Error("Failed to pack commitBatch", "index", batch.Index, "error", err)
			return
		}

		// send transaction
		txID := batch.Hash + "-commit"
		fallbackGasLimit := uint64(float64(batch.TotalL1CommitGas) * r.cfg.L1CommitGasLimitMultiplier)
		if types.RollupStatus(batch.RollupStatus) == types.RollupCommitFailed {
			// use eth_estimateGas if this batch has been committed failed.
			fallbackGasLimit = 0
			log.Warn("Batch commit previously failed, using eth_estimateGas for the re-submission", "hash", batch.Hash)
		}
		txHash, err := r.commitSender.SendTransaction(txID, &r.cfg.RollupContractAddress, big.NewInt(0), calldata, fallbackGasLimit)
		if err != nil {
			log.Error(
				"Failed to send commitBatch tx to layer1",
				"index", batch.Index,
				"hash", batch.Hash,
				"RollupContractAddress", r.cfg.RollupContractAddress,
				"err", err,
			)
			log.Debug(
				"Failed to send commitBatch tx to layer1",
				"index", batch.Index,
				"hash", batch.Hash,
				"RollupContractAddress", r.cfg.RollupContractAddress,
				"calldata", common.Bytes2Hex(calldata),
				"err", err,
			)
			return
		}

		err = r.batchOrm.UpdateCommitTxHashAndRollupStatus(r.ctx, batch.Hash, txHash.String(), types.RollupCommitting)
		if err != nil {
			log.Error("UpdateCommitTxHashAndRollupStatus failed", "hash", batch.Hash, "index", batch.Index, "err", err)
			return
		}
		r.metrics.rollupL2RelayerProcessPendingBatchSuccessTotal.Inc()
		r.processingCommitment.Store(txID, batch.Hash)
		log.Info("Sent the commitBatch tx to layer1", "batch index", batch.Index, "batch hash", batch.Hash, "tx hash", txHash.Hex())
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
		log.Info("Start to roll up zk proof", "hash", batch.Hash)
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
			"ProverAssignedAt", batch.ProverAssignedAt,
			"ProvedAt", batch.ProvedAt,
			"ProofTimeSec", batch.ProofTimeSec,
		)

	default:
		log.Error("encounter unreachable case in ProcessCommittedBatches", "proving status", status)
	}
}

func (r *Layer2Relayer) finalizeBatch(batch *orm.Batch, withProof bool) error {
	// Check batch status before send `finalizeBatch` tx.
	if r.cfg.ChainMonitor.Enabled {
		var batchStatus bool
		batchStatus, err := r.getBatchStatusByIndex(batch)
		if err != nil {
			r.metrics.rollupL2ChainMonitorLatestFailedCall.Inc()
			log.Warn("failed to get batch status, please check chain_monitor api server", "batch_index", batch.Index, "err", err)
			return err
		}
		if !batchStatus {
			r.metrics.rollupL2ChainMonitorLatestFailedBatchStatus.Inc()
			log.Error("the batch status is not right, stop finalize batch and check the reason", "batch_index", batch.Index)
			return err
		}
	}

	var parentBatchStateRoot string
	if batch.Index > 0 {
		var parentBatch *orm.Batch
		parentBatch, err := r.batchOrm.GetBatchByIndex(r.ctx, batch.Index-1)
		// handle unexpected db error
		if err != nil {
			log.Error("Failed to get batch", "index", batch.Index-1, "err", err)
			return err
		}
		parentBatchStateRoot = parentBatch.StateRoot
	}

	var txCalldata []byte
	if withProof {
		aggProof, err := r.batchOrm.GetVerifiedProofByHash(r.ctx, batch.Hash)
		if err != nil {
			log.Error("get verified proof by hash failed", "hash", batch.Hash, "err", err)
			return err
		}

		if err = aggProof.SanityCheck(); err != nil {
			log.Error("agg_proof sanity check fails", "hash", batch.Hash, "error", err)
			return err
		}

		txCalldata, err = r.l1RollupABI.Pack(
			"finalizeBatchWithProof",
			batch.BatchHeader,
			common.HexToHash(parentBatchStateRoot),
			common.HexToHash(batch.StateRoot),
			common.HexToHash(batch.WithdrawRoot),
			aggProof.Proof,
		)
		if err != nil {
			log.Error("Pack finalizeBatchWithProof failed", "err", err)
			return err
		}
	} else {
		var err error
		txCalldata, err = r.l1RollupABI.Pack(
			"finalizeBatch",
			batch.BatchHeader,
			common.HexToHash(parentBatchStateRoot),
			common.HexToHash(batch.StateRoot),
			common.HexToHash(batch.WithdrawRoot),
		)
		if err != nil {
			log.Error("Pack finalizeBatch failed", "err", err)
			return err
		}
	}

	txID := batch.Hash + "-finalize"
	// add suffix `-finalize` to avoid duplication with commit tx in unit tests
	txHash, err := r.finalizeSender.SendTransaction(txID, &r.cfg.RollupContractAddress, big.NewInt(0), txCalldata, 0)
	finalizeTxHash := &txHash
	if err != nil {
		if !errors.Is(err, sender.ErrNoAvailableAccount) && !errors.Is(err, sender.ErrFullPending) {
			// This can happen normally if we try to finalize 2 or more
			// batches around the same time. The 2nd tx might fail since
			// the client does not see the 1st tx's updates at this point.
			// TODO: add more fine-grained error handling
			log.Error(
				"finalizeBatch in layer1 failed",
				"with proof", withProof,
				"index", batch.Index,
				"hash", batch.Hash,
				"RollupContractAddress", r.cfg.RollupContractAddress,
				"err", err,
			)
			log.Debug(
				"finalizeBatch in layer1 failed",
				"with proof", withProof,
				"index", batch.Index,
				"hash", batch.Hash,
				"RollupContractAddress", r.cfg.RollupContractAddress,
				"calldata", common.Bytes2Hex(txCalldata),
				"err", err,
			)
		}
		return err
	}
	log.Info("finalizeBatch in layer1", "with proof", withProof, "index", batch.Index, "batch hash", batch.Hash, "tx hash", batch.Hash)

	// record and sync with db, @todo handle db error
	if err := r.batchOrm.UpdateFinalizeTxHashAndRollupStatus(r.ctx, batch.Hash, finalizeTxHash.String(), types.RollupFinalizing); err != nil {
		log.Error("UpdateFinalizeTxHashAndRollupStatus failed", "index", batch.Index, "batch hash", batch.Hash, "tx hash", finalizeTxHash.String(), "err", err)
		return err
	}
	r.processingFinalization.Store(txID, batch.Hash)
	r.metrics.rollupL2RelayerProcessCommittedBatchesFinalizedSuccessTotal.Inc()
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

func (r *Layer2Relayer) handleConfirmation(confirmation *sender.Confirmation) {
	transactionType := "Unknown"
	// check whether it is CommitBatches transaction
	if batchHash, ok := r.processingCommitment.Load(confirmation.ID); ok {
		transactionType = "BatchesCommitment"
		var status types.RollupStatus
		if confirmation.IsSuccessful {
			status = types.RollupCommitted
		} else {
			status = types.RollupCommitFailed
			r.metrics.rollupL2BatchesCommittedConfirmedFailedTotal.Inc()
			log.Warn("commitBatch transaction confirmed but failed in layer1", "confirmation", confirmation)
		}
		// @todo handle db error
		err := r.batchOrm.UpdateCommitTxHashAndRollupStatus(r.ctx, batchHash.(string), confirmation.TxHash.String(), status)
		if err != nil {
			log.Warn("UpdateCommitTxHashAndRollupStatus failed",
				"batch hash", batchHash.(string),
				"tx hash", confirmation.TxHash.String(), "err", err)
		}
		r.metrics.rollupL2BatchesCommittedConfirmedTotal.Inc()
		r.processingCommitment.Delete(confirmation.ID)
	}

	// check whether it is proof finalization transaction
	if batchHash, ok := r.processingFinalization.Load(confirmation.ID); ok {
		transactionType = "ProofFinalization"
		var status types.RollupStatus
		if confirmation.IsSuccessful {
			status = types.RollupFinalized
		} else {
			status = types.RollupFinalizeFailed
			r.metrics.rollupL2BatchesFinalizedConfirmedFailedTotal.Inc()
			log.Warn("finalizeBatchWithProof transaction confirmed but failed in layer1", "confirmation", confirmation)
		}

		// @todo handle db error
		err := r.batchOrm.UpdateFinalizeTxHashAndRollupStatus(r.ctx, batchHash.(string), confirmation.TxHash.String(), status)
		if err != nil {
			log.Warn("UpdateFinalizeTxHashAndRollupStatus failed",
				"batch hash", batchHash.(string),
				"tx hash", confirmation.TxHash.String(), "err", err)
		}
		r.metrics.rollupL2BatchesFinalizedConfirmedTotal.Inc()
		r.processingFinalization.Delete(confirmation.ID)
	}
	log.Info("transaction confirmed in layer1", "type", transactionType, "confirmation", confirmation)
}

func (r *Layer2Relayer) handleConfirmLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case confirmation := <-r.commitSender.ConfirmChan():
			r.handleConfirmation(confirmation)
		case confirmation := <-r.finalizeSender.ConfirmChan():
			r.handleConfirmation(confirmation)
		case cfm := <-r.gasOracleSender.ConfirmChan():
			r.metrics.rollupL2BatchesGasOraclerConfirmedTotal.Inc()
			if !cfm.IsSuccessful {
				// @discuss: maybe make it pending again?
				err := r.batchOrm.UpdateL2GasOracleStatusAndOracleTxHash(r.ctx, cfm.ID, types.GasOracleFailed, cfm.TxHash.String())
				if err != nil {
					log.Warn("UpdateL2GasOracleStatusAndOracleTxHash failed", "err", err)
				}
				log.Warn("transaction confirmed but failed in layer1", "confirmation", cfm)
			} else {
				// @todo handle db error
				err := r.batchOrm.UpdateL2GasOracleStatusAndOracleTxHash(r.ctx, cfm.ID, types.GasOracleImported, cfm.TxHash.String())
				if err != nil {
					log.Warn("UpdateL2GasOracleStatusAndOracleTxHash failed", "err", err)
				}
				log.Info("transaction confirmed in layer1", "confirmation", cfm)
			}
		}
	}
}
