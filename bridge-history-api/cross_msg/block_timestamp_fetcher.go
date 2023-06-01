package cross_msg

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

type GetEarliestNoBlocktimestampHeight func() (uint64, error)
type BlocktimestampUpdater func(height uint64, timestamp time.Time) error

type BlocktimestampFetcher struct {
	ctx            context.Context
	confirmation   uint
	blockTimeInSec int
	client         *ethclient.Client
	u              BlocktimestampUpdater
	g              GetEarliestNoBlocktimestampHeight
}

func NewBlocktimestampFetcher(ctx context.Context, confirmation uint, blockTimeInSec int, client *ethclient.Client, u BlocktimestampUpdater, g GetEarliestNoBlocktimestampHeight) *BlocktimestampFetcher {
	return &BlocktimestampFetcher{
		ctx:            ctx,
		confirmation:   confirmation,
		blockTimeInSec: blockTimeInSec,
		client:         client,
		g:              g,
		u:              u,
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
					log.Error("Can not get latest block number: ", err)
					continue
				}
				start_height, err := b.g()
				if err != nil {
					log.Error("Can not get latest record without block timestamp: ", err)
					continue
				}
				if start_height <= 0 || number < start_height+uint64(b.confirmation) {
					continue
				}
				for height := start_height; number >= height+uint64(b.confirmation) && height > 0; {
					block, err := b.client.HeaderByNumber(b.ctx, new(big.Int).SetUint64(height))
					if err != nil {
						log.Error("Can not get block by number: ", err)
						break
					}
					err = b.u(height, time.Unix(int64(block.Time), 0))
					if err != nil {
						log.Error("Can not update blocktimstamp into DB: ", err)
						break
					}
					height, err = b.g()
					if err != nil {
						log.Error("Can not get latest record without block timestamp: ", err)
						break
					}
				}
			}
		}
	}()
}

func (b *BlocktimestampFetcher) Stop() {
	log.Info("BlocktimestampFetcher Stop")
	b.ctx.Done()
}
