package utils

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/scroll-tech/go-ethereum/common/math"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/rpc"
)

var (
	tests = []struct {
		input    string
		mustFail bool
		expected rpc.BlockNumber
	}{
		{`"0x"`, true, rpc.BlockNumber(0)},
		{`"0x0"`, false, rpc.BlockNumber(0)},
		{`"0X1"`, false, rpc.BlockNumber(1)},
		{`"0x00"`, true, rpc.BlockNumber(0)},
		{`"0x01"`, true, rpc.BlockNumber(0)},
		{`"0x1"`, false, rpc.BlockNumber(1)},
		{`"0x12"`, false, rpc.BlockNumber(18)},
		{`"0x7fffffffffffffff"`, false, rpc.BlockNumber(math.MaxInt64)},
		{`"0x8000000000000000"`, true, rpc.BlockNumber(0)},
		{"0", true, rpc.BlockNumber(0)},
		{`"ff"`, true, rpc.BlockNumber(0)},
		{`"safe"`, false, rpc.SafeBlockNumber},
		{`"finalized"`, false, rpc.FinalizedBlockNumber},
		{`"pending"`, false, rpc.PendingBlockNumber},
		{`"latest"`, false, rpc.LatestBlockNumber},
		{`"earliest"`, false, rpc.EarliestBlockNumber},
		{`someString`, true, rpc.BlockNumber(0)},
		{`""`, true, rpc.BlockNumber(0)},
		{``, true, rpc.BlockNumber(0)},
	}
)

func TestUnmarshalJSON(t *testing.T) {
	for i, test := range tests {
		var num rpc.BlockNumber
		err := json.Unmarshal([]byte(test.input), &num)
		if test.mustFail && err == nil {
			t.Errorf("Test %d should fail", i)
			continue
		}
		if !test.mustFail && err != nil {
			t.Errorf("Test %d should pass but got err: %v", i, err)
			continue
		}
		if num != test.expected {
			t.Errorf("Test %d got unexpected value, want %d, got %d", i, test.expected, num)
		}
	}
}

func TestMarshalJSON(t *testing.T) {
	for i, test := range tests {
		var num rpc.BlockNumber
		want, err := json.Marshal(test.expected)
		assert.NoError(t, err)
		if !test.mustFail {
			err = json.Unmarshal([]byte(test.input), &num)
			assert.NoError(t, err)
			got, err := json.Marshal(&num)
			assert.NoError(t, err)
			if string(want) != string(got) {
				t.Errorf("Test %d got unexpected value, want %d, got %d", i, test.expected, num)
			}
		}
	}
}

type MockEthClient struct {
	val uint64
}

func (e MockEthClient) BlockNumber(ctx context.Context) (uint64, error) {
	return e.val, nil
}

func (e MockEthClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	var blockNumber int64
	switch number.Int64() {
	case int64(rpc.LatestBlockNumber):
		blockNumber = int64(e.val)
	case int64(rpc.SafeBlockNumber):
		blockNumber = int64(e.val) - 6
	case int64(rpc.FinalizedBlockNumber):
		blockNumber = int64(e.val) - 12
	default:
		blockNumber = number.Int64()
	}
	if blockNumber < 0 {
		blockNumber = 0
	}

	return &types.Header{Number: new(big.Int).SetInt64(blockNumber)}, nil
}

func TestGetLatestConfirmedBlockNumber(t *testing.T) {
	ctx := context.Background()
	client := MockEthClient{}

	testCases := []struct {
		blockNumber    uint64
		confirmation   rpc.BlockNumber
		expectedResult uint64
	}{
		{5, 6, 0},
		{7, 6, 1},
		{10, 2, 8},
		{0, 1, 0},
		{3, 0, 3},
		{15, 15, 0},
		{16, rpc.SafeBlockNumber, 10},
		{22, rpc.FinalizedBlockNumber, 10},
		{10, rpc.LatestBlockNumber, 10},
		{5, rpc.SafeBlockNumber, 0},
		{11, rpc.FinalizedBlockNumber, 0},
	}

	for _, testCase := range testCases {
		client.val = testCase.blockNumber
		confirmed, err := GetLatestConfirmedBlockNumber(ctx, &client, testCase.confirmation)
		assert.NoError(t, err)
		assert.Equal(t, testCase.expectedResult, confirmed)
	}
}
