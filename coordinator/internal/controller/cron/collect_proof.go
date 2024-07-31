package cron

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/orm"
)

// Collector collect the block batch or agg task to send to prover
type Collector struct {
	cfg *config.Config
	db  *gorm.DB
	ctx context.Context

	stopBundleTimeoutChan       chan struct{}
	stopChunkTimeoutChan        chan struct{}
	stopBatchTimeoutChan        chan struct{}
	stopBatchAllChunkReadyChan  chan struct{}
	stopBundleAllBatchReadyChan chan struct{}
	stopCleanChallengeChan      chan struct{}

	proverTaskOrm *orm.ProverTask
	bundleOrm     *orm.Bundle
	chunkOrm      *orm.Chunk
	batchOrm      *orm.Batch
	challenge     *orm.Challenge

	timeoutBundleCheckerRunTotal     prometheus.Counter
	bundleProverTaskTimeoutTotal     prometheus.Counter
	timeoutBatchCheckerRunTotal      prometheus.Counter
	batchProverTaskTimeoutTotal      prometheus.Counter
	timeoutChunkCheckerRunTotal      prometheus.Counter
	chunkProverTaskTimeoutTotal      prometheus.Counter
	checkBatchAllChunkReadyRunTotal  prometheus.Counter
	checkBundleAllBatchReadyRunTotal prometheus.Counter
}

// NewCollector create a collector to cron collect the data to send to prover
func NewCollector(ctx context.Context, db *gorm.DB, cfg *config.Config, reg prometheus.Registerer) *Collector {
	c := &Collector{
		cfg:                         cfg,
		db:                          db,
		ctx:                         ctx,
		stopBundleTimeoutChan:       make(chan struct{}),
		stopChunkTimeoutChan:        make(chan struct{}),
		stopBatchTimeoutChan:        make(chan struct{}),
		stopBatchAllChunkReadyChan:  make(chan struct{}),
		stopBundleAllBatchReadyChan: make(chan struct{}),
		stopCleanChallengeChan:      make(chan struct{}),
		proverTaskOrm:               orm.NewProverTask(db),
		chunkOrm:                    orm.NewChunk(db),
		batchOrm:                    orm.NewBatch(db),
		bundleOrm:                   orm.NewBundle(db),
		challenge:                   orm.NewChallenge(db),

		timeoutBundleCheckerRunTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "coordinator_bundle_timeout_checker_run_total",
			Help: "Total number of bundle timeout checker run.",
		}),
		bundleProverTaskTimeoutTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "coordinator_bundle_prover_task_timeout_total",
			Help: "Total number of bundle timeout prover task.",
		}),
		timeoutBatchCheckerRunTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "coordinator_batch_timeout_checker_run_total",
			Help: "Total number of batch timeout checker run.",
		}),
		batchProverTaskTimeoutTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "coordinator_batch_prover_task_timeout_total",
			Help: "Total number of batch timeout prover task.",
		}),
		timeoutChunkCheckerRunTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "coordinator_chunk_timeout_checker_run_total",
			Help: "Total number of chunk timeout checker run.",
		}),
		chunkProverTaskTimeoutTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "coordinator_chunk_prover_task_timeout_total",
			Help: "Total number of chunk timeout prover task.",
		}),
		checkBatchAllChunkReadyRunTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "coordinator_check_batch_all_chunk_ready_run_total",
			Help: "Total number of check batch all chunks ready total",
		}),
		checkBundleAllBatchReadyRunTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "coordinator_check_bundle_all_batch_ready_run_total",
			Help: "Total number of check bundle all batches ready total",
		}),
	}

	go c.timeoutBundleProofTask()
	go c.timeoutBatchProofTask()
	go c.timeoutChunkProofTask()
	go c.checkBatchAllChunkReady()
	go c.checkBundleAllBatchReady()
	go c.cleanupChallenge()

	log.Info("Start coordinator cron successfully.")

	return c
}

// Stop all the collector
func (c *Collector) Stop() {
	c.stopChunkTimeoutChan <- struct{}{}
	c.stopBatchTimeoutChan <- struct{}{}
	c.stopBundleTimeoutChan <- struct{}{}
	c.stopBatchAllChunkReadyChan <- struct{}{}
	c.stopCleanChallengeChan <- struct{}{}
}

// timeoutBundleProofTask cron checks the send task is timeout. if timeout reached, restore the
// bundle task to unassigned. then the bundle collector can retry it.
func (c *Collector) timeoutBundleProofTask() {
	defer func() {
		if err := recover(); err != nil {
			nerr := fmt.Errorf("timeout bundle proof task panic error:%v", err)
			log.Warn(nerr.Error())
		}
	}()

	ticker := time.NewTicker(time.Second * 2)
	for {
		select {
		case <-ticker.C:
			c.timeoutBundleCheckerRunTotal.Inc()
			timeout := time.Duration(c.cfg.ProverManager.BundleCollectionTimeSec) * time.Second
			assignedProverTasks, err := c.proverTaskOrm.GetTimeoutAssignedProverTasks(c.ctx, 10, message.ProofTypeBundle, timeout)
			if err != nil {
				log.Error("get unassigned session info failure", "error", err)
				break
			}
			c.check(assignedProverTasks, c.bundleProverTaskTimeoutTotal)
		case <-c.ctx.Done():
			if c.ctx.Err() != nil {
				log.Error("manager context canceled with error", "error", c.ctx.Err())
			}
			return
		case <-c.stopBundleTimeoutChan:
			log.Info("the coordinator timeoutBundleProofTask run loop exit")
			return
		}
	}
}

// timeoutBatchProofTask cron check the send task is timeout. if timeout reached, restore the
// chunk/batch task to unassigned. then the batch/chunk collector can retry it.
func (c *Collector) timeoutBatchProofTask() {
	defer func() {
		if err := recover(); err != nil {
			nerr := fmt.Errorf("timeout batch proof task panic error:%v", err)
			log.Warn(nerr.Error())
		}
	}()

	ticker := time.NewTicker(time.Second * 2)
	for {
		select {
		case <-ticker.C:
			c.timeoutBatchCheckerRunTotal.Inc()
			timeout := time.Duration(c.cfg.ProverManager.BatchCollectionTimeSec) * time.Second
			assignedProverTasks, err := c.proverTaskOrm.GetTimeoutAssignedProverTasks(c.ctx, 10, message.ProofTypeBatch, timeout)
			if err != nil {
				log.Error("get unassigned session info failure", "error", err)
				break
			}
			c.check(assignedProverTasks, c.batchProverTaskTimeoutTotal)
		case <-c.ctx.Done():
			if c.ctx.Err() != nil {
				log.Error("manager context canceled with error", "error", c.ctx.Err())
			}
			return
		case <-c.stopBatchTimeoutChan:
			log.Info("the coordinator timeoutBatchProofTask run loop exit")
			return
		}
	}
}

func (c *Collector) timeoutChunkProofTask() {
	defer func() {
		if err := recover(); err != nil {
			nerr := fmt.Errorf("timeout proof chunk task panic error:%v", err)
			log.Warn(nerr.Error())
		}
	}()

	ticker := time.NewTicker(time.Second * 2)
	for {
		select {
		case <-ticker.C:
			c.timeoutChunkCheckerRunTotal.Inc()
			timeout := time.Duration(c.cfg.ProverManager.ChunkCollectionTimeSec) * time.Second
			assignedProverTasks, err := c.proverTaskOrm.GetTimeoutAssignedProverTasks(c.ctx, 10, message.ProofTypeChunk, timeout)
			if err != nil {
				log.Error("get unassigned session info failure", "error", err)
				break
			}
			c.check(assignedProverTasks, c.chunkProverTaskTimeoutTotal)

		case <-c.ctx.Done():
			if c.ctx.Err() != nil {
				log.Error("manager context canceled with error", "error", c.ctx.Err())
			}
			return
		case <-c.stopChunkTimeoutChan:
			log.Info("the coordinator timeoutChunkProofTask run loop exit")
			return
		}
	}
}

func (c *Collector) check(assignedProverTasks []orm.ProverTask, timeout prometheus.Counter) {
	// here not update the block batch proving status failed, because the collector loop will check
	// the attempt times. if reach the times, the collector will set the block batch proving status.
	for _, assignedProverTask := range assignedProverTasks {
		if c.proverTaskOrm.TaskTimeoutMoreThanOnce(c.ctx, message.ProofType(assignedProverTask.TaskType), assignedProverTask.TaskID) {
			log.Warn("Task timeout more than once", "taskType", message.ProofType(assignedProverTask.TaskType).String(), "hash", assignedProverTask.TaskID)
		}

		timeout.Inc()

		log.Warn("proof task have reach the timeout", "task id", assignedProverTask.TaskID,
			"prover public key", assignedProverTask.ProverPublicKey, "prover name", assignedProverTask.ProverName, "task type", assignedProverTask.TaskType)

		err := c.db.Transaction(func(tx *gorm.DB) error {
			if err := c.proverTaskOrm.UpdateProverTaskProvingStatusAndFailureType(c.ctx, assignedProverTask.UUID, types.ProverProofInvalid, types.ProverTaskFailureTypeTimeout, tx); err != nil {
				log.Error("update prover task proving status failure", "uuid", assignedProverTask.UUID, "hash", assignedProverTask.TaskID, "pubKey", assignedProverTask.ProverPublicKey, "err", err)
				return err
			}

			switch message.ProofType(assignedProverTask.TaskType) {
			case message.ProofTypeChunk:
				if err := c.chunkOrm.DecreaseActiveAttemptsByHash(c.ctx, assignedProverTask.TaskID, tx); err != nil {
					log.Error("decrease chunk active attempts failure", "uuid", assignedProverTask.UUID, "hash", assignedProverTask.TaskID, "pubKey", assignedProverTask.ProverPublicKey, "err", err)
					return err
				}

				if err := c.chunkOrm.UpdateProvingStatusFailed(c.ctx, assignedProverTask.TaskID, c.cfg.ProverManager.SessionAttempts, tx); err != nil {
					log.Error("update proving status failed failure", "uuid", assignedProverTask.UUID, "hash", assignedProverTask.TaskID, "pubKey", assignedProverTask.ProverPublicKey, "err", err)
					return err
				}
			case message.ProofTypeBatch:
				if err := c.batchOrm.DecreaseActiveAttemptsByHash(c.ctx, assignedProverTask.TaskID, tx); err != nil {
					log.Error("decrease batch active attempts failure", "uuid", assignedProverTask.UUID, "hash", assignedProverTask.TaskID, "pubKey", assignedProverTask.ProverPublicKey, "err", err)
					return err
				}

				if err := c.batchOrm.UpdateProvingStatusFailed(c.ctx, assignedProverTask.TaskID, c.cfg.ProverManager.SessionAttempts, tx); err != nil {
					log.Error("update proving status failed failure", "uuid", assignedProverTask.UUID, "hash", assignedProverTask.TaskID, "pubKey", assignedProverTask.ProverPublicKey, "err", err)
					return err
				}
			case message.ProofTypeBundle:
				if err := c.bundleOrm.DecreaseActiveAttemptsByHash(c.ctx, assignedProverTask.TaskID, tx); err != nil {
					log.Error("decrease bundle active attempts failure", "uuid", assignedProverTask.UUID, "hash", assignedProverTask.TaskID, "pubKey", assignedProverTask.ProverPublicKey, "err", err)
					return err
				}

				if err := c.bundleOrm.UpdateProvingStatusFailed(c.ctx, assignedProverTask.TaskID, c.cfg.ProverManager.SessionAttempts, tx); err != nil {
					log.Error("update proving status failed failure", "uuid", assignedProverTask.UUID, "hash", assignedProverTask.TaskID, "pubKey", assignedProverTask.ProverPublicKey, "err", err)
					return err
				}
			}

			return nil
		})
		if err != nil {
			log.Error("check task proof is timeout failure", "error", err)
		}
	}
}

func (c *Collector) checkBatchAllChunkReady() {
	defer func() {
		if err := recover(); err != nil {
			nerr := fmt.Errorf("check batch all chunk ready panic error:%v", err)
			log.Warn(nerr.Error())
		}
	}()

	ticker := time.NewTicker(time.Second * 10)
	for {
		select {
		case <-ticker.C:
			c.checkBatchAllChunkReadyRunTotal.Inc()
			page := 1
			pageSize := 50
			for {
				offset := (page - 1) * pageSize
				batches, err := c.batchOrm.GetUnassignedAndChunksUnreadyBatches(c.ctx, offset, pageSize)
				if err != nil {
					log.Warn("checkBatchAllChunkReady GetUnassignedAndChunksUnreadyBatches", "error", err)
					break
				}

				for _, batch := range batches {
					allReady, checkErr := c.chunkOrm.CheckIfBatchChunkProofsAreReady(c.ctx, batch.Hash)
					if checkErr != nil {
						log.Warn("checkBatchAllChunkReady CheckIfBatchChunkProofsAreReady failure", "error", checkErr, "hash", batch.Hash)
						continue
					}

					if !allReady {
						continue
					}

					if updateErr := c.batchOrm.UpdateChunkProofsStatusByBatchHash(c.ctx, batch.Hash, types.ChunkProofsStatusReady); updateErr != nil {
						log.Warn("checkBatchAllChunkReady UpdateChunkProofsStatusByBatchHash failure", "error", checkErr, "hash", batch.Hash)
					}
				}

				if len(batches) < pageSize {
					break
				}
				page++
			}

		case <-c.ctx.Done():
			if c.ctx.Err() != nil {
				log.Error("manager context canceled with error", "error", c.ctx.Err())
			}
			return
		case <-c.stopBatchAllChunkReadyChan:
			log.Info("the coordinator checkBatchAllChunkReady run loop exit")
			return
		}
	}
}

func (c *Collector) checkBundleAllBatchReady() {
	defer func() {
		if err := recover(); err != nil {
			nerr := fmt.Errorf("check batch all batches ready panic error:%v", err)
			log.Warn(nerr.Error())
		}
	}()

	ticker := time.NewTicker(time.Second * 10)
	for {
		select {
		case <-ticker.C:
			c.checkBundleAllBatchReadyRunTotal.Inc()
			page := 1
			pageSize := 50
			for {
				offset := (page - 1) * pageSize
				bundles, err := c.bundleOrm.GetUnassignedAndBatchesUnreadyBundles(c.ctx, offset, pageSize)
				if err != nil {
					log.Warn("checkBundleAllBatchReady GetUnassignedAndBatchesUnreadyBundles", "error", err)
					break
				}

				for _, bundle := range bundles {
					allReady, checkErr := c.batchOrm.CheckIfBundleBatchProofsAreReady(c.ctx, bundle.Hash)
					if checkErr != nil {
						log.Warn("checkBundleAllBatchReady CheckIfBundleBatchProofsAreReady failure", "error", checkErr, "hash", bundle.Hash)
						continue
					}

					if !allReady {
						continue
					}

					if updateErr := c.bundleOrm.UpdateBatchProofsStatusByBundleHash(c.ctx, bundle.Hash, types.BatchProofsStatusReady); updateErr != nil {
						log.Warn("checkBundleAllBatchReady UpdateBatchProofsStatusByBundleHash failure", "error", checkErr, "hash", bundle.Hash)
					}
				}

				if len(bundles) < pageSize {
					break
				}
				page++
			}

		case <-c.ctx.Done():
			if c.ctx.Err() != nil {
				log.Error("manager context canceled with error", "error", c.ctx.Err())
			}
			return
		case <-c.stopBundleAllBatchReadyChan:
			log.Info("the coordinator checkBundleAllBatchReady run loop exit")
			return
		}
	}
}
