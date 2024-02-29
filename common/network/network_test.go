package network

import (
	"math/big"
	"testing"

	"github.com/scroll-tech/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestCollectSortedForkBlocks(t *testing.T) {
	require.Equal(t, CollectSortedForkHeights(&params.ChainConfig{
		EIP155Block:         big.NewInt(4),
		EIP158Block:         big.NewInt(3),
		ByzantiumBlock:      big.NewInt(3),
		ConstantinopleBlock: big.NewInt(0),
	}), []uint64{
		3,
		4,
	})
}
