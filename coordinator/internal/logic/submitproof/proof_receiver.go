package submitproof

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/scroll-tech/go-ethereum/log"
	gethMetrics "github.com/scroll-tech/go-ethereum/metrics"
	"gorm.io/gorm"

	"scroll-tech/common/metrics"
	"scroll-tech/common/types"
	"scroll-tech/common/types/message"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/logic/verifier"
	"scroll-tech/coordinator/internal/orm"
	coordinatorType "scroll-tech/coordinator/internal/types"
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
	// ErrValidatorFailureProverTaskEmpty get none prover task
	ErrValidatorFailureProverTaskEmpty = errors.New("validator failure get none prover task for the proof")
	// ErrValidatorFailureProverInfoHasProofValid proof is vaild
	ErrValidatorFailureProverInfoHasProofValid = errors.New("validator failure prover task info has proof valid")
)

// SubmitProofReceiverLogic the proof receiver logic
type SubmitProofReceiverLogic struct {
	chunkOrm      *orm.Chunk
	batchOrm      *orm.Batch
	proverTaskOrm *orm.ProverTask

	db  *gorm.DB
	cfg *config.ProverManagerConfig

	verifier *verifier.Verifier
}

// NewSubmitProofReceiverLogic create a proof receiver logic
func NewSubmitProofReceiverLogic(cfg *config.ProverManagerConfig, db *gorm.DB) *SubmitProofReceiverLogic {
	vf, err := verifier.NewVerifier(cfg.Verifier)
	if err != nil {
		panic("proof receiver new verifier failure")
	}
	return &SubmitProofReceiverLogic{
		chunkOrm:      orm.NewChunk(db),
		batchOrm:      orm.NewBatch(db),
		proverTaskOrm: orm.NewProverTask(db),

		cfg: cfg,
		db:  db,

		verifier: vf,
	}
}

// HandleZkProof handle a ZkProof submitted from a prover.
// For now only proving/verifying error will lead to setting status as skipped.
// db/unmarshal errors will not because they are errors on the business logic side.
func (m *SubmitProofReceiverLogic) HandleZkProof(ctx *gin.Context, proofMsg *message.ProofMsg) error {
	pk := ctx.GetString(coordinatorType.PublicKey)
	if len(pk) == 0 {
		return fmt.Errorf("get public key from contex failed")
	}

	proverTask, err := m.proverTaskOrm.GetProverTaskByTaskIDAndPubKey(ctx, proofMsg.ID, pk)
	if proverTask == nil || err != nil {
		log.Error("get none prover task for the proof", "key", pk, "taskID", proofMsg.ID, "error", err)
		return ErrValidatorFailureProverTaskEmpty
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
			if dbErr := m.chunkOrm.UpdateProofByHash(ctx, proofMsg.ID, proofMsg.ChunkProof, proofTimeSec, tx); dbErr != nil {
				return fmt.Errorf("failed to store chunk proof into db, err:%w", dbErr)
			}
			if dbErr := m.chunkOrm.UpdateProvingStatus(ctx, proofMsg.ID, types.ProvingTaskProved, tx); dbErr != nil {
				return fmt.Errorf("failed to update chunk task status as proved, error:%w", dbErr)
			}
			return nil
		})
	case message.ProofTypeBatch:
		storeProofErr = m.db.Transaction(func(tx *gorm.DB) error {
			if dbErr := m.batchOrm.UpdateProofByHash(ctx, proofMsg.ID, proofMsg.BatchProof, proofTimeSec, tx); dbErr != nil {
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

	var success bool
	var verifyErr error
	if proofMsg.Type == message.ProofTypeChunk {
		success, verifyErr = m.verifier.VerifyChunkProof(proofMsg.ChunkProof)
	} else if proofMsg.Type == message.ProofTypeBatch {
		success, verifyErr = m.verifier.VerifyBatchProof(proofMsg.BatchProof)
	}

	if verifyErr != nil || !success {
		if verifyErr != nil {
			// TODO: this is only a temp workaround for testnet, we should return err in real cases
			log.Error("failed to verify zk proof", "proof id", proofMsg.ID, "prover pk", pk, "prove type",
				proofMsg.Type, "proof time", proofTimeSec, "error", verifyErr)
		}
		m.proofFailure(ctx, proofMsg.ID, pk, proofMsg.Type)

		// TODO: Prover needs to be slashed if proof is invalid.
		coordinatorProofsVerifiedFailedTimeTimer.Update(proofTime)

		log.Info("proof verified by coordinator failed", "proof id", proofMsg.ID, "prover name", proverTask.ProverName,
			"prover pk", pk, "prove type", proofMsg.Type, "proof time", proofTimeSec, "error", verifyErr)
		return nil
	}

	if err := m.closeProofTask(ctx, proofMsg.ID, pk, proofMsg); err != nil {
		m.proofRecover(ctx, proofMsg.ID, pk, proofMsg.Type)
	}

	coordinatorProofsVerifiedSuccessTimeTimer.Update(proofTime)

	return nil
}

func (m *SubmitProofReceiverLogic) checkAreAllChunkProofsReady(ctx context.Context, chunkHash string) error {
	batchHash, err := m.chunkOrm.GetChunkBatchHash(ctx, chunkHash)
	if err != nil {
		return err
	}

	allReady, err := m.chunkOrm.CheckIfBatchChunkProofsAreReady(ctx, batchHash)
	if err != nil {
		return err
	}
	if allReady {
		err := m.batchOrm.UpdateChunkProofsStatusByBatchHash(ctx, batchHash, types.ChunkProofsStatusReady)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *SubmitProofReceiverLogic) validator(proverTask *orm.ProverTask, pk string, proofMsg *message.ProofMsg) error {
	// Ensure this prover is eligible to participate in the prover task.
	if types.ProverProveStatus(proverTask.ProvingStatus) == types.ProverProofValid {
		// In order to prevent DoS attacks, it is forbidden to repeatedly submit valid proofs.
		// TODO: Defend invalid proof resubmissions by one of the following two methods:
		// (i) slash the prover for each submission of invalid proof
		// (ii) set the maximum failure retry times
		log.Warn("prover has already submitted valid proof in proof session", "prover name", proverTask.ProverName,
			"prover pk", proverTask.ProverPublicKey, "proof type", proverTask.TaskType, "proof id", proofMsg.ProofDetail.ID)
		return ErrValidatorFailureProverInfoHasProofValid
	}

	proofTime := time.Since(proverTask.CreatedAt)
	proofTimeSec := uint64(proofTime.Seconds())

	log.Info("handling zk proof", "proof id", proofMsg.ID, "prover name", proverTask.ProverName,
		"prover pk", pk, "prove type", proverTask.TaskType, "proof time", proofTimeSec)

	if proofMsg.Status != message.StatusOk {
		coordinatorProofsGeneratedFailedTimeTimer.Update(proofTime)

		log.Info("proof generated by prover failed", "proof id", proofMsg.ID, "prover name", proverTask.ProverName,
			"prover pk", pk, "prove type", proofMsg.Type, "proof time", proofTimeSec, "error", proofMsg.Error)
		return ErrValidatorFailureProofMsgStatusNotOk
	}
	return nil
}

func (m *SubmitProofReceiverLogic) proofFailure(ctx context.Context, hash string, pubKey string, proofMsgType message.ProofType) {
	if err := m.updateProofStatus(ctx, hash, pubKey, proofMsgType, types.ProvingTaskFailed); err != nil {
		log.Error("failed to updated proof status ProvingTaskFailed", "hash", hash, "pubKey", pubKey, "error", err)
	}
	coordinatorSessionsFailedTotalCounter.Inc(1)
}

func (m *SubmitProofReceiverLogic) proofRecover(ctx context.Context, hash string, pubKey string, proofMsgType message.ProofType) {
	if err := m.updateProofStatus(ctx, hash, pubKey, proofMsgType, types.ProvingTaskUnassigned); err != nil {
		log.Error("failed to updated proof status ProvingTaskUnassigned", "hash", hash, "pubKey", pubKey, "error", err)
	}
}

func (m *SubmitProofReceiverLogic) closeProofTask(ctx context.Context, hash string, pubKey string, proofMsg *message.ProofMsg) error {
	if err := m.updateProofStatus(ctx, hash, pubKey, proofMsg.Type, types.ProvingTaskVerified); err != nil {
		log.Error("failed to updated proof status ProvingTaskVerified", "hash", hash, "pubKey", pubKey, "error", err)
		return err
	}
	return nil
}

// UpdateProofStatus update the chunk/batch task and session info status
func (m *SubmitProofReceiverLogic) updateProofStatus(ctx context.Context, hash string, proverPublicKey string, proofMsgType message.ProofType, status types.ProvingStatus) error {
	// if the prover task failure type is SessionInfoFailureTimeout,
	// just skip update the status because the proof result come too late.
	if m.checkIsTimeoutFailure(ctx, hash, proverPublicKey) {
		return nil
	}

	var proverTaskStatus types.ProverProveStatus
	switch status {
	case types.ProvingTaskFailed, types.ProvingTaskUnassigned:
		proverTaskStatus = types.ProverProofInvalid
	case types.ProvingTaskVerified:
		proverTaskStatus = types.ProverProofValid
	}

	err := m.db.Transaction(func(tx *gorm.DB) error {
		if updateErr := m.proverTaskOrm.UpdateProverTaskProvingStatus(ctx, proofMsgType, hash, proverPublicKey, proverTaskStatus, tx); updateErr != nil {
			return updateErr
		}

		// if the block batch has proof verified, so the failed status not update block batch proving status
		if status == types.ProvingTaskFailed && m.checkIsTaskSuccess(ctx, hash, proofMsgType) {
			return nil
		}

		switch proofMsgType {
		case message.ProofTypeChunk:
			if err := m.chunkOrm.UpdateProvingStatus(ctx, hash, status, tx); err != nil {
				log.Error("failed to update chunk proving_status as failed", "msg.ID", hash, "error", err)
				return err
			}
		case message.ProofTypeBatch:
			if err := m.batchOrm.UpdateProvingStatus(ctx, hash, status, tx); err != nil {
				log.Error("failed to update batch proving_status as failed", "msg.ID", hash, "error", err)
				return err
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	if status == types.ProvingTaskVerified && proofMsgType == message.ProofTypeChunk {
		if checkReadyErr := m.checkAreAllChunkProofsReady(ctx, hash); checkReadyErr != nil {
			log.Error("failed to check are all chunk proofs ready", "error", checkReadyErr)
			return checkReadyErr
		}
	}

	return nil
}

func (m *SubmitProofReceiverLogic) checkIsTaskSuccess(ctx context.Context, hash string, proofType message.ProofType) bool {
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

	return provingStatus == types.ProvingTaskVerified
}

func (m *SubmitProofReceiverLogic) checkIsTimeoutFailure(ctx context.Context, hash, proverPublicKey string) bool {
	proverTask, err := m.proverTaskOrm.GetProverTaskByTaskIDAndPubKey(ctx, hash, proverPublicKey)
	if err != nil {
		return false
	}

	if types.ProverTaskFailureType(proverTask.FailureType) == types.ProverTaskFailureTypeTimeout {
		return true
	}
	return false
}
