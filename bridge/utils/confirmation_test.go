package utils_test

import (
	"context"
	"math/big"
	"testing"

	"scroll-tech/bridge/utils"

	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
)

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
