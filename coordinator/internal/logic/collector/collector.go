package collector

import (
	"context"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"
	gethMetrics "github.com/scroll-tech/go-ethereum/metrics"

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

// HashTaskPublicKey hash public key pair
type HashTaskPublicKey struct {
	Attempt int
	PubKey  string
}

// BaseCollector a base collector which contain series functions
type BaseCollector struct {
	cfg *config.Config
	ctx context.Context

	batchOrm      *orm.Batch
	chunkOrm      *orm.Chunk
	blockOrm      *orm.L2Block
	proverTaskOrm *orm.ProverTask
}

// checkAttempts use the count of prover task info to check the attempts
func (b *BaseCollector) checkAttemptsExceeded(hash string) bool {
	whereFields := make(map[string]interface{})
	whereFields["hash"] = hash
	proverTasks, err := b.proverTaskOrm.GetProverTasks(whereFields, nil, 0)
	if err != nil {
		log.Error("get session info error", "hash id", hash, "error", err)
		return true
	}

	if len(proverTasks) >= b.cfg.SessionAttempts {
		log.Warn("proof generation prover task %s ended because reach the max attempts", hash)

		var isAllFailed bool
		for _, proverTask := range proverTasks {
			if types.ProvingStatus(proverTask.ProvingStatus) != types.ProvingTaskFailed {
				isAllFailed = false
			}

			if types.ProvingStatus(proverTask.ProvingStatus) == types.ProvingTaskFailed {
				rollermanager.Manager.FreeTaskIDForRoller(proverTask.ProverPublicKey, hash)
			}
		}

		if isAllFailed {
			// Set status as skipped.
			// Note that this is only a workaround for testnet here.
			// TODO: In real cases we should reset to orm.ProvingTaskUnassigned
			// so as to re-distribute the task in the future

			if message.ProofType(proverTasks[0].TaskType) == message.ProofTypeChunk {
				if err := b.chunkOrm.UpdateProvingStatus(b.ctx, hash, types.ProvingTaskFailed); err != nil {
					log.Error("failed to update chunk proving_status as failed", "msg.ID", hash, "error", err)
				}
			}
			if message.ProofType(proverTasks[0].TaskType) == message.ProofTypeBatch {
				if err := b.batchOrm.UpdateProvingStatus(b.ctx, hash, types.ProvingTaskFailed); err != nil {
					log.Error("failed to update batch proving_status as failed", "msg.ID", hash, "error", err)
				}
			}
			coordinatorSessionsTimeoutTotalCounter.Inc(1)
		}

		return false
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
	for i := uint8(0); i < b.cfg.RollersPerSession; i++ {
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
