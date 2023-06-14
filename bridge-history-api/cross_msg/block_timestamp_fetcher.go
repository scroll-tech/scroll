package cross_msg

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

type GetEarliestNoBlocktimestampHeightFunc func() (uint64, error)
type UpdateBlocktimestampFunc func(height uint64, timestamp time.Time) error

type BlocktimestampFetcher struct {
	ctx                                   context.Context
	confirmation                          uint
	blockTimeInSec                        int
	client                                *ethclient.Client
	updateBlocktimestampFunc              UpdateBlocktimestampFunc
	getEarliestNoBlocktimestampHeightFunc GetEarliestNoBlocktimestampHeightFunc
}

func NewBlocktimestampFetcher(ctx context.Context, confirmation uint, blockTimeInSec int, client *ethclient.Client, updateBlocktimestampFunc UpdateBlocktimestampFunc, getEarliestNoBlocktimestampHeightFunc GetEarliestNoBlocktimestampHeightFunc) *BlocktimestampFetcher {
	return &BlocktimestampFetcher{
		ctx:                                   ctx,
		confirmation:                          confirmation,
		blockTimeInSec:                        blockTimeInSec,
		client:                                client,
		getEarliestNoBlocktimestampHeightFunc: getEarliestNoBlocktimestampHeightFunc,
		updateBlocktimestampFunc:              updateBlocktimestampFunc,
	}
}

func (b *BlocktimestampFetcher) Start() {
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
				startHeight, err := b.getEarliestNoBlocktimestampHeightFunc()
				if err != nil {
					log.Error("Can not get latest record without block timestamp", "err", err)
					continue
				}
				for height := startHeight; number >= height+uint64(b.confirmation) && height > 0; {
					block, err := b.client.HeaderByNumber(b.ctx, new(big.Int).SetUint64(height))
					if err != nil {
						log.Error("Can not get block by number", "err", err)
						break
					}
					err = b.updateBlocktimestampFunc(height, time.Unix(int64(block.Time), 0))
					if err != nil {
						log.Error("Can not update blocktimstamp into DB ", "err", err)
						break
					}
					height, err = b.getEarliestNoBlocktimestampHeightFunc()
					if err != nil {
						log.Error("Can not get latest record without block timestamp", "err", err)
						break
					}
				}
			}
		}
	}()
}

func (b *BlocktimestampFetcher) Stop() {
	log.Info("BlocktimestampFetcher Stop")
}
