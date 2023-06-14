package cross_msg

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"bridge-history-api/db"
	"bridge-history-api/message_proof"
)

type BatchInfoFetcher struct {
	ctx                  context.Context
	batchInfoStartNumber uint64
	confirmation         uint
	blockTimeInSec       int
	client               *ethclient.Client
	db                   db.OrmFactory
	msgProofUpdater      *message_proof.MsgProofUpdater
}

func NewBatchInfoFetcher(ctx context.Context, batchInfoStartNumber uint64, confirmation uint, blockTimeInSec int, client *ethclient.Client, db db.OrmFactory, msgProofUpdater *message_proof.MsgProofUpdater) *BatchInfoFetcher {
	return &BatchInfoFetcher{
		ctx:                  ctx,
		batchInfoStartNumber: batchInfoStartNumber,
		confirmation:         confirmation,
		blockTimeInSec:       blockTimeInSec,
		client:               client,
		db:                   db,
		msgProofUpdater:      msgProofUpdater,
	}
}

func (b *BatchInfoFetcher) Start() {
	err := b.fetchBatchInfo()
	if err != nil {
		log.Error("fetch batch info at begining failed: ", "err", err)
	}
	go b.msgProofUpdater.Start()
	go func() {
		tick := time.NewTicker(time.Duration(b.blockTimeInSec) * time.Second)
		for {
			select {
			case <-b.ctx.Done():
				tick.Stop()
				return
			case <-tick.C:
				err := b.fetchBatchInfo()
				if err != nil {
					log.Error("fetch batch info failed: ", "err", err)
				}
			}
		}
	}()
}

func (b *BatchInfoFetcher) Stop() {
	log.Info("BatchInfoFetcher Stop")
	b.msgProofUpdater.Stop()
}

func (b *BatchInfoFetcher) fetchBatchInfo() error {
	number, err := b.client.BlockNumber(b.ctx)
	if err != nil {
		log.Error("Can not get latest block number: ", "err", err)
		return err
	}
	latestBatch, err := b.db.GetLatestBridgeBatch()
	if err != nil {
		log.Error("Can not get latest BatchInfo: ", "err", err)
		return err
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
	return nil
}
