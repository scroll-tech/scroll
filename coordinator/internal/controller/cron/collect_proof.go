package cron

import (
	"context"
	"fmt"
	"time"

	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/logic/collector"
	"scroll-tech/coordinator/internal/orm"
)

// Collector collect the block batch or agg task to send to prover
type Collector struct {
	cfg *config.Config
	db  *gorm.DB
	ctx context.Context

	stopRunChan     chan struct{}
	stopTimeoutChan chan struct{}

	collectors map[message.ProofType]collector.Collector

	proverTaskOrm *orm.ProverTask
	chunkOrm      *orm.Chunk
	batchOrm      *orm.Batch
}

// NewCollector create a collector to cron collect the data to send to prover
func NewCollector(ctx context.Context, db *gorm.DB, cfg *config.Config) *Collector {
	c := &Collector{
		cfg:             cfg,
		db:              db,
		ctx:             ctx,
		stopRunChan:     make(chan struct{}),
		stopTimeoutChan: make(chan struct{}),
		collectors:      make(map[message.ProofType]collector.Collector),
		proverTaskOrm:   orm.NewProverTask(db),
		chunkOrm:        orm.NewChunk(db),
		batchOrm:        orm.NewBatch(db),
	}

	c.collectors[message.ProofTypeBatch] = collector.NewBatchProofCollector(cfg, db)
	c.collectors[message.ProofTypeChunk] = collector.NewChunkProofCollector(cfg, db)

	go c.run()
	go c.timeoutProofTask()

	log.Info("Start coordinator successfully.")

	return c
}

// Stop all the collector
func (c *Collector) Stop() {
	c.stopRunChan <- struct{}{}
	c.stopTimeoutChan <- struct{}{}
}

// run loop and cron collect
func (c *Collector) run() {
	defer func() {
		if err := recover(); err != nil {
			nerr := fmt.Errorf("collector panic error:%v", err)
			log.Warn(nerr.Error())
		}
	}()

	ticker := time.NewTicker(time.Second * 2)
	for {
		select {
		case <-ticker.C:
			for _, tmpCollector := range c.collectors {
				if err := tmpCollector.Collect(c.ctx); err != nil {
					log.Warn("collect data to prover failure", "collector name", tmpCollector.Name(), "error", err)
				}
			}
		case <-c.ctx.Done():
			if c.ctx.Err() != nil {
				log.Error("manager context canceled with error", "error", c.ctx.Err())
			}
			return
		case <-c.stopRunChan:
			log.Info("the coordinator run loop exit")
			return
		}
	}
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
			assignedProverTasks, err := c.proverTaskOrm.GetAssignedProverTasks(c.ctx, 10)
			if err != nil {
				log.Error("get unassigned session info failure", "error", err)
				break
			}

			for _, assignedProverTask := range assignedProverTasks {
				timeoutDuration := time.Duration(c.cfg.RollerManagerConfig.CollectionTime) * time.Minute
				// here not update the block batch proving status failed, because the collector loop will check
				// the attempt times. if reach the times, the collector will set the block batch proving status.
				if time.Since(assignedProverTask.AssignedAt) >= timeoutDuration {
					log.Warn("proof task have reach the timeout", "task id", assignedProverTask.TaskID,
						"prover public key", assignedProverTask.ProverPublicKey, "prover name", assignedProverTask.ProverName, "task type", assignedProverTask.TaskType)
					err = c.db.Transaction(func(tx *gorm.DB) error {
						// update prover task proving status as RollerProofInvalid
						if err = c.proverTaskOrm.UpdateProverTaskProvingStatus(c.ctx, message.ProofType(assignedProverTask.TaskType),
							assignedProverTask.TaskID, assignedProverTask.ProverPublicKey, types.RollerProofInvalid, tx); err != nil {
							log.Error("update prover task proving status failure", "hash", assignedProverTask.TaskID, "pubKey", assignedProverTask.ProverPublicKey, "err", err)
							return err
						}

						// update prover task failure type
						if err = c.proverTaskOrm.UpdateProverTaskFailureType(c.ctx, message.ProofType(assignedProverTask.TaskType),
							assignedProverTask.TaskID, assignedProverTask.ProverPublicKey, types.ProverTaskFailureTypeTimeout, tx); err != nil {
							log.Error("update prover task failure type failure", "hash", assignedProverTask.TaskID, "pubKey", assignedProverTask.ProverPublicKey, "err", err)
							return err
						}

						// update the task to unassigned, let collector restart it
						if message.ProofType(assignedProverTask.TaskType) == message.ProofTypeChunk {
							if err = c.chunkOrm.UpdateProvingStatus(c.ctx, assignedProverTask.TaskID, types.ProvingTaskUnassigned, tx); err != nil {
								log.Error("update chunk proving status to unassigned to restart it failure", "hash", assignedProverTask.TaskID, "err", err)
							}
						}
						if message.ProofType(assignedProverTask.TaskType) == message.ProofTypeBatch {
							if err = c.batchOrm.UpdateProvingStatus(c.ctx, assignedProverTask.TaskID, types.ProvingTaskUnassigned, tx); err != nil {
								log.Error("update batch proving status to unassigned to restart it failure", "hash", assignedProverTask.TaskID, "err", err)
							}
						}
						return nil
					})
					if err != nil {
						log.Error("check task proof is timeout failure", "error", err)
					}
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
