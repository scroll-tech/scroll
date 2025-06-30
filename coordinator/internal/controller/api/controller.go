package api

import (
	"encoding/json"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/params"
	"gorm.io/gorm"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/logic/libzkp"
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

	log.Info("verifier created", "openVmVerifier", vf.OpenVMVkMap)

	// TODO: enable this when the libzkp has been updated
	l2cfg := cfg.L2.Endpoint
	if l2cfg == nil {
		panic("l2geth is not specified")
	}
	l2cfgBytes, err := json.Marshal(l2cfg)
	if err != nil {
		panic(err)
	}
	libzkp.InitL2geth(string(l2cfgBytes))

	Auth = NewAuthController(db, cfg, vf)
	GetTask = NewGetTaskController(cfg, chainCfg, db, vf, reg)
	SubmitProof = NewSubmitProofController(cfg, chainCfg, db, vf, reg)
}
