package utils_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/utils"
)

// TestComputetTraceCost test ComputeTraceGasCost function
func TestComputetTraceCost(t *testing.T) {
	templateBlockTrace, err := os.ReadFile("../testdata/blockTrace_03.json")
	assert.NoError(t, err)
	// unmarshal blockTrace
	blockTrace := &types.BlockTrace{}
	err = json.Unmarshal(templateBlockTrace, blockTrace)
	assert.NoError(t, err)

	res := utils.ComputeTraceGasCost(blockTrace)
	assert.Equal(t, uint64(113874), res)
}
