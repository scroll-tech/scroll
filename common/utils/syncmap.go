package utils

import (
	"sync"
	"sync/atomic"
)

// SyncMap warped by sync.Map, let the usage be more simpler.
type SyncMap[K, V any] struct {
	count int64
	data  sync.Map
	zero  V
}

// Store sets the value for a key.
func (m *SyncMap[K, V]) Store(key K, value V) {
	m.LoadOrStore(key, value)
}

// Load gets value by key, if not exist return zero value.
func (m *SyncMap[K, V]) Load(key K) (V, bool) {
	val, ok := m.data.Load(key)
	if !ok {
		return m.zero, ok
	}
	return val.(V), ok
}

// Delete delete value by key, if not exist do nothing.
func (m *SyncMap[K, V]) Delete(key K) {
	_, _ = m.LoadAndDelete(key)
}

// LoadOrStore same to `sync.Map`'s loadOrStore api.
func (m *SyncMap[K, V]) LoadOrStore(key K, value V) (V, bool) {
	val, loaded := m.data.LoadOrStore(key, value)
	if !loaded {
		atomic.AddInt64(&m.count, 1)
	}
	return val.(V), loaded
}

// LoadAndDelete same to `sync.Map`'s loadAndDelete api.
func (m *SyncMap[K, V]) LoadAndDelete(key K) (V, bool) {
	val, loaded := m.data.LoadAndDelete(key)
	if loaded {
		atomic.AddInt64(&m.count, -1)
		return val.(V), loaded
	}
	return m.zero, loaded
}

// Range range k, v by real types.
func (m *SyncMap[K, V]) Range(fn func(key K, value V) bool) {
	m.data.Range(func(key, value any) bool {
		return fn(key.(K), value.(V))
	})
}

// Count returns the count members.
func (m *SyncMap[K, V]) Count() int64 {
	return atomic.LoadInt64(&m.count)
}
