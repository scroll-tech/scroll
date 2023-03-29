package testdata

import (
	"encoding/json"
	"os"

	"github.com/scroll-tech/go-ethereum/core/types"
)

// GetTrace returns trace by file name.
func GetTrace(file string) *types.BlockTrace {
	data, err := os.ReadFile(file)
	if err != nil {
		panic(err)
	}
	trace := &types.BlockTrace{}
	if err = json.Unmarshal(data, &trace); err != nil {
		panic(err)
	}
	return trace
}
