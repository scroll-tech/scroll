package provertask

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/params"
	"gorm.io/gorm"

	"scroll-tech/common/forks"
	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
	"scroll-tech/common/utils"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/orm"
	coordinatorType "scroll-tech/coordinator/internal/types"
)

// ChunkProverTask the chunk prover task
type ChunkProverTask struct {
	BaseProverTask

	chunkAttemptsExceedTotal prometheus.Counter
	chunkTaskGetTaskTotal    *prometheus.CounterVec
	chunkTaskGetTaskProver   *prometheus.CounterVec
}

// NewChunkProverTask new a chunk prover task
func NewChunkProverTask(cfg *config.Config, chainCfg *params.ChainConfig, db *gorm.DB, vk string, reg prometheus.Registerer) *ChunkProverTask {
	forkHeights, _, nameForkMap := forks.CollectSortedForkHeights(chainCfg)
	log.Info("new chunk prover task", "forkHeights", forkHeights, "nameForks", nameForkMap)
	cp := &ChunkProverTask{
		BaseProverTask: BaseProverTask{
			vk:                 vk,
			db:                 db,
			cfg:                cfg,
			nameForkMap:        nameForkMap,
			forkHeights:        forkHeights,
			chunkOrm:           orm.NewChunk(db),
			blockOrm:           orm.NewL2Block(db),
			proverTaskOrm:      orm.NewProverTask(db),
			proverBlockListOrm: orm.NewProverBlockList(db),
		},
		chunkAttemptsExceedTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "coordinator_chunk_attempts_exceed_total",
			Help: "Total number of chunk attempts exceed.",
		}),
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
	taskCtx, err := cp.checkParameter(ctx, getTaskParameter)
	if err != nil || taskCtx == nil {
		return nil, fmt.Errorf("check prover task parameter failed, error:%w", err)
	}

	hardForkNumber, err := cp.getHardForkNumberByName(getTaskParameter.HardForkName)
	if err != nil {
		log.Error("chunk assign failure because of the hard fork name don't exist", "fork name", getTaskParameter.HardForkName)
		return nil, err
	}

	fromBlockNum, toBlockNum := forks.BlockRange(hardForkNumber, cp.forkHeights)
	if toBlockNum > getTaskParameter.ProverHeight {
		toBlockNum = getTaskParameter.ProverHeight + 1
	}

	maxActiveAttempts := cp.cfg.ProverManager.ProversPerSession
	maxTotalAttempts := cp.cfg.ProverManager.SessionAttempts
	var chunkTask *orm.Chunk
	for i := 0; i < 5; i++ {
		var getTaskError error
		var tmpChunkTask *orm.Chunk
		tmpChunkTask, getTaskError = cp.chunkOrm.GetAssignedChunk(ctx, fromBlockNum, toBlockNum, maxActiveAttempts, maxTotalAttempts)
		if getTaskError != nil {
			log.Error("failed to get assigned chunk proving tasks", "height", getTaskParameter.ProverHeight, "err", getTaskError)
			return nil, ErrCoordinatorInternalFailure
		}

		// Why here need get again? In order to support a task can assign to multiple prover, need also assign `ProvingTaskAssigned`
		// chunk to prover. But use `proving_status in (1, 2)` will not use the postgres index. So need split the sql.
		if tmpChunkTask == nil {
			tmpChunkTask, getTaskError = cp.chunkOrm.GetUnassignedChunk(ctx, fromBlockNum, toBlockNum, maxActiveAttempts, maxTotalAttempts)
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

	taskMsg, err := cp.formatProverTask(ctx, &proverTask)
	if err != nil {
		cp.recoverActiveAttempts(ctx, chunkTask)
		log.Error("format prover task failure", "hash", chunkTask.Hash, "err", err)
		return nil, ErrCoordinatorInternalFailure
	}

	cp.chunkTaskGetTaskTotal.WithLabelValues(getTaskParameter.HardForkName).Inc()
	cp.chunkTaskGetTaskProver.With(prometheus.Labels{
		coordinatorType.LabelProverName:      proverTask.ProverName,
		coordinatorType.LabelProverPublicKey: proverTask.ProverPublicKey,
		coordinatorType.LabelProverVersion:   proverTask.ProverVersion,
	}).Inc()

	return taskMsg, nil
}

func (cp *ChunkProverTask) formatProverTask(ctx context.Context, task *orm.ProverTask) (*coordinatorType.GetTaskSchema, error) {
	// Get block hashes.
	blockHashes, dbErr := cp.blockOrm.GetL2BlockHashesByChunkHash(ctx, task.TaskID)
	if dbErr != nil || len(blockHashes) == 0 {
		return nil, fmt.Errorf("failed to fetch block hashes of a chunk, chunk hash:%s err:%w", task.TaskID, dbErr)
	}

	taskDetail := message.ChunkTaskDetail{
		BlockHashes: blockHashes,
	}
	blockHashesBytes, err := json.Marshal(taskDetail)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal block hashes hash:%s, err:%w", task.TaskID, err)
	}

	proverTaskSchema := &coordinatorType.GetTaskSchema{
		UUID:     task.UUID.String(),
		TaskID:   task.TaskID,
		TaskType: int(message.ProofTypeChunk),
		TaskData: string(blockHashesBytes),
	}

	return proverTaskSchema, nil
}

func (cp *ChunkProverTask) recoverActiveAttempts(ctx *gin.Context, chunkTask *orm.Chunk) {
	if err := cp.chunkOrm.DecreaseActiveAttemptsByHash(ctx, chunkTask.Hash); err != nil {
		log.Error("failed to recover chunk active attempts", "hash", chunkTask.Hash, "error", err)
	}
}
