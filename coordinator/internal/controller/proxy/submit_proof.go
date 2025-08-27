package proxy

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/scroll-tech/go-ethereum/params"
	"gorm.io/gorm"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/logic/submitproof"
	"scroll-tech/coordinator/internal/logic/verifier"
)

// SubmitProofController the submit proof api controller
type SubmitProofController struct {
	submitProofReceiverLogic *submitproof.ProofReceiverLogic
}

// NewSubmitProofController create the submit proof api controller instance
func NewSubmitProofController(cfg *config.Config, chainCfg *params.ChainConfig, db *gorm.DB, vf *verifier.Verifier, reg prometheus.Registerer) *SubmitProofController {
	return &SubmitProofController{
		submitProofReceiverLogic: submitproof.NewSubmitProofReceiverLogic(cfg.ProverManager, chainCfg, db, vf, reg),
	}
}

// SubmitProof prover submit the proof to coordinator
func (spc *SubmitProofController) SubmitProof(ctx *gin.Context) {
	// TODO: implement proxy submit proof logic
}