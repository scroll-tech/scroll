package collector

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/patrickmn/go-cache"
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

var coordinatorSessionsTimeoutTotalCounter = gethMetrics.NewRegisteredCounter("coordinator/sessions/timeout/total", metrics.ScrollRegistry)

// Collector the interface of a collector who send data to prover
type Collector interface {
	Type() message.ProveType
	Collect(ctx context.Context) error
	Start()
	Pause()
	IsPaused() bool
}

// HashTaskPublicKey hash public key pair
type HashTaskPublicKey struct {
	Attempt int
	PubKey  string
}

// BaseCollector a base collector which contain series functions
type BaseCollector struct {
	cfg   *config.Config
	cache *cache.Cache

	isPaused atomic.Bool //nolint:typecheck

	aggTaskOrm     *orm.AggTask
	blockBatchOrm  *orm.BlockBatch
	blockTraceOrm  *orm.BlockTrace
	sessionInfoOrm *orm.SessionInfo
}

func (b *BaseCollector) Start() {
	b.isPaused.Store(false)
}

func (b *BaseCollector) Pause() {
	b.isPaused.Store(true)
}

func (b *BaseCollector) IsPaused() bool {
	return b.isPaused.Load()
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

		rollermanager.Manager.FreeTaskIDForRoller(hashTaskPk.PubKey, hash)
		coordinatorSessionsTimeoutTotalCounter.Inc(1)
		return false
	}
	hashTaskPk.Attempt++
	b.cache.SetDefault(hash, hashTaskPk)
	return true
}

func (b *BaseCollector) sendTask(proveType message.ProveType, taskID string, traces []common.Hash, subProofs []*message.AggProof) (map[string]*coordinatorType.RollerStatus, error) {
	var err1 error
	rollers := make(map[string]*coordinatorType.RollerStatus)
	for i := 0; i < int(b.cfg.RollerManagerConfig.RollersPerSession); i++ {
		sendMsg := &message.TaskMsg{
			ID:          taskID,
			Type:        proveType,
			BlockHashes: traces,
			SubProofs:   subProofs,
		}

		rollerPubKey, rollerName, sendErr := rollermanager.Manager.SendTask(proveType, sendMsg)
		if sendErr != nil {
			err1 = sendErr
			continue
		}

		rollermanager.Manager.UpdateMetricRollerProofsLastAssignedTimestampGauge(rollerPubKey)

		rollerStatus := &coordinatorType.RollerStatus{
			PublicKey: rollerPubKey,
			Name:      rollerName,
			Status:    types.RollerAssigned,
		}
		rollers[rollerPubKey] = rollerStatus

		if val, ok := b.cache.Get(taskID); ok {
			if hashTaskPk, isHashTaskPk := val.(*HashTaskPublicKey); !isHashTaskPk {
				hashTaskPk.PubKey = rollerPubKey
				b.cache.SetDefault(taskID, hashTaskPk)
			}
		}
	}

	rollersInfo := &coordinatorType.RollersInfo{
		ID:             taskID,
		Rollers:        rollers,
		ProveType:      message.BasicProve,
		StartTimestamp: time.Now().Unix(),
	}
	rollermanager.Manager.AddRollerInfo(rollersInfo)

	return rollers, err1
}
