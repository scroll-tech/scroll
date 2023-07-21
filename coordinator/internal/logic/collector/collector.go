package collector

import (
	"context"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"
	gethMetrics "github.com/scroll-tech/go-ethereum/metrics"
	"gorm.io/gorm"

	"scroll-tech/common/metrics"
	"scroll-tech/common/types"
	"scroll-tech/common/types/message"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/logic/rollermanager"
	"scroll-tech/coordinator/internal/orm"
	coordinatorType "scroll-tech/coordinator/internal/types"
)

const (
	// BatchCollectorName the name of batch collector
	BatchCollectorName = "batch_collector"
	// ChunkCollectorName the name of chunk collector
	ChunkCollectorName = "chunk_collector"
)

var coordinatorSessionsTimeoutTotalCounter = gethMetrics.NewRegisteredCounter("coordinator/sessions/timeout/total", metrics.ScrollRegistry)

// Collector the interface of a collector who send data to prover
type Collector interface {
	Name() string
	Collect(ctx context.Context) error
}

// BaseCollector a base collector which contain series functions
type BaseCollector struct {
	cfg *config.Config
	ctx context.Context
	db  *gorm.DB

	batchOrm      *orm.Batch
	chunkOrm      *orm.Chunk
	blockOrm      *orm.L2Block
	proverTaskOrm *orm.ProverTask
}

// checkAttempts use the count of prover task info to check the attempts
func (b *BaseCollector) checkAttemptsExceeded(hash string, taskType message.ProofType) bool {
	whereFields := make(map[string]interface{})
	whereFields["task_id"] = hash
	whereFields["task_type"] = int16(taskType)
	proverTasks, err := b.proverTaskOrm.GetProverTasks(whereFields, nil, 0)
	if err != nil {
		log.Error("get prover task error", "hash id", hash, "error", err)
		return true
	}

	if len(proverTasks) >= int(b.cfg.RollerManagerConfig.SessionAttempts) {
		coordinatorSessionsTimeoutTotalCounter.Inc(1)

		log.Warn("proof generation prover task %s ended because reach the max attempts", hash)

		for _, proverTask := range proverTasks {
			if types.ProvingStatus(proverTask.ProvingStatus) == types.ProvingTaskFailed {
				rollermanager.Manager.FreeTaskIDForRoller(proverTask.ProverPublicKey, hash)
			}
		}

		transErr := b.db.Transaction(func(tx *gorm.DB) error {
			switch message.ProofType(proverTasks[0].TaskType) {
			case message.ProofTypeChunk:
				if err := b.chunkOrm.UpdateProvingStatus(b.ctx, hash, types.ProvingTaskFailed, tx); err != nil {
					log.Error("failed to update chunk proving_status as failed", "msg.ID", hash, "error", err)
				}
			case message.ProofTypeBatch:
				if err := b.batchOrm.UpdateProvingStatus(b.ctx, hash, types.ProvingTaskFailed, tx); err != nil {
					log.Error("failed to update batch proving_status as failed", "msg.ID", hash, "error", err)
				}
			}
			// update the prover task status to let timeout checker don't check it.
			if err := b.proverTaskOrm.UpdateAllProverTaskProvingStatusOfTaskID(b.ctx, message.ProofType(proverTasks[0].TaskType), hash, types.RollerProofInvalid, tx); err != nil {
				log.Error("failed to update prover task proving_status as failed", "msg.ID", hash, "error", err)
			}
			return nil
		})
		if transErr == nil {
			return false
		}
	}
	return true
}

func (b *BaseCollector) sendTask(proveType message.ProofType, hash string, blockHashes []common.Hash, subProofs []*message.AggProof) ([]*coordinatorType.RollerStatus, error) {
	sendMsg := &message.TaskMsg{
		ID:          hash,
		Type:        proveType,
		BlockHashes: blockHashes,
		SubProofs:   subProofs,
	}

	var err error
	var rollerStatusList []*coordinatorType.RollerStatus
	for i := uint8(0); i < b.cfg.RollerManagerConfig.RollersPerSession; i++ {
		rollerPubKey, rollerName, sendErr := rollermanager.Manager.SendTask(proveType, sendMsg)
		if sendErr != nil {
			err = sendErr
			continue
		}

		rollermanager.Manager.UpdateMetricRollerProofsLastAssignedTimestampGauge(rollerPubKey)

		rollerStatus := &coordinatorType.RollerStatus{
			PublicKey: rollerPubKey,
			Name:      rollerName,
			Status:    types.RollerAssigned,
		}
		rollerStatusList = append(rollerStatusList, rollerStatus)
	}

	if err != nil {
		return nil, err
	}
	return rollerStatusList, nil
}
