package l2

import (
	"context"
	"fmt"
	"math/big"
	"runtime"

	"github.com/scroll-tech/go-ethereum/log"
	"golang.org/x/sync/errgroup"
	"modernc.org/mathutil"

	"scroll-tech/common/types"
)

func (w *WatcherClient) initCache(ctx context.Context) error {
	var (
		// Use at most half of the system threads.
		parallel = (runtime.GOMAXPROCS(0) + 1) / 2
		db       = w.orm
		index    uint64
	)

	// Fill unsigned block traces.
	for {
		batches, err := db.GetBlockBatches(
			map[string]interface{}{"proving_status": types.ProvingTaskUnassigned},
			fmt.Sprintf("AND index > %d", index),
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
			index = mathutil.MaxUint64(index, batch.Index)
			eg.Go(func() error {
				return w.fillTraceByBlockNumber(ctx, batch.StartBlockNumber, batch.EndBlockNumber)
			})
		}
		if err = eg.Wait(); err != nil {
			return err
		}
	}

	// Fill assigned and under proofing block traces into cache.
	hashes, err := w.orm.GetAssignedBatchHashes()
	if err != nil {
		return err
	}
	for _, hash := range hashes {
		err = w.fillTraceByBatchHash(ctx, hash)
		if err != nil {
			log.Error("failed to fill traces by hash", "hash", hash, "err", err)
			return err
		}
	}

	// Fill pending block traces into cache.
	for {
		hashes, err = w.orm.GetPendingBatches(uint64(parallel))
		if err != nil {
			log.Error("failed to get pending batch hashes", "err", err)
			return err
		}
		if len(hashes) == 0 {
			log.Info("L2 WatcherClient initCache done")
			return nil
		}
		for _, hash := range hashes {
			err = w.fillTraceByBatchHash(ctx, hash)
			if err != nil {
				log.Error("failed to fill traces by hash", "hash", hash, "err", err)
				return err
			}
		}
	}
}

// fillTraceByBatchHash Fill block traces by batch hash.
func (w *WatcherClient) fillTraceByBatchHash(ctx context.Context, hash string) error {
	batches, err := w.orm.GetBlockBatches(map[string]interface{}{"hash": hash})
	if err != nil || len(batches) == 0 {
		return err
	}
	batch := batches[0]
	err = w.fillTraceByBlockNumber(ctx, mathutil.MaxUint64(1, batch.StartBlockNumber), batch.EndBlockNumber)
	if err != nil {
		return err
	}
	return nil
}

// Start number must bigger than 0, because genesis block has no trace.
func (w *WatcherClient) fillTraceByBlockNumber(ctx context.Context, start, end uint64) error {
	var (
		rdb    = w.orm.GetCache()
		client = w.Client
	)
	// Block number start from 1.
	start = mathutil.MaxUint64(1, start)
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
