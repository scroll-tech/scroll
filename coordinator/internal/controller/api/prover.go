package api

import (
	"context"
	"errors"
	"time"

	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rpc"
	"gorm.io/gorm"

	"scroll-tech/common/types/message"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/logic/proof"
)

// ProverController the prover api controller
type ProverController struct {
	proofReceiver *proof.ZKProofReceiver
	taskWorker    *proof.TaskWorker
	tokenExpire   time.Duration
	jwtSecret     []byte
}

// NewProverController create a prover controller
func NewProverController(cfg *config.ProverManagerConfig, db *gorm.DB) *ProverController {
	tokenExpire := time.Duration(cfg.TokenTimeToLive) * time.Second
	return &ProverController{
		proofReceiver: proof.NewZKProofReceiver(cfg, db),
		taskWorker:    proof.NewTaskWorker(),
		tokenExpire:   tokenExpire,
		jwtSecret:     []byte(cfg.JwtSecret),
	}
}

// RequestToken get request token of authMsg
func (r *ProverController) RequestToken(authMsg *message.AuthMsg) (string, error) {
	if ok, err := authMsg.Verify(); !ok {
		if err != nil {
			log.Error("failed to verify auth message", "error", err)
		}
		return "", errors.New("signature verification failed")
	}
	token, err := message.GenerateToken(r.tokenExpire, r.jwtSecret)
	if err != nil {
		return "", errors.New("token generation failed")
	}
	return token, nil
}

// VerifyToken verifies JWT for token and expiration time
func (r *ProverController) verifyToken(tokenStr string) (bool, error) {
	return message.VerifyToken(r.jwtSecret, tokenStr)
}

// Register register api for prover
func (r *ProverController) Register(ctx context.Context, authMsg *message.AuthMsg) (*rpc.Subscription, error) {
	// Verify register message.
	if ok, err := authMsg.Verify(); !ok {
		if err != nil {
			log.Error("failed to verify auth message", "error", err)
		}
		return nil, errors.New("signature verification failed")
	}
	// Verify the jwt
	if ok, err := r.verifyToken(authMsg.Identity.Token); !ok {
		return nil, err
	}

	rpcSub, err := r.taskWorker.AllocTaskWorker(ctx, authMsg)
	if err != nil {
		return rpcSub, err
	}
	return rpcSub, nil
}

// SubmitProof prover pull proof
func (r *ProverController) SubmitProof(proof *message.ProofMsg) error {
	// Verify the signature
	if ok, err := proof.Verify(); !ok {
		if err != nil {
			log.Error("failed to verify proof message", "error", err)
		}
		return errors.New("auth signature verify fail")
	}

	err := r.proofReceiver.HandleZkProof(context.Background(), proof)
	if err != nil {
		return err
	}

	return nil
}
