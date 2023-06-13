package cross_msg

import (
	"bridge-history-api/db"
	"context"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

type BatchInfoFetcher struct {
	ctx            context.Context
	confirmation   uint
	blockTimeInSec int
	client         *ethclient.Client
	db             db.OrmFactory
}

func NewBatchInfoFetcher(ctx context.Context, confirmation uint, blockTimeInSec int, client *ethclient.Client, db db.OrmFactory) *BatchInfoFetcher {
	return &BatchInfoFetcher{
		ctx:            ctx,
		confirmation:   confirmation,
		blockTimeInSec: blockTimeInSec,
		client:         client,
		db:             db,
	}
}

func (b *BatchInfoFetcher) Start() {
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
				latestBatch, err := b.db.GetLatestBridgeBatch()
				if err != nil {
					log.Error("Can not get latest record without block timestamp: ", err)
					continue
				}
				startHeight := latestBatch.EndBlockNumber + 1
				for height := startHeight; number >= height+uint64(b.confirmation); height += uint64(FETCH_LIMIT) {
					iter_end := height + uint64(FETCH_LIMIT) - 1
					if iter_end > number {
						iter_end = number
					}
					// filerlog to update bridge batch

				}
			}
		}
	}()
}

func (b *BatchInfoFetcher) Stop() {
	log.Info("BatchInfoFetcher Stop")
	b.ctx.Done()
}
