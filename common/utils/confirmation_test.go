package utils_test

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"testing"

	"scroll-tech/common/utils"

	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
)

func TestUnmarshalJSON(t *testing.T) {
	var params utils.ConfirmationParams

	decoder := json.NewDecoder(strings.NewReader(`"finalized"`))
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&params)
	assert.Nil(t, err)
	assert.Equal(t, utils.Finalized, params.Type)

	decoder = json.NewDecoder(strings.NewReader(`"safe"`))
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&params)
	assert.Nil(t, err)
	assert.Equal(t, utils.Safe, params.Type)

	decoder = json.NewDecoder(strings.NewReader(`"number=6"`))
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&params)
	assert.Nil(t, err)
	assert.Equal(t, utils.Number, params.Type)
	assert.Equal(t, uint64(6), params.Number)

	decoder = json.NewDecoder(strings.NewReader(`"number=999"`))
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&params)
	assert.Nil(t, err)
	assert.Equal(t, utils.Number, params.Type)
	assert.Equal(t, uint64(999), params.Number)

	decoder = json.NewDecoder(strings.NewReader(`"number=1000"`))
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&params)
	assert.NotNil(t, err)

	decoder = json.NewDecoder(strings.NewReader(`"number=6x"`))
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&params)
	assert.NotNil(t, err)

	decoder = json.NewDecoder(strings.NewReader(`"latest"`))
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&params)
	assert.NotNil(t, err)
}

func TestMarshalJSON(t *testing.T) {
	bytes, err := json.Marshal(&utils.ConfirmationParams{Type: utils.Finalized, Number: 6})
	assert.Nil(t, err)
	assert.Equal(t, `"finalized"`, string(bytes))

	bytes, err = json.Marshal(&utils.ConfirmationParams{Type: utils.Safe, Number: 6})
	assert.Nil(t, err)
	assert.Equal(t, `"safe"`, string(bytes))

	bytes, err = json.Marshal(&utils.ConfirmationParams{Type: utils.Number, Number: 6})
	assert.Nil(t, err)
	assert.Equal(t, `"number=6"`, string(bytes))
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
	confirmed, err := utils.GetLatestConfirmedBlockNumber(ctx, &client, utils.ConfirmationParams{Type: utils.Number, Number: 6})
	assert.Nil(t, err)
	assert.Equal(t, uint64(0), confirmed)

	client.val = 7
	confirmed, err = utils.GetLatestConfirmedBlockNumber(ctx, &client, utils.ConfirmationParams{Type: utils.Number, Number: 6})
	assert.Nil(t, err)
	assert.Equal(t, uint64(1), confirmed)
}
