package collector

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
	"scroll-tech/common/utils"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/logic/provermanager"
	"scroll-tech/coordinator/internal/orm"
	coordinatorType "scroll-tech/coordinator/internal/types"
)

// BatchProofCollector is collector implement for batch proof
type BatchProofCollector struct {
	BaseCollector
}

// NewBatchProofCollector new a batch collector
func NewBatchProofCollector(cfg *config.Config, db *gorm.DB) *BatchProofCollector {
	bp := &BatchProofCollector{
		BaseCollector: BaseCollector{
			db:            db,
			cfg:           cfg,
			chunkOrm:      orm.NewChunk(db),
			batchOrm:      orm.NewBatch(db),
			proverTaskOrm: orm.NewProverTask(db),
		},
	}
	return bp
}

// Name return the batch proof collector name
func (bp *BatchProofCollector) Name() string {
	return BatchCollectorName
}

// Collect load and send batch tasks
func (bp *BatchProofCollector) Collect(ctx context.Context) error {
	batchTasks, err := bp.batchOrm.GetUnassignedBatches(ctx, 1)
	if err != nil {
		return fmt.Errorf("failed to get unassigned batch proving tasks, error:%w", err)
	}

	if len(batchTasks) == 0 {
		return nil
	}

	if len(batchTasks) != 1 {
		return fmt.Errorf("get unassigned batch proving task len not 1, batch tasks:%v", batchTasks)
	}

	batchTask := batchTasks[0]
	log.Info("start batch proof generation session", "id", batchTask.Hash)

	if provermanager.Manager.GetNumberOfIdleProvers(message.ProofTypeBatch) == 0 {
		return fmt.Errorf("no idle common prover when starting proof generation session, id:%s", batchTask.Hash)
	}

	if !bp.checkAttemptsExceeded(batchTask.Hash, message.ProofTypeBatch) {
		return fmt.Errorf("the batch task id:%s check attempts have reach the maximum", batchTask.Hash)
	}

	proverStatusList, err := bp.sendTask(ctx, batchTask.Hash)
	if err != nil {
		return fmt.Errorf("send batch task id:%s err:%w", batchTask.Hash, err)
	}

	transErr := bp.db.Transaction(func(tx *gorm.DB) error {
		// Update session proving status as assigned.
		if err = bp.batchOrm.UpdateProvingStatus(ctx, batchTask.Hash, types.ProvingTaskAssigned, tx); err != nil {
			return fmt.Errorf("failed to update task status, id:%s, error:%w", batchTask.Hash, err)
		}

		for _, proverStatus := range proverStatusList {
			proverTask := orm.ProverTask{
				TaskID:          batchTask.Hash,
				ProverPublicKey: proverStatus.PublicKey,
				TaskType:        int16(message.ProofTypeBatch),
				ProverName:      proverStatus.Name,
				ProvingStatus:   int16(types.ProverAssigned),
				FailureType:     int16(types.ProverTaskFailureTypeUndefined),
				// here why need use UTC time. see scroll/common/databased/db.go
				AssignedAt: utils.NowUTC(),
			}

			// Store session info.
			if err = bp.proverTaskOrm.SetProverTask(ctx, &proverTask, tx); err != nil {
				return fmt.Errorf("db set session info fail, session id:%s, error:%w", proverTask.TaskID, err)
			}
		}
		return nil
	})
	return transErr
}

func (bp *BatchProofCollector) sendTask(ctx context.Context, hash string) ([]*coordinatorType.ProverStatus, error) {
	// get chunks from db
	chunks, err := bp.chunkOrm.GetChunksByBatchHash(ctx, hash)
	if err != nil {
		err = fmt.Errorf("failed to get chunk proofs for batch task id:%s err:%w ", hash, err)
		return nil, err
	}

	taskDetail := &message.BatchTaskDetail{}
	for _, chunk := range chunks {
		chunkInfo:= &message.ChunkInfo{
			ChainID:       bp.cfg.L2Config.ChainID,
			PrevStateRoot: common.HexToHash(chunk.ParentChunkStateRoot),
			PostStateRoot: common.HexToHash(chunk.StateRoot),
			WithdrawRoot:  common.HexToHash(chunk.WithdrawRoot),
			DataHash:      common.HexToHash(chunk.Hash),
			IsPadding:     false,
		}
		taskDetail.ChunkInfos = append(taskDetail.ChunkInfos, chunkInfo)

		chunkProof := &message.ChunkProof{}
		if err := json.Unmarshal(chunk.Proof, chunkProof); err != nil {
			return nil, fmt.Errorf("json Unmarshal ChunkProof error: %w, chunk hash: %v", err, chunk.Hash)
		}
		taskDetail.SubProofs = append(taskDetail.SubProofs, chunkProof)
	}

	taskMsg := &message.TaskMsg{
		ID:              hash,
		Type:            message.ProofTypeBatch,
		ChunkTaskDetail: nil,
		BatchTaskDetail: taskDetail,
	}
	return bp.BaseCollector.sendTask(taskMsg)
}
