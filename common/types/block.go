package types

import (
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
)

// BlockWithWithdrawTrieRoot contains the block's Header and transactions, also the WithdrawTrieRoot hash.
type BlockWithWithdrawTrieRoot struct {
	*types.Block

	WithdrawTrieRoot common.Hash `json:"withdraw_trie_root,omitempty"`
}
