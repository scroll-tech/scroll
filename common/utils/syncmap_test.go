package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSyncMap(t *testing.T) {
	key, value := "1", "1"

	mp := SyncMap[string, *string]{}
	mp.LoadOrStore(key, nil)
	assert.Equal(t, int64(1), mp.Count())

	// test store and load
	mp.Store("1", &value)
	assert.Equal(t, int64(1), mp.Count())
	expect, ok := mp.Load("1")
	assert.Equal(t, true, ok)
	assert.Equal(t, value, *expect)

	expect, ok = mp.LoadAndDelete(key)
	assert.Equal(t, true, ok)
	assert.Equal(t, value, *expect)
	assert.Equal(t, true, mp.Count() == 0)

	mp.Store("1", nil)
	value = "2"
	mp.Store("2", &value)
	val1 := "3"
	mp.Store("3", &val1)

	mp.Range(func(key string, value *string) bool {
		t.Logf("key: %s, value: %v\n", key, value)
		return true
	})
}
