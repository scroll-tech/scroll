package types

import (
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
)

// BlockWithWithdrawTrieRoot contains the block's Header, Transactions and WithdrawTrieRoot hash.
type BlockWithWithdrawTrieRoot struct {
	Header           *types.Header            `json:"header"`
	Transactions     []*types.TransactionData `json:"transactions"`
	WithdrawTrieRoot common.Hash              `json:"withdraw_trie_root,omitempty"`
}
