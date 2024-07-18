package provertask

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/scroll-tech/go-ethereum/common"
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

// BatchProverTask is prover task implement for batch proof
type BatchProverTask struct {
	BaseProverTask

	batchAttemptsExceedTotal prometheus.Counter
	batchTaskGetTaskTotal    *prometheus.CounterVec
	batchTaskGetTaskProver   *prometheus.CounterVec
}

// NewBatchProverTask new a batch collector
func NewBatchProverTask(cfg *config.Config, chainCfg *params.ChainConfig, db *gorm.DB, vkMap map[string]string, reg prometheus.Registerer) *BatchProverTask {
	forkHeights, _, nameForkMap := forks.CollectSortedForkHeights(chainCfg)
	log.Info("new batch prover task", "forkHeights", forkHeights, "nameForks", nameForkMap)

	bp := &BatchProverTask{
		BaseProverTask: BaseProverTask{
			vkMap:              vkMap,
			reverseVkMap:       reverseMap(vkMap),
			db:                 db,
			cfg:                cfg,
			nameForkMap:        nameForkMap,
			forkHeights:        forkHeights,
			chunkOrm:           orm.NewChunk(db),
			batchOrm:           orm.NewBatch(db),
			proverTaskOrm:      orm.NewProverTask(db),
			proverBlockListOrm: orm.NewProverBlockList(db),
		},
		batchAttemptsExceedTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "coordinator_batch_attempts_exceed_total",
			Help: "Total number of batch attempts exceed.",
		}),
		batchTaskGetTaskTotal: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Name: "coordinator_batch_get_task_total",
			Help: "Total number of batch get task.",
		}, []string{"fork_name"}),
		batchTaskGetTaskProver: newGetTaskCounterVec(promauto.With(reg), "batch"),
	}
	return bp
}

type chunkIndexRange struct {
	start uint64
	end   uint64
}

func (r *chunkIndexRange) merge(o chunkIndexRange) *chunkIndexRange {
	var start, end = r.start, r.end
	if o.start < r.start {
		start = o.start
	}
	if o.end > r.end {
		end = o.end
	}
	return &chunkIndexRange{start, end}
}

func (r *chunkIndexRange) contains(start, end uint64) bool {
	return r.start <= start && r.end > end
}

type getHardForkNameByBatchFunc func(*orm.Batch) (string, error)

func (bp *BatchProverTask) doAssignTaskWithinChunkRange(ctx *gin.Context, taskCtx *proverTaskContext,
	chunkRange *chunkIndexRange, getTaskParameter *coordinatorType.GetTaskParameter, getHardForkName getHardForkNameByBatchFunc) (*coordinatorType.GetTaskSchema, error) {
	startChunkIndex, endChunkIndex := chunkRange.start, chunkRange.end
	maxActiveAttempts := bp.cfg.ProverManager.ProversPerSession
	maxTotalAttempts := bp.cfg.ProverManager.SessionAttempts
	var batchTask *orm.Batch
	for i := 0; i < 5; i++ {
		var getTaskError error
		var tmpBatchTask *orm.Batch
		tmpBatchTask, getTaskError = bp.batchOrm.GetAssignedBatch(ctx.Copy(), startChunkIndex, endChunkIndex, maxActiveAttempts, maxTotalAttempts)
		if getTaskError != nil {
			log.Error("failed to get assigned batch proving tasks", "height", getTaskParameter.ProverHeight, "err", getTaskError)
			return nil, ErrCoordinatorInternalFailure
		}

		// Why here need get again? In order to support a task can assign to multiple prover, need also assign `ProvingTaskAssigned`
		// batch to prover. But use `proving_status in (1, 2)` will not use the postgres index. So need split the sql.
		if tmpBatchTask == nil {
			tmpBatchTask, getTaskError = bp.batchOrm.GetUnassignedBatch(ctx.Copy(), startChunkIndex, endChunkIndex, maxActiveAttempts, maxTotalAttempts)
			if getTaskError != nil {
				log.Error("failed to get unassigned batch proving tasks", "height", getTaskParameter.ProverHeight, "err", getTaskError)
				return nil, ErrCoordinatorInternalFailure
			}
		}

		if tmpBatchTask == nil {
			log.Debug("get empty batch", "height", getTaskParameter.ProverHeight)
			return nil, nil
		}

		rowsAffected, updateAttemptsErr := bp.batchOrm.UpdateBatchAttempts(ctx.Copy(), tmpBatchTask.Index, tmpBatchTask.ActiveAttempts, tmpBatchTask.TotalAttempts)
		if updateAttemptsErr != nil {
			log.Error("failed to update batch attempts", "height", getTaskParameter.ProverHeight, "err", updateAttemptsErr)
			return nil, ErrCoordinatorInternalFailure
		}

		if rowsAffected == 0 {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		batchTask = tmpBatchTask
		break
	}

	if batchTask == nil {
		log.Debug("get empty unassigned batch after retry 5 times", "height", getTaskParameter.ProverHeight)
		return nil, nil
	}

	log.Info("start batch proof generation session", "task_id", batchTask.Hash, "public key", taskCtx.PublicKey, "prover name", taskCtx.ProverName)
	var (
		proverVersion = taskCtx.ProverVersion
		hardForkName  = taskCtx.HardForkName
	)
	var err error
	if getHardForkName != nil {
		hardForkName, err = getHardForkName(batchTask)
		if err != nil {
			log.Error("failed to get hard fork name by batch", "task_id", batchTask.Hash, "error", err.Error())
			return nil, ErrCoordinatorInternalFailure
		}
	}

	proverTask := orm.ProverTask{
		TaskID:          batchTask.Hash,
		ProverPublicKey: taskCtx.PublicKey,
		TaskType:        int16(message.ProofTypeBatch),
		ProverName:      taskCtx.ProverName,
		ProverVersion:   proverVersion,
		ProvingStatus:   int16(types.ProverAssigned),
		FailureType:     int16(types.ProverTaskFailureTypeUndefined),
		// here why need use UTC time. see scroll/common/databased/db.go
		AssignedAt: utils.NowUTC(),
	}

	// Store session info.
	if err = bp.proverTaskOrm.InsertProverTask(ctx.Copy(), &proverTask); err != nil {
		bp.recoverActiveAttempts(ctx, batchTask)
		log.Error("insert batch prover task info fail", "task_id", batchTask.Hash, "publicKey", taskCtx.PublicKey, "err", err)
		return nil, ErrCoordinatorInternalFailure
	}

	taskMsg, err := bp.formatProverTask(ctx.Copy(), &proverTask)
	if err != nil {
		bp.recoverActiveAttempts(ctx, batchTask)
		log.Error("format prover task failure", "task_id", batchTask.Hash, "err", err)
		return nil, ErrCoordinatorInternalFailure
	}

	bp.batchTaskGetTaskTotal.WithLabelValues(hardForkName).Inc()
	bp.batchTaskGetTaskProver.With(prometheus.Labels{
		coordinatorType.LabelProverName:      proverTask.ProverName,
		coordinatorType.LabelProverPublicKey: proverTask.ProverPublicKey,
		coordinatorType.LabelProverVersion:   proverTask.ProverVersion,
	}).Inc()

	return taskMsg, nil
}

func (bp *BatchProverTask) getChunkRangeByName(ctx *gin.Context, hardForkName string) (*chunkIndexRange, error) {
	hardForkNumber, err := bp.getHardForkNumberByName(hardForkName)
	if err != nil {
		// log.Error("batch assign failure because of the hard fork name don't exist", "fork name", hardForkName)
		return nil, err
	}

	// if the hard fork number set, rollup relayer must generate the chunk from hard fork number,
	// so the hard fork chunk's start_block_number must be ForkBlockNumber
	var startChunkIndex uint64 = 0
	var endChunkIndex uint64 = math.MaxInt64
	fromBlockNum, toBlockNum := forks.BlockRange(hardForkNumber, bp.forkHeights)
	if fromBlockNum != 0 {
		startChunk, chunkErr := bp.chunkOrm.GetChunkByStartBlockNumber(ctx.Copy(), fromBlockNum)
		if chunkErr != nil {
			log.Error("failed to get fork start chunk index", "forkName", hardForkName, "fromBlockNumber", fromBlockNum, "err", chunkErr)
			return nil, ErrCoordinatorInternalFailure
		}
		if startChunk == nil {
			return nil, nil
		}
		startChunkIndex = startChunk.Index
	}
	if toBlockNum != math.MaxInt64 {
		toChunk, chunkErr := bp.chunkOrm.GetChunkByStartBlockNumber(ctx.Copy(), toBlockNum)
		if chunkErr != nil {
			log.Error("failed to get fork end chunk index", "forkName", hardForkName, "toBlockNumber", toBlockNum, "err", chunkErr)
			return nil, ErrCoordinatorInternalFailure
		}
		if toChunk != nil {
			// toChunk being nil only indicates that we haven't yet reached the fork boundary
			// don't need change the endChunkIndex of math.MaxInt64
			endChunkIndex = toChunk.Index
		}
	}
	return &chunkIndexRange{startChunkIndex, endChunkIndex}, nil
}

func (bp *BatchProverTask) assignWithSingleCircuit(ctx *gin.Context, taskCtx *proverTaskContext, getTaskParameter *coordinatorType.GetTaskParameter) (*coordinatorType.GetTaskSchema, error) {
	chunkRange, err := bp.getChunkRangeByName(ctx, taskCtx.HardForkName)
	if err != nil {
		return nil, err
	}
	if chunkRange == nil {
		return nil, nil
	}
	return bp.doAssignTaskWithinChunkRange(ctx, taskCtx, chunkRange, getTaskParameter, nil)
}

func (bp *BatchProverTask) assignWithTwoCircuits(ctx *gin.Context, taskCtx *proverTaskContext, getTaskParameter *coordinatorType.GetTaskParameter) (*coordinatorType.GetTaskSchema, error) {
	var (
		hardForkNames [2]string
		chunkRanges   [2]*chunkIndexRange
		err           error
	)
	var chunkRange *chunkIndexRange
	for i := 0; i < 2; i++ {
		hardForkNames[i] = bp.reverseVkMap[getTaskParameter.VKs[i]]
		chunkRanges[i], err = bp.getChunkRangeByName(ctx, hardForkNames[i])
		if err == nil && chunkRanges[i] != nil {
			if chunkRange == nil {
				chunkRange = chunkRanges[i]
			} else {
				chunkRange = chunkRange.merge(*chunkRanges[i])
			}
		}
	}
	if chunkRange == nil {
		log.Error("chunkRange empty")
		return nil, errors.New("chunkRange empty")
	}
	var hardForkName string
	getHardForkName := func(batch *orm.Batch) (string, error) {
		for i := 0; i < 2; i++ {
			if chunkRanges[i] != nil && chunkRanges[i].contains(batch.StartChunkIndex, batch.EndChunkIndex) {
				hardForkName = hardForkNames[i]
				break
			}
		}
		if hardForkName == "" {
			log.Warn("get batch not belongs to any hard fork name", "batch id", batch.Index)
			return "", fmt.Errorf("get batch not belongs to any hard fork name, batch id: %d", batch.Index)
		}
		return hardForkName, nil
	}
	schema, err := bp.doAssignTaskWithinChunkRange(ctx, taskCtx, chunkRange, getTaskParameter, getHardForkName)
	if schema != nil && err == nil {
		schema.HardForkName = hardForkName
		return schema, nil
	}
	return schema, err
}

// Assign load and assign batch tasks
func (bp *BatchProverTask) Assign(ctx *gin.Context, getTaskParameter *coordinatorType.GetTaskParameter) (*coordinatorType.GetTaskSchema, error) {
	taskCtx, err := bp.checkParameter(ctx, getTaskParameter)
	if err != nil || taskCtx == nil {
		return nil, fmt.Errorf("check prover task parameter failed, error:%w", err)
	}

	if len(getTaskParameter.VKs) > 0 {
		return bp.assignWithTwoCircuits(ctx, taskCtx, getTaskParameter)
	}
	return bp.assignWithSingleCircuit(ctx, taskCtx, getTaskParameter)
}

func (bp *BatchProverTask) formatProverTask(ctx context.Context, task *orm.ProverTask) (*coordinatorType.GetTaskSchema, error) {
	// get chunk from db
	chunks, err := bp.chunkOrm.GetChunksByBatchHash(ctx, task.TaskID)
	if err != nil {
		err = fmt.Errorf("failed to get chunk proofs for batch task id:%s err:%w ", task.TaskID, err)
		return nil, err
	}

	var chunkProofs []*message.ChunkProof
	var chunkInfos []*message.ChunkInfo
	for _, chunk := range chunks {
		var proof message.ChunkProof
		if encodeErr := json.Unmarshal(chunk.Proof, &proof); encodeErr != nil {
			return nil, fmt.Errorf("Chunk.GetProofsByBatchHash unmarshal proof error: %w, batch hash: %v, chunk hash: %v", encodeErr, task.TaskID, chunk.Hash)
		}
		chunkProofs = append(chunkProofs, &proof)

		chunkInfo := message.ChunkInfo{
			ChainID:       bp.cfg.L2.ChainID,
			PrevStateRoot: common.HexToHash(chunk.ParentChunkStateRoot),
			PostStateRoot: common.HexToHash(chunk.StateRoot),
			WithdrawRoot:  common.HexToHash(chunk.WithdrawRoot),
			DataHash:      common.HexToHash(chunk.Hash),
			IsPadding:     false,
		}
		if proof.ChunkInfo != nil {
			chunkInfo.TxBytes = proof.ChunkInfo.TxBytes
		}
		chunkInfos = append(chunkInfos, &chunkInfo)
	}

	taskDetail := message.BatchTaskDetail{
		ChunkInfos:  chunkInfos,
		ChunkProofs: chunkProofs,
	}

	chunkProofsBytes, err := json.Marshal(taskDetail)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal chunk proofs, taskID:%s err:%w", task.TaskID, err)
	}

	taskMsg := &coordinatorType.GetTaskSchema{
		UUID:     task.UUID.String(),
		TaskID:   task.TaskID,
		TaskType: int(message.ProofTypeBatch),
		TaskData: string(chunkProofsBytes),
	}
	return taskMsg, nil
}

func (bp *BatchProverTask) recoverActiveAttempts(ctx *gin.Context, batchTask *orm.Batch) {
	if err := bp.chunkOrm.DecreaseActiveAttemptsByHash(ctx.Copy(), batchTask.Hash); err != nil {
		log.Error("failed to recover batch active attempts", "hash", batchTask.Hash, "error", err)
	}
}
