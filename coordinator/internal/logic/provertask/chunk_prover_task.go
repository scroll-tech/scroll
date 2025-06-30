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
	"github.com/scroll-tech/go-ethereum/params"
	"gorm.io/gorm"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
	"scroll-tech/common/utils"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/orm"
	coordinatorType "scroll-tech/coordinator/internal/types"
	cutils "scroll-tech/coordinator/internal/utils"
)

// ChunkProverTask the chunk prover task
type ChunkProverTask struct {
	BaseProverTask

	chunkTaskGetTaskTotal  *prometheus.CounterVec
	chunkTaskGetTaskProver *prometheus.CounterVec
}

// NewChunkProverTask new a chunk prover task
func NewChunkProverTask(cfg *config.Config, chainCfg *params.ChainConfig, db *gorm.DB, expectedVk map[string][]byte, reg prometheus.Registerer) *ChunkProverTask {
	cp := &ChunkProverTask{
		BaseProverTask: BaseProverTask{
			db:                 db,
			cfg:                cfg,
			chainCfg:           chainCfg,
			expectedVk:         expectedVk,
			chunkOrm:           orm.NewChunk(db),
			blockOrm:           orm.NewL2Block(db),
			proverTaskOrm:      orm.NewProverTask(db),
			proverBlockListOrm: orm.NewProverBlockList(db),
		},
		chunkTaskGetTaskTotal: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Name: "coordinator_chunk_get_task_total",
			Help: "Total number of chunk get task.",
		}, []string{"fork_name"}),
		chunkTaskGetTaskProver: newGetTaskCounterVec(promauto.With(reg), "chunk"),
	}
	return cp
}

// Assign the chunk proof which need to prove
func (cp *ChunkProverTask) Assign(ctx *gin.Context, getTaskParameter *coordinatorType.GetTaskParameter) (*coordinatorType.GetTaskSchema, error) {
	taskCtx, err := cp.checkParameter(ctx)
	if err != nil || taskCtx == nil {
		return nil, fmt.Errorf("check prover task parameter failed, error:%w", err)
	}

	maxActiveAttempts := cp.cfg.ProverManager.ProversPerSession
	maxTotalAttempts := cp.cfg.ProverManager.SessionAttempts
	if taskCtx.ProverProviderType == uint8(coordinatorType.ProverProviderTypeExternal) {
		unassignedChunkCount, getCountError := cp.chunkOrm.GetUnassignedChunkCount(ctx.Copy(), maxActiveAttempts, maxTotalAttempts, getTaskParameter.ProverHeight)
		if getCountError != nil {
			log.Error("failed to get unassigned chunk proving tasks count", "height", getTaskParameter.ProverHeight, "err", getCountError)
			return nil, ErrCoordinatorInternalFailure
		}
		// Assign external prover if unassigned task number exceeds threshold
		if unassignedChunkCount < cp.cfg.ProverManager.ExternalProverThreshold {
			return nil, nil
		}
	}

	var chunkTask *orm.Chunk
	var hardForkName string
	for i := 0; i < 5; i++ {
		var getTaskError error
		var tmpChunkTask *orm.Chunk
		if taskCtx.hasAssignedTask != nil {
			log.Debug("retrieved assigned task chunk", "taskID", taskCtx.hasAssignedTask.TaskID, "prover", taskCtx.ProverName)
			tmpChunkTask, getTaskError = cp.chunkOrm.GetChunkByHash(ctx.Copy(), taskCtx.hasAssignedTask.TaskID)
			if getTaskError != nil {
				log.Error("failed to get chunk has assigned to prover", "taskID", taskCtx.hasAssignedTask.TaskID, "err", getTaskError)
				return nil, ErrCoordinatorInternalFailure
			} else if tmpChunkTask == nil {
				// if the assigned chunk dropped, there would be too much issue to assign another
				return nil, fmt.Errorf("prover with publicKey %s is already assigned a dropped chunk. ProverName: %s, ProverVersion: %s",
					taskCtx.PublicKey, taskCtx.ProverName, taskCtx.ProverVersion)
			}
		}

		if tmpChunkTask == nil {
			tmpChunkTask, getTaskError = cp.chunkOrm.GetAssignedChunk(ctx.Copy(), maxActiveAttempts, maxTotalAttempts, getTaskParameter.ProverHeight)
			if getTaskError != nil {
				log.Error("failed to get assigned chunk proving tasks", "height", getTaskParameter.ProverHeight, "err", getTaskError)
				return nil, ErrCoordinatorInternalFailure
			}
		}
		// Why here need get again? In order to support a task can assign to multiple prover, need also assign `ProvingTaskAssigned`
		// chunk to prover. But use `proving_status in (1, 2)` will not use the postgres index. So need split the sql.
		if tmpChunkTask == nil {
			tmpChunkTask, getTaskError = cp.chunkOrm.GetUnassignedChunk(ctx.Copy(), maxActiveAttempts, maxTotalAttempts, getTaskParameter.ProverHeight)
			if getTaskError != nil {
				log.Error("failed to get unassigned chunk proving tasks", "height", getTaskParameter.ProverHeight, "err", getTaskError)
				return nil, ErrCoordinatorInternalFailure
			}
		}

		if tmpChunkTask == nil {
			log.Debug("get empty chunk", "height", getTaskParameter.ProverHeight)
			return nil, nil
		}

		taskCtx.taskType = message.ProofTypeChunk
		taskCtx.chunkTask = tmpChunkTask

		var checkErr error
		hardForkName, checkErr = cp.hardForkSanityCheck(ctx, taskCtx)
		if checkErr != nil {
			log.Debug("hard fork sanity check failed", "height", getTaskParameter.ProverHeight, "err", checkErr)
			return nil, nil
		}

		// we are simply pick the chunk which has been assigned, so don't bother to update attempts or check failed before
		if taskCtx.hasAssignedTask == nil {
			// Don't dispatch the same failing job to the same prover
			proverTasks, getFailedTaskError := cp.proverTaskOrm.GetFailedProverTasksByHash(ctx.Copy(), message.ProofTypeChunk, tmpChunkTask.Hash, 2)
			if getFailedTaskError != nil {
				log.Error("failed to get prover tasks", "proof type", message.ProofTypeChunk.String(), "task ID", tmpChunkTask.Hash, "error", getFailedTaskError)
				return nil, ErrCoordinatorInternalFailure
			}
			for i := 0; i < len(proverTasks); i++ {
				if proverTasks[i].ProverPublicKey == taskCtx.PublicKey ||
					taskCtx.ProverProviderType == uint8(coordinatorType.ProverProviderTypeExternal) && cutils.IsExternalProverNameMatch(proverTasks[i].ProverName, taskCtx.ProverName) {
					log.Debug("get empty chunk, the prover already failed this task", "height", getTaskParameter.ProverHeight, "task ID", tmpChunkTask.Hash, "prover name", taskCtx.ProverName, "prover public key", taskCtx.PublicKey)
					return nil, nil
				}
			}

			rowsAffected, updateAttemptsErr := cp.chunkOrm.UpdateChunkAttempts(ctx.Copy(), tmpChunkTask.Index, tmpChunkTask.ActiveAttempts, tmpChunkTask.TotalAttempts)
			if updateAttemptsErr != nil {
				log.Error("failed to update chunk attempts", "height", getTaskParameter.ProverHeight, "err", updateAttemptsErr)
				return nil, ErrCoordinatorInternalFailure
			}

			if rowsAffected == 0 {
				time.Sleep(100 * time.Millisecond)
				continue
			}
		}
		chunkTask = tmpChunkTask
		break
	}

	if chunkTask == nil {
		log.Debug("get empty unassigned chunk after retry 5 times", "height", getTaskParameter.ProverHeight)
		return nil, nil
	}

	log.Info("start chunk generation session", "task_id", chunkTask.Hash, "public key", taskCtx.PublicKey, "prover name", taskCtx.ProverName)
	var proverTask *orm.ProverTask
	if taskCtx.hasAssignedTask == nil {
		proverTask = &orm.ProverTask{
			TaskID:          chunkTask.Hash,
			ProverPublicKey: taskCtx.PublicKey,
			TaskType:        int16(message.ProofTypeChunk),
			ProverName:      taskCtx.ProverName,
			ProverVersion:   taskCtx.ProverVersion,
			ProvingStatus:   int16(types.ProverAssigned),
			FailureType:     int16(types.ProverTaskFailureTypeUndefined),
			// here why need use UTC time. see scroll/common/database/db.go
			AssignedAt: utils.NowUTC(),
		}
	} else {
		proverTask = taskCtx.hasAssignedTask
	}

	taskMsg, err := cp.formatProverTask(ctx.Copy(), proverTask, chunkTask, hardForkName)
	if err != nil {
		cp.recoverActiveAttempts(ctx, chunkTask)
		log.Error("format prover task failure", "task_id", chunkTask.Hash, "err", err)
		return nil, ErrCoordinatorInternalFailure
	}

	if getTaskParameter.Universal {
		var metadata []byte
		taskMsg, metadata, err = cp.applyUniversal(taskMsg)
		if err != nil {
			cp.recoverActiveAttempts(ctx, chunkTask)
			log.Error("Generate universal prover task failure", "task_id", chunkTask.Hash, "type", "chunk")
			return nil, ErrCoordinatorInternalFailure
		}
		proverTask.Metadata = metadata
	}

	if taskCtx.hasAssignedTask == nil {
		if err = cp.proverTaskOrm.InsertProverTask(ctx.Copy(), proverTask); err != nil {
			cp.recoverActiveAttempts(ctx, chunkTask)
			log.Error("insert chunk prover task fail", "task_id", chunkTask.Hash, "publicKey", taskCtx.PublicKey, "err", err)
			return nil, ErrCoordinatorInternalFailure
		}
	}
	// notice uuid is set as a side effect of InsertProverTask
	taskMsg.UUID = proverTask.UUID.String()

	cp.chunkTaskGetTaskTotal.WithLabelValues(hardForkName).Inc()
	cp.chunkTaskGetTaskProver.With(prometheus.Labels{
		coordinatorType.LabelProverName:      proverTask.ProverName,
		coordinatorType.LabelProverPublicKey: proverTask.ProverPublicKey,
		coordinatorType.LabelProverVersion:   proverTask.ProverVersion,
	}).Inc()

	return taskMsg, nil
}

func (cp *ChunkProverTask) formatProverTask(ctx context.Context, task *orm.ProverTask, chunk *orm.Chunk, hardForkName string) (*coordinatorType.GetTaskSchema, error) {
	// Get block hashes.
	blockHashes, dbErr := cp.blockOrm.GetL2BlockHashesByChunkHash(ctx, task.TaskID)
	if dbErr != nil || len(blockHashes) == 0 {
		return nil, fmt.Errorf("failed to fetch block hashes of a chunk, chunk hash:%s err:%w", task.TaskID, dbErr)
	}

	var taskDetailBytes []byte
	taskDetail := message.ChunkTaskDetail{
		BlockHashes:      blockHashes,
		PrevMsgQueueHash: common.HexToHash(chunk.PrevL1MessageQueueHash),
	}

	if hardForkName == message.EuclidV2Fork {
		taskDetail.ForkName = message.EuclidV2ForkNameForProver
	} else {
		log.Error("unsupported hard fork name", "hard_fork_name", hardForkName)
		return nil, fmt.Errorf("unsupported hard fork name: %s", hardForkName)
	}

	var err error
	taskDetailBytes, err = json.Marshal(taskDetail)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal block hashes hash:%s, err:%w", task.TaskID, err)
	}

	proverTaskSchema := &coordinatorType.GetTaskSchema{
		TaskID:       task.TaskID,
		TaskType:     int(message.ProofTypeChunk),
		TaskData:     string(taskDetailBytes),
		HardForkName: hardForkName,
	}

	log.Debug("TaskData", "task_id", task.TaskID, "task_type", message.ProofTypeChunk.String(), "hard_fork_name", hardForkName, "task_data", proverTaskSchema.TaskData)

	return proverTaskSchema, nil
}

func (cp *ChunkProverTask) recoverActiveAttempts(ctx *gin.Context, chunkTask *orm.Chunk) {
	if err := cp.chunkOrm.DecreaseActiveAttemptsByHash(ctx, chunkTask.Hash); err != nil {
		log.Error("failed to recover chunk active attempts", "hash", chunkTask.Hash, "error", err)
	}
}
