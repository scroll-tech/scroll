package l2

import (
	"context"

	"github.com/scroll-tech/go-ethereum/core/types"
)

// WatcherAPI watcher api service
type WatcherAPI interface {
	GetTracesByBatchIndex(ctx context.Context, id string) ([]*types.BlockTrace, error)
}

// GetTracesByBatchIndex get traces by batch_id.
func (w *WatcherClient) GetTracesByBatchIndex(ctx context.Context, index int64) ([]*types.BlockTrace, error) {
	id, err := w.orm.GetIDByIndex(index)
	if err != nil {
		return nil, err
	}
	return w.orm.GetBlockTraces(map[string]interface{}{"batch_id": id})
}
