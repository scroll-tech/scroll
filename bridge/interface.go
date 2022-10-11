package bridge

import (
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/rpc"
)

// L2GethClient Provide a unified external l2 call interface.
type L2GethClient interface {
	APIs() []rpc.API
}

// L1GethClient Provide a unified external l1 call interface.
type L1GethClient interface {
	SendTransaction(tx *types.Transaction) error
}

// MockL2BackendClient Provide a l2 test interface.
type MockL2BackendClient interface {
	L2GethClient
	MockBlockResult(blockResult *types.BlockResult)

	Start() error
	Stop()
}

// RelayerClient Provider a unified external layer 2 relayer interface.
type RelayerClient interface {
	ProcessSavedEvents()

	Start()
	Stop()
}
