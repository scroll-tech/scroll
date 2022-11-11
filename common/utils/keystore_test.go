package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

var keyDir = "key-dir"

func TestLoadOrCreateKey(t *testing.T) {
	err := os.RemoveAll(keyDir)
	assert.NoError(t, err)
	// no dir.
	ksPath := filepath.Join(keyDir, "my-key")
	_, err = LoadOrCreateKey(ksPath, "pwd")
	assert.NoError(t, err)
	err = os.RemoveAll(ksPath)
	assert.NoError(t, err)

	// only has dir, no file.
	err = os.MkdirAll(keyDir, os.ModeDir)
	assert.NoError(t, err)
	_, err = LoadOrCreateKey(ksPath, "pwd")
	assert.NoError(t, err)

	// load keystore
	_, err = LoadOrCreateKey(ksPath, "pwd")
	assert.NoError(t, err)
	os.RemoveAll(keyDir)
}
