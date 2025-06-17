package provertask

import (
	"sync"
)

type CachedTaskData struct {
	MetaData  string
	UTaskData string
}

// very simple size limited lru caching for task
type TaskCache struct {
	sync.RWMutex
	cache          map[string]*CachedTaskData
	dropping_cache map[string]*CachedTaskData
	limit          int
}

func newCache(limit int) *TaskCache {
	t := TaskCache{
		cache:          make(map[string]*CachedTaskData),
		dropping_cache: make(map[string]*CachedTaskData),
		limit:          limit,
	}

	return &t
}

func (c *TaskCache) Add(k string, itm *CachedTaskData) {
	c.Lock()
	defer c.Unlock()

	if len(c.cache) >= c.limit {
		c.dropping_cache = c.cache
		c.cache = make(map[string]*CachedTaskData)
	}
	c.cache[k] = itm
}

func (c *TaskCache) Query(key string) *CachedTaskData {
	c.RLock()
	if v, ok := c.cache[key]; ok {
		c.RUnlock()
		return v
	}

	if v, ok := c.dropping_cache[key]; ok {
		c.RUnlock()
		c.Add(key, v)
		return v
	}

	c.RUnlock()
	return nil
}
