package utils

import (
	"sync"
)

// SyncMap wraps sync.Map to make it easier to use.
type SyncMap[K, V any] struct {
	data sync.Map
	zero V
}

// Store sets the value for a key.
func (m *SyncMap[K, V]) Store(key K, value V) {
	m.data.Store(key, value)
}

// Load returns the value stored in the map for a key, or nil if no value is present. The ok result indicates whether value was found in the map.
func (m *SyncMap[K, V]) Load(key K) (V, bool) {
	val, ok := m.data.Load(key)
	if !ok {
		return m.zero, ok
	}
	return val.(V), ok
}

// Delete deletes the value for a key.
func (m *SyncMap[K, V]) Delete(key K) {
	m.data.Delete(key)
}

// LoadOrStore returns the existing value for the key if present. Otherwise, it stores and returns the given value. The loaded result is true if the value was loaded, false if stored.
func (m *SyncMap[K, V]) LoadOrStore(key K, value V) (V, bool) {
	val, loaded := m.data.LoadOrStore(key, value)
	return val.(V), loaded
}

// LoadAndDelete deletes the value for a key, returning the previous value if any. The loaded result reports whether the key was present.
func (m *SyncMap[K, V]) LoadAndDelete(key K) (V, bool) {
	val, loaded := m.data.LoadAndDelete(key)
	if loaded {
		return val.(V), loaded
	}
	return m.zero, loaded
}

// Range calls f sequentially for each key and value present in the map. If f returns false, range stops the iteration.
func (m *SyncMap[K, V]) Range(fn func(key K, value V) bool) {
	m.data.Range(func(key, value any) bool {
		return fn(key.(K), value.(V))
	})
}

// Count returns the number of elements in the map.
func (m *SyncMap[K, V]) Count() (count int64) {
	m.data.Range(func(key, value any) bool {
		count++
		return true
	})
	return count
}
