package proxy

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/scroll-tech/go-ethereum/params"
	"gorm.io/gorm"

	"scroll-tech/common/types/message"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/logic/provertask"
	"scroll-tech/coordinator/internal/logic/verifier"
	coordinatorType "scroll-tech/coordinator/internal/types"
)

// GetTaskController the get prover task api controller
type GetTaskController struct {
	proverTasks map[message.ProofType]provertask.ProverTask

	getTaskAccessCounter *prometheus.CounterVec
}

// NewGetTaskController create a get prover task controller
func NewGetTaskController(cfg *config.Config, chainCfg *params.ChainConfig, db *gorm.DB, verifier *verifier.Verifier, reg prometheus.Registerer) *GetTaskController {
	// TODO: implement proxy get task controller initialization
	return &GetTaskController{
		proverTasks: make(map[message.ProofType]provertask.ProverTask),
	}
}

func (ptc *GetTaskController) incGetTaskAccessCounter(ctx *gin.Context) error {
	// TODO: implement proxy get task access counter
	return nil
}

// GetTasks get assigned chunk/batch task
func (ptc *GetTaskController) GetTasks(ctx *gin.Context) {
	// TODO: implement proxy get tasks logic
}

func (ptc *GetTaskController) proofType(para *coordinatorType.GetTaskParameter) message.ProofType {
	// TODO: implement proxy proof type logic
	return message.ProofTypeChunk
}