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

	eventUpdateLogic     *logic.EventUpdateLogic
	l1FetcherLogic       *logic.L1FetcherLogic
	l1ReorgHandlingLogic *logic.L1ReorgHandlingLogic
}

// NewL1MessageFetcher creates a new L1MessageFetcher instance.
func NewL1MessageFetcher(ctx context.Context, cfg *config.LayerConfig, db *gorm.DB, client *ethclient.Client, syncInfo *SyncInfo) (*L1MessageFetcher, error) {
	return &L1MessageFetcher{
		ctx:                  ctx,
		cfg:                  cfg,
		client:               client,
		syncInfo:             syncInfo,
		eventUpdateLogic:     logic.NewEventUpdateLogic(db),
		l1FetcherLogic:       logic.NewL1FetcherLogic(cfg, db, client),
		l1ReorgHandlingLogic: logic.NewL1ReorgHandlingLogic(db, client),
	}, nil
}

// Start starts the L1 message fetching process.
func (c *L1MessageFetcher) Start() {
	messageSyncedHeight, batchSyncedHeight, dbErr := c.eventUpdateLogic.GetL1SyncHeight(c.ctx)
	if dbErr != nil {
		log.Crit("L1MessageFetcher start failed", "err", dbErr)
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
	endHeight, rpcErr := utils.GetBlockNumber(c.ctx, c.client, confirmation)
	if rpcErr != nil {
		log.Error("failed to get L1 safe block number", "err", rpcErr)
		return
	}
	log.Info("fetch and save missing L1 events", "start height", startHeight, "end height", endHeight)

	for from := startHeight; from <= endHeight; from += c.cfg.FetchLimit {
		to := from + c.cfg.FetchLimit - 1
		if to > endHeight {
			to = endHeight
		}

		if endHeight-to >= logic.L1ReorgSafeDepth {
			isReorg, resyncHeight, handleErr := c.l1ReorgHandlingLogic.HandleL1Reorg(c.ctx)
			if handleErr != nil {
				log.Error("failed to Handle L1 Reorg", "err", handleErr)
				return
			}

			if isReorg {
				c.l1ScanHeight = resyncHeight
				log.Warn("L1 reorg happened, exit and re-enter fetchAndSaveEvents", "restart height", c.l1ScanHeight)
				return
			}
		}

		fetcherResult, fetcherErr := c.l1FetcherLogic.L1Fetcher(c.ctx, from, to)
		if fetcherErr != nil {
			log.Error("failed to fetch L1 events", "from", from, "to", to, "err", fetcherErr)
			return
		}

		if insertUpdateErr := c.eventUpdateLogic.L1InsertOrUpdate(c.ctx, fetcherResult); insertUpdateErr != nil {
			log.Error("failed to save L1 events", "from", from, "to", to, "err", insertUpdateErr)
			return
		}
		c.l1ScanHeight = to

		l2ScannedHeight := c.syncInfo.GetL2ScanHeight()
		if l2ScannedHeight == 0 {
			log.Error("L2 fetcher has not successfully synced at least one round yet")
			return
		}

		if updateErr := c.eventUpdateLogic.UpdateL1BatchIndexAndStatus(c.ctx, l2ScannedHeight); updateErr != nil {
			log.Error("failed to update L1 batch index and status", "from", from, "to", to, "err", updateErr)
			return
		}
	}
}
