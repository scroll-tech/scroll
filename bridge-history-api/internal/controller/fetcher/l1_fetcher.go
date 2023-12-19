package fetcher

import (
	"context"
	"math/big"
	"time"

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
	cfg    *config.LayerConfig
	client *ethclient.Client

	syncInfo            *SyncInfo
	l1SyncHeight        uint64
	l1LastSyncBlockHash common.Hash

	eventUpdateLogic *logic.EventUpdateLogic
	l1FetcherLogic   *logic.L1FetcherLogic

	// reorg number: counter
	// sync height: gauge
}

// NewL1MessageFetcher creates a new L1MessageFetcher instance.
func NewL1MessageFetcher(ctx context.Context, cfg *config.LayerConfig, db *gorm.DB, client *ethclient.Client, syncInfo *SyncInfo) *L1MessageFetcher {
	return &L1MessageFetcher{
		ctx:              ctx,
		cfg:              cfg,
		client:           client,
		syncInfo:         syncInfo,
		eventUpdateLogic: logic.NewEventUpdateLogic(db),
		l1FetcherLogic:   logic.NewL1FetcherLogic(cfg, db, client),
	}
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

	if updateErr := c.updateL1SyncHeight(l1SyncHeight); updateErr != nil {
		log.Crit("failed to update L1 sync height", "height", l1SyncHeight, "err", updateErr)
		return
	}

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

		isReorg, resyncHeight, l1FetcherResult, fetcherErr := c.l1FetcherLogic.L1Fetcher(c.ctx, from, to, c.l1LastSyncBlockHash)
		if fetcherErr != nil {
			log.Error("failed to fetch L1 events", "from", from, "to", to, "err", fetcherErr)
			return
		}

		if isReorg {
			log.Warn("L1 reorg happened, exit and re-enter fetchAndSaveEvents", "re-sync height", resyncHeight)
			if updateErr := c.updateL1SyncHeight(resyncHeight); updateErr != nil {
				log.Error("failed to update L1 sync height", "height", to, "err", updateErr)
				return
			}
			return
		}

		if insertUpdateErr := c.eventUpdateLogic.L1InsertOrUpdate(c.ctx, l1FetcherResult); insertUpdateErr != nil {
			log.Error("failed to save L1 events", "from", from, "to", to, "err", insertUpdateErr)
			return
		}

		if updateErr := c.updateL1SyncHeight(to); updateErr != nil {
			log.Error("failed to update L1 sync height", "height", to, "err", updateErr)
			return
		}

		l2ScannedHeight := c.syncInfo.GetL2SyncHeight()
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

func (c *L1MessageFetcher) updateL1SyncHeight(height uint64) error {
	blockHeader, err := c.client.HeaderByNumber(c.ctx, new(big.Int).SetUint64(height))
	if err != nil {
		log.Error("failed to get L1 header by number", "block number", height, "err", err)
		return err
	}
	c.l1LastSyncBlockHash = blockHeader.Hash()
	c.l1SyncHeight = height
	return nil
}
