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

// ChunkProverTask the chunk prover task
type ChunkProverTask struct {
	BaseProverTask

	chunkAttemptsExceedTotal prometheus.Counter
	chunkTaskGetTaskTotal    prometheus.Counter
}

// NewChunkProverTask new a chunk prover task
func NewChunkProverTask(cfg *config.Config, db *gorm.DB, reg prometheus.Registerer) *ChunkProverTask {
	cp := &ChunkProverTask{
		BaseProverTask: BaseProverTask{
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

func (cp *ChunkProverTask) selectAndSetAvailableChunk(ctx context.Context, proverHeight int,
	publicKey string, proverName string, proverVersion string, dbTX *gorm.DB) (*orm.ProverTask, error) {
	// TODO: add a transaction lock.

	var taskIDs []string
	dbProver := dbTX.WithContext(ctx)
	dbProver = dbProver.Table("prover_task")
	dbProver = dbProver.Select("task_id")
	dbProver = dbProver.Group("task_id")
	dbProver = dbProver.Having("COUNT(task_id) >= ? OR COUNT(CASE WHEN prover_task.proving_status = ? THEN 1 ELSE NULL END) > ?",
		cp.cfg.ProverManager.SessionAttempts, types.ProverAssigned, cp.cfg.ProverManager.ProversPerSession)
	if err := dbProver.Find(&taskIDs).Error; err != nil {
		dbTX.Rollback()
		return nil, fmt.Errorf("select unavailable prover task error: %w", err)
	}

	var chunk orm.Chunk
	dbChunk := dbTX.WithContext(ctx)
	dbChunk = dbChunk.Table("chunk")
	if len(taskIDs) > 0 {
		dbChunk = dbChunk.Where("hash NOT IN ?", taskIDs)
	}
	dbChunk = dbChunk.Where("proving_status != ?", types.ProvingTaskVerified)
	dbChunk = dbChunk.Where("proving_status != ?", types.ProvingTaskFailed)
	dbChunk = dbChunk.Where("end_block_number <= ?", proverHeight)
	dbChunk = dbChunk.Order("index ASC")

	if err := dbChunk.First(&chunk).Error; err != nil {
		dbTX.Rollback()
		return nil, fmt.Errorf("select available chunk error: %w", err)
	}

	proverTask := &orm.ProverTask{
		TaskID:          chunk.Hash,
		ProverPublicKey: publicKey,
		TaskType:        int16(message.ProofTypeChunk),
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

// Assign the chunk proof which need to prove
func (cp *ChunkProverTask) Assign(ctx *gin.Context, getTaskParameter *coordinatorType.GetTaskParameter) (*coordinatorType.GetTaskSchema, error) {
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
		return nil, fmt.Errorf("incompatible prover version. please upgrade your prover, expect version: %s, actual version: %s",
			version.Version, proverVersion.(string))
	}

	isAssigned, err := cp.proverTaskOrm.IsProverAssigned(ctx, publicKey.(string))
	if err != nil {
		return nil, fmt.Errorf("failed to check if prover is assigned a task: %w", err)
	}

	if isAssigned {
		return nil, fmt.Errorf("prover with publicKey %s is already assigned a task", publicKey)
	}

	dbTX := cp.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			dbTX.Rollback()
		}
	}()

	// select and set chunk tasks
	proverTask, err := cp.selectAndSetAvailableChunk(
		ctx, getTaskParameter.ProverHeight, publicKey.(string),
		proverName.(string), proverVersion.(string), dbTX)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		dbTX.Rollback()
		return nil, fmt.Errorf("failed to select and set available batch: %w", err)
	}

	log.Info("start chunk generation session",
		"id", proverTask.TaskID,
		"public key", publicKey,
		"prover name", proverName)

	taskMsg, err := cp.formatProverTask(ctx, proverTask.TaskID, dbTX)
	if err != nil {
		dbTX.Rollback()
		return nil, fmt.Errorf("failed to format prover task, ID: %v, err: %v", proverTask.TaskID, err)
	}

	cp.chunkTaskGetTaskTotal.Inc()
	if err := dbTX.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit db change: %w", err)
	}

	return taskMsg, nil
}

func (cp *ChunkProverTask) formatProverTask(ctx context.Context, hash string, dbTX *gorm.DB) (*coordinatorType.GetTaskSchema, error) {
	// Get block hashes.
	wrappedBlocks, wrappedErr := cp.blockOrm.GetL2BlocksByChunkHash(ctx, hash, dbTX)
	if wrappedErr != nil || len(wrappedBlocks) == 0 {
		return nil, fmt.Errorf("failed to fetch wrapped blocks, batch hash:%s err:%w", hash, wrappedErr)
	}

	blockHashes := make([]common.Hash, len(wrappedBlocks))
	for i, wrappedBlock := range wrappedBlocks {
		blockHashes[i] = wrappedBlock.Header.Hash()
	}

	taskDetail := message.ChunkTaskDetail{
		BlockHashes: blockHashes,
	}
	blockHashesBytes, err := json.Marshal(taskDetail)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal block hashes hash:%s, err:%w", hash, err)
	}

	proverTaskSchema := &coordinatorType.GetTaskSchema{
		TaskID:   hash,
		TaskType: int(message.ProofTypeChunk),
		TaskData: string(blockHashesBytes),
	}

	return proverTaskSchema, nil
}
