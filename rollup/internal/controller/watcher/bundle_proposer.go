package watcher

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/params"
	"gorm.io/gorm"

	"scroll-tech/rollup/internal/config"
	"scroll-tech/rollup/internal/orm"
)

// BundleProposer proposes bundles based on available unbundled batches.
type BundleProposer struct {
	ctx context.Context
	db  *gorm.DB

	chunkOrm  *orm.Chunk
	batchOrm  *orm.Batch
	bundleOrm *orm.Bundle

	cfg *config.BundleProposerConfig

	minCodecVersion encoding.CodecVersion
	chainCfg        *params.ChainConfig

	bundleProposerCircleTotal           prometheus.Counter
	proposeBundleFailureTotal           prometheus.Counter
	proposeBundleUpdateInfoTotal        prometheus.Counter
	proposeBundleUpdateInfoFailureTotal prometheus.Counter
	bundleBatchesNum                    prometheus.Gauge
	bundleFirstBlockTimeoutReached      prometheus.Counter
	bundleBatchesProposeNotEnoughTotal  prometheus.Counter
}

// NewBundleProposer creates a new BundleProposer instance.
func NewBundleProposer(ctx context.Context, cfg *config.BundleProposerConfig, minCodecVersion encoding.CodecVersion, chainCfg *params.ChainConfig, db *gorm.DB, reg prometheus.Registerer) *BundleProposer {
	log.Info("new bundle proposer", "bundleBatchesNum", cfg.MaxBatchNumPerBundle, "bundleTimeoutSec", cfg.BundleTimeoutSec)

	p := &BundleProposer{
		ctx:             ctx,
		db:              db,
		chunkOrm:        orm.NewChunk(db),
		batchOrm:        orm.NewBatch(db),
		bundleOrm:       orm.NewBundle(db),
		cfg:             cfg,
		minCodecVersion: minCodecVersion,
		chainCfg:        chainCfg,

		bundleProposerCircleTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "rollup_propose_bundle_circle_total",
			Help: "Total number of propose bundle attempts.",
		}),
		proposeBundleFailureTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "rollup_propose_bundle_failure_total",
			Help: "Total number of propose bundle failures.",
		}),
		proposeBundleUpdateInfoTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "rollup_propose_bundle_update_info_total",
			Help: "Total number of propose bundle update info attempts.",
		}),
		proposeBundleUpdateInfoFailureTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "rollup_propose_bundle_update_info_failure_total",
			Help: "Total number of propose bundle update info failures.",
		}),
		bundleBatchesNum: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Name: "rollup_propose_bundle_batches_number",
			Help: "The number of batches in the current bundle.",
		}),
		bundleFirstBlockTimeoutReached: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "rollup_propose_bundle_first_block_timeout_reached_total",
			Help: "Total times the first block in a bundle reached the timeout.",
		}),
		bundleBatchesProposeNotEnoughTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "rollup_propose_bundle_batches_propose_not_enough_total",
			Help: "Total number of times there were not enough batches to propose a bundle.",
		}),
	}

	return p
}

// TryProposeBundle tries to propose a new bundle.
func (p *BundleProposer) TryProposeBundle() {
	p.bundleProposerCircleTotal.Inc()
	if err := p.proposeBundle(); err != nil {
		p.proposeBundleFailureTotal.Inc()
		log.Error("propose new bundle failed", "err", err)
		return
	}
}

func (p *BundleProposer) updateDBBundleInfo(batches []*orm.Batch, codecVersion encoding.CodecVersion) error {
	if len(batches) == 0 {
		return nil
	}

	p.proposeBundleUpdateInfoTotal.Inc()
	err := p.db.Transaction(func(dbTX *gorm.DB) error {
		bundle, err := p.bundleOrm.InsertBundle(p.ctx, batches, codecVersion, dbTX)
		if err != nil {
			log.Warn("BundleProposer.InsertBundle failed", "err", err)
			return err
		}
		if err := p.batchOrm.UpdateBundleHashInRange(p.ctx, bundle.StartBatchIndex, bundle.EndBatchIndex, bundle.Hash, dbTX); err != nil {
			log.Error("failed to update bundle_hash for batches", "bundle hash", bundle.Hash, "start batch index", bundle.StartBatchIndex, "end batch index", bundle.EndBatchIndex, "err", err)
			return err
		}
		return nil
	})
	if err != nil {
		p.proposeBundleUpdateInfoFailureTotal.Inc()
		log.Error("update chunk info in orm failed", "err", err)
		return err
	}
	return nil
}

func (p *BundleProposer) proposeBundle() error {
	firstUnbundledBatchIndex, err := p.bundleOrm.GetFirstUnbundledBatchIndex(p.ctx)
	if err != nil {
		return err
	}

	// select at most maxBlocksThisChunk blocks
	maxBatchesThisBundle := p.cfg.MaxBatchNumPerBundle
	batches, err := p.batchOrm.GetCommittedBatchesGEIndexGECodecVersion(p.ctx, firstUnbundledBatchIndex, p.minCodecVersion, int(maxBatchesThisBundle))
	if err != nil {
		return err
	}

	if len(batches) == 0 {
		return nil
	}

	// Ensure all blocks in the same chunk use the same hardfork name
	// If a different hardfork name is found, truncate the blocks slice at that point
	firstChunk, err := p.chunkOrm.GetChunkByIndex(p.ctx, batches[0].StartChunkIndex)
	if err != nil {
		return err
	}

	if firstChunk == nil {
		log.Error("first chunk not found", "start chunk index", batches[0].StartChunkIndex, "start batch index", batches[0].Index, "firstUnbundledBatchIndex", firstUnbundledBatchIndex)
		return errors.New("first chunk not found in proposeBundle")
	}

	hardforkName := encoding.GetHardforkName(p.chainCfg, firstChunk.StartBlockNumber, firstChunk.StartBlockTime)
	codecVersion := encoding.CodecVersion(batches[0].CodecVersion)

	if codecVersion < p.minCodecVersion {
		return fmt.Errorf("unsupported codec version: %v, expected at least %v", codecVersion, p.minCodecVersion)
	}

	for i := 1; i < len(batches); i++ {
		// Make sure that all batches have been committed.
		if len(batches[i].CommitTxHash) == 0 {
			return fmt.Errorf("commit tx hash is empty for batch %v %s", batches[i].Index, batches[i].Hash)
		}

		chunk, err := p.chunkOrm.GetChunkByIndex(p.ctx, batches[i].StartChunkIndex)
		if err != nil {
			return err
		}
		currentHardfork := encoding.GetHardforkName(p.chainCfg, chunk.StartBlockNumber, chunk.StartBlockTime)
		if currentHardfork != hardforkName {
			batches = batches[:i]
			maxBatchesThisBundle = uint64(i) // update maxBlocksThisChunk to trigger chunking, because these blocks are the last blocks before the hardfork
			break
		}
	}

	if uint64(len(batches)) == maxBatchesThisBundle {
		log.Info("reached maximum number of batches per bundle", "batch count", len(batches), "start batch index", batches[0].Index, "end batch index", batches[len(batches)-1].Index)

		batches, err = p.allBatchesCommittedInSameTXIncluded(batches)
		if err != nil {
			return fmt.Errorf("failed to include all batches committed in the same tx: %w", err)
		}

		p.bundleFirstBlockTimeoutReached.Inc()
		p.bundleBatchesNum.Set(float64(len(batches)))
		return p.updateDBBundleInfo(batches, codecVersion)
	}

	currentTimeSec := uint64(time.Now().Unix())
	if firstChunk.StartBlockTime+p.cfg.BundleTimeoutSec < currentTimeSec {
		log.Info("first block timeout", "batch count", len(batches), "start block number", firstChunk.StartBlockNumber, "start block timestamp", firstChunk.StartBlockTime, "bundle timeout", p.cfg.BundleTimeoutSec, "current time", currentTimeSec)

		batches, err = p.allBatchesCommittedInSameTXIncluded(batches)
		if err != nil {
			return fmt.Errorf("failed to include all batches committed in the same tx: %w", err)
		}

		p.bundleFirstBlockTimeoutReached.Inc()
		p.bundleBatchesNum.Set(float64(len(batches)))
		return p.updateDBBundleInfo(batches, codecVersion)
	}

	log.Debug("pending batches are not enough and do not contain a timeout batch")
	p.bundleBatchesProposeNotEnoughTotal.Inc()
	return nil
}

// allBatchesCommittedInSameTXIncluded makes sure that all batches that were committed in the same tx are included in the bundle.
// If the last batch of the input batches was committed in the same tx as other batches but has not the highest index amongst those,
// we need to remove all batches with the same commit tx hash.
// As a result, all batches with the same commit tx hash will always be included in a single bundle.
func (p *BundleProposer) allBatchesCommittedInSameTXIncluded(batches []*orm.Batch) ([]*orm.Batch, error) {
	lastBatch := batches[len(batches)-1]
	fields := map[string]interface{}{
		"commit_tx_hash = ?": lastBatch.CommitTxHash,
	}

	// get all batches with the same commit tx hash as lastBatch
	batchesWithSameCommitTX, err := p.batchOrm.GetBatches(p.ctx, fields, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get batches with the same commit tx hash: %w", err)
	}

	// This should never happen as we take the commit tx hash from the last batch which should always exist in this returned list
	if len(batchesWithSameCommitTX) == 0 {
		return nil, fmt.Errorf("no matching batches found for commit tx hash %s", lastBatch.CommitTxHash)
	}

	// get the batch with the highest index amongst the batches with the same commit tx hash as lastBatch
	lastBatchWithSameCommitTX := batchesWithSameCommitTX[len(batchesWithSameCommitTX)-1]

	// check if lastBatchWithSameCommitTX is included in the input batches -> if not, we need to remove all batches with the same commit tx hash
	batchIncluded := lastBatch.Index == lastBatchWithSameCommitTX.Index
	if !batchIncluded {
		// we need to remove all batches with the same commit tx hash
		for i := 0; i < len(batches); i++ {
			if batches[i].CommitTxHash != lastBatchWithSameCommitTX.CommitTxHash {
				continue
			}

			batches = batches[:i]
			break
		}
	}

	if len(batches) == 0 {
		return nil, fmt.Errorf("no batches anymore after cleaning up batches with the same commit tx hash %s", lastBatch.CommitTxHash)
	}

	return batches, nil
}
