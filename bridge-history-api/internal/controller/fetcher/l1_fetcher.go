package fetcher

import (
	"context"
	"time"

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
	cfg    *config.LayerConfig
	client *ethclient.Client

	syncInfo     *SyncInfo
	l1ScanHeight uint64

	eventUpdateLogic *logic.EventUpdateLogic
	l1FetcherLogic   *logic.L1FetcherLogic
}

// NewL1MessageFetcher creates a new L1MessageFetcher instance.
func NewL1MessageFetcher(ctx context.Context, cfg *config.LayerConfig, db *gorm.DB, client *ethclient.Client, syncInfo *SyncInfo) (*L1MessageFetcher, error) {
	return &L1MessageFetcher{
		ctx:              ctx,
		cfg:              cfg,
		client:           client,
		syncInfo:         syncInfo,
		eventUpdateLogic: logic.NewEventUpdateLogic(db),
		l1FetcherLogic:   logic.NewL1FetcherLogic(cfg, db, client),
	}, nil
}

// Start starts the L1 message fetching process.
func (c *L1MessageFetcher) Start() {
	messageSyncedHeight, batchSyncedHeight, err := c.eventUpdateLogic.GetL1SyncHeight(c.ctx)
	if err != nil {
		log.Crit("L1MessageFetcher start failed", "error", err)
	}

	c.l1ScanHeight = messageSyncedHeight
	if batchSyncedHeight > c.l1ScanHeight {
		c.l1ScanHeight = batchSyncedHeight
	}
	if c.cfg.StartHeight > c.l1ScanHeight {
		c.l1ScanHeight = c.cfg.StartHeight - 1
	}

	log.Info("Start L1 message fetcher", "message synced height", messageSyncedHeight, "batch synced height", batchSyncedHeight, "config start height", c.cfg.StartHeight)

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
	startHeight := c.l1ScanHeight + 1
	endHeight, err := utils.GetBlockNumber(c.ctx, c.client, confirmation)
	if err != nil {
		log.Error("failed to get L1 safe block number", "err", err)
		return
	}
	log.Info("fetch and save missing L1 events", "start height", startHeight, "end height", endHeight)

	for from := startHeight; from <= endHeight; from += c.cfg.FetchLimit {
		to := from + c.cfg.FetchLimit - 1
		if to > endHeight {
			to = endHeight
		}

		fetcherResult, fetcherErr := c.l1FetcherLogic.L1Fetcher(c.ctx, from, to)
		if fetcherErr != nil {
			log.Error("failed to fetch L1 events", "from", from, "to", to, "err", err)
			return
		}

		if insertUpdateErr := c.eventUpdateLogic.L1InsertOrUpdate(c.ctx, fetcherResult); insertUpdateErr != nil {
			log.Error("failed to save L1 events", "from", from, "to", to, "err", err)
			return
		}
		c.l1ScanHeight = to

		l2ScannedHeight := c.syncInfo.GetL2ScanHeight()
		if l2ScannedHeight == 0 {
			log.Error("L2 fetcher has not successfully synced at least one round yet")
			return
		}

		if updateErr := c.eventUpdateLogic.UpdateL1BatchIndexAndStatus(c.ctx, l2ScannedHeight); updateErr != nil {
			log.Error("failed to update L1 batch index and status", "from", from, "to", to, "err", err)
			return
		}
	}
}
