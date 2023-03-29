package utils

import (
	"sync"
	"sync/atomic"
)

// TODO: Add more test cases.

type SyncMap[K, V any] struct {
	count int64
	data  sync.Map
	zero  V
}

func (m *SyncMap[K, V]) Store(key K, value V) {
	m.LoadOrStore(key, value)
}

func (m *SyncMap[K, V]) Load(key K) (V, bool) {
	val, ok := m.data.Load(key)
	if !ok {
		return m.zero, ok
	}
	return val.(V), ok
}

func (m *SyncMap[K, V]) Delete(key K) {
	_, _ = m.LoadAndDelete(key)
}

func (m *SyncMap[K, V]) LoadOrStore(key K, value V) (V, bool) {
	val, loaded := m.data.LoadOrStore(key, value)
	if !loaded {
		atomic.AddInt64(&m.count, 1)
	}
	return val.(V), loaded
}

func (m *SyncMap[K, V]) LoadAndDelete(key K) (V, bool) {
	val, loaded := m.data.LoadAndDelete(key)
	if loaded {
		atomic.AddInt64(&m.count, -1)
		return val.(V), loaded
	}
	return m.zero, loaded
}

func (m *SyncMap[K, V]) Range(fn func(key K, value V) bool) {
	m.data.Range(func(key, value any) bool {
		return fn(key.(K), value.(V))
	})
}

func (m *SyncMap[K, V]) Count() int64 {
	return atomic.LoadInt64(&m.count)
}
