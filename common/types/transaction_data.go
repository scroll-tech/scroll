package types

import (
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
)

// This is needed as we include the L1 Block Hashes Tx into the chunk blocks.
// The transaction hash is generated with the inclusion of the fields below.
type TransactionData struct {
	gethTypes.TransactionData
	FirstAppliedL1Block *hexutil.Uint64 `json:"firstAppliedL1Block,omitempty"`
	LastAppliedL1Block  *hexutil.Uint64 `json:"lastAppliedL1Block,omitempty"`
	BlockRangeHash      []common.Hash   `json:"blockRangeHash,omitempty"`
}
