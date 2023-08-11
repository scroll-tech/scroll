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
}

// NewCollector create a collector to cron collect the data to send to prover
func NewCollector(ctx context.Context, db *gorm.DB, cfg *config.Config) *Collector {
	c := &Collector{
		cfg:             cfg,
		db:              db,
		ctx:             ctx,
		stopTimeoutChan: make(chan struct{}),
		proverTaskOrm:   orm.NewProverTask(db),
		chunkOrm:        orm.NewChunk(db),
		batchOrm:        orm.NewBatch(db),
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
			timeout := time.Duration(c.cfg.ProverManager.CollectionTimeSec) * time.Second
			assignedProverTasks, err := c.proverTaskOrm.GetTimeoutAssignedProverTasks(c.ctx, 10, timeout)
			if err != nil {
				log.Error("get unassigned session info failure", "error", err)
				break
			}

			// here not update the block batch proving status failed, because the collector loop will check
			// the attempt times. if reach the times, the collector will set the block batch proving status.
			for _, assignedProverTask := range assignedProverTasks {
				log.Warn("proof task have reach the timeout", "task id", assignedProverTask.TaskID,
					"prover public key", assignedProverTask.ProverPublicKey, "prover name", assignedProverTask.ProverName, "task type", assignedProverTask.TaskType)
				err = c.db.Transaction(func(tx *gorm.DB) error {
					// update prover task proving status as ProverProofInvalid
					if err = c.proverTaskOrm.UpdateProverTaskProvingStatus(c.ctx, message.ProofType(assignedProverTask.TaskType),
						assignedProverTask.TaskID, assignedProverTask.ProverPublicKey, types.ProverProofInvalid, tx); err != nil {
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
