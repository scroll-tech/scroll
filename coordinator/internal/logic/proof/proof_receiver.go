package proof

import (
	"context"
	"encoding/json"
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
	validatorFailureProofMsgStatusNotOk     = errors.New("validator failure proof msg status not ok")
	validatorFailureRollerEmpty             = errors.New("validator failure get none rollers for the proof")
	validatorFailureRollerInfoHasProofValid = errors.New("validator failure roller info has proof valid")
)

type ProofReceiver struct {
	blockBatchOrm  *orm.BlockBatch
	aggTaskOrm     *orm.AggTask
	sessionInfoOrm *orm.SessionInfo

	verifier verifier.Verifier
	cfg      *config.Config
}

func NewProofReceiver(cfg *config.Config, db *gorm.DB) *ProofReceiver {
	return &ProofReceiver{
		cfg:            cfg,
		blockBatchOrm:  orm.NewBlockBatch(db),
		aggTaskOrm:     orm.NewAggTask(db),
		sessionInfoOrm: orm.NewSessionInfo(db),
	}
}

// HandleZkProof handle a ZkProof submitted from a roller.
// For now only proving/verifying error will lead to setting status as skipped.
// db/unmarshal errors will not because they are errors on the business logic side.
func (m *ProofReceiver) HandleZkProof(ctx context.Context, proofMsg *message.ProofMsg) error {
	pk, _ := proofMsg.PublicKey()
	rollerInfo, ok := rollermanager.Manager.RollersInfo(pk, proofMsg.ID)
	if !ok {
		m.proofFailure(proofMsg.Type, proofMsg.ID)
		return fmt.Errorf("proof generation session for id %v does not existID", proofMsg.ID)
	}

	if err := m.validator(pk, rollerInfo, proofMsg); err != nil {
		m.proofFailure(proofMsg.Type, proofMsg.ID)
		return nil
	}

	proofTime := time.Since(time.Unix(rollerInfo.StartTimestamp, 0))
	proofTimeSec := uint64(proofTime.Seconds())

	proofByt, err := json.Marshal(proofMsg.Proof)
	if err != nil {
		m.proofFailure(proofMsg.Type, proofMsg.ID)
		return err
	}

	// store proof content
	var storeProofErr error
	switch proofMsg.Type {
	case message.BasicProve:
		storeProofErr = m.blockBatchOrm.UpdateProofAndHashByHash(ctx, proofMsg.ID, proofByt, proofTimeSec, types.ProvingTaskProved)
	case message.AggregatorProve:
		storeProofErr = m.aggTaskOrm.UpdateProofForAggTask(proofMsg.ID, proofByt)
	}
	if storeProofErr != nil {
		m.proofFailure(proofMsg.Type, proofMsg.ID)
		log.Error("failed to store basic proof into db", "error", storeProofErr)
		return storeProofErr
	}

	coordinatorProofsReceivedTotalCounter.Inc(1)

	// TODO: wrap both basic verifier and aggregator verifier
	success, verifyErr := m.verifier.VerifyProof(proofMsg.Proof)
	if verifyErr != nil || !success {
		m.proofFailure(proofMsg.Type, proofMsg.ID)

		// TODO: this is only a temp workaround for testnet, we should return err in real cases
		log.Error("Failed to verify zk proof", "proof id", proofMsg.ID, "roller pk", pk, "prove type",
			proofMsg.Type, "proof time", proofTimeSec, "error", verifyErr)

		// TODO: Roller needs to be slashed if proof is invalid.
		coordinatorProofsVerifiedFailedTimeTimer.Update(proofTime)

		rollermanager.Manager.UpdateMetricRollerProofsVerifiedFailedTimeTimer(pk, proofTime)

		log.Info("proof verified by coordinator failed", "proof id", proofMsg.ID, "roller name", "roller pk",
			pk, "prove type", proofMsg.Type, "proof time", proofTimeSec, "error", verifyErr)
		return nil
	}

	if err := m.closeProofTask(proofMsg, rollerInfo); err != nil {
		if proofMsg.Type == message.BasicProve {
			if err := m.blockBatchOrm.UpdateProvingStatus(proofMsg.ID, types.ProvingTaskUnassigned); err != nil {
				log.Error("fail to reset basic task status as Unassigned", "msg.ID", proofMsg.ID)
			}
		}
		if proofMsg.Type == message.AggregatorProve {
			if err := m.aggTaskOrm.UpdateAggTaskStatus(proofMsg.ID, types.ProvingTaskUnassigned); err != nil {
				log.Error("fail to reset aggregator task status as Unassigned", "msg.ID", proofMsg.ID)
			}
		}
	}

	coordinatorProofsVerifiedSuccessTimeTimer.Update(proofTime)
	rollermanager.Manager.UpdateMetricRollerProofsVerifiedSuccessTimeTimer(pk, proofTime)

	return nil
}

func (m *ProofReceiver) validator(pk string, rollersInfo *coordinatorType.RollersInfo, proofMsg *message.ProofMsg) error {
	pubKey, _ := proofMsg.PublicKey()
	rollermanager.Manager.UpdateMetricRollerProofsLastFinishedTimestampGauge(pubKey)

	// Ensure this roller is eligible to participate in the session.
	rollers, ok := rollersInfo.Rollers[pk]
	if !ok {
		log.Error("get none rollers for the proof key", rollers.PublicKey, "id", proofMsg.ID)
		return validatorFailureRollerEmpty
	}

	if rollers.Status == types.RollerProofValid {
		// In order to prevent DoS attacks, it is forbidden to repeatedly submit valid proofs.
		// TODO: Defend invalid proof resubmissions by one of the following two methods:
		// (i) slash the roller for each submission of invalid proof
		// (ii) set the maximum failure retry times
		log.Warn("roller has already submitted valid proof in proof session", "roller name", rollers.Name, "roller pk",
			rollers.PublicKey, "prove type", rollersInfo.ProveType, "proof id", proofMsg.ID)
		return validatorFailureRollerInfoHasProofValid
	}

	proofTime := time.Since(time.Unix(rollersInfo.StartTimestamp, 0))
	proofTimeSec := uint64(proofTime.Seconds())

	log.Info("handling zk proof", "proof id", proofMsg.ID, "roller name", rollers.Name, "roller pk", rollers.PublicKey,
		"prove type", rollersInfo.ProveType, "proof time", proofTimeSec)

	if proofMsg.Status != message.StatusOk {
		coordinatorProofsGeneratedFailedTimeTimer.Update(proofTime)
		rollermanager.Manager.UpdateMetricRollerProofsGeneratedFailedTimeTimer(rollers.PublicKey, proofTime)
		log.Info("proof generated by roller failed", "proof id", proofMsg.ID, "roller name", rollers.Name, "roller pk",
			rollers.PublicKey, "prove type", proofMsg.Type, "proof time", proofTimeSec, "error", proofMsg.Error)
		return validatorFailureProofMsgStatusNotOk
	}
	return nil
}

func (m *ProofReceiver) proofFailure(proofMsgType message.ProveType, taskId string) {
	switch proofMsgType {
	case message.BasicProve:
		if err := m.blockBatchOrm.UpdateProvingStatus(taskId, types.ProvingTaskFailed); err != nil {
			log.Error("failed to update basic proving_status as failed", "msg.ID", taskId, "error", err)
		}
	case message.AggregatorProve:
		if err := m.aggTaskOrm.UpdateAggTaskStatus(taskId, types.ProvingTaskFailed); err != nil {
			log.Error("failed to update aggregator proving_status as failed", "msg.ID", taskId, "error", err)
		}
	}

	coordinatorSessionsFailedTotalCounter.Inc(1)
}

func (m *ProofReceiver) closeProofTask(proofMsg *message.ProofMsg, rollersInfo *coordinatorType.RollersInfo) error {
	switch proofMsg.Type {
	case message.AggregatorProve:
		dbErr := m.aggTaskOrm.UpdateAggTaskStatus(proofMsg.ID, types.ProvingTaskVerified)
		if dbErr != nil {
			log.Error("failed to update aggregator proving_status", "msg.ID", proofMsg.ID, "status", types.ProvingTaskVerified, "error", dbErr)
			return dbErr
		}
	case message.BasicProve:
		dbErr := m.blockBatchOrm.UpdateProvingStatus(proofMsg.ID, types.ProvingTaskVerified)
		if dbErr != nil {
			log.Error("failed to update basic proving_status", "msg.ID", proofMsg.ID, "status", types.ProvingTaskVerified, "error", dbErr)
			return dbErr
		}
	}

	pk, _ := proofMsg.PublicKey()

	log.Info("proof verified by coordinator success", "proof id", proofMsg.ID, "roller pk", pk, "prove type", proofMsg.Type)

	if err := m.sessionInfoOrm.InsertSessionInfo(rollersInfo); err != nil {
		log.Error("db set session info fail", "pk", pk, "error", err)
	}

	for pk := range rollersInfo.Rollers {
		rollermanager.Manager.FreeTaskIDForRoller(pk, rollersInfo.ID)
	}
	return nil
}
