package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSyncMap(t *testing.T) {
	mp := SyncMap[string, *string]{}
	mp.LoadOrStore("1", nil)
	assert.Equal(t, int64(1), mp.Count())

	val := "1"
	mp.Store("1", &val)
	assert.Equal(t, int64(1), mp.Count())
}
