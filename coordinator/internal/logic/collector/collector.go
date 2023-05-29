package collector

import (
	"context"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"
	gethMetrics "github.com/scroll-tech/go-ethereum/metrics"

	"scroll-tech/common/metrics"
	"scroll-tech/common/types"
	"scroll-tech/common/types/message"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/logic/roller_manager"
	"scroll-tech/coordinator/internal/orm"
	coordinatorType "scroll-tech/coordinator/internal/types"
)

const (
	AggTaskCollectorName    = "agg_task_collector"
	BlockBatchCollectorName = "block_batch_collector"
)

var coordinatorSessionsTimeoutTotalCounter = gethMetrics.NewRegisteredCounter("coordinator/sessions/timeout/total", metrics.ScrollRegistry)

// Collector the interface of a collector who send data to prover
type Collector interface {
	Name() string
	Collect(ctx context.Context) error
}

type HashTaskPublicKey struct {
	Attempt int
	PubKey  string
}

type BaseCollector struct {
	cfg   *config.Config
	cache *cache.Cache

	aggTaskOrm     *orm.AggTask
	blockBatchOrm  *orm.BlockBatch
	blockTraceOrm  *orm.BlockTrace
	sessionInfoOrm *orm.SessionInfo
}

func (b *BaseCollector) checkAttempts(hash string) bool {
	val, ok := b.cache.Get(hash)
	if !ok {
		hashPk := &HashTaskPublicKey{
			Attempt: 1,
		}
		b.cache.SetDefault(hash, hashPk)
		return true
	}

	hashTaskPk, isHashPk := val.(*HashTaskPublicKey)
	if !isHashPk {
		log.Warn("store task id:%s which attempt count is not int", hash)
		return true
	}

	if hashTaskPk.Attempt >= b.cfg.RollerManagerConfig.SessionAttempts {
		log.Warn("proof generation session %s ended because reach the max attempts", hash)
		// Set status as skipped.
		// Note that this is only a workaround for testnet here.
		// TODO: In real cases we should reset to orm.ProvingTaskUnassigned
		// so as to re-distribute the task in the future
		if err := b.blockBatchOrm.UpdateProvingStatus(hash, types.ProvingTaskFailed); err != nil {
			log.Error("fail to reset basic task_status as Unassigned", "id", hash, "err", err)
		}

		roller_manager.Manager.FreeTaskIDForRoller(hashTaskPk.PubKey, hash)
		coordinatorSessionsTimeoutTotalCounter.Inc(1)
		return false
	}
	hashTaskPk.Attempt += 1
	b.cache.SetDefault(hash, hashTaskPk)
	return true
}

func (b *BaseCollector) sendTask(proveType message.ProveType, taskId string, traces []common.Hash, subProofs []*message.AggProof) (map[string]*coordinatorType.RollerStatus, error) {
	var err1 error
	rollers := make(map[string]*coordinatorType.RollerStatus)
	for i := 0; i < int(b.cfg.RollerManagerConfig.RollersPerSession); i++ {
		sendMsg := &message.TaskMsg{
			ID:          taskId,
			Type:        proveType,
			BlockHashes: traces,
			SubProofs:   subProofs,
		}

		rollerPubKey, rollerName, sendErr := roller_manager.Manager.SendTask(proveType, sendMsg)
		if sendErr != nil {
			err1 = sendErr
			continue
		}

		roller_manager.Manager.UpdateMetricRollerProofsLastAssignedTimestampGauge(rollerPubKey)

		rollerStatus := &coordinatorType.RollerStatus{
			PublicKey: rollerPubKey,
			Name:      rollerName,
			Status:    types.RollerAssigned,
		}
		rollers[rollerPubKey] = rollerStatus

		if val, ok := b.cache.Get(taskId); ok {
			if hashTaskPk, isHashTaskPk := val.(*HashTaskPublicKey); !isHashTaskPk {
				hashTaskPk.PubKey = rollerPubKey
				b.cache.SetDefault(taskId, hashTaskPk)
			}
		}
	}

	rollersInfo := &coordinatorType.RollersInfo{
		ID:             taskId,
		Rollers:        rollers,
		ProveType:      message.BasicProve,
		StartTimestamp: time.Now().Unix(),
	}
	roller_manager.Manager.AddRollerInfo(rollersInfo)

	return rollers, err1
}
