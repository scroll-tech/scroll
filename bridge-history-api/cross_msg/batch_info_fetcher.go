package cross_msg

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"bridge-history-api/db"
)

type BatchInfoFetcher struct {
	ctx                  context.Context
	batchInfoStartNumber uint64
	confirmation         uint
	blockTimeInSec       int
	client               *ethclient.Client
	db                   db.OrmFactory
}

func NewBatchInfoFetcher(ctx context.Context, batchInfoStartNumber uint64, confirmation uint, blockTimeInSec int, client *ethclient.Client, db db.OrmFactory) *BatchInfoFetcher {
	return &BatchInfoFetcher{
		ctx:                  ctx,
		batchInfoStartNumber: batchInfoStartNumber,
		confirmation:         confirmation,
		blockTimeInSec:       blockTimeInSec,
		client:               client,
		db:                   db,
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
					log.Error("Can not get latest BatchInfo: ", err)
					continue
				}
				var startHeight uint64
				if latestBatch == nil {
					startHeight = b.batchInfoStartNumber
				} else {
					startHeight = latestBatch.EndBlockNumber + 1
				}
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
