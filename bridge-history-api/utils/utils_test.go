package utils_test

import (
	"os"
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

func TestGetBatchRangeFromCalldataV1(t *testing.T) {
	calldata, err := os.ReadFile("../testdata/commit-batches-0x3095e91db7ba4a6fbf4654d607db322e58ff5579c502219c8024acaea74cf311.txt")
	assert.NoError(t, err)

	// multiple batches
	batchIndices, startBlocks, finishBlocks, err := utils.GetBatchRangeFromCalldataV1(common.Hex2Bytes(string(calldata[:])))
	assert.NoError(t, err)
	assert.Equal(t, len(batchIndices), 5)
	assert.Equal(t, len(startBlocks), 5)
	assert.Equal(t, len(finishBlocks), 5)
	assert.Equal(t, batchIndices[0], uint64(1))
	assert.Equal(t, batchIndices[1], uint64(2))
	assert.Equal(t, batchIndices[2], uint64(3))
	assert.Equal(t, batchIndices[3], uint64(4))
	assert.Equal(t, batchIndices[4], uint64(5))
	assert.Equal(t, startBlocks[0], uint64(1))
	assert.Equal(t, startBlocks[1], uint64(6))
	assert.Equal(t, startBlocks[2], uint64(7))
	assert.Equal(t, startBlocks[3], uint64(19))
	assert.Equal(t, startBlocks[4], uint64(20))
	assert.Equal(t, finishBlocks[0], uint64(5))
	assert.Equal(t, finishBlocks[1], uint64(6))
	assert.Equal(t, finishBlocks[2], uint64(18))
	assert.Equal(t, finishBlocks[3], uint64(19))
	assert.Equal(t, finishBlocks[4], uint64(20))
}
