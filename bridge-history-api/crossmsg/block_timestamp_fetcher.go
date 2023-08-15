package crossmsg

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

// GetEarliestNoBlockTimestampHeightFunc is a function type that gets the earliest record without block timestamp from database
type GetEarliestNoBlockTimestampHeightFunc func(ctx context.Context) (uint64, error)

// UpdateBlockTimestampFunc is a function type that updates block timestamp into database
type UpdateBlockTimestampFunc func(ctx context.Context, height uint64, timestamp time.Time) error

// BlockTimestampFetcher fetches block timestamp from blockchain and saves them to database
type BlockTimestampFetcher struct {
	ctx                                   context.Context
	confirmation                          uint64
	blockTimeInSec                        int
	client                                *ethclient.Client
	updateBlockTimestampFunc              UpdateBlockTimestampFunc
	getEarliestNoBlockTimestampHeightFunc GetEarliestNoBlockTimestampHeightFunc
}

// NewBlockTimestampFetcher creates a new BlockTimestampFetcher instance
func NewBlockTimestampFetcher(ctx context.Context, confirmation uint64, blockTimeInSec int, client *ethclient.Client, updateBlockTimestampFunc UpdateBlockTimestampFunc, getEarliestNoBlockTimestampHeightFunc GetEarliestNoBlockTimestampHeightFunc) *BlockTimestampFetcher {
	return &BlockTimestampFetcher{
		ctx:                                   ctx,
		confirmation:                          confirmation,
		blockTimeInSec:                        blockTimeInSec,
		client:                                client,
		getEarliestNoBlockTimestampHeightFunc: getEarliestNoBlockTimestampHeightFunc,
		updateBlockTimestampFunc:              updateBlockTimestampFunc,
	}
}

// Start the BlockTimestampFetcher
func (b *BlockTimestampFetcher) Start() {
	go func() {
		tick := time.NewTicker(time.Duration(b.blockTimeInSec) * time.Second)
		for {
			select {
			case <-b.ctx.Done():
				tick.Stop()
				return
			case <-tick.C:
				number, err := b.client.BlockNumber(b.ctx)
				if err != nil {
					log.Error("Can not get latest block number", "err", err)
					continue
				}
				startHeight, err := b.getEarliestNoBlockTimestampHeightFunc(b.ctx)
				if err != nil {
					log.Error("Can not get latest record without block timestamp", "err", err)
					continue
				}
				for height := startHeight; number >= height+b.confirmation && height > 0; {
					block, err := b.client.HeaderByNumber(b.ctx, new(big.Int).SetUint64(height))
					if err != nil {
						log.Error("Can not get block by number", "err", err)
						break
					}
					err = b.updateBlockTimestampFunc(b.ctx, height, time.Unix(int64(block.Time), 0))
					if err != nil {
						log.Error("Can not update blockTimestamp into DB ", "err", err)
						break
					}
					height, err = b.getEarliestNoBlockTimestampHeightFunc(b.ctx)
					if err != nil {
						log.Error("Can not get latest record without block timestamp", "err", err)
						break
					}
				}
			}
		}
	}()
}

// Stop the BlockTimestampFetcher and log the info
func (b *BlockTimestampFetcher) Stop() {
	log.Info("BlockTimestampFetcher Stop")
}
