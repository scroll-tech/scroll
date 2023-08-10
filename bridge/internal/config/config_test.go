package config

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	t.Run("Success Case", func(t *testing.T) {
		cfg, err := NewConfig("../../conf/config.json")
		assert.NoError(t, err)

		assert.Len(t, cfg.L1Config.RelayerConfig.MessageSenderPrivateKeys, 1)
		assert.Len(t, cfg.L2Config.RelayerConfig.MessageSenderPrivateKeys, 1)
		assert.Len(t, cfg.L2Config.RelayerConfig.RollupSenderPrivateKeys, 1)

		data, err := json.Marshal(cfg)
		assert.NoError(t, err)

		tmpJSON := fmt.Sprintf("/tmp/%d_bridge_config.json", time.Now().Nanosecond())
		defer func() {
			if _, err = os.Stat(tmpJSON); err == nil {
				assert.NoError(t, os.Remove(tmpJSON))
			}
		}()

		assert.NoError(t, os.WriteFile(tmpJSON, data, 0644))

		cfg2, err := NewConfig(tmpJSON)
		assert.NoError(t, err)

		assert.Equal(t, cfg.L1Config, cfg2.L1Config)
		assert.Equal(t, cfg.L2Config, cfg2.L2Config)
		assert.Equal(t, cfg.DBConfig, cfg2.DBConfig)
	})

	t.Run("File Not Found", func(t *testing.T) {
		_, err := NewConfig("non_existent_file.json")
		assert.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("Invalid JSON Content", func(t *testing.T) {
		// Create a temporary file with invalid JSON content
		tmpFile, err := os.CreateTemp("", "invalid_json_config.json")
		assert.NoError(t, err)
		defer func() {
			assert.NoError(t, tmpFile.Close())
			assert.NoError(t, os.Remove(tmpFile.Name()))
		}()

		_, err = tmpFile.WriteString("{ invalid_json: ")
		assert.NoError(t, err)

		_, err = NewConfig(tmpFile.Name())
		assert.Error(t, err)
	})
}
