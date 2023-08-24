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

// ErrCoordinatorInternalFailure coordinator internal db failure
var ErrCoordinatorInternalFailure = fmt.Errorf("coordinator internal error")

// ChunkProverTask the chunk prover task
type ChunkProverTask struct {
	BaseProverTask
	vk string

	chunkAttemptsExceedTotal prometheus.Counter
	chunkTaskGetTaskTotal    prometheus.Counter
}

// NewChunkProverTask new a chunk prover task
func NewChunkProverTask(cfg *config.Config, db *gorm.DB, vk string, reg prometheus.Registerer) *ChunkProverTask {
	cp := &ChunkProverTask{
		BaseProverTask: BaseProverTask{
			db:            db,
			cfg:           cfg,
			chunkOrm:      orm.NewChunk(db),
			blockOrm:      orm.NewL2Block(db),
			proverTaskOrm: orm.NewProverTask(db),
		},
		vk: vk,
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
	if getTaskParameter.VK == "" { // allow vk being empty, because for the first time the prover may not know its vk
		if !version.CheckScrollProverVersionTag(proverVersion.(string)) { // but reject too-old provers
			return nil, fmt.Errorf("incompatible prover version. please upgrade your prover, expect version: %s, actual version: %s", version.Version, proverVersion.(string))
		}
	} else if getTaskParameter.VK != cp.vk { // non-empty vk but different
		if version.CheckScrollProverVersion(proverVersion.(string)) { // same prover version but different vks
			return nil, fmt.Errorf("incompatible vk. please check your params files or config files")
		}
		// different prover versions and different vks
		return nil, fmt.Errorf("incompatible prover version. please upgrade your prover, expect version: %s, actual version: %s", version.Version, proverVersion.(string))
	}

	isAssigned, err := cp.proverTaskOrm.IsProverAssigned(ctx, publicKey.(string))
	if err != nil {
		return nil, fmt.Errorf("failed to check if prover is assigned a task: %w", err)
	}

	if isAssigned {
		return nil, fmt.Errorf("prover with publicKey %s is already assigned a task", publicKey)
	}

	// load and send chunk tasks
	chunkTasks, err := cp.chunkOrm.UpdateUnassignedChunkReturning(ctx, getTaskParameter.ProverHeight, 1)
	if err != nil {
		log.Error("failed to get unassigned chunk proving tasks", "height", getTaskParameter.ProverHeight, "err", err)
		return nil, ErrCoordinatorInternalFailure
	}

	if len(chunkTasks) == 0 {
		return nil, nil
	}

	if len(chunkTasks) != 1 {
		log.Error("get unassigned chunk proving task len not 1", "length", len(chunkTasks), "chunk tasks", chunkTasks)
		return nil, ErrCoordinatorInternalFailure
	}

	chunkTask := chunkTasks[0]

	log.Info("start chunk generation session", "id", chunkTask.Hash, "public key", publicKey, "prover name", proverName)

	if !cp.checkAttemptsExceeded(chunkTask.Hash, message.ProofTypeChunk) {
		cp.chunkAttemptsExceedTotal.Inc()
		// TODO: retry fetching unassigned chunk proving task
		log.Error("chunk task proving attempts reach the maximum", "hash", chunkTask.Hash)
		return nil, ErrCoordinatorInternalFailure
	}

	proverTask := orm.ProverTask{
		TaskID:          chunkTask.Hash,
		ProverPublicKey: publicKey.(string),
		TaskType:        int16(message.ProofTypeChunk),
		ProverName:      proverName.(string),
		ProverVersion:   proverVersion.(string),
		ProvingStatus:   int16(types.ProverAssigned),
		FailureType:     int16(types.ProverTaskFailureTypeUndefined),
		// here why need use UTC time. see scroll/common/databased/db.go
		AssignedAt: utils.NowUTC(),
	}
	if err = cp.proverTaskOrm.SetProverTask(ctx, &proverTask); err != nil {
		cp.recoverProvingStatus(ctx, chunkTask)
		log.Error("db set session info fail", "task hash", chunkTask.Hash, "prover name", proverName.(string), "prover pubKey", publicKey.(string), "err", err)
		return nil, ErrCoordinatorInternalFailure
	}

	taskMsg, err := cp.formatProverTask(ctx, chunkTask.Hash)
	if err != nil {
		cp.recoverProvingStatus(ctx, chunkTask)
		log.Error("format prover task failure", "hash", chunkTask.Hash, "err", err)
		return nil, ErrCoordinatorInternalFailure
	}

	cp.chunkTaskGetTaskTotal.Inc()

	return taskMsg, nil
}

func (cp *ChunkProverTask) formatProverTask(ctx context.Context, hash string) (*coordinatorType.GetTaskSchema, error) {
	// Get block hashes.
	wrappedBlocks, wrappedErr := cp.blockOrm.GetL2BlocksByChunkHash(ctx, hash)
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

// recoverProvingStatus if not return the batch task to prover success,
// need recover the proving status to unassigned
func (cp *ChunkProverTask) recoverProvingStatus(ctx *gin.Context, chunkTask *orm.Chunk) {
	if err := cp.chunkOrm.UpdateProvingStatus(ctx, chunkTask.Hash, types.ProvingTaskUnassigned); err != nil {
		log.Warn("failed to recover chunk proving status", "hash:", chunkTask.Hash, "error", err)
	}
}
