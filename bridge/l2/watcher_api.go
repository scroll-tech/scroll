package l2

import (
	"context"
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
)

// WatcherAPI watcher api service
type WatcherAPI interface {
	GetBlockTraceByHash(ctx context.Context, blockHash common.Hash) (*types.BlockTrace, error)
	GetBlockTraceByNumber(ctx context.Context, number *big.Int) (*types.BlockTrace, error)
}
