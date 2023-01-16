package cache

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"

	"github.com/go-redis/redis/v8"
)

// RedisClient handle redis client and some expires.
type RedisClient struct {
	*redis.Client
	traceExpire time.Duration
}

// NewRedisClient create a redis client and become CacheOrm interface.
func NewRedisClient(url string, traceExpire time.Duration) (CacheOrm, error) {
	op, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}
	return &RedisClient{
		Client:      redis.NewClient(op),
		traceExpire: traceExpire,
	}, nil
}

// SetBlockTrace Set trace to redis.
func (r *RedisClient) SetBlockTrace(ctx context.Context, trace *types.BlockTrace) error {
	hash, number := trace.Header.Hash().String(), trace.Header.Number.String()

	// If exist the trace or return error, interrupt and return.
	if exist, err := r.HExists(ctx, number, hash).Result(); err != nil || exist {
		return err
	}
	// Set trace expire time.
	r.Expire(ctx, number, r.traceExpire)
	data, err := json.Marshal(trace)
	if err != nil {
		return err
	}
	return r.HSet(ctx, number, hash, data).Err()
}

// GetBlockTrace get block trace by number, hash.
func (r *RedisClient) GetBlockTrace(ctx context.Context, number *big.Int, hash common.Hash) (*types.BlockTrace, error) {
	// Get trace content.
	data, err := r.HGet(ctx, number.String(), hash.String()).Bytes()
	if err != nil {
		return nil, err
	}

	// Unmarshal trace and return result.
	var trace types.BlockTrace
	return &trace, json.Unmarshal(data, &trace)
}
