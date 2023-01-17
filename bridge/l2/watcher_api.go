package l2

import (
	"context"
	"github.com/scroll-tech/go-ethereum/core/types"
)

// WatcherAPI watcher api service
type WatcherAPI interface {
	GetBatchByID(ctx context.Context, id string) ([]*types.BlockTrace, error)
}

func (w *WatcherClient) GetBatchByID(ctx context.Context, id string) ([]*types.BlockTrace, error) {
	return w.orm.GetBlockTraces(map[string]interface{}{"batch_id": id})
}
