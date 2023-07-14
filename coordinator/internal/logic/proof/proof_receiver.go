package proof

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/scroll-tech/go-ethereum/log"
	gethMetrics "github.com/scroll-tech/go-ethereum/metrics"
	"gorm.io/gorm"

	"scroll-tech/common/metrics"
	"scroll-tech/common/types"
	"scroll-tech/common/types/message"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/logic/rollermanager"
	"scroll-tech/coordinator/internal/logic/verifier"
	"scroll-tech/coordinator/internal/orm"
	types2 "scroll-tech/coordinator/internal/types"
)

var (
	coordinatorProofsGeneratedFailedTimeTimer = gethMetrics.NewRegisteredTimer("coordinator/proofs/generated/failed/time", metrics.ScrollRegistry)
	coordinatorProofsReceivedTotalCounter     = gethMetrics.NewRegisteredCounter("coordinator/proofs/received/total", metrics.ScrollRegistry)
	coordinatorProofsVerifiedSuccessTimeTimer = gethMetrics.NewRegisteredTimer("coordinator/proofs/verified/success/time", metrics.ScrollRegistry)
	coordinatorProofsVerifiedFailedTimeTimer  = gethMetrics.NewRegisteredTimer("coordinator/proofs/verified/failed/time", metrics.ScrollRegistry)
	coordinatorSessionsFailedTotalCounter     = gethMetrics.NewRegisteredCounter("coordinator/sessions/failed/total", metrics.ScrollRegistry)
)

var (
	// ErrValidatorFailureProofMsgStatusNotOk proof msg status not ok
	ErrValidatorFailureProofMsgStatusNotOk = errors.New("validator failure proof msg status not ok")
	// ErrValidatorFailureRollerEmpty get none rollers
	ErrValidatorFailureRollerEmpty = errors.New("validator failure get none rollers for the proof")
	// ErrValidatorFailureRollerInfoHasProofValid proof is vaild
	ErrValidatorFailureRollerInfoHasProofValid = errors.New("validator failure roller info has proof valid")
)

// ZKProofReceiver the proof receiver
type ZKProofReceiver struct {
	chunkOrm      *orm.Chunk
	batchOrm      *orm.Batch
	proverTaskOrm *orm.ProverTask

	db  *gorm.DB
	cfg *config.Config

	verifier *verifier.Verifier
}

// NewZKProofReceiver create a proof receiver
func NewZKProofReceiver(cfg *config.Config, db *gorm.DB) *ZKProofReceiver {
	vf, err := verifier.NewVerifier(cfg.Verifier)
	if err != nil {
		panic("proof receiver new verifier failure")
	}
	return &ZKProofReceiver{
		chunkOrm:      orm.NewChunk(db),
		batchOrm:      orm.NewBatch(db),
		proverTaskOrm: orm.NewProverTask(db),

		cfg: cfg,
		db:  db,

		verifier: vf,
	}
}

// HandleZkProof handle a ZkProof submitted from a roller.
// For now only proving/verifying error will lead to setting status as skipped.
// db/unmarshal errors will not because they are errors on the business logic side.
func (m *ZKProofReceiver) HandleZkProof(ctx context.Context, proofMsg *message.ProofMsg) error {
	pk, _ := proofMsg.PublicKey()
	rollermanager.Manager.UpdateMetricRollerProofsLastFinishedTimestampGauge(pk)

	proverTask, err := m.proverTaskOrm.GetProverTaskByHashAndPubKey(ctx, proofMsg.ProofDetail.ID, pk)
	if proverTask == nil || err != nil {
		log.Error("get none rollers for the proof key", pk, "id", proofMsg.ProofDetail.ID)
		return ErrValidatorFailureRollerEmpty
	}

	if err = m.validator(proverTask, pk, proofMsg); err != nil {
		if errors.Is(err, ErrValidatorFailureProofMsgStatusNotOk) {
			m.proofFailure(ctx, proofMsg.ID, pk, proofMsg.Type)
		}
		return nil
	}

	proofTime := time.Since(proverTask.CreatedAt)
	proofTimeSec := uint64(proofTime.Seconds())

	// store proof content
	var storeProofErr error
	switch proofMsg.Type {
	case message.ProofTypeChunk:
		storeProofErr = m.db.Transaction(func(tx *gorm.DB) error {
			if dbErr := m.chunkOrm.UpdateProofByHash(ctx, proofMsg.ID, proofMsg.Proof, proofTimeSec, tx); dbErr != nil {
				return fmt.Errorf("failed to store chunk proof into db, err:%w", dbErr)
			}
			if dbErr := m.chunkOrm.UpdateProvingStatus(ctx, proofMsg.ID, types.ProvingTaskProved, tx); dbErr != nil {
				return fmt.Errorf("failed to update chunk task status as proved, error:%w", dbErr)
			}
			return nil
		})
	case message.ProofTypeBatch:
		storeProofErr = m.db.Transaction(func(tx *gorm.DB) error {
			if dbErr := m.batchOrm.UpdateProofByHash(ctx, proofMsg.ID, proofMsg.Proof, proofTimeSec, tx); dbErr != nil {
				return fmt.Errorf("failed to store batch proof into db, error:%w", dbErr)
			}
			if dbErr := m.batchOrm.UpdateProvingStatus(ctx, proofMsg.ID, types.ProvingTaskProved, tx); dbErr != nil {
				return fmt.Errorf("failed to update batch task status as proved, error:%w", dbErr)
			}
			return nil
		})
	}
	if storeProofErr != nil {
		m.proofFailure(ctx, proofMsg.ID, pk, proofMsg.Type)
		log.Error("failed to store basic proof into db", "error", storeProofErr)
		return storeProofErr
	}

	coordinatorProofsReceivedTotalCounter.Inc(1)

	// TODO: wrap both basic verifier and aggregator verifier
	success, verifyErr := m.verifier.VerifyProof(proofMsg.Proof)
	if verifyErr != nil || !success {
		m.proofFailure(ctx, proofMsg.ID, pk, proofMsg.Type)

		// TODO: this is only a temp workaround for testnet, we should return err in real cases
		log.Error("Failed to verify zk proof", "proof id", proofMsg.ID, "roller pk", pk, "prove type",
			proofMsg.Type, "proof time", proofTimeSec, "error", verifyErr)

		// TODO: Roller needs to be slashed if proof is invalid.
		coordinatorProofsVerifiedFailedTimeTimer.Update(proofTime)

		rollermanager.Manager.UpdateMetricRollerProofsVerifiedFailedTimeTimer(pk, proofTime)

		log.Info("proof verified by coordinator failed", "proof id", proofMsg.ID, "roller name", "roller pk", pk,
			"prove type", proofMsg.Type, "proof time", proofTimeSec, "error", verifyErr)
		return nil
	}

	if err := m.closeProofTask(ctx, proofMsg.ID, pk, proofMsg); err != nil {
		m.proofRecover(ctx, proofMsg.ID, pk, proofMsg.Type)
	}

	coordinatorProofsVerifiedSuccessTimeTimer.Update(proofTime)
	rollermanager.Manager.UpdateMetricRollerProofsVerifiedSuccessTimeTimer(pk, proofTime)

	return nil
}

func (m *ZKProofReceiver) checkAreAllChunkProofsReady(ctx context.Context, chunkHash string) error {
	batchHash, err := m.chunkOrm.GetChunkBatchHash(ctx, chunkHash)
	if err != nil {
		return err
	}

	allReady, err := m.chunkOrm.CheckIfBatchChunkProofsAreReady(ctx, batchHash)
	if err != nil {
		return err
	}
	if allReady {
		err := m.chunkOrm.UpdateChunkProofsStatusByBatchHash(ctx, batchHash, types.ChunkProofsStatusReady)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *ZKProofReceiver) validator(proverTask *orm.ProverTask, pk string, proofMsg *message.ProofMsg) error {
	// Ensure this roller is eligible to participate in the prover task.
	if types.RollerProveStatus(proverTask.ProvingStatus) == types.RollerProofValid {
		// In order to prevent DoS attacks, it is forbidden to repeatedly submit valid proofs.
		// TODO: Defend invalid proof resubmissions by one of the following two methods:
		// (i) slash the roller for each submission of invalid proof
		// (ii) set the maximum failure retry times
		log.Warn("roller has already submitted valid proof in proof session", "roller name", proverTask.ProverName,
			"roller pk", proverTask.ProverPublicKey, "proof type", proverTask.TaskType, "proof id", proofMsg.ProofDetail.ID)
		return ErrValidatorFailureRollerInfoHasProofValid
	}

	proofTime := time.Since(proverTask.CreatedAt)
	proofTimeSec := uint64(proofTime.Seconds())

	log.Info("handling zk proof", "proof id", proofMsg.ID, "roller name", proverTask.ProverName,
		"roller pk", pk, "prove type", proverTask.TaskType, "proof time", proofTimeSec)

	if proofMsg.Status != message.StatusOk {
		coordinatorProofsGeneratedFailedTimeTimer.Update(proofTime)

		rollermanager.Manager.UpdateMetricRollerProofsGeneratedFailedTimeTimer(pk, proofTime)

		log.Info("proof generated by roller failed", "proof id", proofMsg.ID, "roller name", proverTask.ProverName,
			"roller pk", pk, "prove type", proofMsg.Type, "proof time", proofTimeSec, "error", proofMsg.Error)
		return ErrValidatorFailureProofMsgStatusNotOk
	}
	return nil
}

func (m *ZKProofReceiver) proofFailure(ctx context.Context, hash string, pubKey string, proofMsgType message.ProofType) {
	if err := m.updateProofStatus(ctx, hash, pubKey, proofMsgType, types.ProvingTaskFailed); err != nil {
		log.Error("failed to updated proof status ProvingTaskFailed", "hash", hash, "pubKey", pubKey, "error", err)
	}
	coordinatorSessionsFailedTotalCounter.Inc(1)
}

func (m *ZKProofReceiver) proofRecover(ctx context.Context, hash string, pubKey string, proofMsgType message.ProofType) {
	if err := m.updateProofStatus(ctx, hash, pubKey, proofMsgType, types.ProvingTaskUnassigned); err != nil {
		log.Error("failed to updated proof status ProvingTaskUnassigned", "hash", hash, "pubKey", pubKey, "error", err)
	}
}

func (m *ZKProofReceiver) closeProofTask(ctx context.Context, hash string, pubKey string, proofMsg *message.ProofMsg) error {
	if err := m.updateProofStatus(ctx, hash, pubKey, proofMsg.Type, types.ProvingTaskVerified); err != nil {
		log.Error("failed to updated proof status ProvingTaskVerified", "hash", hash, "pubKey", pubKey, "error", err)
		return err
	}

	rollermanager.Manager.FreeTaskIDForRoller(pubKey, hash)
	return nil
}

// UpdateProofStatus update the block batch/agg task and session info status
func (m *ZKProofReceiver) updateProofStatus(ctx context.Context, hash string, proverPublicKey string, proofMsgType message.ProofType, status types.ProvingStatus) error {
	// if the prover task failure type is SessionInfoFailureTimeout,
	// just skip update the status because the proof result come so slow.
	if m.checkIsTimeoutFailure(ctx, hash, proverPublicKey) {
		return nil
	}

	var proverTaskStatus types.RollerProveStatus
	switch status {
	case types.ProvingTaskProved, types.ProvingTaskUnassigned:
		proverTaskStatus = types.RollerProofInvalid
	case types.ProvingTaskVerified:
		proverTaskStatus = types.RollerProofValid
	}

	err := m.db.Transaction(func(tx *gorm.DB) error {
		if updateErr := m.proverTaskOrm.UpdateProverTaskProvingStatus(ctx, proofMsgType, hash, proverPublicKey, proverTaskStatus); updateErr != nil {
			return updateErr
		}

		// if the block batch has proof verified, so the failed status not update block batch proving status
		if status == types.ProvingTaskFailed && m.checkIsTaskSuccess(ctx, hash, proofMsgType) {
			return nil
		}

		switch proofMsgType {
		case message.ProofTypeChunk:
			if err := m.chunkOrm.UpdateProvingStatus(ctx, hash, status, tx); err != nil {
				log.Error("failed to update basic proving_status as failed", "msg.ID", hash, "error", err)
				return err
			}
			if status == types.ProvingTaskVerified {
				if err := m.checkAreAllChunkProofsReady(ctx, hash); err != nil {
					log.Error("failed to check are all chunk proofs ready", "error", err)
					return err
				}
			}
		case message.ProofTypeBatch:
			if err := m.batchOrm.UpdateProvingStatus(ctx, hash, status, tx); err != nil {
				log.Error("failed to update aggregator proving_status as failed", "msg.ID", hash, "error", err)
				return err
			}
		}
		return nil
	})

	return err
}

func (m *ZKProofReceiver) checkIsTaskSuccess(ctx context.Context, hash string, proofType message.ProofType) bool {
	var provingStatus types.ProvingStatus
	var err error

	switch proofType {
	case message.ProofTypeChunk:
		provingStatus, err = m.chunkOrm.GetProvingStatusByHash(ctx, hash)
		if err != nil {
			return false
		}
	case message.ProofTypeBatch:
		provingStatus, err = m.batchOrm.GetProvingStatusByHash(ctx, hash)
		if err != nil {
			return false
		}
	}
	if provingStatus == types.ProvingTaskVerified || provingStatus == types.ProvingTaskProved {
		return true
	}
	return false
}

func (m *ZKProofReceiver) checkIsTimeoutFailure(ctx context.Context, hash, proverPublicKey string) bool {
	proverTask, err := m.proverTaskOrm.GetProverTaskByHashAndPubKey(ctx, hash, proverPublicKey)
	if err != nil {
		return false
	}

	if types2.ProverTaskFailureType(proverTask.FailureType) == types2.ProverTaskFailureTypeTimeout {
		return true
	}
	return false
}
