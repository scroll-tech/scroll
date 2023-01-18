package cache

import (
	"context"
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
)

// Cache cache common interface.
type Cache interface {
	ExistTrace(ctx context.Context, number *big.Int) (bool, error)
	GetBlockTrace(ctx context.Context, number *big.Int, hash common.Hash) (*types.BlockTrace, error)
	SetBlockTrace(ctx context.Context, trace *types.BlockTrace) error
}
