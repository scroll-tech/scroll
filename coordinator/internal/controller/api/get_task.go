package api

import (
	"fmt"
	"math/rand"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"gorm.io/gorm"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/logic/provertask"
	coordinatorType "scroll-tech/coordinator/internal/types"
)

// GetTaskController the get prover task api controller
type GetTaskController struct {
	proverTasks map[message.ProofType]provertask.ProverTask
}

// NewGetTaskController create a get prover task controller
func NewGetTaskController(cfg *config.Config, db *gorm.DB, reg prometheus.Registerer) *GetTaskController {
	chunkProverTask := provertask.NewChunkProverTask(cfg, db, reg)
	batchProverTask := provertask.NewBatchProverTask(cfg, db, reg)

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
		nerr := fmt.Errorf("prover tasks parameter invalid, err:%w", err)
		coordinatorType.RenderJSON(ctx, types.ErrCoordinatorParameterInvalidNo, nerr, nil)
		return
	}

	proofType := ptc.proofType(&getTaskParameter)
	proverTask, isExist := ptc.proverTasks[proofType]
	if !isExist {
		nerr := fmt.Errorf("parameter wrong proof type")
		coordinatorType.RenderJSON(ctx, types.ErrCoordinatorParameterInvalidNo, nerr, nil)
		return
	}

	result, err := proverTask.Assign(ctx, &getTaskParameter)
	if err != nil {
		nerr := fmt.Errorf("return prover task err:%w", err)
		coordinatorType.RenderJSON(ctx, types.ErrCoordinatorGetTaskFailure, nerr, nil)
		return
	}

	if result == nil {
		nerr := fmt.Errorf("get empty prover task")
		coordinatorType.RenderJSON(ctx, types.ErrCoordinatorEmptyProofData, nerr, nil)
		return
	}

	coordinatorType.RenderJSON(ctx, types.Success, nil, result)
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
