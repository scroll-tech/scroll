package l2

import (
	"errors"

	"github.com/scroll-tech/go-ethereum/rpc"
)

// WatcherAPI watcher api service
type WatcherAPI interface {
	ReplayBlockResultByHash(blockNrOrHash rpc.BlockNumberOrHash) (bool, error)
}

// ReplayBlockResultByHash temporary interface for easy testing.
func (r *WatcherClient) ReplayBlockResultByHash(blockNrOrHash rpc.BlockNumberOrHash) (bool, error) {
	orm := r.orm
	params := make(map[string]interface{})
	if number, ok := blockNrOrHash.Number(); ok {
		params["number"] = int64(number)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		params["hash"] = hash.String()
	}
	if len(params) == 0 {
		return false, errors.New("empty params")
	}
	trace, err := orm.GetBlockTraces(params)
	if err != nil {
		return false, err
	}
	r.Send(&trace[0])
	return true, nil
}
