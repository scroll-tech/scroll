package testdata

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/scroll-tech/go-ethereum/core/types"
)

var (
	TraceList = map[string]*types.BlockTrace{
		"blockTrace_02.json":       nil,
		"blockTrace_03.json":       nil,
		"blockTrace_delegate.json": nil,
	}
)

// Load Load trace list.
func Load() {
	dir, _ := os.Getwd()
	index := strings.LastIndex(dir, "scroll-tech/scroll")
	pwd := dir[:index] + "scroll-tech/scroll/common/testdata/"
	for file, val := range TraceList {
		if val != nil {
			continue
		}
		data, err := os.ReadFile(pwd + file)
		if err != nil {
			panic(err)
		}
		trace := &types.BlockTrace{}
		if err = json.Unmarshal(data, &trace); err != nil {
			panic(err)
		}
		TraceList[file] = trace
	}
}
