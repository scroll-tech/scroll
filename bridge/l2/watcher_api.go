package l2

import (
	"context"
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"

	"scroll-tech/database/cache"
)

// WatcherAPI watcher api service
type WatcherAPI interface {
	GetBlockTraceByHash(ctx context.Context, blockHash common.Hash) (*types.BlockTrace, error)
	GetBlockTraceByNumber(ctx context.Context, number *big.Int) (*types.BlockTrace, error)
}

// GetBlockTraceByHash get trace by hash.
func (w *WatcherClient) GetBlockTraceByHash(ctx context.Context, blockHash common.Hash) (*types.BlockTrace, error) {
	traces, err := w.orm.GetBlockTraces(map[string]interface{}{"hash": blockHash.String()})
	if err != nil {
		return nil, err
	}
	// If trace don't exist in cache, get it return and write into cache.
	if len(traces) == 0 {
		rdb := w.orm.(cache.Cache)
		trace, err := w.Client.GetBlockTraceByHash(ctx, blockHash)
		if err != nil {
			return nil, err
		}
		return trace, rdb.SetBlockTrace(ctx, trace)
	}
	return traces[0], nil
}

// GetBlockTraceByNumber get trace by number.
func (w *WatcherClient) GetBlockTraceByNumber(ctx context.Context, number *big.Int) (*types.BlockTrace, error) {
	traces, err := w.orm.GetBlockTraces(map[string]interface{}{"number": number.Uint64()})
	if err != nil {
		return nil, err
	}
	// If trace don't exist in cache, get it return and write into cache.
	if len(traces) == 0 {
		rdb := w.orm.(cache.Cache)
		trace, err := w.Client.GetBlockTraceByNumber(ctx, number)
		if err != nil {
			return nil, err
		}
		return trace, rdb.SetBlockTrace(ctx, trace)
	}
	return traces[0], nil
}
