package testdata

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime/debug"
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

// Load trace list.
func init() {
	dir, _ := os.Getwd()
	index := strings.LastIndex(dir, "scroll-tech/scroll")
	if index == -1 {
		fmt.Println("call stack is: ", string(debug.Stack()))
	}
	pwd := dir[:index] + "scroll-tech/scroll/common/testdata/"
	for file := range TraceList {
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
