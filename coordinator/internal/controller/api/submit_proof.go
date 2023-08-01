package api

import (
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"scroll-tech/common/types/message"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/logic/submit_proof"
	"scroll-tech/coordinator/internal/types"
)

// SubmitProofController the submit proof api controller
type SubmitProofController struct {
	submitProofReceiverLogic *submit_proof.SubmitProofReceiverLogic
}

// NewSubmitProofController create the submit proof api controller instance
func NewSubmitProofController(cfg *config.Config, db *gorm.DB) *SubmitProofController {
	return &SubmitProofController{
		submitProofReceiverLogic: submit_proof.NewSubmitProofReceiverLogic(cfg.ProverManagerConfig, db),
	}
}

// SubmitProof prover submit the proof to coordinator
func (spc *SubmitProofController) SubmitProof(ctx *gin.Context) {
	var spp types.SubmitProofParameter
	if err := ctx.ShouldBind(&spp); err != nil {
		nerr := fmt.Errorf("parameter invalid, err:%w", err)
		types.RenderJSON(ctx, types.ErrParameterInvalidNo, nerr, nil)
		return
	}

	proofMsg := message.ProofMsg{
		ProofDetail: &message.ProofDetail{
			ID:     spp.TaskID,
			Type:   message.ProofType(spp.ProofType),
			Status: message.RespStatus(spp.Status),
			Error:  spp.Error,
		},
		Signature: spp.Signature,
	}

	switch message.ProofType(spp.ProofType) {
	case message.ProofTypeChunk:
		var tmpChunkProof message.ChunkProof
		if err := json.Unmarshal([]byte(spp.Proof), &tmpChunkProof); err != nil {
			nerr := fmt.Errorf("unmarshal parameter chunk proof invalid, err:%w", err)
			types.RenderJSON(ctx, types.ErrParameterInvalidNo, nerr, nil)
			return
		}
		proofMsg.ChunkProof = &tmpChunkProof
	case message.ProofTypeBatch:
		var tmpBatchProof message.BatchProof
		if err := json.Unmarshal([]byte(spp.Proof), &tmpBatchProof); err != nil {
			nerr := fmt.Errorf("unmarshal parameter batch proof invalid, err:%w", err)
			types.RenderJSON(ctx, types.ErrParameterInvalidNo, nerr, nil)
			return
		}
		proofMsg.BatchProof = &tmpBatchProof
	}

	if err := spc.submitProofReceiverLogic.HandleZkProof(ctx, &proofMsg); err != nil {
		nerr := fmt.Errorf("handle zk proof failure, err:%w", err)
		types.RenderJSON(ctx, types.ErrHandleZkProofFailure, nerr, nil)
		return
	}
	types.RenderJSON(ctx, types.Success, nil, nil)
}
