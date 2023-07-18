package collector

import (
	"context"
	"fmt"

	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/logic/rollermanager"
	"scroll-tech/coordinator/internal/orm"
	coordinatorType "scroll-tech/coordinator/internal/types"
)

// BatchProofCollector is collector implement for batch proof
type BatchProofCollector struct {
	BaseCollector
}

// NewBatchProofCollector new a batch collector
func NewBatchProofCollector(cfg *config.Config, db *gorm.DB) *BatchProofCollector {
	ac := &BatchProofCollector{
		BaseCollector: BaseCollector{
			db:            db,
			cfg:           cfg,
			chunkOrm:      orm.NewChunk(db),
			batchOrm:      orm.NewBatch(db),
			proverTaskOrm: orm.NewProverTask(db),
		},
	}
	return ac
}

// Name return the batch proof collector name
func (ac *BatchProofCollector) Name() string {
	return BatchCollectorName
}

// Collect load and send batch tasks
func (ac *BatchProofCollector) Collect(ctx context.Context) error {
	batchTasks, err := ac.batchOrm.GetUnassignedBatches(ctx, 1)
	if err != nil {
		return fmt.Errorf("failed to get unassigned batch proving tasks, error:%w", err)
	}

	if len(batchTasks) == 0 {
		return nil
	}

	if len(batchTasks) != 1 {
		return fmt.Errorf("get unassigned batch proving task len not 1")
	}

	batchTask := batchTasks[0]
	log.Info("start batch proof generation session", "id", batchTask.Hash)

	if rollermanager.Manager.GetNumberOfIdleRollers(message.ProofTypeBatch) == 0 {
		return fmt.Errorf("no idle common roller when starting proof generation session, id:%s", batchTask.Hash)
	}

	if !ac.checkAttemptsExceeded(batchTask.Hash, message.ProofTypeBatch) {
		return fmt.Errorf("the batch task id:%s check attempts have reach the maximum", batchTask.Hash)
	}

	rollerStatusList, err := ac.sendTask(ctx, batchTask.Hash)
	if err != nil {
		return fmt.Errorf("send batch task id:%s err:%w", batchTask.Hash, err)
	}

	transErr := ac.db.Transaction(func(tx *gorm.DB) error {
		// Update session proving status as assigned.
		if err = ac.batchOrm.UpdateProvingStatus(ctx, batchTask.Hash, types.ProvingTaskAssigned, tx); err != nil {
			return fmt.Errorf("failed to update task status, id:%s, error:%w", batchTask.Hash, err)
		}

		for _, rollerStatus := range rollerStatusList {
			proverTask := orm.ProverTask{
				TaskID:          batchTask.Hash,
				ProverPublicKey: rollerStatus.PublicKey,
				TaskType:        int16(message.ProofTypeBatch),
				ProverName:      rollerStatus.Name,
				ProvingStatus:   int16(types.RollerAssigned),
				FailureType:     int16(types.ProverTaskFailureTypeUndefined),
			}

			// Store session info.
			if err = ac.proverTaskOrm.SetProverTask(ctx, &proverTask, tx); err != nil {
				return fmt.Errorf("db set session info fail, session id:%s, error:%w", proverTask.TaskID, err)
			}
		}
		return nil
	})
	return transErr
}

func (ac *BatchProofCollector) sendTask(ctx context.Context, taskID string) ([]*coordinatorType.RollerStatus, error) {
	// get chunk proofs from db
	chunkProofs, err := ac.chunkOrm.GetProofsByBatchHash(ctx, taskID)
	if err != nil {
		err = fmt.Errorf("failed to get chunk proofs for batch task id:%s err:%w ", taskID, err)
		return nil, err
	}
	return ac.BaseCollector.sendTask(message.ProofTypeBatch, taskID, nil, chunkProofs)
}
