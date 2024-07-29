package submitproof

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/logic/verifier"
	"scroll-tech/coordinator/internal/orm"
	coordinatorType "scroll-tech/coordinator/internal/types"
)

var (
	// ErrValidatorFailureProofMsgStatusNotOk proof msg status not ok
	ErrValidatorFailureProofMsgStatusNotOk = errors.New("validator failure proof msg status not ok")
	// ErrValidatorFailureProverTaskEmpty get none prover task
	ErrValidatorFailureProverTaskEmpty = errors.New("validator failure get none prover task for the proof")
	// ErrValidatorFailureProverTaskCannotSubmitTwice prove task can not submit proof twice
	ErrValidatorFailureProverTaskCannotSubmitTwice = errors.New("validator failure prove task cannot submit proof twice")
	// ErrValidatorFailureProofTimeout the submit proof is timeout
	ErrValidatorFailureProofTimeout = errors.New("validator failure submit proof timeout")
	// ErrValidatorFailureTaskHaveVerifiedSuccess have proved success and verified success
	ErrValidatorFailureTaskHaveVerifiedSuccess = errors.New("validator failure chunk/batch have proved and verified success")
	// ErrValidatorFailureVerifiedFailed failed to verify and the verifier returns error
	ErrValidatorFailureVerifiedFailed = fmt.Errorf("verification failed, verifier returns error")
	// ErrValidatorSuccessInvalidProof successful verified and the proof is invalid
	ErrValidatorSuccessInvalidProof = fmt.Errorf("verification succeeded, it's an invalid proof")
	// ErrCoordinatorInternalFailure coordinator internal db failure
	ErrCoordinatorInternalFailure = fmt.Errorf("coordinator internal error")
)

// ProofReceiverLogic the proof receiver logic
type ProofReceiverLogic struct {
	chunkOrm      *orm.Chunk
	batchOrm      *orm.Batch
	proverTaskOrm *orm.ProverTask

	db  *gorm.DB
	cfg *config.ProverManager

	verifier *verifier.Verifier

	proofReceivedTotal                    prometheus.Counter
	proofSubmitFailure                    prometheus.Counter
	verifierTotal                         *prometheus.CounterVec
	verifierFailureTotal                  *prometheus.CounterVec
	proverTaskProveDuration               prometheus.Histogram
	validateFailureTotal                  prometheus.Counter
	validateFailureProverTaskSubmitTwice  prometheus.Counter
	validateFailureProverTaskStatusNotOk  prometheus.Counter
	validateFailureProverTaskTimeout      prometheus.Counter
	validateFailureProverTaskHaveVerifier prometheus.Counter
}

// NewSubmitProofReceiverLogic create a proof receiver logic
func NewSubmitProofReceiverLogic(cfg *config.ProverManager, db *gorm.DB, vf *verifier.Verifier, reg prometheus.Registerer) *ProofReceiverLogic {
	return &ProofReceiverLogic{
		chunkOrm:      orm.NewChunk(db),
		batchOrm:      orm.NewBatch(db),
		proverTaskOrm: orm.NewProverTask(db),

		cfg: cfg,
		db:  db,

		verifier: vf,

		proofReceivedTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "coordinator_submit_proof_total",
			Help: "Total number of submit proof.",
		}),
		proofSubmitFailure: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "coordinator_submit_proof_failure_total",
			Help: "Total number of submit proof failure.",
		}),
		verifierTotal: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Name: "coordinator_verifier_total",
			Help: "Total number of verifier.",
		}, []string{"version"}),
		verifierFailureTotal: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Name: "coordinator_verifier_failure_total",
			Help: "Total number of verifier failure.",
		}, []string{"version"}),
		proverTaskProveDuration: promauto.With(reg).NewHistogram(prometheus.HistogramOpts{
			Name:    "coordinator_task_prove_duration_seconds",
			Help:    "Time spend by prover prove task.",
			Buckets: []float64{180, 300, 480, 600, 900, 1200, 1800},
		}),
		validateFailureTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "coordinator_validate_failure_total",
			Help: "Total number of submit proof validate failure.",
		}),
		validateFailureProverTaskSubmitTwice: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "coordinator_validate_failure_submit_twice_total",
			Help: "Total number of submit proof validate failure submit twice.",
		}),
		validateFailureProverTaskStatusNotOk: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "coordinator_validate_failure_submit_status_not_ok",
			Help: "Total number of submit proof validate failure proof status not ok.",
		}),
		validateFailureProverTaskTimeout: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "coordinator_validate_failure_submit_timeout",
			Help: "Total number of submit proof validate failure timeout.",
		}),
		validateFailureProverTaskHaveVerifier: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "coordinator_validate_failure_submit_have_been_verifier",
			Help: "Total number of submit proof validate failure proof have been verifier.",
		}),
	}
}

// HandleZkProof handle a ZkProof submitted from a prover.
// For now only proving/verifying error will lead to setting status as skipped.
// db/unmarshal errors will not because they are errors on the business logic side.
func (m *ProofReceiverLogic) HandleZkProof(ctx *gin.Context, proofMsg *message.ProofMsg, proofParameter coordinatorType.SubmitProofParameter) error {
	m.proofReceivedTotal.Inc()
	pk := ctx.GetString(coordinatorType.PublicKey)
	if len(pk) == 0 {
		return fmt.Errorf("get public key from context failed")
	}
	pv := ctx.GetString(coordinatorType.ProverVersion)
	if len(pv) == 0 {
		return fmt.Errorf("get ProverVersion from context failed")
	}
	// use hard_fork_name from parameter first
	// if prover support multi hard_forks, the real hard_fork_name is not set to the gin context
	hardForkName := proofParameter.HardForkName
	if hardForkName == "" {
		hardForkName = ctx.GetString(coordinatorType.HardForkName)
	}

	var proverTask *orm.ProverTask
	var err error
	if proofParameter.UUID != "" {
		proverTask, err = m.proverTaskOrm.GetProverTaskByUUIDAndPublicKey(ctx.Copy(), proofParameter.UUID, pk)
		if proverTask == nil || err != nil {
			log.Error("get none prover task for the proof", "uuid", proofParameter.UUID, "key", pk, "taskID", proofMsg.ID, "error", err)
			return ErrValidatorFailureProverTaskEmpty
		}
	} else {
		// TODO When prover all have upgrade, need delete this logic
		proverTask, err = m.proverTaskOrm.GetAssignedProverTaskByTaskIDAndProver(ctx.Copy(), proofMsg.Type, proofMsg.ID, pk, pv)
		if proverTask == nil || err != nil {
			log.Error("get none prover task for the proof", "key", pk, "taskID", proofMsg.ID, "error", err)
			return ErrValidatorFailureProverTaskEmpty
		}
	}

	proofTime := time.Since(proverTask.CreatedAt)
	proofTimeSec := uint64(proofTime.Seconds())

	log.Info("handling zk proof", "proofID", proofMsg.ID, "proverName", proverTask.ProverName,
		"proverPublicKey", pk, "proveType", proverTask.TaskType, "proofTime", proofTimeSec, "hardForkName", hardForkName)

	if err = m.validator(ctx.Copy(), proverTask, pk, proofMsg, proofParameter, hardForkName); err != nil {
		return err
	}

	m.verifierTotal.WithLabelValues(pv).Inc()

	success := true
	var verifyErr error
	// only verify batch proof. chunk proof verifier have been disabled after Bernoulli
	if proofMsg.Type == message.ProofTypeBatch {
		success, verifyErr = m.verifier.VerifyBatchProof(proofMsg.BatchProof, hardForkName)
	}

	if verifyErr != nil || !success {
		m.verifierFailureTotal.WithLabelValues(pv).Inc()

		m.proofRecover(ctx.Copy(), proverTask, types.ProverTaskFailureTypeVerifiedFailed, proofMsg)

		log.Info("proof verified by coordinator failed", "proof id", proofMsg.ID, "prover name", proverTask.ProverName,
			"prover pk", pk, "forkName", hardForkName, "prove type", proofMsg.Type, "proof time", proofTimeSec, "error", verifyErr)

		if verifyErr != nil {
			return ErrValidatorFailureVerifiedFailed
		}
		return ErrValidatorSuccessInvalidProof
	}

	m.proverTaskProveDuration.Observe(time.Since(proverTask.CreatedAt).Seconds())

	log.Info("proof verified and valid", "proof id", proofMsg.ID, "prover name", proverTask.ProverName,
		"prover pk", pk, "prove type", proofMsg.Type, "proof time", proofTimeSec, "forkName", hardForkName)

	if err := m.closeProofTask(ctx.Copy(), proverTask, proofMsg, proofTimeSec); err != nil {
		m.proofSubmitFailure.Inc()

		m.proofRecover(ctx.Copy(), proverTask, types.ProverTaskFailureTypeServerError, proofMsg)

		return ErrCoordinatorInternalFailure
	}

	return nil
}

func (m *ProofReceiverLogic) checkAreAllChunkProofsReady(ctx context.Context, chunkHash string) error {
	batch, err := m.chunkOrm.GetChunkByHash(ctx, chunkHash)
	if err != nil {
		return err
	}

	allReady, err := m.chunkOrm.CheckIfBatchChunkProofsAreReady(ctx, batch.BatchHash)
	if err != nil {
		return err
	}
	if allReady {
		err := m.batchOrm.UpdateChunkProofsStatusByBatchHash(ctx, batch.BatchHash, types.ChunkProofsStatusReady)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *ProofReceiverLogic) validator(ctx context.Context, proverTask *orm.ProverTask, pk string, proofMsg *message.ProofMsg, proofParameter coordinatorType.SubmitProofParameter, forkName string) (err error) {
	defer func() {
		if err != nil {
			m.validateFailureTotal.Inc()
		}
	}()

	// Ensure this prover is eligible to participate in the prover task.
	if types.ProverProveStatus(proverTask.ProvingStatus) == types.ProverProofValid ||
		types.ProverProveStatus(proverTask.ProvingStatus) == types.ProverProofInvalid {
		m.validateFailureProverTaskSubmitTwice.Inc()
		// In order to prevent DoS attacks, it is forbidden to repeatedly submit valid proofs.
		// TODO: Defend invalid proof resubmissions by one of the following two methods:
		// (i) slash the prover for each submission of invalid proof
		// (ii) set the maximum failure retry times
		log.Warn(
			"cannot submit valid proof for a prover task twice",
			"taskType", proverTask.TaskType, "hash", proofMsg.ID,
			"proverName", proverTask.ProverName, "proverVersion", proverTask.ProverVersion,
			"proverPublicKey", proverTask.ProverPublicKey, "forkName", forkName,
		)
		return ErrValidatorFailureProverTaskCannotSubmitTwice
	}

	proofTime := time.Since(proverTask.CreatedAt)
	proofTimeSec := uint64(proofTime.Seconds())

	if proofMsg.Status != message.StatusOk {
		// Temporarily replace "panic" with "pa-nic" to prevent triggering the alert based on logs.
		failureMsg := strings.Replace(proofParameter.FailureMsg, "panic", "pa-nic", -1)

		m.proofRecover(ctx, proverTask, types.ProverTaskFailureTypeSubmitStatusNotOk, proofMsg)

		m.validateFailureProverTaskStatusNotOk.Inc()

		log.Info("proof generated by prover failed",
			"taskType", proofMsg.Type, "hash", proofMsg.ID, "proverName", proverTask.ProverName,
			"proverVersion", proverTask.ProverVersion, "proverPublicKey", pk, "failureType", proofParameter.FailureType,
			"failureMessage", failureMsg, "forkName", forkName)
		return ErrValidatorFailureProofMsgStatusNotOk
	}

	// if prover task FailureType is SessionInfoFailureTimeout, the submit proof is timeout, need skip it
	if types.ProverTaskFailureType(proverTask.FailureType) == types.ProverTaskFailureTypeTimeout {
		m.validateFailureProverTaskTimeout.Inc()
		log.Info("proof submit proof have timeout, skip this submit proof", "hash", proofMsg.ID, "taskType", proverTask.TaskType,
			"proverName", proverTask.ProverName, "proverPublicKey", pk, "proofTime", proofTimeSec, "forkName", forkName)
		return ErrValidatorFailureProofTimeout
	}

	// store the proof to prover task
	if updateTaskProofErr := m.updateProverTaskProof(ctx, proverTask, proofMsg); updateTaskProofErr != nil {
		log.Warn("update prover task proof failure", "hash", proofMsg.ID, "proverPublicKey", pk, "forkName", forkName,
			"taskType", proverTask.TaskType, "proverName", proverTask.ProverName, "error", updateTaskProofErr)
	}

	// if the batch/chunk have proved and verifier success, need skip this submit proof
	if m.checkIsTaskSuccess(ctx, proofMsg.ID, proofMsg.Type) {
		m.proofRecover(ctx, proverTask, types.ProverTaskFailureTypeObjectAlreadyVerified, proofMsg)

		m.validateFailureProverTaskHaveVerifier.Inc()
		log.Info("the prove task have proved and verifier success, skip this submit proof", "hash", proofMsg.ID,
			"taskType", proverTask.TaskType, "proverName", proverTask.ProverName, "proverPublicKey", pk, "forkName", forkName)
		return ErrValidatorFailureTaskHaveVerifiedSuccess
	}
	return nil
}

func (m *ProofReceiverLogic) proofRecover(ctx context.Context, proverTask *orm.ProverTask, failureType types.ProverTaskFailureType, proofMsg *message.ProofMsg) {
	log.Info("proof recover update proof status", "hash", proverTask.TaskID, "proverPublicKey", proverTask.ProverPublicKey,
		"taskType", message.ProofType(proverTask.TaskType).String(), "status", types.ProvingTaskUnassigned.String())

	if err := m.updateProofStatus(ctx, proverTask, proofMsg, types.ProverProofInvalid, failureType, 0); err != nil {
		log.Error("failed to updated proof status ProvingTaskUnassigned", "hash", proverTask.TaskID, "pubKey", proverTask.ProverPublicKey, "error", err)
	}
}

func (m *ProofReceiverLogic) closeProofTask(ctx context.Context, proverTask *orm.ProverTask, proofMsg *message.ProofMsg, proofTimeSec uint64) error {
	log.Info("proof close task update proof status", "hash", proverTask.TaskID, "proverPublicKey", proverTask.ProverPublicKey,
		"taskType", message.ProofType(proverTask.TaskType).String(), "status", types.ProvingTaskVerified.String())

	if err := m.updateProofStatus(ctx, proverTask, proofMsg, types.ProverProofValid, types.ProverTaskFailureTypeUndefined, proofTimeSec); err != nil {
		log.Error("failed to updated proof status ProvingTaskVerified", "hash", proverTask.TaskID, "proverPublicKey", proverTask.ProverPublicKey, "error", err)
		return err
	}
	return nil
}

// UpdateProofStatus update the chunk/batch task and session info status
func (m *ProofReceiverLogic) updateProofStatus(ctx context.Context, proverTask *orm.ProverTask,
	proofMsg *message.ProofMsg, status types.ProverProveStatus, failureType types.ProverTaskFailureType, proofTimeSec uint64) error {
	err := m.db.Transaction(func(tx *gorm.DB) error {
		if updateErr := m.proverTaskOrm.UpdateProverTaskProvingStatusAndFailureType(ctx, proverTask.UUID, status, failureType, tx); updateErr != nil {
			log.Error("failed to update prover task proving status and failure type", "uuid", proverTask.UUID, "error", updateErr)
			return updateErr
		}

		switch proofMsg.Type {
		case message.ProofTypeChunk:
			if err := m.chunkOrm.DecreaseActiveAttemptsByHash(ctx, proverTask.TaskID, tx); err != nil {
				log.Error("failed to update chunk proving_status as failed", "hash", proverTask.TaskID, "error", err)
				return err
			}
		case message.ProofTypeBatch:
			if err := m.batchOrm.DecreaseActiveAttemptsByHash(ctx, proverTask.TaskID, tx); err != nil {
				log.Error("failed to update batch proving_status as failed", "hash", proverTask.TaskID, "error", err)
				return err
			}
		}

		// if the block batch has proof verified, so the failed status not update block batch proving status
		if m.checkIsTaskSuccess(ctx, proverTask.TaskID, proofMsg.Type) {
			log.Info("update proof status skip because this chunk/batch has been verified", "hash", proverTask.TaskID, "public key", proverTask.ProverPublicKey)
			return nil
		}

		if status == types.ProverProofValid {
			var storeProofErr error
			switch proofMsg.Type {
			case message.ProofTypeChunk:
				storeProofErr = m.chunkOrm.UpdateProofAndProvingStatusByHash(ctx, proofMsg.ID, proofMsg.ChunkProof, types.ProvingTaskVerified, proofTimeSec, tx)
			case message.ProofTypeBatch:
				storeProofErr = m.batchOrm.UpdateProofAndProvingStatusByHash(ctx, proofMsg.ID, proofMsg.BatchProof, types.ProvingTaskVerified, proofTimeSec, tx)
			}
			if storeProofErr != nil {
				log.Error("failed to store chunk/batch proof and proving status", "hash", proverTask.TaskID, "public key", proverTask.ProverPublicKey, "error", storeProofErr)
				return storeProofErr
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	if status == types.ProverProofValid && proofMsg.Type == message.ProofTypeChunk {
		if checkReadyErr := m.checkAreAllChunkProofsReady(ctx, proverTask.TaskID); checkReadyErr != nil {
			log.Error("failed to check are all chunk proofs ready", "error", checkReadyErr)
			return checkReadyErr
		}
	}

	return nil
}

func (m *ProofReceiverLogic) checkIsTaskSuccess(ctx context.Context, hash string, proofType message.ProofType) bool {
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

func (m *ProofReceiverLogic) updateProverTaskProof(ctx context.Context, proverTask *orm.ProverTask, proofMsg *message.ProofMsg) error {
	// store the proof to prover task
	var proofBytes []byte
	var marshalErr error
	switch proofMsg.Type {
	case message.ProofTypeChunk:
		proofBytes, marshalErr = json.Marshal(proofMsg.ChunkProof)
	case message.ProofTypeBatch:
		proofBytes, marshalErr = json.Marshal(proofMsg.BatchProof)
	}

	if len(proofBytes) == 0 || marshalErr != nil {
		return fmt.Errorf("updateProverTaskProof marshal proof error:%w", marshalErr)
	}
	return m.proverTaskOrm.UpdateProverTaskProof(ctx, proverTask.UUID, proofBytes)
}
