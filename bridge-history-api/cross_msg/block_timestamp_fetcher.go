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
		u:              u}
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
					continue
				}
				start_height, err := b.g()
				if err != nil {
					log.Error("Can not get latest record without block timestamp: ", err)
				}
				if start_height <= 0 || int64(number-(uint64(b.confirmation))) < int64(start_height) {
					continue
				}
				for i := start_height; i <= uint64(number-(uint64(b.confirmation))) && i > 0; {
					block, err := b.client.BlockByNumber(b.ctx, big.NewInt(int64(start_height)))
					if err != nil {
						log.Error("Can not get block by number: ", err)
						break
					}
					err = b.u(i, time.Unix(int64(block.Time()), 0))
					if err != nil {
						log.Error("Can not update blocktimstamp into DB: ", err)
						break
					}
					i, err = b.g()
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
