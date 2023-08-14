package provertask

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

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

func (bp *BatchProverTask) selectAndSetAvailableBatch(ctx context.Context, publicKey string,
	proverName string, proverVersion string, dbTX *gorm.DB) (*orm.ProverTask, error) {
	// TODO: add a transaction lock.

	var taskIDs []string
	dbProver := dbTX.WithContext(ctx)
	dbProver = dbProver.Table("prover_task")
	dbProver = dbProver.Select("task_id")
	dbProver = dbProver.Group("task_id")
	dbProver = dbProver.Having("COUNT(task_id) >= ? OR COUNT(CASE WHEN prover_task.proving_status = ? THEN 1 ELSE NULL END) > ?",
		bp.cfg.ProverManager.SessionAttempts, types.ProverAssigned, bp.cfg.ProverManager.ProversPerSession)
	if err := dbProver.Find(&taskIDs).Error; err != nil {
		dbTX.Rollback()
		return nil, fmt.Errorf("select unavailable prover task error: %w", err)
	}

	var batch orm.Batch
	dbBatch := dbTX.WithContext(ctx)
	dbBatch = dbBatch.Table("batch")
	if len(taskIDs) > 0 {
		dbBatch = dbBatch.Where("hash NOT IN ?", taskIDs)
	}
	dbBatch = dbBatch.Where("proving_status != ?", types.ProvingTaskVerified)
	dbBatch = dbBatch.Where("proving_status != ?", types.ProvingTaskFailed)
	dbBatch = dbBatch.Where("chunk_proofs_status = ?", types.ChunkProofsStatusReady)
	dbBatch = dbBatch.Order("index ASC")

	if err := dbBatch.First(&batch).Error; err != nil {
		dbTX.Rollback()
		return nil, fmt.Errorf("select available chunk error: %w", err)
	}

	proverTask := &orm.ProverTask{
		TaskID:          batch.Hash,
		ProverPublicKey: publicKey,
		TaskType:        int16(message.ProofTypeBatch),
		ProverName:      proverName,
		ProverVersion:   proverVersion,
		ProvingStatus:   int16(types.ProverAssigned),
		FailureType:     int16(types.ProverTaskFailureTypeUndefined),
		AssignedAt:      utils.NowUTC(),
	}

	// set prover task
	dbProver = dbTX.Model(&orm.ProverTask{})
	dbProver = dbProver.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "task_type"}, {Name: "task_id"}, {Name: "prover_public_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"prover_version", "proving_status", "failure_type", "assigned_at"}),
	})

	if err := dbProver.Create(proverTask).Error; err != nil {
		dbTX.Rollback()
		return nil, fmt.Errorf("set prover task failed: %w, prover task: %v", err, proverTask)
	}
	return proverTask, nil
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

	dbTX := bp.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			dbTX.Rollback()
		}
	}()

	// Select and set batch tasks
	proverTask, err := bp.selectAndSetAvailableBatch(
		ctx, publicKey.(string), proverName.(string), proverVersion.(string), dbTX)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		dbTX.Rollback()
		return nil, fmt.Errorf("failed to select and set available batch: %w", err)
	}

	log.Info("start batch proof generation session",
		"id", proverTask.TaskID,
		"public key", publicKey,
		"prover name", proverName)

	taskMsg, err := bp.formatProverTask(ctx, proverTask.TaskID, dbTX)
	if err != nil {
		dbTX.Rollback()
		return nil, fmt.Errorf("failed to format prover task, ID: %v, err: %v", proverTask.TaskID, err)
	}

	bp.batchTaskGetTaskTotal.Inc()
	if err := dbTX.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit db change: %w", err)
	}

	return taskMsg, nil
}

func (bp *BatchProverTask) formatProverTask(ctx context.Context, taskID string, dbTX *gorm.DB) (*coordinatorType.GetTaskSchema, error) {
	// get chunk from db
	chunks, err := bp.chunkOrm.GetChunksByBatchHash(ctx, taskID, dbTX)
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
