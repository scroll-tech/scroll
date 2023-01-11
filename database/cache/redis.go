package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/spf13/cast"
)

// RedisClient handle redis client and some expires.
type RedisClient struct {
	*redis.Client
	traceExpire time.Duration
}

// NewRedisClient create a redis client and become CacheOrm interface.
func NewRedisClient(option *redis.Options, traceExpire time.Duration) CacheOrm {
	return &RedisClient{
		Client:      redis.NewClient(option),
		traceExpire: traceExpire,
	}
}

// GetBlockTraceByNumber get trace by number.
func (r *RedisClient) GetBlockTraceByNumber(ctx context.Context, number *big.Int) (*types.BlockTrace, error) {
	hashes, err := r.HKeys(ctx, cast.ToString(number.Uint64())).Result()
	if err != nil {
		return nil, err
	}
	if len(hashes) == 0 {
		return nil, fmt.Errorf("don't have such trace in redis, number: %d", number.Uint64())
	}

	return r.getTrace(ctx, cast.ToString(number.Uint64()), hashes[len(hashes)-1])
}

// GetBlockTraceByHash get trace by hash.
func (r *RedisClient) GetBlockTraceByHash(ctx context.Context, hash common.Hash) (*types.BlockTrace, error) {
	// Get number by hash.
	number, err := r.Get(ctx, hash.String()).Result()
	if err != nil {
		return nil, err
	}

	return r.getTrace(ctx, number, hash.String())
}

// SetBlockTrace Set trace to redis.
func (r *RedisClient) SetBlockTrace(ctx context.Context, trace *types.BlockTrace) error {
	hash, number := trace.Header.Hash().String(), cast.ToString(trace.Header.Number.Uint64())

	// Set Hash => number kv index and set the expire time.
	exist, err := r.SetNX(ctx, hash, cast.ToString(number), r.traceExpire).Result()
	if err != nil {
		return err
	}
	if exist {
		log.Warn("the trace already exist don't need to insert into redis cache again", "number", number, "hash", hash)
		return nil
	}

	// Set trace expire time.
	r.Expire(ctx, number, r.traceExpire)
	data, err := json.Marshal(trace)
	if err != nil {
		return err
	}
	return r.HSet(ctx, number, data).Err()
}

func (r *RedisClient) getTrace(ctx context.Context, number, hash string) (*types.BlockTrace, error) {
	// Get trace content.
	data, err := r.HGet(ctx, number, hash).Bytes()
	if err != nil {
		return nil, err
	}

	// Unmarshal trace and return result.
	var trace types.BlockTrace
	return &trace, json.Unmarshal(data, &trace)
}
