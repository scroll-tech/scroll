package types

import (
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
)

type BlockWithWithdrawTrieRoot struct {
	*types.Block

	WithdrawTrieRoot common.Hash `json:"withdraw_trie_root,omitempty"`
}
