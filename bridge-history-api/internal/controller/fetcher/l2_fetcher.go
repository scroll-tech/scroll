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

// L2MessageFetcher fetches cross message events from L2 and saves them to database.
type L2MessageFetcher struct {
	ctx                 context.Context
	cfg                 *config.FetcherConfig
	db                  *gorm.DB
	client              *ethclient.Client
	l2SyncHeight        uint64
	l2LastSyncBlockHash common.Hash

	eventUpdateLogic *logic.EventUpdateLogic
	l2FetcherLogic   *logic.L2FetcherLogic

	l2MessageFetcherRunningTotal prometheus.Counter
	l2MessageFetcherReorgTotal   prometheus.Counter
	l2MessageFetcherSyncHeight   prometheus.Gauge
}

// NewL2MessageFetcher creates a new L2MessageFetcher instance.
func NewL2MessageFetcher(ctx context.Context, cfg *config.FetcherConfig, db *gorm.DB, client *ethclient.Client) *L2MessageFetcher {
	c := &L2MessageFetcher{
		ctx:              ctx,
		cfg:              cfg,
		db:               db,
		client:           client,
		eventUpdateLogic: logic.NewEventUpdateLogic(db, false),
		l2FetcherLogic:   logic.NewL2FetcherLogic(cfg, db, client),
	}

	reg := prometheus.DefaultRegisterer
	c.l2MessageFetcherRunningTotal = promauto.With(reg).NewCounter(prometheus.CounterOpts{
		Name: "L2_message_fetcher_running_total",
		Help: "Current count of running L2 message fetcher instances.",
	})
	c.l2MessageFetcherReorgTotal = promauto.With(reg).NewCounter(prometheus.CounterOpts{
		Name: "L2_message_fetcher_reorg_total",
		Help: "Total count of blockchain reorgs encountered by the L2 message fetcher.",
	})
	c.l2MessageFetcherSyncHeight = promauto.With(reg).NewGauge(prometheus.GaugeOpts{
		Name: "L2_message_fetcher_sync_height",
		Help: "Latest blockchain height the L2 message fetcher has synced with.",
	})

	return c
}

// Start starts the L2 message fetching process.
func (c *L2MessageFetcher) Start() {
	l2SentMessageSyncedHeight, l2BridgeBatchDepositSyncedHeight, dbErr := c.eventUpdateLogic.GetL2MessageSyncedHeightInDB(c.ctx)
	if dbErr != nil {
		log.Crit("failed to get L2 cross message processed height", "err", dbErr)
		return
	}

	l2SyncHeight := l2SentMessageSyncedHeight
	if l2BridgeBatchDepositSyncedHeight > l2SyncHeight {
		l2SyncHeight = l2BridgeBatchDepositSyncedHeight
	}

	// Sync from an older block to prevent reorg during restart.
	if l2SyncHeight < logic.L2ReorgSafeDepth {
		l2SyncHeight = 0
	} else {
		l2SyncHeight -= logic.L2ReorgSafeDepth
	}

	header, err := c.client.HeaderByNumber(c.ctx, new(big.Int).SetUint64(l2SyncHeight))
	if err != nil {
		log.Crit("failed to get L2 header by number", "block number", l2SyncHeight, "err", err)
		return
	}

	c.updateL2SyncHeight(l2SyncHeight, header.Hash())

	log.Info("Start L2 message fetcher", "l2 sent message synced height", l2SentMessageSyncedHeight,
		"bridge batch deposit synced height", l2BridgeBatchDepositSyncedHeight, "sync start height", l2SyncHeight+1)

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

func (c *L2MessageFetcher) fetchAndSaveEvents(confirmation uint64) {
	startHeight := c.l2SyncHeight + 1
	endHeight, rpcErr := utils.GetBlockNumber(c.ctx, c.client, confirmation)
	if rpcErr != nil {
		log.Error("failed to get L2 block number", "confirmation", confirmation, "err", rpcErr)
		return
	}
	log.Info("fetch and save missing L2 events", "start height", startHeight, "end height", endHeight, "confirmation", confirmation)

	for from := startHeight; from <= endHeight; from += c.cfg.FetchLimit {
		to := from + c.cfg.FetchLimit - 1
		if to > endHeight {
			to = endHeight
		}

		isReorg, resyncHeight, lastBlockHash, l2FetcherResult, fetcherErr := c.l2FetcherLogic.L2Fetcher(c.ctx, from, to, c.l2LastSyncBlockHash)
		if fetcherErr != nil {
			log.Error("failed to fetch L2 events", "from", from, "to", to, "err", fetcherErr)
			return
		}

		if isReorg {
			c.l2MessageFetcherReorgTotal.Inc()
			log.Warn("L2 reorg happened, exit and re-enter fetchAndSaveEvents", "re-sync height", resyncHeight)
			c.updateL2SyncHeight(resyncHeight, lastBlockHash)
			c.l2MessageFetcherRunningTotal.Inc()
			return
		}

		if insertUpdateErr := c.eventUpdateLogic.L2InsertOrUpdate(c.ctx, l2FetcherResult); insertUpdateErr != nil {
			log.Error("failed to save L2 events", "from", from, "to", to, "err", insertUpdateErr)
			return
		}

		if updateErr := c.eventUpdateLogic.UpdateL2WithdrawMessageProofs(c.ctx, c.l2SyncHeight); updateErr != nil {
			log.Error("failed to update L1 batch index and status", "from", from, "to", to, "err", updateErr)
			return
		}

		if updateErr := c.eventUpdateLogic.UpdateL2BridgeBatchDepositEvent(c.ctx, l2FetcherResult.BridgeBatchDepositMessage); updateErr != nil {
			log.Error("failed to update L1 batch index and status", "from", from, "to", to, "err", updateErr)
			return
		}

		c.updateL2SyncHeight(to, lastBlockHash)
		c.l2MessageFetcherRunningTotal.Inc()
	}
}

func (c *L2MessageFetcher) updateL2SyncHeight(height uint64, blockHash common.Hash) {
	c.l2MessageFetcherSyncHeight.Set(float64(height))
	c.l2LastSyncBlockHash = blockHash
	c.l2SyncHeight = height
}
