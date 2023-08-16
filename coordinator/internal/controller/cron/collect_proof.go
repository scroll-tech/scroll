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

	stopTimeoutChan chan struct{}

	proverTaskOrm *orm.ProverTask
	chunkOrm      *orm.Chunk
	batchOrm      *orm.Batch

	timeoutCheckerRunTotal prometheus.Counter
	proverTaskTimeoutTotal prometheus.Counter
}

// NewCollector create a collector to cron collect the data to send to prover
func NewCollector(ctx context.Context, db *gorm.DB, cfg *config.Config, reg prometheus.Registerer) *Collector {
	c := &Collector{
		cfg:             cfg,
		db:              db,
		ctx:             ctx,
		stopTimeoutChan: make(chan struct{}),
		proverTaskOrm:   orm.NewProverTask(db),
		chunkOrm:        orm.NewChunk(db),
		batchOrm:        orm.NewBatch(db),

		timeoutCheckerRunTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "coordinator_timeout_checker_run_total",
			Help: "Total number of timeout checker run.",
		}),
		proverTaskTimeoutTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "coordinator_prover_task_timeout_total",
			Help: "Total number of timeout prover task.",
		}),
	}

	go c.timeoutProofTask()

	log.Info("Start coordinator successfully.")

	return c
}

// Stop all the collector
func (c *Collector) Stop() {
	c.stopTimeoutChan <- struct{}{}
}

// timeoutTask cron check the send task is timeout. if timeout reached, restore the
// chunk/batch task to unassigned. then the batch/chunk collector can retry it.
func (c *Collector) timeoutProofTask() {
	defer func() {
		if err := recover(); err != nil {
			nerr := fmt.Errorf("timeout proof task panic error:%v", err)
			log.Warn(nerr.Error())
		}
	}()

	ticker := time.NewTicker(time.Second * 2)
	for {
		select {
		case <-ticker.C:
			c.timeoutCheckerRunTotal.Inc()
			timeout := time.Duration(c.cfg.ProverManager.CollectionTimeSec) * time.Second
			assignedProverTasks, err := c.proverTaskOrm.GetTimeoutAssignedProverTasks(c.ctx, 10, timeout)
			if err != nil {
				log.Error("get unassigned session info failure", "error", err)
				break
			}

			for _, assignedProverTask := range assignedProverTasks {
				c.proverTaskTimeoutTotal.Inc()
				log.Warn("proof task have reach the timeout", "task id", assignedProverTask.TaskID,
					"prover public key", assignedProverTask.ProverPublicKey, "prover name", assignedProverTask.ProverName, "task type", assignedProverTask.TaskType)
				err = c.db.Transaction(func(tx *gorm.DB) error {
					// update prover task proving status as ProverProofInvalid
					if err = c.proverTaskOrm.UpdateProverTaskProvingStatus(c.ctx, message.ProofType(assignedProverTask.TaskType),
						assignedProverTask.TaskID, assignedProverTask.ProverPublicKey, types.ProverProofInvalid, tx); err != nil {
						log.Error("update prover task proving status failure",
							"hash", assignedProverTask.TaskID, "pubKey", assignedProverTask.ProverPublicKey,
							"prover proving status", types.ProverProofInvalid, "err", err)
						return err
					}

					// update prover task failure type as ProverTaskFailureTypeTimeout
					if err = c.proverTaskOrm.UpdateProverTaskFailureType(c.ctx, message.ProofType(assignedProverTask.TaskType),
						assignedProverTask.TaskID, assignedProverTask.ProverPublicKey, types.ProverTaskFailureTypeTimeout, tx); err != nil {
						log.Error("update prover task failure type failure",
							"hash", assignedProverTask.TaskID, "pubKey", assignedProverTask.ProverPublicKey,
							"prover failure type", types.ProverTaskFailureTypeTimeout, "err", err)
						return err
					}

					if message.ProofType(assignedProverTask.TaskType) == message.ProofTypeChunk {
						if err = c.chunkOrm.DecreaseActiveAttemptsByHash(c.ctx, assignedProverTask.TaskID, tx); err != nil {
							log.Error("decrease active attempts of chunk failure", "hash", assignedProverTask.TaskID, "err", err)
							return err
						}
					}
					if message.ProofType(assignedProverTask.TaskType) == message.ProofTypeBatch {
						if err = c.batchOrm.DecreaseActiveAttemptsByHash(c.ctx, assignedProverTask.TaskID, tx); err != nil {
							log.Error("decrease active attempts of batch failure", "hash", assignedProverTask.TaskID, "err", err)
							return err
						}
					}

					var failedAssignmentCount uint64
					failedAssignmentCount, err = c.proverTaskOrm.GetFailedTaskAssignmentCount(c.ctx, assignedProverTask.TaskID, tx)
					if err != nil {
						log.Error("get failed task assignment count failure",
							"taskID", assignedProverTask.TaskID, "err", err)
						return err
					}

					if failedAssignmentCount >= uint64(c.cfg.ProverManager.SessionAttempts) {
						if message.ProofType(assignedProverTask.TaskType) == message.ProofTypeChunk {
							if err = c.chunkOrm.UpdateProvingStatus(c.ctx, assignedProverTask.TaskID, types.ProvingTaskFailed, tx); err != nil {
								log.Error("update chunk proving status failure", "hash", assignedProverTask.TaskID,
									"status", types.ProvingTaskFailed, "err", err)
								return err
							}
						}
						if message.ProofType(assignedProverTask.TaskType) == message.ProofTypeBatch {
							if err = c.batchOrm.UpdateProvingStatus(c.ctx, assignedProverTask.TaskID, types.ProvingTaskFailed, tx); err != nil {
								log.Error("update batch proving status failure", "hash", assignedProverTask.TaskID,
									"status", types.ProvingTaskFailed, "err", err)
								return err
							}
						}
					}
					return nil
				})
				if err != nil {
					log.Error("check task proof is timeout failure", "error", err)
				}
			}
		case <-c.ctx.Done():
			if c.ctx.Err() != nil {
				log.Error("manager context canceled with error", "error", c.ctx.Err())
			}
			return
		case <-c.stopTimeoutChan:
			log.Info("the coordinator run loop exit")
			return
		}
	}
}
