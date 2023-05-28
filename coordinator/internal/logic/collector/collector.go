package collector

import (
	"context"

	"github.com/patrickmn/go-cache"
	"github.com/scroll-tech/go-ethereum/log"
	gethMetrics "github.com/scroll-tech/go-ethereum/metrics"

	"scroll-tech/common/metrics"
	"scroll-tech/common/types"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/logic/roller_manager"
	"scroll-tech/coordinator/internal/orm"
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

type BasicCollector struct {
	cfg   *config.Config
	cache *cache.Cache

	blockBatchOrm  *orm.BlockBatch
	blockTraceOrm  *orm.BlockTrace
	sessionInfoOrm *orm.SessionInfo
}

func (b *BasicCollector) checkAttempts(hash string) bool {
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
