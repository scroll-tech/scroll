package forks

import (
	"math/big"
	"testing"

	"github.com/scroll-tech/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestCollectSortedForkBlocks(t *testing.T) {
	l, m, n := CollectSortedForkHeights(&params.ChainConfig{
		EIP155Block:         big.NewInt(4),
		EIP158Block:         big.NewInt(3),
		ByzantiumBlock:      big.NewInt(3),
		ConstantinopleBlock: big.NewInt(0),
	})
	require.Equal(t, l, []uint64{
		3,
		4,
	})
	require.Equal(t, map[uint64]bool{
		3: true,
		4: true,
	}, m)
	require.Equal(t, map[string]uint64{
		"eip155": 4,
		"eip158": 3,
	}, n)
}

func TestBlocksUntilFork(t *testing.T) {
	tests := map[string]struct {
		block    uint64
		forks    []uint64
		expected uint64
	}{
		"NoFork": {
			block:    44,
			forks:    []uint64{},
			expected: 0,
		},
		"BeforeFork": {
			block:    0,
			forks:    []uint64{1, 5},
			expected: 1,
		},
		"OnFork": {
			block:    1,
			forks:    []uint64{1, 5},
			expected: 4,
		},
		"OnLastFork": {
			block:    5,
			forks:    []uint64{1, 5},
			expected: 0,
		},
		"AfterFork": {
			block:    5,
			forks:    []uint64{1, 5},
			expected: 0,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, test.expected, BlocksUntilFork(test.block, test.forks))
		})
	}
}
