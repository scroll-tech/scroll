package provertask

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/params"
	"gorm.io/gorm"

	"scroll-tech/common/types/message"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/logic/libzkp"
	"scroll-tech/coordinator/internal/orm"
	coordinatorType "scroll-tech/coordinator/internal/types"
)

var (
	// ErrCoordinatorInternalFailure coordinator internal db failure
	ErrCoordinatorInternalFailure = errors.New("coordinator internal error")
)

var (
	getTaskCounterInitOnce sync.Once
	getTaskCounterVec      *prometheus.CounterVec = nil
)

// ProverTask the interface of a collector who send data to prover
type ProverTask interface {
	Assign(ctx *gin.Context, getTaskParameter *coordinatorType.GetTaskParameter) (*coordinatorType.GetTaskSchema, error)
}

// BaseProverTask a base prover task which contain series functions
type BaseProverTask struct {
	cfg        *config.Config
	chainCfg   *params.ChainConfig
	db         *gorm.DB
	expectedVk map[string][]byte

	batchOrm           *orm.Batch
	chunkOrm           *orm.Chunk
	bundleOrm          *orm.Bundle
	blockOrm           *orm.L2Block
	proverTaskOrm      *orm.ProverTask
	proverBlockListOrm *orm.ProverBlockList
}

type proverTaskContext struct {
	PublicKey          string
	ProverName         string
	ProverVersion      string
	ProverProviderType uint8
	HardForkNames      map[string]struct{}

	taskType        message.ProofType
	chunkTask       *orm.Chunk
	batchTask       *orm.Batch
	bundleTask      *orm.Bundle
	hasAssignedTask *orm.ProverTask
}

// hardForkName get the chunk/batch/bundle hard fork name
func (b *BaseProverTask) hardForkName(ctx *gin.Context, taskCtx *proverTaskContext) (string, error) {
	switch {
	case taskCtx.taskType == message.ProofTypeChunk:
		if taskCtx.chunkTask == nil {
			return "", errors.New("chunk task is nil")
		}
		l2Block, getBlockErr := b.blockOrm.GetL2BlockByNumber(ctx.Copy(), taskCtx.chunkTask.StartBlockNumber)
		if getBlockErr != nil {
			return "", getBlockErr
		}
		hardForkName := encoding.GetHardforkName(b.chainCfg, l2Block.Number, l2Block.BlockTimestamp)
		return hardForkName, nil

	case taskCtx.taskType == message.ProofTypeBatch:
		if taskCtx.batchTask == nil {
			return "", errors.New("batch task is nil")
		}
		startChunk, getChunkErr := b.chunkOrm.GetChunkByHash(ctx, taskCtx.batchTask.StartChunkHash)
		if getChunkErr != nil {
			return "", getChunkErr
		}
		l2Block, getBlockErr := b.blockOrm.GetL2BlockByNumber(ctx.Copy(), startChunk.StartBlockNumber)
		if getBlockErr != nil {
			return "", getBlockErr
		}
		hardForkName := encoding.GetHardforkName(b.chainCfg, l2Block.Number, l2Block.BlockTimestamp)
		return hardForkName, nil

	case taskCtx.taskType == message.ProofTypeBundle:
		if taskCtx.bundleTask == nil {
			return "", errors.New("bundle task is nil")
		}
		startBatch, getBatchErr := b.batchOrm.GetBatchByHash(ctx, taskCtx.bundleTask.StartBatchHash)
		if getBatchErr != nil {
			return "", getBatchErr
		}
		startChunk, getChunkErr := b.chunkOrm.GetChunkByHash(ctx, startBatch.StartChunkHash)
		if getChunkErr != nil {
			return "", getChunkErr
		}
		l2Block, getBlockErr := b.blockOrm.GetL2BlockByNumber(ctx.Copy(), startChunk.StartBlockNumber)
		if getBlockErr != nil {
			return "", getBlockErr
		}
		hardForkName := encoding.GetHardforkName(b.chainCfg, l2Block.Number, l2Block.BlockTimestamp)
		return hardForkName, nil
	default:
		return "", errors.New("illegal task type")
	}
}

// hardForkSanityCheck check the task's hard fork name is the same as prover
func (b *BaseProverTask) hardForkSanityCheck(ctx *gin.Context, taskCtx *proverTaskContext) (string, error) {
	hardForkName, getHardForkErr := b.hardForkName(ctx, taskCtx)
	if getHardForkErr != nil {
		return "", getHardForkErr
	}

	if _, ok := taskCtx.HardForkNames[hardForkName]; !ok {
		return "", fmt.Errorf("to be assigned prover task's hard-fork name is not the same as prover, proverName: %s, proverVersion: %s, proverSupportHardForkNames: %s, taskHardForkName: %v", taskCtx.ProverName, taskCtx.ProverVersion, taskCtx.HardForkNames, hardForkName)
	}
	return hardForkName, nil
}

// checkParameter check the prover task parameter illegal
func (b *BaseProverTask) checkParameter(ctx *gin.Context) (*proverTaskContext, error) {
	var ptc proverTaskContext
	ptc.HardForkNames = make(map[string]struct{})

	publicKey, publicKeyExist := ctx.Get(coordinatorType.PublicKey)
	if !publicKeyExist {
		return nil, errors.New("get public key from context failed")
	}
	ptc.PublicKey = publicKey.(string)

	proverName, proverNameExist := ctx.Get(coordinatorType.ProverName)
	if !proverNameExist {
		return nil, errors.New("get prover name from context failed")
	}
	ptc.ProverName = proverName.(string)

	proverVersion, proverVersionExist := ctx.Get(coordinatorType.ProverVersion)
	if !proverVersionExist {
		return nil, errors.New("get prover version from context failed")
	}
	ptc.ProverVersion = proverVersion.(string)

	ProverProviderType, ProverProviderTypeExist := ctx.Get(coordinatorType.ProverProviderTypeKey)
	if !ProverProviderTypeExist {
		// for backward compatibility, set ProverProviderType as internal
		ProverProviderType = float64(coordinatorType.ProverProviderTypeInternal)
	}
	ptc.ProverProviderType = uint8(ProverProviderType.(float64))

	hardForkNamesStr, hardForkNameExist := ctx.Get(coordinatorType.HardForkName)
	if !hardForkNameExist {
		return nil, errors.New("get hard fork name from context failed")
	}
	hardForkNames := strings.Split(hardForkNamesStr.(string), ",")
	for _, hardForkName := range hardForkNames {
		ptc.HardForkNames[hardForkName] = struct{}{}
	}

	isBlocked, err := b.proverBlockListOrm.IsPublicKeyBlocked(ctx.Copy(), publicKey.(string))
	if err != nil {
		return nil, fmt.Errorf("failed to check whether the public key %s is blocked before assigning a chunk task, err: %w, proverName: %s, proverVersion: %s", publicKey, err, proverName, proverVersion)
	}
	if isBlocked {
		return nil, fmt.Errorf("public key %s is blocked from fetching tasks. ProverName: %s, ProverVersion: %s", publicKey, proverName, proverVersion)
	}

	assigned, err := b.proverTaskOrm.IsProverAssigned(ctx.Copy(), publicKey.(string))
	if err != nil {
		return nil, fmt.Errorf("failed to check if prover %s is assigned a task, err: %w", publicKey.(string), err)
	}

	ptc.hasAssignedTask = assigned
	return &ptc, nil
}

func (b *BaseProverTask) applyUniversal(schema *coordinatorType.GetTaskSchema) (*coordinatorType.GetTaskSchema, []byte, error) {
	expectedVk, ok := b.expectedVk[schema.HardForkName]
	if !ok {
		return nil, nil, fmt.Errorf("no expectedVk found from hardfork %s", schema.HardForkName)
	}

	ok, uTaskData, metadata, _ := libzkp.GenerateUniversalTask(schema.TaskType, schema.TaskData, schema.HardForkName, expectedVk)
	if !ok {
		return nil, nil, fmt.Errorf("can not generate universal task, see coordinator log for the reason")
	}

	schema.TaskData = uTaskData
	return schema, []byte(metadata), nil
}

func newGetTaskCounterVec(factory promauto.Factory, taskType string) *prometheus.CounterVec {
	getTaskCounterInitOnce.Do(func() {
		getTaskCounterVec = factory.NewCounterVec(prometheus.CounterOpts{
			Name: "coordinator_get_task_count",
			Help: "Multi dimensions get task counter.",
		}, []string{"task_type",
			coordinatorType.LabelProverName,
			coordinatorType.LabelProverPublicKey,
			coordinatorType.LabelProverVersion})
	})

	return getTaskCounterVec.MustCurryWith(prometheus.Labels{"task_type": taskType})
}
