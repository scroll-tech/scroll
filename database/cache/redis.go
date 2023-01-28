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

// RedisConfig redis cache config.
type RedisConfig struct {
	URL         string           `json:"url"`
	Expirations map[string]int64 `json:"expirations,omitempty"`
}

// RedisClient handle redis client and some expires.
type RedisClient struct {
	*redis.Client
	traceExpire time.Duration
}

// NewRedisClient create a redis client and become Cache interface.
func NewRedisClient(redisConfig *RedisConfig) (Cache, error) {
	op, err := redis.ParseURL(redisConfig.URL)
	if err != nil {
		return nil, err
	}

	var traceExpire = time.Second * 60
	if val, exist := redisConfig.Expirations["trace"]; exist {
		traceExpire = time.Duration(val) * time.Second
	}
	return &RedisClient{
		Client:      redis.NewClient(op),
		traceExpire: traceExpire,
	}, nil
}

// ExistTrace check the trace is exist or not.
func (r *RedisClient) ExistTrace(ctx context.Context, number *big.Int) (bool, error) {
	n, err := r.Exists(ctx, number.String()).Result()
	return err == nil && n > 0, err
}

// SetBlockTrace Set trace to redis.
func (r *RedisClient) SetBlockTrace(ctx context.Context, trace *types.BlockTrace) (setErr error) {
	hash, number := trace.Header.Hash().String(), trace.Header.Number.String()

	// If return error or the trace is exist return this function.
	n, err := r.Exists(ctx, hash).Result()
	if err != nil || n > 0 {
		return err
	}
	// Set trace expire time.
	defer func() {
		if setErr == nil {
			r.Set(ctx, number, hash, r.traceExpire)
		}
	}()

	var data []byte
	data, setErr = json.Marshal(trace)
	if setErr != nil {
		return setErr
	}
	return r.Set(ctx, hash, data, r.traceExpire).Err()
}

// GetBlockTrace get block trace by number, hash.
func (r *RedisClient) GetBlockTrace(ctx context.Context, hash common.Hash) (*types.BlockTrace, error) {
	// Get trace content.
	data, err := r.Get(ctx, hash.String()).Bytes()
	if err != nil {
		return nil, err
	}

	// Unmarshal trace and return result.
	var trace types.BlockTrace
	return &trace, json.Unmarshal(data, &trace)
}
