package cross_msg

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"bridge-history-api/cross_msg/message_proof"
	"bridge-history-api/db"
)

type BatchInfoFetcher struct {
	ctx                  context.Context
	scrollChainAddr      common.Address
	batchInfoStartNumber uint64
	confirmation         uint64
	blockTimeInSec       int
	client               *ethclient.Client
	db                   db.OrmFactory
	msgProofUpdater      *message_proof.MsgProofUpdater
}

func NewBatchInfoFetcher(ctx context.Context, scrollChainAddr common.Address, batchInfoStartNumber uint64, confirmation uint64, blockTimeInSec int, client *ethclient.Client, db db.OrmFactory, msgProofUpdater *message_proof.MsgProofUpdater) *BatchInfoFetcher {
	return &BatchInfoFetcher{
		ctx:                  ctx,
		scrollChainAddr:      scrollChainAddr,
		batchInfoStartNumber: batchInfoStartNumber,
		confirmation:         confirmation,
		blockTimeInSec:       blockTimeInSec,
		client:               client,
		db:                   db,
		msgProofUpdater:      msgProofUpdater,
	}
}

func (b *BatchInfoFetcher) Start() {
	log.Info("BatchInfoFetcher Start")
	// Fetch batch info at begining
	// Then start msg proof updater after db have some bridge batch
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
		startHeight = latestBatch.CommitHeight + 1
	}
	for from := startHeight; number >= from+b.confirmation; from += uint64(fetchLimit) {
		to := from + uint64(fetchLimit) - 1
		// number - confirmation can never less than 0 since the for loop condition
		// but watch out the overflow
		if to > number-b.confirmation {
			to = number - b.confirmation
		}
		// filter logs to fetch batches
		err = FetchAndSaveBatchIndex(b.ctx, b.client, b.db, int64(from), int64(to), b.scrollChainAddr)
		if err != nil {
			log.Error("Can not fetch and save from chain: ", "err", err)
			return err
		}
	}
	return nil
}
