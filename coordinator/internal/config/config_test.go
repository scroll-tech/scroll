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
	configTemplate := `{
		"compression_level": 9,
		"rollers_per_session": 1,
		"session_attempts": %d,
		"collection_time": 180,
		"token_time_to_live": 60,
		"verifier": {
			"mock_mode": true,
			"params_path": "",
			"agg_vk_path": ""
		},
		"max_verifier_workers": %d,
		"order_session": "%s",
		"db_config": {
			"driver_name": "postgres",
			"dsn": "postgres://admin:123456@localhost/test?sslmode=disable",
			"maxOpenNum": 200,
			"maxIdleNum": 20
		},
		"l2_config": {
			"endpoint": "/var/lib/jenkins/workspace/SequencerPipeline/MyPrivateNetwork/geth.ipc"
		}
	}`

	t.Run("Success Case", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "example")
		assert.NoError(t, err)
		defer func() {
			assert.NoError(t, tmpFile.Close())
			assert.NoError(t, os.Remove(tmpFile.Name()))
		}()
		config := fmt.Sprintf(configTemplate, defaultNumberOfSessionRetryAttempts, defaultNumberOfVerifierWorkers, "ASC")
		_, err = tmpFile.WriteString(config)
		assert.NoError(t, err)

		cfg, err := NewConfig(tmpFile.Name())
		assert.NoError(t, err)

		data, err := json.Marshal(cfg)
		assert.NoError(t, err)
		tmpJSON := fmt.Sprintf("/tmp/%d_config.json", time.Now().Nanosecond())
		defer func() {
			if _, err = os.Stat(tmpJSON); err == nil {
				assert.NoError(t, os.Remove(tmpJSON))
			}
		}()

		assert.NoError(t, os.WriteFile(tmpJSON, data, 0o644))

		cfg2, err := NewConfig(tmpJSON)
		assert.NoError(t, err)
		assert.Equal(t, cfg, cfg2)
	})

	t.Run("File Not Found", func(t *testing.T) {
		_, err := NewConfig("non_existent_file.json")
		assert.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("Invalid JSON Content", func(t *testing.T) {
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

	t.Run("Invalid Order Session", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "example")
		assert.NoError(t, err)
		defer func() {
			assert.NoError(t, tmpFile.Close())
			assert.NoError(t, os.Remove(tmpFile.Name()))
		}()
		config := fmt.Sprintf(configTemplate, defaultNumberOfSessionRetryAttempts, defaultNumberOfVerifierWorkers, "INVALID")
		_, err = tmpFile.WriteString(config)
		assert.NoError(t, err)

		_, err = NewConfig(tmpFile.Name())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "roller config's order session is invalid")
	})

	t.Run("Default MaxVerifierWorkers", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "example")
		assert.NoError(t, err)
		defer func() {
			assert.NoError(t, tmpFile.Close())
			assert.NoError(t, os.Remove(tmpFile.Name()))
		}()
		config := fmt.Sprintf(configTemplate, defaultNumberOfSessionRetryAttempts, 0, "ASC")
		_, err = tmpFile.WriteString(config)
		assert.NoError(t, err)

		cfg, err := NewConfig(tmpFile.Name())
		assert.NoError(t, err)
		assert.Equal(t, defaultNumberOfVerifierWorkers, cfg.MaxVerifierWorkers)
	})

	t.Run("Default SessionAttempts", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "example")
		assert.NoError(t, err)
		defer func() {
			assert.NoError(t, tmpFile.Close())
			assert.NoError(t, os.Remove(tmpFile.Name()))
		}()
		config := fmt.Sprintf(configTemplate, 0, defaultNumberOfVerifierWorkers, "ASC")
		_, err = tmpFile.WriteString(config)
		assert.NoError(t, err)

		cfg, err := NewConfig(tmpFile.Name())
		assert.NoError(t, err)
		assert.Equal(t, defaultNumberOfSessionRetryAttempts, cfg.SessionAttempts)
	})
}
