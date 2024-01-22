package fetcher

import (
	"context"
	"math/big"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/bridge-history-api/internal/config"
	"scroll-tech/bridge-history-api/internal/logic"
	"scroll-tech/bridge-history-api/internal/utils"
)

// L1MessageFetcher fetches cross message events from L1 and saves them to database.
type L1MessageFetcher struct {
	ctx    context.Context
	cfg    *config.FetcherConfig
	client *ethclient.Client

	l1SyncHeight        uint64
	l1LastSyncBlockHash common.Hash

	eventUpdateLogic *logic.EventUpdateLogic
	l1FetcherLogic   *logic.L1FetcherLogic

	l1MessageFetcherRunningTotal prometheus.Counter
	l1MessageFetcherReorgTotal   prometheus.Counter
	l1MessageFetcherSyncHeight   prometheus.Gauge
}

// NewL1MessageFetcher creates a new L1MessageFetcher instance.
func NewL1MessageFetcher(ctx context.Context, cfg *config.FetcherConfig, db *gorm.DB, client *ethclient.Client) *L1MessageFetcher {
	c := &L1MessageFetcher{
		ctx:              ctx,
		cfg:              cfg,
		client:           client,
		eventUpdateLogic: logic.NewEventUpdateLogic(db, true),
		l1FetcherLogic:   logic.NewL1FetcherLogic(cfg, db, client),
	}

	reg := prometheus.DefaultRegisterer
	c.l1MessageFetcherRunningTotal = promauto.With(reg).NewCounter(prometheus.CounterOpts{
		Name: "L1_message_fetcher_running_total",
		Help: "Current count of running L1 message fetcher instances.",
	})
	c.l1MessageFetcherReorgTotal = promauto.With(reg).NewCounter(prometheus.CounterOpts{
		Name: "L1_message_fetcher_reorg_total",
		Help: "Total count of blockchain reorgs encountered by the L1 message fetcher.",
	})
	c.l1MessageFetcherSyncHeight = promauto.With(reg).NewGauge(prometheus.GaugeOpts{
		Name: "L1_message_fetcher_sync_height",
		Help: "Latest blockchain height the L1 message fetcher has synced with.",
	})

	return c
}

// Start starts the L1 message fetching process.
func (c *L1MessageFetcher) Start() {
	messageSyncedHeight, batchSyncedHeight, dbErr := c.eventUpdateLogic.GetL1SyncHeight(c.ctx)
	if dbErr != nil {
		log.Crit("L1MessageFetcher start failed", "err", dbErr)
	}

	l1SyncHeight := messageSyncedHeight
	if batchSyncedHeight > l1SyncHeight {
		l1SyncHeight = batchSyncedHeight
	}
	if c.cfg.StartHeight > l1SyncHeight {
		l1SyncHeight = c.cfg.StartHeight - 1
	}

	// Sync from an older block to prevent reorg during restart.
	if l1SyncHeight < logic.L1ReorgSafeDepth {
		l1SyncHeight = 0
	} else {
		l1SyncHeight -= logic.L1ReorgSafeDepth
	}

	header, err := c.client.HeaderByNumber(c.ctx, new(big.Int).SetUint64(l1SyncHeight))
	if err != nil {
		log.Crit("failed to get L1 header by number", "block number", l1SyncHeight, "err", err)
		return
	}

	c.updateL1SyncHeight(l1SyncHeight, header.Hash())

	log.Info("Start L1 message fetcher", "message synced height", messageSyncedHeight, "batch synced height", batchSyncedHeight, "config start height", c.cfg.StartHeight, "sync start height", c.l1SyncHeight+1)

	tick := time.NewTicker(time.Duration(c.cfg.BlockTime) * time.Second)
	go func() {
		for {
			select {
			case <-c.ctx.Done():
				tick.Stop()
				return
			case <-tick.C:
				c.fetchAndSaveEvents(c.cfg.Confirmation)
			}
		}
	}()
}

func (c *L1MessageFetcher) fetchAndSaveEvents(confirmation uint64) {
	c.l1MessageFetcherRunningTotal.Inc()
	startHeight := c.l1SyncHeight + 1
	endHeight, rpcErr := utils.GetBlockNumber(c.ctx, c.client, confirmation)
	if rpcErr != nil {
		log.Error("failed to get L1 block number", "confirmation", confirmation, "err", rpcErr)
		return
	}

	log.Info("fetch and save missing L1 events", "start height", startHeight, "end height", endHeight, "confirmation", confirmation)

	for from := startHeight; from <= endHeight; from += c.cfg.FetchLimit {
		to := from + c.cfg.FetchLimit - 1
		if to > endHeight {
			to = endHeight
		}

		isReorg, resyncHeight, lastBlockHash, l1FetcherResult, fetcherErr := c.l1FetcherLogic.L1Fetcher(c.ctx, from, to, c.l1LastSyncBlockHash)
		if fetcherErr != nil {
			log.Error("failed to fetch L1 events", "from", from, "to", to, "err", fetcherErr)
			return
		}

		if isReorg {
			c.l1MessageFetcherReorgTotal.Inc()
			log.Warn("L1 reorg happened, exit and re-enter fetchAndSaveEvents", "re-sync height", resyncHeight)
			c.updateL1SyncHeight(resyncHeight, lastBlockHash)
			return
		}

		if insertUpdateErr := c.eventUpdateLogic.L1InsertOrUpdate(c.ctx, l1FetcherResult); insertUpdateErr != nil {
			log.Error("failed to save L1 events", "from", from, "to", to, "err", insertUpdateErr)
			return
		}

		c.updateL1SyncHeight(to, lastBlockHash)
	}
}

func (c *L1MessageFetcher) updateL1SyncHeight(height uint64, blockHash common.Hash) {
	c.l1MessageFetcherSyncHeight.Set(float64(height))
	c.l1LastSyncBlockHash = blockHash
	c.l1SyncHeight = height
}
