package forks

import (
	"errors"
	"math"
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

func TestBlockRange(t *testing.T) {
	tests := []struct {
		name         string
		forkHeight   uint64
		forkHeights  []uint64
		expectedFrom uint64
		expectedTo   uint64
		err          error
	}{
		{
			name:         "ToInfinite",
			forkHeight:   300,
			forkHeights:  []uint64{100, 200, 300},
			expectedFrom: 300,
			expectedTo:   math.MaxUint64,
			err:          nil,
		},
		{
			name:         "To300",
			forkHeight:   200,
			forkHeights:  []uint64{100, 200, 300},
			expectedFrom: 200,
			expectedTo:   300,
			err:          nil,
		},
		{
			name:         "To200",
			forkHeight:   100,
			forkHeights:  []uint64{100, 200, 300},
			expectedFrom: 100,
			expectedTo:   200,
			err:          nil,
		},
		{
			name:         "To100",
			forkHeight:   0,
			forkHeights:  []uint64{100, 200, 300},
			expectedFrom: 0,
			expectedTo:   100,
			err:          nil,
		},
		{
			name:         "To200-1",
			forkHeight:   100,
			forkHeights:  []uint64{100, 200},
			expectedFrom: 100,
			expectedTo:   200,
			err:          nil,
		},
		{
			name:         "to2",
			forkHeight:   1,
			forkHeights:  []uint64{1, 2},
			expectedFrom: 1,
			expectedTo:   2,
			err:          nil,
		},
		{
			name:         "to0",
			forkHeight:   0,
			forkHeights:  []uint64{0, 0},
			expectedFrom: 0,
			expectedTo:   0,
			err:          errors.New("forkHeights contains duplicated number, forkHeights:[0 0]"),
		},
		{
			name:         "100NotInside",
			forkHeight:   100,
			forkHeights:  []uint64{200, 300},
			expectedFrom: 0,
			expectedTo:   0,
			err:          errors.New("forkHeights:[200 300] don't contains forkHeight:100"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			from, to, err := BlockRange(test.forkHeight, test.forkHeights)
			require.Equal(t, test.expectedFrom, from)
			require.Equal(t, test.expectedTo, to)
			require.Equal(t, err.Error(), test.err.Error())
		})
	}
}
