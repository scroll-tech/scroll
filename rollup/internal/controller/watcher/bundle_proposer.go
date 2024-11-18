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

	maxBatchNumPerBundle uint64
	bundleTimeoutSec     uint64

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
		ctx:                  ctx,
		db:                   db,
		chunkOrm:             orm.NewChunk(db),
		batchOrm:             orm.NewBatch(db),
		bundleOrm:            orm.NewBundle(db),
		maxBatchNumPerBundle: cfg.MaxBatchNumPerBundle,
		bundleTimeoutSec:     cfg.BundleTimeoutSec,
		minCodecVersion:      minCodecVersion,
		chainCfg:             chainCfg,

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
	maxBatchesThisBundle := p.maxBatchNumPerBundle
	batches, err := p.batchOrm.GetBatchesGEIndexGECodecVersion(p.ctx, firstUnbundledBatchIndex, p.minCodecVersion, int(maxBatchesThisBundle))
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
		p.bundleFirstBlockTimeoutReached.Inc()
		p.bundleBatchesNum.Set(float64(len(batches)))
		return p.updateDBBundleInfo(batches, codecVersion)
	}

	currentTimeSec := uint64(time.Now().Unix())
	if firstChunk.StartBlockTime+p.bundleTimeoutSec < currentTimeSec {
		log.Info("first block timeout", "batch count", len(batches), "start block number", firstChunk.StartBlockNumber, "start block timestamp", firstChunk.StartBlockTime, "current time", currentTimeSec)
		p.bundleFirstBlockTimeoutReached.Inc()
		p.bundleBatchesNum.Set(float64(len(batches)))
		return p.updateDBBundleInfo(batches, codecVersion)
	}

	log.Debug("pending batches are not enough and do not contain a timeout batch")
	p.bundleBatchesProposeNotEnoughTotal.Inc()
	return nil
}
