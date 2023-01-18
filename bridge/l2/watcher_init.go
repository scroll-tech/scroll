package l2

import (
	"context"
	"fmt"
	"math/big"
	"runtime"
	"time"

	"github.com/scroll-tech/go-ethereum/log"
	"golang.org/x/sync/errgroup"

	"scroll-tech/database/cache"
	"scroll-tech/database/orm"
)

func (w *WatcherClient) initCache(timeout time.Duration) error {
	var (
		// Use at most half of the system threads.
		parallel = (runtime.GOMAXPROCS(0) + 1) / 2
		db       = w.orm
	)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Fill unsigned block traces.
	for {
		batches, err := db.GetBlockBatches(
			map[string]interface{}{"proving_status": orm.ProvingTaskUnassigned},
			fmt.Sprintf("ORDER BY index ASC LIMIT %d;", parallel),
		)
		if err != nil {
			log.Error("failed to get block batch", "err", err)
			return err
		}
		if len(batches) == 0 {
			break
		}

		var eg errgroup.Group
		for _, batch := range batches {
			batch := batch
			eg.Go(func() error {
				return w.fillTraceInCache(ctx, batch.StartBlockNumber, batch.EndBlockNumber)
			})
		}
		if err = eg.Wait(); err != nil {
			return err
		}
	}

	// Fill assigned and under proofing block traces into cache.
	ids, err := w.orm.GetAssignedBatchIDs()
	if err != nil {
		return err
	}
	prevSessions, err := db.GetSessionInfosByIDs(ids)
	if err != nil {
		return err
	}
	for _, v := range prevSessions {
		id := v.ID
		batches, err := db.GetBlockBatches(map[string]interface{}{"id": id})
		if err != nil {
			log.Error("Failed to GetBlockBatches", "batch_id", id, "err", err)
			return err
		}
		if len(batches) == 0 {
			break
		}
		batch := batches[0]
		err = w.fillTraceInCache(ctx, batch.StartBlockNumber, batch.EndBlockNumber)
		if err != nil {
			log.Error("failed to fill batch block traces into cache", "start number", batch.StartBlockNumber, "end number", batch.EndBlockNumber, "err", err)
			return err
		}
	}

	return nil
}

func (w *WatcherClient) fillTraceInCache(ctx context.Context, start, end uint64) error {
	var (
		rdb    = w.orm.(cache.Cache)
		client = w.Client
	)
	for height := start; height <= end; height++ {
		number := big.NewInt(0).SetUint64(height)
		exist, err := rdb.ExistTrace(ctx, number)
		if err != nil {
			return err
		}
		if exist {
			continue
		}
		trace, err := client.GetBlockTraceByNumber(ctx, number)
		if err != nil {
			return err
		}
		err = rdb.SetBlockTrace(ctx, trace)
		if err != nil {
			return err
		}
	}
	return nil
}
