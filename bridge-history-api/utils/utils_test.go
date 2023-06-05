package utils_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"

	"bridge-history-api/utils"
)

func TestKeccak2(t *testing.T) {
	a := common.HexToHash("0xe90b7bceb6e7df5418fb78d8ee546e97c83a08bbccc01a0644d599ccd2a7c2e0")
	b := common.HexToHash("0x222ff5e0b5877792c2bc1670e2ccd0c2c97cd7bb1672a57d598db05092d3d72c")
	c := utils.Keccak2(a, b)
	assert.NotEmpty(t, c)
	assert.NotEqual(t, a, c)
	assert.NotEqual(t, b, c)
	assert.Equal(t, "0xc0ffbd7f501bd3d49721b0724b2bff657cb2378f15d5a9b97cd7ea5bf630d512", c.Hex())
}
