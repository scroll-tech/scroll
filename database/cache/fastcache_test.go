package cache_test

import (
	"context"
	"encoding/json"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"os"
	"scroll-tech/database/cache"
	"testing"
)

func TestFastCache(t *testing.T) {
	fdb := cache.NewFastCache(context.Background(), &cache.FastConfig{})

	var (
		data  []byte
		trace = &types.BlockTrace{}
	)
	data, err := os.ReadFile("../../common/testdata/blockTrace_02.json")
	assert.NoError(t, err)

	assert.NoError(t, json.Unmarshal(data, &trace))
	assert.NoError(t, fdb.SetBlockTrace(context.Background(), trace))
}
