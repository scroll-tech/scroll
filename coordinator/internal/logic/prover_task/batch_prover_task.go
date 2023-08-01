package prover_task

import (
	"context"
	"fmt"
	"github.com/goccy/go-json"

	"github.com/gin-gonic/gin"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
	"scroll-tech/common/utils"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/orm"
	coordinatorType "scroll-tech/coordinator/internal/types"
)

// BatchProverTask is prover task implement for batch proof
type BatchProverTask struct {
	BaseCollector
}

// NewBatchProverTask new a batch collector
func NewBatchProverTask(cfg *config.Config, db *gorm.DB) *BatchProverTask {
	bp := &BatchProverTask{
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

// Collect load and send batch tasks
func (bp *BatchProverTask) Collect(ctx *gin.Context) (*coordinatorType.ProverTaskSchema, error) {
	batchTasks, err := bp.batchOrm.GetUnassignedBatches(ctx, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to get unassigned batch proving tasks, error:%w", err)
	}

	if len(batchTasks) == 0 {
		return nil, nil
	}

	if len(batchTasks) != 1 {
		return nil, fmt.Errorf("get unassigned batch proving task len not 1, batch tasks:%v", batchTasks)
	}

	batchTask := batchTasks[0]
	log.Info("start batch proof generation session", "id", batchTask.Hash)

	if !bp.checkAttemptsExceeded(batchTask.Hash, message.ProofTypeBatch) {
		return nil, fmt.Errorf("the batch task id:%s check attempts have reach the maximum", batchTask.Hash)
	}

	publicKey, publicKeyExist := ctx.Get(coordinatorType.PublicKeyCtxKey)
	if !publicKeyExist {
		return nil, fmt.Errorf("get public key from contex failed")
	}

	proverName, proverNameExist := ctx.Get(coordinatorType.ProverNameCtxKey)
	if !proverNameExist {
		return nil, fmt.Errorf("get prover name from contex failed")
	}

	transErr := bp.db.Transaction(func(tx *gorm.DB) error {
		// Update session proving status as assigned.
		if err = bp.batchOrm.UpdateProvingStatus(ctx, batchTask.Hash, types.ProvingTaskAssigned, tx); err != nil {
			return fmt.Errorf("failed to update task status, id:%s, error:%w", batchTask.Hash, err)
		}

		proverTask := orm.ProverTask{
			TaskID:          batchTask.Hash,
			ProverPublicKey: publicKey.(string),
			TaskType:        int16(message.ProofTypeBatch),
			ProverName:      proverName.(string),
			ProvingStatus:   int16(types.ProverAssigned),
			FailureType:     int16(types.ProverTaskFailureTypeUndefined),
			// here why need use UTC time. see scroll/common/databased/db.go
			AssignedAt: utils.NowUTC(),
		}

		// Store session info.
		if err = bp.proverTaskOrm.SetProverTask(ctx, &proverTask, tx); err != nil {
			return fmt.Errorf("db set session info fail, session id:%s, error:%w", proverTask.TaskID, err)
		}

		return nil
	})

	if transErr != nil {
		return nil, transErr
	}

	taskMsg, err := bp.formatProverTask(ctx, batchTask.Hash)
	if err != nil {
		return nil, fmt.Errorf("format prover failure, id:%s error:%w", batchTask.Hash, err)
	}

	return taskMsg, nil
}

func (bp *BatchProverTask) formatProverTask(ctx context.Context, taskID string) (*coordinatorType.ProverTaskSchema, error) {
	// get chunk proofs from db
	chunkProofs, err := bp.chunkOrm.GetProofsByBatchHash(ctx, taskID)
	if err != nil {
		err = fmt.Errorf("failed to get chunk proofs for batch task id:%s err:%w ", taskID, err)
		return nil, err
	}

	chunkProofsBytes, err := json.Marshal(chunkProofs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal chunk proofs, taskID:%w err:%w", taskID, err)
	}

	taskMsg := &coordinatorType.ProverTaskSchema{
		TaskID:    taskID,
		ProofType: int(message.ProofTypeBatch),
		ProofData: chunkProofsBytes,
	}
	return taskMsg, nil
}
