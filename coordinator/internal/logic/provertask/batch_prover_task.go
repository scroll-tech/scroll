package provertask

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
	"scroll-tech/common/utils"
	"scroll-tech/common/version"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/orm"
	coordinatorType "scroll-tech/coordinator/internal/types"
)

// BatchProverTask is prover task implement for batch proof
type BatchProverTask struct {
	BaseProverTask

	batchAttemptsExceedTotal prometheus.Counter
	batchTaskGetTaskTotal    prometheus.Counter
}

// NewBatchProverTask new a batch collector
func NewBatchProverTask(cfg *config.Config, db *gorm.DB, reg prometheus.Registerer) *BatchProverTask {
	bp := &BatchProverTask{
		BaseProverTask: BaseProverTask{
			db:            db,
			cfg:           cfg,
			chunkOrm:      orm.NewChunk(db),
			batchOrm:      orm.NewBatch(db),
			proverTaskOrm: orm.NewProverTask(db),
		},
		batchAttemptsExceedTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "coordinator_batch_attempts_exceed_total",
			Help: "Total number of batch attempts exceed.",
		}),
		batchTaskGetTaskTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "coordinator_batch_get_task_total",
			Help: "Total number of batch get task.",
		}),
	}
	return bp
}

// Assign load and assign batch tasks
func (bp *BatchProverTask) Assign(ctx *gin.Context, getTaskParameter *coordinatorType.GetTaskParameter) (*coordinatorType.GetTaskSchema, error) {
	publicKey, publicKeyExist := ctx.Get(coordinatorType.PublicKey)
	if !publicKeyExist {
		return nil, fmt.Errorf("get public key from context failed")
	}

	proverName, proverNameExist := ctx.Get(coordinatorType.ProverName)
	if !proverNameExist {
		return nil, fmt.Errorf("get prover name from context failed")
	}

	proverVersion, proverVersionExist := ctx.Get(coordinatorType.ProverVersion)
	if !proverVersionExist {
		return nil, fmt.Errorf("get prover version from context failed")
	}
	if !version.CheckScrollProverVersion(proverVersion.(string)) {
		return nil, fmt.Errorf("incompatible prover version. please upgrade your prover, expect version: %s, actual version: %s", proverVersion.(string), version.Version)
	}

	isAssigned, err := bp.proverTaskOrm.IsProverAssigned(ctx, publicKey.(string))
	if err != nil {
		return nil, fmt.Errorf("failed to check if prover is assigned a task: %w", err)
	}

	if isAssigned {
		return nil, fmt.Errorf("prover with publicKey %s is already assigned a task", publicKey)
	}

	batchTasks, err := bp.batchOrm.UpdateUnassignedBatchReturning(ctx, 1)
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
	log.Info("start batch proof generation session", "id", batchTask.Hash, "public key", publicKey, "prover name", proverName)

	if !bp.checkAttemptsExceeded(batchTask.Hash, message.ProofTypeBatch) {
		bp.batchAttemptsExceedTotal.Inc()
		return nil, fmt.Errorf("the batch task id:%s check attempts have reach the maximum", batchTask.Hash)
	}

	proverTask := orm.ProverTask{
		TaskID:          batchTask.Hash,
		ProverPublicKey: publicKey.(string),
		TaskType:        int16(message.ProofTypeBatch),
		ProverName:      proverName.(string),
		ProverVersion:   proverVersion.(string),
		ProvingStatus:   int16(types.ProverAssigned),
		FailureType:     int16(types.ProverTaskFailureTypeUndefined),
		// here why need use UTC time. see scroll/common/databased/db.go
		AssignedAt: utils.NowUTC(),
	}

	// Store session info.
	if err = bp.proverTaskOrm.SetProverTask(ctx, &proverTask); err != nil {
		bp.recoverProvingStatus(ctx, batchTask)
		return nil, fmt.Errorf("db set session info fail, session id:%s, error:%w", proverTask.TaskID, err)
	}

	taskMsg, err := bp.formatProverTask(ctx, batchTask.Hash)
	if err != nil {
		bp.recoverProvingStatus(ctx, batchTask)
		return nil, fmt.Errorf("format prover failure, id:%s error:%w", batchTask.Hash, err)
	}

	bp.batchTaskGetTaskTotal.Inc()

	return taskMsg, nil
}

func (bp *BatchProverTask) formatProverTask(ctx context.Context, taskID string) (*coordinatorType.GetTaskSchema, error) {
	// get chunk from db
	chunks, err := bp.chunkOrm.GetChunksByBatchHash(ctx, taskID)
	if err != nil {
		err = fmt.Errorf("failed to get chunk proofs for batch task id:%s err:%w ", taskID, err)
		return nil, err
	}

	var chunkProofs []*message.ChunkProof
	var chunkInfos []*message.ChunkInfo
	for _, chunk := range chunks {
		var proof message.ChunkProof
		if encodeErr := json.Unmarshal(chunk.Proof, &proof); encodeErr != nil {
			return nil, fmt.Errorf("Chunk.GetProofsByBatchHash unmarshal proof error: %w, batch hash: %v, chunk hash: %v", encodeErr, taskID, chunk.Hash)
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
		chunkInfos = append(chunkInfos, &chunkInfo)
	}

	taskDetail := message.BatchTaskDetail{
		ChunkInfos:  chunkInfos,
		ChunkProofs: chunkProofs,
	}

	chunkProofsBytes, err := json.Marshal(taskDetail)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal chunk proofs, taskID:%s err:%w", taskID, err)
	}

	taskMsg := &coordinatorType.GetTaskSchema{
		TaskID:   taskID,
		TaskType: int(message.ProofTypeBatch),
		TaskData: string(chunkProofsBytes),
	}
	return taskMsg, nil
}

// recoverProvingStatus if not return the batch task to prover success,
// need recover the proving status to unassigned
func (bp *BatchProverTask) recoverProvingStatus(ctx *gin.Context, batchTask *orm.Batch) {
	if err := bp.batchOrm.UpdateProvingStatus(ctx, batchTask.Hash, types.ProvingTaskUnassigned); err != nil {
		log.Warn("failed to recover batch proving status", "hash:", batchTask.Hash, "error", err)
	}
}
