package relayer

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"sort"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/params"
	"gorm.io/gorm"

	"scroll-tech/common/types"
	"scroll-tech/common/types/encoding"
	"scroll-tech/common/types/encoding/codecv0"
	"scroll-tech/common/types/encoding/codecv1"
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

	commitSender     *sender.Sender
	finalizeSender   *sender.Sender
	blobCommitSender *sender.Sender // TODO: consider how to add blob sender support.
	l1RollupABI      *abi.ABI

	gasOracleSender *sender.Sender
	l2GasOracleABI  *abi.ABI

	lastGasPrice uint64
	minGasPrice  uint64
	gasPriceDiff uint64

	// Used to get batch status from chain_monitor api.
	chainMonitorClient *resty.Client

	metrics *l2RelayerMetrics

	banachForkHeight uint64
}

// NewLayer2Relayer will return a new instance of Layer2RelayerClient
func NewLayer2Relayer(ctx context.Context, l2Client *ethclient.Client, db *gorm.DB, cfg *config.RelayerConfig, chainCfg *params.ChainConfig, initGenesis bool, serviceType ServiceType, reg prometheus.Registerer) (*Layer2Relayer, error) {
	var gasOracleSender, commitSender, finalizeSender, blobCommitSender *sender.Sender
	var err error

	switch serviceType {
	case ServiceTypeL2GasOracle:
		gasOracleSender, err = sender.NewSender(ctx, cfg.SenderConfig, cfg.GasOracleSenderPrivateKey, "l2_relayer", "gas_oracle_sender", types.SenderTypeL2GasOracle, db, reg)
		if err != nil {
			addr := crypto.PubkeyToAddress(cfg.GasOracleSenderPrivateKey.PublicKey)
			return nil, fmt.Errorf("new gas oracle sender failed for address %s, err: %w", addr.Hex(), err)
		}

		// Ensure test features aren't enabled on the ethereum mainnet.
		if gasOracleSender.GetChainID().Cmp(big.NewInt(1)) == 0 && cfg.EnableTestEnvBypassFeatures {
			return nil, fmt.Errorf("cannot enable test env features in mainnet")
		}

	case ServiceTypeL2RollupRelayer:
		commitSender, err = sender.NewSender(ctx, cfg.SenderConfig, cfg.CommitSenderPrivateKey, "l2_relayer", "commit_sender", types.SenderTypeCommitBatch, db, reg)
		if err != nil {
			addr := crypto.PubkeyToAddress(cfg.CommitSenderPrivateKey.PublicKey)
			return nil, fmt.Errorf("new commit sender failed for address %s, err: %w", addr.Hex(), err)
		}

		finalizeSender, err = sender.NewSender(ctx, cfg.SenderConfig, cfg.FinalizeSenderPrivateKey, "l2_relayer", "finalize_sender", types.SenderTypeFinalizeBatch, db, reg)
		if err != nil {
			addr := crypto.PubkeyToAddress(cfg.FinalizeSenderPrivateKey.PublicKey)
			return nil, fmt.Errorf("new finalize sender failed for address %s, err: %w", addr.Hex(), err)
		}

		blobCommitSender, err = sender.NewSender(ctx, cfg.SenderConfig, cfg.BlobCommitSenderPrivateKey, "l2_relayer", "blob_commit_sender", types.SenderTypeCommitBlobBatch, db, reg)
		if err != nil {
			addr := crypto.PubkeyToAddress(cfg.CommitSenderPrivateKey.PublicKey)
			return nil, fmt.Errorf("new blob commit sender failed for address %s, err: %w", addr.Hex(), err)
		}

		// Ensure test features aren't enabled on the ethereum mainnet.
		if commitSender.GetChainID().Cmp(big.NewInt(1)) == 0 && cfg.EnableTestEnvBypassFeatures {
			return nil, fmt.Errorf("cannot enable test env features in mainnet")
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

		batchOrm:   orm.NewBatch(db),
		l2BlockOrm: orm.NewL2Block(db),
		chunkOrm:   orm.NewChunk(db),

		l2Client: l2Client,

		commitSender:     commitSender,
		finalizeSender:   finalizeSender,
		blobCommitSender: blobCommitSender,
		l1RollupABI:      bridgeAbi.ScrollChainABI,

		gasOracleSender: gasOracleSender,
		l2GasOracleABI:  bridgeAbi.L2GasPriceOracleABI,

		minGasPrice:  minGasPrice,
		gasPriceDiff: gasPriceDiff,

		cfg: cfg,
	}

	// If BanachBlock is not set in chain's genesis config, banachForkHeight is inf,
	// which means chunk proposer uses the codecv0 version by default.
	// TODO: Must change it to real fork name.
	if chainCfg.BanachBlock != nil {
		layer2Relayer.banachForkHeight = chainCfg.BanachBlock.Uint64()
	} else {
		layer2Relayer.banachForkHeight = math.MaxUint64
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
		dbChunk, err = r.chunkOrm.InsertChunk(r.ctx, chunk, dbTX)
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
		dbBatch, err = r.batchOrm.InsertBatch(r.ctx, batch, dbTX)
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
	calldata, err := r.l1RollupABI.Pack("importGenesisBatch", batchHeader, stateRoot)
	if err != nil {
		return fmt.Errorf("failed to pack importGenesisBatch with batch header: %v and state root: %v. error: %v", common.Bytes2Hex(batchHeader), stateRoot, err)
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
		if r.lastGasPrice > 0 && expectedDelta == 0 {
			expectedDelta = 1
		}

		// last is undefine or (suggestGasPriceUint64 >= minGasPrice && exceed diff)
		if r.lastGasPrice == 0 || (suggestGasPriceUint64 >= r.minGasPrice && (suggestGasPriceUint64 >= r.lastGasPrice+expectedDelta || suggestGasPriceUint64 <= r.lastGasPrice-expectedDelta)) {
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
	batches, err := r.batchOrm.GetFailedAndPendingBatches(r.ctx, 5)
	if err != nil {
		log.Error("Failed to fetch pending L2 batches", "err", err)
		return
	}
	for _, batch := range batches {
		r.metrics.rollupL2RelayerProcessPendingBatchTotal.Inc()

		txHash, err := r.sendCommitBatchTransaction(batch)
		if err != nil {
			log.Error("Failed to send commitBatch transaction", "index", batch.Index, "error", err)
			return
		}

		err = r.batchOrm.UpdateCommitTxHashAndRollupStatus(r.ctx, batch.Hash, txHash.Hex(), types.RollupCommitting)
		if err != nil {
			log.Error("UpdateCommitTxHashAndRollupStatus failed", "hash", batch.Hash, "index", batch.Index, "err", err)
			return
		}
		r.metrics.rollupL2RelayerProcessPendingBatchSuccessTotal.Inc()
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

	calldata, err := r.constructFinalizeBatchPayload(batch, withProof)
	if err != nil {
		log.Error("failed to construct finalizeBatch payload", "index", batch.Index, "error", err)
		return err
	}

	txHash, err := r.finalizeSender.SendTransaction(batch.Hash, &r.cfg.RollupContractAddress, calldata, nil, 0)
	if err != nil {
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
			"calldata", common.Bytes2Hex(calldata),
			"err", err,
		)
		return err
	}

	log.Info("finalizeBatch in layer1", "with proof", withProof, "index", batch.Index, "batch hash", batch.Hash, "tx hash", txHash)

	// record and sync with db, @todo handle db error
	if err := r.batchOrm.UpdateFinalizeTxHashAndRollupStatus(r.ctx, batch.Hash, txHash.String(), types.RollupFinalizing); err != nil {
		log.Error("UpdateFinalizeTxHashAndRollupStatus failed", "index", batch.Index, "batch hash", batch.Hash, "tx hash", txHash.String(), "err", err)
		return err
	}
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

func (r *Layer2Relayer) sendCommitBatchTransaction(dbBatch *orm.Batch) (common.Hash, error) {
	dbChunks, err := r.chunkOrm.GetChunksInRange(r.ctx, dbBatch.StartChunkIndex, dbBatch.EndChunkIndex)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to fetch chunks: %w", err)
	}

	chunks := make([]*encoding.Chunk, len(dbChunks))
	for i, c := range dbChunks {
		blocks, getErr := r.l2BlockOrm.GetL2BlocksInRange(r.ctx, c.StartBlockNumber, c.EndBlockNumber)
		if getErr != nil {
			return common.Hash{}, fmt.Errorf("failed to fetch blocks: %w", getErr)
		}
		chunks[i] = &encoding.Chunk{Blocks: blocks}
	}

	var parentBatchHeader []byte
	var parentBatchHash common.Hash
	if dbBatch.Index > 0 {
		var parentDBBatch *orm.Batch
		parentDBBatch, getErr := r.batchOrm.GetBatchByIndex(r.ctx, dbBatch.Index-1)
		if getErr != nil {
			return common.Hash{}, fmt.Errorf("failed to get parent batch header: %w", getErr)
		}
		if parentDBBatch != nil { // TODO: remove this check, return error when nil.
			parentBatchHeader = parentDBBatch.BatchHeader
			parentBatchHash = common.HexToHash(parentDBBatch.Hash)
		}
	}

	startBlockNumber := dbChunks[0].StartBlockNumber
	if startBlockNumber >= r.banachForkHeight { // codecv1
		batch := &encoding.Batch{
			Index:                      dbBatch.Index,
			TotalL1MessagePoppedBefore: dbChunks[len(dbChunks)-1].TotalL1MessagesPoppedBefore + dbChunks[len(dbChunks)-1].TotalL1MessagesPoppedInChunk,
			ParentBatchHash:            parentBatchHash,
			Chunks:                     chunks,
		}

		var daBatch *codecv1.DABatch
		daBatch, err = codecv1.NewDABatch(batch)
		if err != nil {
			return common.Hash{}, fmt.Errorf("failed to initialize new DA batch: %w", err)
		}

		encodedChunks := make([][]byte, len(dbChunks))
		for i, c := range dbChunks {
			daChunk, createErr := codecv1.NewDAChunk(chunks[i], c.TotalL1MessagesPoppedBefore)
			if createErr != nil {
				return common.Hash{}, fmt.Errorf("failed to initialize new DA chunk: %w", createErr)
			}
			encodedChunks[i] = daChunk.Encode()
		}

		calldata, packErr := r.l1RollupABI.Pack("commitBatch", daBatch.Version, parentBatchHeader, encodedChunks, daBatch.SkippedL1MessageBitmap)
		if packErr != nil {
			return common.Hash{}, fmt.Errorf("failed to pack commitBatch: %w", packErr)
		}

		txHash, sendErr := r.blobCommitSender.SendTransaction(dbBatch.Hash, &r.cfg.RollupContractAddress, calldata, daBatch.Blob(), 0)
		if sendErr != nil {
			log.Error(
				"Failed to send commitBatch tx to layer1",
				"index", dbBatch.Index,
				"hash", dbBatch.Hash,
				"RollupContractAddress", r.cfg.RollupContractAddress,
				"err", sendErr,
			)
			log.Debug(
				"Failed to send commitBatch tx to layer1",
				"index", dbBatch.Index,
				"hash", dbBatch.Hash,
				"RollupContractAddress", r.cfg.RollupContractAddress,
				"calldata", common.Bytes2Hex(calldata),
				"err", sendErr,
			)
			return common.Hash{}, sendErr
		}
		return txHash, nil
	}

	// codecv0
	daBatch, err := codecv0.NewDABatchFromBytes(dbBatch.BatchHeader)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to initialize new DA batch from bytes: %w", err)
	}

	encodedChunks := make([][]byte, len(dbChunks))
	for i, c := range dbChunks {
		daChunk, createErr := codecv0.NewDAChunk(chunks[i], c.TotalL1MessagesPoppedBefore)
		if createErr != nil {
			return common.Hash{}, fmt.Errorf("failed to initialize new DA chunk: %w", createErr)
		}
		daChunkBytes, encodeErr := daChunk.Encode()
		if encodeErr != nil {
			return common.Hash{}, fmt.Errorf("failed to encode DA chunk: %w", encodeErr)
		}
		encodedChunks[i] = daChunkBytes
	}

	calldata, packErr := r.l1RollupABI.Pack("commitBatch", daBatch.Version, parentBatchHeader, encodedChunks, daBatch.SkippedL1MessageBitmap)
	if packErr != nil {
		return common.Hash{}, fmt.Errorf("failed to pack commitBatch: %w", packErr)
	}

	fallbackGasLimit := uint64(float64(dbBatch.TotalL1CommitGas) * r.cfg.L1CommitGasLimitMultiplier)
	if types.RollupStatus(dbBatch.RollupStatus) == types.RollupCommitFailed {
		// use eth_estimateGas if this batch has been committed and failed at least once.
		fallbackGasLimit = 0
		log.Warn("Batch commit previously failed, using eth_estimateGas for the re-submission", "hash", dbBatch.Hash)
	}
	txHash, err := r.commitSender.SendTransaction(dbBatch.Hash, &r.cfg.RollupContractAddress, calldata, nil, fallbackGasLimit)
	if err != nil {
		log.Error(
			"Failed to send commitBatch tx to layer1",
			"index", dbBatch.Index,
			"hash", dbBatch.Hash,
			"RollupContractAddress", r.cfg.RollupContractAddress,
			"err", err,
		)
		log.Debug(
			"Failed to send commitBatch tx to layer1",
			"index", dbBatch.Index,
			"hash", dbBatch.Hash,
			"RollupContractAddress", r.cfg.RollupContractAddress,
			"calldata", common.Bytes2Hex(calldata),
			"err", err,
		)
		return common.Hash{}, err
	}
	return txHash, nil
}

func (r *Layer2Relayer) constructFinalizeBatchPayload(dbBatch *orm.Batch, withProof bool) ([]byte, error) {
	var parentBatchStateRoot string
	var parentBatchHash common.Hash
	if dbBatch.Index > 0 { // TODO: remove this check, return error when nil.
		var parentDBBatch *orm.Batch
		parentDBBatch, err := r.batchOrm.GetBatchByIndex(r.ctx, dbBatch.Index-1)
		if err != nil {
			return nil, fmt.Errorf("failed to get batch, index: %d, err: %w", dbBatch.Index-1, err)
		}
		parentBatchStateRoot = parentDBBatch.StateRoot
		parentBatchHash = common.HexToHash(parentDBBatch.Hash)
	}

	dbChunks, err := r.chunkOrm.GetChunksInRange(r.ctx, dbBatch.StartChunkIndex, dbBatch.EndChunkIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch chunks: %w", err)
	}

	startBlockNumber := dbChunks[0].StartBlockNumber
	if startBlockNumber >= r.banachForkHeight { // codecv1
		if withProof {
			aggProof, err := r.batchOrm.GetVerifiedProofByHash(r.ctx, dbBatch.Hash)
			if err != nil {
				return nil, fmt.Errorf("failed to get verified proof by hash, index: %d, err: %w", dbBatch.Index, err)
			}

			if err = aggProof.SanityCheck(); err != nil {
				return nil, fmt.Errorf("failed to check agg_proof sanity, index: %d, err: %w", dbBatch.Index, err)
			}

			chunks := make([]*encoding.Chunk, len(dbChunks))
			for i, c := range dbChunks {
				blocks, getErr := r.l2BlockOrm.GetL2BlocksInRange(r.ctx, c.StartBlockNumber, c.EndBlockNumber)
				if getErr != nil {
					return nil, fmt.Errorf("failed to fetch blocks: %w", getErr)
				}
				chunks[i] = &encoding.Chunk{Blocks: blocks}
			}

			batch := &encoding.Batch{
				Index:                      dbBatch.Index,
				TotalL1MessagePoppedBefore: dbChunks[len(dbChunks)-1].TotalL1MessagesPoppedBefore + dbChunks[len(dbChunks)-1].TotalL1MessagesPoppedInChunk,
				ParentBatchHash:            parentBatchHash,
				Chunks:                     chunks,
			}

			daBatch, err := codecv1.NewDABatch(batch)
			if err != nil {
				return nil, fmt.Errorf("failed to initialize new DA batch: %w", err)
			}

			blobDataProof, err := daBatch.BlobDataProof()
			if err != nil {
				return nil, fmt.Errorf("failed to get blob data proof: %w", err)
			}

			calldata, err := r.l1RollupABI.Pack(
				"finalizeBatchWithProof4844",
				dbBatch.BatchHeader,
				common.HexToHash(parentBatchStateRoot),
				common.HexToHash(dbBatch.StateRoot),
				common.HexToHash(dbBatch.WithdrawRoot),
				blobDataProof,
				aggProof.Proof,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to pack finalizeBatchWithProof4844: %w", err)
			}
			return calldata, nil
		}

		return nil, fmt.Errorf("failed to finalizeBatch4844 without proof: unsupported feature")
	}

	// codecv0
	var calldata []byte
	if withProof {
		aggProof, err := r.batchOrm.GetVerifiedProofByHash(r.ctx, dbBatch.Hash)
		if err != nil {
			return nil, fmt.Errorf("failed to get verified proof by hash, index: %d, err: %w", dbBatch.Index, err)
		}

		if err = aggProof.SanityCheck(); err != nil {
			return nil, fmt.Errorf("failed to check agg_proof sanity, index: %d, err: %w", dbBatch.Index, err)
		}

		calldata, err = r.l1RollupABI.Pack(
			"finalizeBatchWithProof",
			dbBatch.BatchHeader,
			common.HexToHash(parentBatchStateRoot),
			common.HexToHash(dbBatch.StateRoot),
			common.HexToHash(dbBatch.WithdrawRoot),
			aggProof.Proof,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to pack finalizeBatchWithProof: %w", err)
		}
	} else {
		var err error
		calldata, err = r.l1RollupABI.Pack(
			"finalizeBatch",
			dbBatch.BatchHeader,
			common.HexToHash(parentBatchStateRoot),
			common.HexToHash(dbBatch.StateRoot),
			common.HexToHash(dbBatch.WithdrawRoot),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to pack finalizeBatch: %w", err)
		}
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

	if r.blobCommitSender != nil {
		r.blobCommitSender.Stop()
	}
}
