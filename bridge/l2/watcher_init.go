package l2

import (
	"context"
	"fmt"
	"math/big"
	"runtime"

	"github.com/scroll-tech/go-ethereum/log"
	"golang.org/x/sync/errgroup"

	"scroll-tech/database/cache"
	"scroll-tech/database/orm"
)

func (w *WatcherClient) initCache(ctx context.Context) error {
	var (
		// Use at most half of the system threads.
		parallel = (runtime.GOMAXPROCS(0) + 1) / 2
		db       = w.orm
	)

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
				return w.fillTraceByNumber(ctx, batch.StartBlockNumber, batch.EndBlockNumber)
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
	for _, id := range ids {
		err = w.fillTraceByID(ctx, id)
		if err != nil {
			log.Error("failed to fill traces by id", "id", id, "err", err)
			return err
		}
	}

	// Fill pending block traces into cache.
	for {
		ids, err = w.orm.GetPendingBatches(uint64(parallel))
		if err != nil {
			log.Error("failed to get pending batch ids", "err", err)
			return err
		}
		if len(ids) == 0 {
			log.Info("L2 WatcherClient initCache done")
			return nil
		}
		for _, id := range ids {
			err = w.fillTraceByID(ctx, id)
			if err != nil {
				log.Error("failed to fill traces by id", "id", id, "err", err)
				return err
			}
		}
	}
}

// fillTraceByID Fill block traces by batch id.
func (w *WatcherClient) fillTraceByID(ctx context.Context, id string) error {
	batches, err := w.orm.GetBlockBatches(map[string]interface{}{"id": id})
	if err != nil || len(batches) == 0 {
		return err
	}
	batch := batches[0]
	err = w.fillTraceByNumber(ctx, batch.StartBlockNumber, batch.EndBlockNumber)
	if err != nil {
		return err
	}
	return nil
}

func (w *WatcherClient) fillTraceByNumber(ctx context.Context, start, end uint64) error {
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
