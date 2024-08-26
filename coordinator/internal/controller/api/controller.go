package api

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/params"
	"gorm.io/gorm"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/logic/verifier"
)

var (
	// GetTask the prover task controller
	GetTask *GetTaskController
	// SubmitProof the submit proof controller
	SubmitProof *SubmitProofController
	// Auth the auth controller
	Auth *AuthController
)

// InitController inits Controller with database
func InitController(cfg *config.Config, chainCfg *params.ChainConfig, db *gorm.DB, reg prometheus.Registerer) {
	vf, err := verifier.NewVerifier(cfg.ProverManager.Verifier)
	if err != nil {
		panic("proof receiver new verifier failure")
	}

	log.Info("verifier created", "chunkVerifier", vf.ChunkVKMap, "batchVerifier", vf.BatchVKMap, "bundleVerifier", vf.BundleVkMap)

	Auth = NewAuthController(db, cfg, vf)
	GetTask = NewGetTaskController(cfg, chainCfg, db, reg)
	SubmitProof = NewSubmitProofController(cfg, chainCfg, db, vf, reg)
}
