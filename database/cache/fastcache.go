package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/VictoriaMetrics/fastcache"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"math/big"
	"sync"
	"time"
)

var (
	defFastConfig = &FastConfig{
		CacheSize:      64 * 1024 * 1024,
		TraceExpireSec: 3600,
	}
)

type FastConfig struct {
	CacheSize      int
	TraceExpireSec int64
}

type FastCache struct {
	// trace expire time second.
	traceExpireSec int64
	// fast cache handler.
	cache *fastcache.Cache
	// Handle all the key and it's insert time.
	expireTime sync.Map
}

func NewFastCache(ctx context.Context, cfg *FastConfig) Cache {
	if cfg == nil {
		cfg = defFastConfig
	}
	cache := &FastCache{
		cache:          fastcache.New(cfg.CacheSize),
		traceExpireSec: cfg.TraceExpireSec,
	}

	go cache.loop(ctx)
	return cache
}

func (f *FastCache) loop(ctx context.Context) {
	tick := time.NewTicker(time.Second * 10)
	for {
		select {
		case <-ctx.Done():
			// Free cache.
			f.cache.Reset()
			return
		case <-tick.C:
			f.expireTime.Range(func(key, value any) bool {
				number := key.(*big.Int)
				if f.isExpired(number) {
					// Delete key in expire map.
					f.freeTrace(number)
				}
				return true
			})
		}
	}
}

func (f *FastCache) ExistTrace(ctx context.Context, number *big.Int) (bool, error) {
	// Get hash by number.
	if f.isExpired(number) {
		f.freeTrace(number)
		return false, nil
	}
	return true, nil
}

func (f *FastCache) GetBlockTrace(ctx context.Context, hash common.Hash) (*types.BlockTrace, error) {
	data := f.cache.Get(nil, hash.Bytes())
	if data == nil {
		return nil, fmt.Errorf("trace if not stored in fastcache, hash: %s", hash.String())
	}
	var trace = &types.BlockTrace{}
	return trace, json.Unmarshal(data, &trace)
}

func (f *FastCache) SetBlockTrace(ctx context.Context, trace *types.BlockTrace) error {
	hash := trace.Header.Hash()
	number := trace.Header.Number

	// The trace is not expired, don't need to reset.
	if !f.isExpired(number) {
		return nil
	} else {
		// Try to delete the fragment data.
		f.freeTrace(number)
	}

	// Unmarshal trace.
	data, err := json.Marshal(trace)
	if err != nil {
		return err
	}

	// Get time stamp(second unit).
	curSec := time.Now().Unix()
	// Set expire time.
	f.expireTime.Store(number, curSec)
	// Set index of number to hash.
	f.cache.Set(number.Bytes(), hash.Bytes())
	// Set trace content in memory cache.
	f.cache.Set(hash.Bytes(), data)

	return nil
}

// If the trace is expired delete the number and
func (f *FastCache) freeTrace(number *big.Int) {
	val := f.cache.Get(nil, number.Bytes())
	if val == nil {
		return
	}
	// delete number index.
	f.cache.Del(number.Bytes())
	// delete trace content.
	f.cache.Del(val)
	// delete expire key.
	f.expireTime.Delete(number)
}

func (f *FastCache) isExpired(key interface{}) bool {
	curSec := time.Now().Unix()
	val, ok := f.expireTime.Load(key)
	if ok {
		return curSec-val.(int64) > f.traceExpireSec
	}
	return false
}
