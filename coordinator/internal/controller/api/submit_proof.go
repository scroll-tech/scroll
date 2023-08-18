package api

import (
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"gorm.io/gorm"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/logic/submitproof"
	coodinatorType "scroll-tech/coordinator/internal/types"
)

// SubmitProofController the submit proof api controller
type SubmitProofController struct {
	submitProofReceiverLogic *submitproof.ProofReceiverLogic
}

// NewSubmitProofController create the submit proof api controller instance
func NewSubmitProofController(cfg *config.Config, db *gorm.DB, reg prometheus.Registerer) *SubmitProofController {
	return &SubmitProofController{
		submitProofReceiverLogic: submitproof.NewSubmitProofReceiverLogic(cfg.ProverManager, db, reg),
	}
}

// SubmitProof prover submit the proof to coordinator
func (spc *SubmitProofController) SubmitProof(ctx *gin.Context) {
	var spp = &coodinatorType.SubmitProofParameter{}
	if err := ctx.ShouldBind(&spp); err != nil {
		nerr := fmt.Errorf("parameter invalid, err:%w", err)
		coodinatorType.RenderJSON(ctx, types.ErrCoordinatorParameterInvalidNo, nerr, nil)
		return
	}

	proofMsg := message.ProofMsg{
		ProofDetail: &message.ProofDetail{
			ID:     spp.TaskID,
			Type:   message.ProofType(spp.TaskType),
			Status: message.RespStatus(spp.Status),
		},
	}

	if spp.Status == int(message.StatusOk) {
		switch message.ProofType(spp.TaskType) {
		case message.ProofTypeChunk:
			var tmpChunkProof message.ChunkProof
			if err := json.Unmarshal([]byte(spp.Proof), &tmpChunkProof); err != nil {
				nerr := fmt.Errorf("unmarshal parameter chunk proof invalid, err:%w", err)
				coodinatorType.RenderJSON(ctx, types.ErrCoordinatorParameterInvalidNo, nerr, nil)
				return
			}
			proofMsg.ChunkProof = &tmpChunkProof
		case message.ProofTypeBatch:
			var tmpBatchProof message.BatchProof
			if err := json.Unmarshal([]byte(spp.Proof), &tmpBatchProof); err != nil {
				nerr := fmt.Errorf("unmarshal parameter batch proof invalid, err:%w", err)
				coodinatorType.RenderJSON(ctx, types.ErrCoordinatorParameterInvalidNo, nerr, nil)
				return
			}
			proofMsg.BatchProof = &tmpBatchProof
		}
	}

	if err := spc.submitProofReceiverLogic.HandleZkProof(ctx, &proofMsg, spp); err != nil {
		nerr := fmt.Errorf("handle zk proof failure, err:%w", err)
		coodinatorType.RenderJSON(ctx, types.ErrCoordinatorHandleZkProofFailure, nerr, nil)
		return
	}
	coodinatorType.RenderJSON(ctx, types.Success, nil, nil)
}
