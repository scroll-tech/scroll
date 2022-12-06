package utils_test

import (
	"encoding/json"
	"os"
	"scroll-tech/common/utils"
	"testing"

	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
)

// TestComputetTraceCost test ComputeTraceGasCost function
func TestComputetTraceCost(t *testing.T) {
	templateBlockTrace, err := os.ReadFile("../testdata/blockTrace_03.json")
	assert.NoError(t, err)
	// unmarshal blockTrace
	blockTrace := &types.BlockTrace{}
	err = json.Unmarshal(templateBlockTrace, blockTrace)
	assert.NoError(t, err)

	// Insert into db
	res := utils.ComputeTraceGasCost(blockTrace)
	assert.NotEqual(t, res, blockTrace.Header.GasUsed)
	assert.Greater(t, res, uint64(0))
}
