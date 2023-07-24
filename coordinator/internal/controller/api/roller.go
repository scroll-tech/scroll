package api

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rpc"
	"gorm.io/gorm"

	"scroll-tech/common/types/message"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/logic/proof"
)

// RollerController the prover api controller
type RollerController struct {
	tokenCache    *cache.Cache
	proofReceiver *proof.ZKProofReceiver
	taskWorker    *proof.TaskWorker
}

// NewRollerController create a prover controller
func NewRollerController(cfg *config.RollerManagerConfig, db *gorm.DB) *RollerController {
	return &RollerController{
		proofReceiver: proof.NewZKProofReceiver(cfg, db),
		taskWorker:    proof.NewTaskWorker(),
		tokenCache:    cache.New(time.Duration(cfg.TokenTimeToLive)*time.Second, 1*time.Hour),
	}
}

// RequestToken get request token of authMsg
func (r *RollerController) RequestToken(authMsg *message.AuthMsg) (string, error) {
	if ok, err := authMsg.Verify(); !ok {
		if err != nil {
			log.Error("failed to verify auth message", "error", err)
		}
		return "", errors.New("signature verification failed")
	}
	pubkey, err := authMsg.PublicKey()
	if err != nil {
		return "", fmt.Errorf("RequestToken auth msg public key error:%w", err)
	}
	if token, ok := r.tokenCache.Get(pubkey); ok {
		return token.(string), nil
	}
	token, err := message.GenerateToken()
	if err != nil {
		return "", errors.New("token generation failed")
	}
	r.tokenCache.SetDefault(pubkey, token)
	return token, nil
}

// VerifyToken verifies pubkey for token and expiration time
func (r *RollerController) verifyToken(authMsg *message.AuthMsg) (bool, error) {
	pubkey, err := authMsg.PublicKey()
	if err != nil {
		return false, fmt.Errorf("verify token auth msg public key error:%w", err)
	}
	// GetValue returns nil if value is expired
	if token, ok := r.tokenCache.Get(pubkey); !ok || token != authMsg.Identity.Token {
		return false, fmt.Errorf("failed to find corresponding token. prover name: %s prover pk: %s", authMsg.Identity.Name, pubkey)
	}
	return true, nil
}

// Register register api for prover
func (r *RollerController) Register(ctx context.Context, authMsg *message.AuthMsg) (*rpc.Subscription, error) {
	// Verify register message.
	if ok, err := authMsg.Verify(); !ok {
		if err != nil {
			log.Error("failed to verify auth message", "error", err)
		}
		return nil, errors.New("signature verification failed")
	}
	// Lock here to avoid malicious prover message replay before cleanup of token
	if ok, err := r.verifyToken(authMsg); !ok {
		return nil, err
	}
	pubkey, err := authMsg.PublicKey()
	if err != nil {
		return nil, fmt.Errorf("register auth msg public key error:%w", err)
	}
	// prover successfully registered, remove token associated with this prover
	r.tokenCache.Delete(pubkey)

	rpcSub, err := r.taskWorker.AllocTaskWorker(ctx, authMsg)
	if err != nil {
		return rpcSub, err
	}
	return rpcSub, nil
}

// SubmitProof prover pull proof
func (r *RollerController) SubmitProof(proof *message.ProofMsg) error {
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
