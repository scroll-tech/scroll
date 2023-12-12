package provertask

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
	"scroll-tech/common/utils"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/orm"
	coordinatorType "scroll-tech/coordinator/internal/types"
)

// ErrCoordinatorInternalFailure coordinator internal db failure
var ErrCoordinatorInternalFailure = fmt.Errorf("coordinator internal error")

// ChunkProverTask the chunk prover task
type ChunkProverTask struct {
	BaseProverTask

	chunkAttemptsExceedTotal prometheus.Counter
	chunkTaskGetTaskTotal    prometheus.Counter
}

// NewChunkProverTask new a chunk prover task
func NewChunkProverTask(cfg *config.Config, db *gorm.DB, vk string, reg prometheus.Registerer) *ChunkProverTask {
	cp := &ChunkProverTask{
		BaseProverTask: BaseProverTask{
			vk:            vk,
			db:            db,
			cfg:           cfg,
			chunkOrm:      orm.NewChunk(db),
			blockOrm:      orm.NewL2Block(db),
			proverTaskOrm: orm.NewProverTask(db),
		},
		chunkAttemptsExceedTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "coordinator_chunk_attempts_exceed_total",
			Help: "Total number of chunk attempts exceed.",
		}),
		chunkTaskGetTaskTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "coordinator_chunk_get_task_total",
			Help: "Total number of chunk get task.",
		}),
	}
	return cp
}

// Assign the chunk proof which need to prove
func (cp *ChunkProverTask) Assign(ctx *gin.Context, getTaskParameter *coordinatorType.GetTaskParameter) (*coordinatorType.GetTaskSchema, error) {
	taskCtx, err := cp.checkParameter(ctx, getTaskParameter)
	if err != nil || taskCtx == nil {
		return nil, fmt.Errorf("check prover task parameter failed, error:%w", err)
	}

	maxActiveAttempts := cp.cfg.ProverManager.ProversPerSession
	maxTotalAttempts := cp.cfg.ProverManager.SessionAttempts
	var chunkTask *orm.Chunk
	for i := 0; i < 5; i++ {
		var getTaskError error
		var tmpChunkTask *orm.Chunk
		tmpChunkTask, getTaskError = cp.chunkOrm.GetAssignedChunk(ctx, getTaskParameter.ProverHeight, maxActiveAttempts, maxTotalAttempts)
		if getTaskError != nil {
			log.Error("failed to get assigned chunk proving tasks", "height", getTaskParameter.ProverHeight, "err", getTaskError)
			return nil, ErrCoordinatorInternalFailure
		}

		// Why here need get again? In order to support a task can assign to multiple prover, need also assign `ProvingTaskAssigned`
		// chunk to prover. But use `proving_status in (1, 2)` will not use the postgres index. So need split the sql.
		if tmpChunkTask == nil {
			tmpChunkTask, getTaskError = cp.chunkOrm.GetUnassignedChunk(ctx, getTaskParameter.ProverHeight, maxActiveAttempts, maxTotalAttempts)
			if getTaskError != nil {
				log.Error("failed to get unassigned chunk proving tasks", "height", getTaskParameter.ProverHeight, "err", getTaskError)
				return nil, ErrCoordinatorInternalFailure
			}
		}

		if tmpChunkTask == nil {
			log.Debug("get empty chunk", "height", getTaskParameter.ProverHeight)
			return nil, nil
		}

		rowsAffected, updateAttemptsErr := cp.chunkOrm.UpdateChunkAttempts(ctx, tmpChunkTask.Index, tmpChunkTask.ActiveAttempts, tmpChunkTask.TotalAttempts)
		if updateAttemptsErr != nil {
			log.Error("failed to update chunk attempts", "height", getTaskParameter.ProverHeight, "err", updateAttemptsErr)
			return nil, ErrCoordinatorInternalFailure
		}

		if rowsAffected == 0 {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		chunkTask = tmpChunkTask
		break
	}

	if chunkTask == nil {
		log.Debug("get empty unassigned chunk after retry 5 times", "height", getTaskParameter.ProverHeight)
		return nil, nil
	}

	log.Info("start chunk generation session", "id", chunkTask.Hash, "public key", taskCtx.PublicKey, "prover name", taskCtx.ProverName)

	proverTask := orm.ProverTask{
		TaskID:          chunkTask.Hash,
		ProverPublicKey: taskCtx.PublicKey,
		TaskType:        int16(message.ProofTypeChunk),
		ProverName:      taskCtx.ProverName,
		ProverVersion:   taskCtx.ProverVersion,
		ProvingStatus:   int16(types.ProverAssigned),
		FailureType:     int16(types.ProverTaskFailureTypeUndefined),
		// here why need use UTC time. see scroll/common/databased/db.go
		AssignedAt: utils.NowUTC(),
	}

	if err = cp.proverTaskOrm.InsertProverTask(ctx, &proverTask); err != nil {
		cp.recoverActiveAttempts(ctx, chunkTask)
		log.Error("insert chunk prover task fail", "taskID", chunkTask.Hash, "publicKey", taskCtx.PublicKey, "err", err)
		return nil, ErrCoordinatorInternalFailure
	}

	taskMsg, err := cp.formatProverTask(ctx, &proverTask, chunkTask)
	if err != nil {
		cp.recoverActiveAttempts(ctx, chunkTask)
		log.Error("format prover task failure", "hash", chunkTask.Hash, "err", err)
		return nil, ErrCoordinatorInternalFailure
	}

	cp.chunkTaskGetTaskTotal.Inc()

	return taskMsg, nil
}

func (cp *ChunkProverTask) formatProverTask(ctx context.Context, task *orm.ProverTask, chunk *orm.Chunk) (*coordinatorType.GetTaskSchema, error) {
	// Get block hashes.
	wrappedBlocks, wrappedErr := cp.blockOrm.GetL2BlocksByChunkHash(ctx, task.TaskID)
	if wrappedErr != nil || len(wrappedBlocks) == 0 {
		return nil, fmt.Errorf("failed to fetch wrapped blocks, chunk hash:%s err:%w", task.TaskID, wrappedErr)
	}

	blockHashes := make([]common.Hash, len(wrappedBlocks))
	for i, wrappedBlock := range wrappedBlocks {
		blockHashes[i] = wrappedBlock.Header.Hash()
	}

	parentChunk, err := cp.chunkOrm.GetChunkByHash(ctx, chunk.ParentChunkHash)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch parent chunk blocks, chunk hash:%s err:%w", chunk.ParentChunkHash, err)
	}

	taskDetail := message.ChunkTaskDetail{
		BlockHashes: blockHashes,
		PrevLastAppliedL1Block: func() uint64 {
			if parentChunk != nil {
				return parentChunk.LastAppliedL1Block
			}
			return 0
		}(),
		LastAppliedL1Block: chunk.LastAppliedL1Block,
		L1BlockRangeHash:   common.HexToHash(chunk.L1BlockRangeHash),
	}
	taskDataBytes, err := json.Marshal(taskDetail)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal block hashes hash:%s, err:%w", task.TaskID, err)
	}

	proverTaskSchema := &coordinatorType.GetTaskSchema{
		UUID:     task.UUID.String(),
		TaskID:   task.TaskID,
		TaskType: int(message.ProofTypeChunk),
		TaskData: string(taskDataBytes),
	}

	return proverTaskSchema, nil
}

func (cp *ChunkProverTask) recoverActiveAttempts(ctx *gin.Context, chunkTask *orm.Chunk) {
	if err := cp.chunkOrm.DecreaseActiveAttemptsByHash(ctx, chunkTask.Hash); err != nil {
		log.Error("failed to recover chunk active attempts", "hash", chunkTask.Hash, "error", err)
	}
}
