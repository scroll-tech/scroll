package utils

import "sync"

// SafeMap the exposed interface of safeMap.
type SafeMap[K comparable, V comparable] interface {
	Store(key K, value V)
	Load(key K) (V, bool)
	Delete(key K)
	LoadOrStore(key K, value V) (V, bool)
	LoadAndDelete(key K) (V, bool)
	Range(fn func(key K, value V))
	Keys() []K
	Count() (count int64)
}

// safeMap wraps normal map to make it easier to use.
type safeMap[K comparable, V comparable] struct {
	data map[K]V
	zero V
	mu   sync.RWMutex
}

// NewSafeMap creates a new SafeMap instance.
func NewSafeMap[K, V comparable](cap int64) SafeMap[K, V] {
	return &safeMap[K, V]{
		data: make(map[K]V, cap),
	}
}

// Store sets the value for a key.
func (m *safeMap[K, V]) Store(key K, value V) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Don't need to store nil type of value.
	if value == m.zero {
		return
	}
	m.data[key] = value
}

// Load returns the value stored in the map for a key, or nil if no value is present. The ok result indicates whether value was found in the map.
func (m *safeMap[K, V]) Load(key K) (V, bool) {
	value := m.data[key]
	if value == m.zero {
		return m.zero, false
	}
	return value, true
}

// Delete deletes the value for a key.
func (m *safeMap[K, V]) Delete(key K) {
	m.mu.Lock()
	delete(m.data, key)
	m.mu.Unlock()
}

// LoadOrStore returns the existing value for the key if present. Otherwise, it stores and returns the given value. The loaded result is true if the value was loaded, false if stored.
func (m *safeMap[K, V]) LoadOrStore(key K, value V) (V, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// We can consider nil type of value is exist by default
	if value == m.zero {
		return value, true
	}
	val := m.data[key]
	if val == m.zero {
		m.data[key] = value
		return val, false
	}
	return val, true
}

// LoadAndDelete deletes the value for a key, returning the previous value if any. The loaded result reports whether the key was present.
func (m *safeMap[K, V]) LoadAndDelete(key K) (V, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	val := m.data[key]
	if val != m.zero {
		delete(m.data, key)
		return val, true
	}
	return m.zero, false
}

// Range calls f sequentially for each key and value present in the map. If f returns false, range stops the iteration.
func (m *safeMap[K, V]) Range(fn func(key K, value V)) {
	m.mu.RLock()
	for key, value := range m.data {
		fn(key, value)
	}
	m.mu.RUnlock()
}

// Count returns the number of elements in the map.
func (m *safeMap[K, V]) Count() (count int64) {
	m.mu.RLock()
	count = int64(len(m.data))
	m.mu.RUnlock()
	return count
}

// Keys returns key list of elements in the map.
func (m *safeMap[K, V]) Keys() []K {
	var keys []K
	m.mu.RLock()
	for key := range m.data {
		keys = append(keys, key)
	}
	m.mu.RUnlock()
	return keys
}
