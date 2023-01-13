package cache

import (
	"context"
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
)

type CacheOrm interface {
	GetBlockTrace(ctx context.Context, number *big.Int, hash common.Hash) (*types.BlockTrace, error)
	SetBlockTrace(ctx context.Context, trace *types.BlockTrace) error
}
