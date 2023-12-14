package fetcher

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/bridge-history-api/internal/config"
	"scroll-tech/bridge-history-api/internal/logic"
	"scroll-tech/bridge-history-api/internal/orm"
	"scroll-tech/bridge-history-api/internal/utils"
)

// L2MessageFetcher fetches cross message events from L2 and saves them to database.
type L2MessageFetcher struct {
	ctx      context.Context
	cfg      *config.LayerConfig
	db       *gorm.DB
	client   *ethclient.Client
	syncInfo *SyncInfo

	eventUpdateLogic *logic.EventUpdateLogic
	l2FetcherLogic   *logic.L2FetcherLogic
}

// NewL2MessageFetcher creates a new L2MessageFetcher instance.
func NewL2MessageFetcher(ctx context.Context, cfg *config.LayerConfig, db *gorm.DB, client *ethclient.Client, syncInfo *SyncInfo) (*L2MessageFetcher, error) {
	return &L2MessageFetcher{
		ctx:              ctx,
		cfg:              cfg,
		db:               db,
		syncInfo:         syncInfo,
		client:           client,
		eventUpdateLogic: logic.NewEventUpdateLogic(db),
		l2FetcherLogic:   logic.NewL2FetcherLogic(cfg, db, client),
	}, nil
}

// Start starts the L2 message fetching process.
func (c *L2MessageFetcher) Start() {
	l2SentMessageSyncedHeight, err := c.eventUpdateLogic.GetL2MessageSyncedHeightInDB(c.ctx)
	if err != nil {
		log.Error("failed to get L2 cross message processed height", "err", err)
		return
	}

	c.syncInfo.SetL2ScanHeight(l2SentMessageSyncedHeight)
	log.Info("Start L2 message fetcher", "message synced height", l2SentMessageSyncedHeight)

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
	startHeight := c.syncInfo.GetL2ScanHeight() + 1
	endHeight, err := utils.GetBlockNumber(c.ctx, c.client, confirmation)
	if err != nil {
		log.Error("failed to get L1 safe block number", "err", err)
		return
	}
	log.Info("fetch and save missing L2 events", "start height", startHeight, "end height", endHeight)

	for from := startHeight; from <= endHeight; from += c.cfg.FetchLimit {
		to := from + c.cfg.FetchLimit - 1
		if to > endHeight {
			to = endHeight
		}

		l2FilterResult, err := c.l2FetcherLogic.L2Fetcher(c.ctx, from, to)
		if err != nil {
			log.Error("failed to fetch L2 events", "from", from, "to", to, "err", err)
			return
		}

		if updateWithdrawErr := c.updateL2WithdrawMessageProofs(c.ctx, l2FilterResult.WithdrawMessages, to); updateWithdrawErr != nil {
			log.Error("failed to update L2 withdraw message", "from", from, "to", to, "err", err)
			return
		}

		if insertUpdateErr := c.eventUpdateLogic.L2InsertOrUpdate(c.ctx, l2FilterResult); insertUpdateErr != nil {
			log.Error("failed to save L2 events", "from", from, "to", to, "err", err)
			return
		}

		c.syncInfo.SetL2ScanHeight(to)
	}
}

func (c *L2MessageFetcher) updateL2WithdrawMessageProofs(ctx context.Context, l2WithdrawMessages []*orm.CrossMessage, endBlock uint64) error {
	withdrawTrie := utils.NewWithdrawTrie()
	message, err := c.eventUpdateLogic.GetL2LatestWithdrawal(ctx)
	if err != nil {
		log.Error("failed to get latest L2 sent message event", "err", err)
		return err
	}

	if message != nil {
		withdrawTrie.Initialize(message.MessageNonce, common.HexToHash(message.MessageHash), message.MerkleProof)
	}

	messageHashes := make([]common.Hash, len(l2WithdrawMessages))
	for i, message := range l2WithdrawMessages {
		messageHashes[i] = common.HexToHash(message.MessageHash)
	}

	proofs := withdrawTrie.AppendMessages(messageHashes)
	if len(l2WithdrawMessages) != len(proofs) {
		log.Error("invalid proof array length", "L2 withdrawal messages length", len(l2WithdrawMessages), "proofs length", len(proofs))
		return fmt.Errorf("invalid proof array length: got %d proofs for %d l2WithdrawMessages", len(proofs), len(l2WithdrawMessages))
	}

	for i, proof := range proofs {
		l2WithdrawMessages[i].MerkleProof = proof
	}

	// Verify if local info is correct.
	withdrawRoot, err := c.client.StorageAt(ctx, common.HexToAddress(c.cfg.MessageQueueAddr), common.Hash{}, new(big.Int).SetUint64(endBlock))
	if err != nil {
		log.Error("failed to get withdraw root", "number", endBlock, "error", err)
		return fmt.Errorf("failed to get withdraw root: %v, number: %v", err, endBlock)
	}

	if common.BytesToHash(withdrawRoot) != withdrawTrie.MessageRoot() {
		log.Error("withdraw root mismatch", "expected", common.BytesToHash(withdrawRoot).String(), "got", withdrawTrie.MessageRoot().String())
		return fmt.Errorf("withdraw root mismatch. expected: %v, got: %v", common.BytesToHash(withdrawRoot), withdrawTrie.MessageRoot())
	}
	return nil
}
