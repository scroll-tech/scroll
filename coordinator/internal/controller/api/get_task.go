package api

import (
	"fmt"
	"math/rand"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/scroll-tech/go-ethereum/params"
	"gorm.io/gorm"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/logic/provertask"
	"scroll-tech/coordinator/internal/logic/verifier"
	coordinatorType "scroll-tech/coordinator/internal/types"
)

// GetTaskController the get prover task api controller
type GetTaskController struct {
	proverTasks map[message.ProofType]provertask.ProverTask
}

// NewGetTaskController create a get prover task controller
func NewGetTaskController(cfg *config.Config, chainCfg *params.ChainConfig, db *gorm.DB, vf *verifier.Verifier, reg prometheus.Registerer) *GetTaskController {
	chunkProverTask := provertask.NewChunkProverTask(cfg, chainCfg, db, vf.ChunkVK, reg)
	batchProverTask := provertask.NewBatchProverTask(cfg, chainCfg, db, vf.BatchVK, reg)

	ptc := &GetTaskController{
		proverTasks: make(map[message.ProofType]provertask.ProverTask),
	}

	ptc.proverTasks[message.ProofTypeChunk] = chunkProverTask
	ptc.proverTasks[message.ProofTypeBatch] = batchProverTask

	return ptc
}

// GetTasks get assigned chunk/batch task
func (ptc *GetTaskController) GetTasks(ctx *gin.Context) {
	var getTaskParameter coordinatorType.GetTaskParameter
	if err := ctx.ShouldBind(&getTaskParameter); err != nil {
		nerr := fmt.Errorf("prover task parameter invalid, err:%w", err)
		types.RenderFailure(ctx, types.ErrCoordinatorParameterInvalidNo, nerr)
		return
	}

	proofType := ptc.proofType(&getTaskParameter)
	proverTask, isExist := ptc.proverTasks[proofType]
	if !isExist {
		nerr := fmt.Errorf("parameter wrong proof type:%v", proofType)
		types.RenderFailure(ctx, types.ErrCoordinatorParameterInvalidNo, nerr)
		return
	}

	result, err := proverTask.Assign(ctx, &getTaskParameter)
	if err != nil {
		nerr := fmt.Errorf("return prover task err:%w", err)
		types.RenderFailure(ctx, types.ErrCoordinatorGetTaskFailure, nerr)
		return
	}

	if result == nil {
		nerr := fmt.Errorf("get empty prover task")
		types.RenderFailure(ctx, types.ErrCoordinatorEmptyProofData, nerr)
		return
	}

	types.RenderSuccess(ctx, result)
}

func (ptc *GetTaskController) proofType(para *coordinatorType.GetTaskParameter) message.ProofType {
	proofType := message.ProofType(para.TaskType)

	proofTypes := []message.ProofType{
		message.ProofTypeChunk,
		message.ProofTypeBatch,
	}

	if proofType == message.ProofTypeUndefined {
		rand.Shuffle(len(proofTypes), func(i, j int) {
			proofTypes[i], proofTypes[j] = proofTypes[j], proofTypes[i]
		})
		proofType = proofTypes[0]
	}
	return proofType
}
