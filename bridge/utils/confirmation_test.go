package utils_test

import (
	"context"
	"encoding/json"
	"github.com/scroll-tech/go-ethereum/common/math"
	"github.com/scroll-tech/go-ethereum/rpc"
	"math/big"
	"testing"

	"scroll-tech/bridge/utils"

	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
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
		assert.Nil(t, err)
		if !test.mustFail {
			err = json.Unmarshal([]byte(test.input), &num)
			assert.Nil(t, err)
			got, err := json.Marshal(&num)
			assert.Nil(t, err)
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
	return &types.Header{Number: new(big.Int).SetUint64(e.val)}, nil
}

func TestGetLatestConfirmedBlockNumber(t *testing.T) {
	ctx := context.Background()
	client := MockEthClient{}

	client.val = 5
	confirmed, err := utils.GetLatestConfirmedBlockNumber(ctx, &client, 6)
	assert.Nil(t, err)
	assert.Equal(t, uint64(0), confirmed)

	client.val = 7
	confirmed, err = utils.GetLatestConfirmedBlockNumber(ctx, &client, 6)
	assert.Nil(t, err)
	assert.Equal(t, uint64(1), confirmed)
}
