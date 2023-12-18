package crossmsg

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/modern-go/reflect2"
	"gorm.io/gorm"

	"bridge-history-api/config"
	"bridge-history-api/utils"
)

// MsgFetcher fetches cross message events from blockchain and saves them to database
type MsgFetcher struct {
	ctx           context.Context
	config        *config.LayerConfig
	db            *gorm.DB
	client        *ethclient.Client
	worker        *FetchEventWorker
	reorgHandling ReorgHandling
	addressList   []common.Address
	cachedHeaders []*types.Header
	mu            sync.Mutex
	reorgStartCh  chan struct{}
	reorgEndCh    chan struct{}
}

// NewMsgFetcher creates a new MsgFetcher instance
func NewMsgFetcher(ctx context.Context, config *config.LayerConfig, db *gorm.DB, client *ethclient.Client, worker *FetchEventWorker, addressList []common.Address, reorg ReorgHandling) (*MsgFetcher, error) {
	msgFetcher := &MsgFetcher{
		ctx:           ctx,
		config:        config,
		db:            db,
		client:        client,
		worker:        worker,
		reorgHandling: reorg,
		addressList:   addressList,
		cachedHeaders: make([]*types.Header, 0),
		reorgStartCh:  make(chan struct{}),
		reorgEndCh:    make(chan struct{}),
	}
	return msgFetcher, nil
}

// Start the MsgFetcher
func (c *MsgFetcher) Start() {
	log.Info("MsgFetcher Start")
	// fetch missing events from finalized blocks, we don't handle reorgs here
	c.forwardFetchAndSaveMissingEvents(c.config.Confirmation)

	tick := time.NewTicker(time.Duration(c.config.BlockTime) * time.Second)
	headerTick := time.NewTicker(time.Duration(c.config.BlockTime/2) * time.Second)
	go func() {
		for {
			select {
			case <-c.reorgStartCh:
				// create timeout here
				timeout := time.NewTicker(300 * time.Second)
				select {
				case <-c.reorgEndCh:
					log.Info("Reorg finished")
					timeout.Stop()
				case <-timeout.C:
					// TODO: need to notify the on-call members to handle reorg manually
					timeout.Stop()
					log.Crit("Reorg timeout")
				}
			case <-c.ctx.Done():
				tick.Stop()
				return
			case <-tick.C:
				c.mu.Lock()
				c.forwardFetchAndSaveMissingEvents(1)
				c.mu.Unlock()
			}
		}
	}()

	go func() {
		for {
			select {
			case <-c.ctx.Done():
				headerTick.Stop()
				return
			case <-headerTick.C:
				c.fetchMissingLatestHeaders()
			}
		}
	}()
}

// Stop the MsgFetcher and log the info
func (c *MsgFetcher) Stop() {
	log.Info("MsgFetcher Stop")
}

// forwardFetchAndSaveMissingEvents will fetch all events from the latest processed height to the latest block number.
func (c *MsgFetcher) forwardFetchAndSaveMissingEvents(confirmation uint64) {
	// if we fetch to the latest block, shall not exceed cachedHeaders
	var number uint64
	var err error
	if len(c.cachedHeaders) != 0 && confirmation == 0 {
		number = c.cachedHeaders[len(c.cachedHeaders)-1].Number.Uint64() - 1
	} else {
		number, err = utils.GetSafeBlockNumber(c.ctx, c.client, confirmation)
		if err != nil {
			log.Error(fmt.Sprintf("%s: can not get the safe block number", c.worker.Name), "err", err)
			return
		}
	}
	if reflect2.IsNil(c.worker.G) || reflect2.IsNil(c.worker.F) {
		log.Error(fmt.Sprintf("%s: invalid get/fetch function", c.worker.Name))
		return
	}
	processedHeight, err := c.worker.G(c.ctx, c.db)
	if err != nil {
		log.Error(fmt.Sprintf("%s: can not get latest processed block height", c.worker.Name))
	}
	log.Info(fmt.Sprintf("%s: ", c.worker.Name), "height", processedHeight)
	if processedHeight <= 0 || processedHeight < c.config.StartHeight {
		processedHeight = c.config.StartHeight
	} else {
		processedHeight++
	}
	for from := processedHeight; from <= number; from += fetchLimit {
		to := from + fetchLimit - 1
		if to > number {
			to = number
		}
		// watch for overflow here, though its unlikely to happen
		err := c.worker.F(c.ctx, c.client, c.db, int64(from), int64(to), c.addressList)
		if err != nil {
			log.Error(fmt.Sprintf("%s: failed!", c.worker.Name), "err", err)
			break
		}
	}
}

func (c *MsgFetcher) fetchMissingLatestHeaders() {
	var start int64
	number, err := c.client.BlockNumber(c.ctx)
	if err != nil {
		log.Error("fetchMissingLatestHeaders(): can not get the latest block number", "err", err)
		return
	}

	if len(c.cachedHeaders) > 0 {
		start = c.cachedHeaders[len(c.cachedHeaders)-1].Number.Int64() + 1
	} else {
		start = int64(number - c.config.Confirmation)
	}
	for i := start; i <= int64(number); i++ {
		select {
		case <-c.ctx.Done():
			close(c.reorgStartCh)
			close(c.reorgEndCh)
			return
		default:
			header, err := c.client.HeaderByNumber(c.ctx, big.NewInt(i))
			if err != nil {
				log.Error("failed to get latest header", "err", err)
				return
			}
			if len(c.cachedHeaders) == 0 {
				c.cachedHeaders = MergeAddIntoHeaderList(c.cachedHeaders, []*types.Header{header}, int(c.config.Confirmation))
				return
			}
			//check if the fetched header is child from the last cached header
			if IsParentAndChild(c.cachedHeaders[len(c.cachedHeaders)-1], header) {
				c.cachedHeaders = MergeAddIntoHeaderList(c.cachedHeaders, []*types.Header{header}, int(c.config.Confirmation))
				log.Debug("fetched block into cache", "height", header.Number, "parent hash", header.ParentHash.Hex(), "block hash", c.cachedHeaders[len(c.cachedHeaders)-1].Hash().Hex(), "len", len(c.cachedHeaders))
				continue
			}
			// reorg happened
			log.Warn("Reorg happened", "height", header.Number, "parent hash", header.ParentHash.Hex(), "last cached hash", c.cachedHeaders[len(c.cachedHeaders)-1].Hash().Hex(), "last cached height", c.cachedHeaders[len(c.cachedHeaders)-1].Number)
			c.reorgStartCh <- struct{}{}
			// waiting here if there is fetcher running
			c.mu.Lock()
			index, ok, validHeaders := BackwardFindReorgBlock(c.ctx, c.cachedHeaders, c.client, header)
			if !ok {
				log.Error("Reorg happened too earlier than cached headers", "reorg height", header.Number)
				num, getSafeErr := utils.GetSafeBlockNumber(c.ctx, c.client, c.config.Confirmation)
				if getSafeErr != nil {
					log.Crit("Can not get safe number during reorg, quit the process", "err", err)
				}
				// clear all our saved data, because no data is safe now
				err = c.reorgHandling(c.ctx, num, c.db)
				// if handling success then we can update the cachedHeaders
				if err == nil {
					c.cachedHeaders = c.cachedHeaders[:0]
				}
				c.mu.Unlock()
				c.reorgEndCh <- struct{}{}
				return
			}
			err = c.reorgHandling(c.ctx, c.cachedHeaders[index].Number.Uint64(), c.db)
			// if handling success then we can update the cachedHeaders
			if err == nil {
				c.cachedHeaders = c.cachedHeaders[:index+1]
				c.cachedHeaders = MergeAddIntoHeaderList(c.cachedHeaders, validHeaders, int(c.config.Confirmation))
			}
			c.mu.Unlock()
			c.reorgEndCh <- struct{}{}
		}
	}

}
