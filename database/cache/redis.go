package cache

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"math/big"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"

	"github.com/redis/go-redis/v9"
)

// RedisConfig redis cache config.
type RedisConfig struct {
	URL         string           `json:"url"`
	Mode        string           `json:"mode,omitempty"`
	OpenTLS     bool             `json:"openTLS,omitempty"`
	Expirations map[string]int64 `json:"expirations,omitempty"`
}

// RedisClientWrapper handle redis client and some expires.
type RedisClientWrapper struct {
	client      redisClient
	traceExpire time.Duration
}

// redisClient wrap around single-redis-node / redis-cluster
type redisClient interface {
	Exists(context.Context, ...string) *redis.IntCmd
	Set(context.Context, string, interface{}, time.Duration) *redis.StatusCmd
	Get(context.Context, string) *redis.StringCmd
}

// NewRedisClientWrapper create a redis client and become Cache interface.
func NewRedisClientWrapper(redisConfig *RedisConfig) (Cache, error) {
	var traceExpire = time.Second * 60
	if val, exist := redisConfig.Expirations["trace"]; exist {
		traceExpire = time.Duration(val) * time.Second
	}

	var tlsCfg *tls.Config
	if redisConfig.OpenTLS {
		tlsCfg = &tls.Config{InsecureSkipVerify: true}
	}
	if redisConfig.Mode == "cluster" {
		op, err := redis.ParseClusterURL(redisConfig.URL)
		if err != nil {
			return nil, err
		}
		op.TLSConfig = tlsCfg
		return &RedisClientWrapper{
			client:      redis.NewClusterClient(op),
			traceExpire: traceExpire,
		}, nil
	}

	op, err := redis.ParseURL(redisConfig.URL)
	if err != nil {
		return nil, err
	}
	op.TLSConfig = tlsCfg
	return &RedisClientWrapper{
		client:      redis.NewClient(op),
		traceExpire: traceExpire,
	}, nil
}

// ExistTrace check the trace is exist or not.
func (r *RedisClientWrapper) ExistTrace(ctx context.Context, number *big.Int) (bool, error) {
	n, err := r.client.Exists(ctx, number.String()).Result()
	return err == nil && n > 0, err
}

// SetBlockTrace Set trace to redis.
func (r *RedisClientWrapper) SetBlockTrace(ctx context.Context, trace *types.BlockTrace) (setErr error) {
	hash, number := trace.Header.Hash().String(), trace.Header.Number.String()

	// If return error or the trace is exist return this function.
	n, err := r.client.Exists(ctx, hash).Result()
	if err != nil || n > 0 {
		return err
	}
	// Set trace expire time.
	defer func() {
		if setErr == nil {
			r.client.Set(ctx, number, hash, r.traceExpire)
		}
	}()

	var data []byte
	data, setErr = json.Marshal(trace)
	if setErr != nil {
		return setErr
	}
	return r.client.Set(ctx, hash, data, r.traceExpire).Err()
}

// GetBlockTrace get block trace by number, hash.
func (r *RedisClientWrapper) GetBlockTrace(ctx context.Context, hash common.Hash) (*types.BlockTrace, error) {
	// Get trace content.
	data, err := r.client.Get(ctx, hash.String()).Bytes()
	if err != nil {
		return nil, err
	}

	// Unmarshal trace and return result.
	var trace types.BlockTrace
	return &trace, json.Unmarshal(data, &trace)
}
