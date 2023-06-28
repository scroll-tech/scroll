package cross_msg

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

type GetEarliestNoBlockTimestampHeightFunc func() (uint64, error)
type UpdateBlockTimestampFunc func(height uint64, timestamp time.Time) error

type BlockTimestampFetcher struct {
	ctx                                   context.Context
	confirmation                          uint64
	blockTimeInSec                        int
	client                                *ethclient.Client
	updateBlockTimestampFunc              UpdateBlockTimestampFunc
	getEarliestNoBlockTimestampHeightFunc GetEarliestNoBlockTimestampHeightFunc
}

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
				startHeight, err := b.getEarliestNoBlockTimestampHeightFunc()
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
					err = b.updateBlockTimestampFunc(height, time.Unix(int64(block.Time), 0))
					if err != nil {
						log.Error("Can not update blockTimestamp into DB ", "err", err)
						break
					}
					height, err = b.getEarliestNoBlockTimestampHeightFunc()
					if err != nil {
						log.Error("Can not get latest record without block timestamp", "err", err)
						break
					}
				}
			}
		}
	}()
}

func (b *BlockTimestampFetcher) Stop() {
	log.Info("BlockTimestampFetcher Stop")
}
