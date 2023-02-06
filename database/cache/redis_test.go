package cache_test

import (
	"context"
	"encoding/json"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"os"
	"scroll-tech/common/docker"
	"scroll-tech/database/cache"
	"testing"
)

func TestRedisCache(t *testing.T) {
	redisImg := docker.NewTestRedisDocker(t)
	defer redisImg.Stop()
	rdb, err := cache.NewRedisClientWrapper(&cache.RedisConfig{
		URL: redisImg.Endpoint(),
		Expirations: map[string]int64{
			"trace": 3600,
		},
	})
	assert.NoError(t, err)

	var (
		data  []byte
		trace = &types.BlockTrace{}
	)
	data, err = os.ReadFile("../../common/testdata/blockTrace_02.json")
	assert.NoError(t, err)

	assert.NoError(t, json.Unmarshal(data, &trace))
	assert.NoError(t, rdb.SetBlockTrace(context.Background(), trace))

	exist, err := rdb.ExistTrace(context.Background(), trace.Header.Number)
	assert.NoError(t, err)
	assert.Equal(t, true, exist)
}
