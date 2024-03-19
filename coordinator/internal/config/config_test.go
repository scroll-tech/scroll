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
		"prover_manager": {
			"provers_per_session": 1,
			"session_attempts": 5,
			"batch_collection_time_sec": 180,
			"chunk_collection_time_sec": 180,
			"verifier": {
				"mock_mode": true,
				"params_path": "",
				"agg_vk_path": ""
			},
			"max_verifier_workers": 4,
			"min_prover_version": "v1.0.0"
		},
		"db": {
			"driver_name": "postgres",
			"dsn": "postgres://admin:123456@localhost/test?sslmode=disable",
			"maxOpenNum": 200,
			"maxIdleNum": 20
		},
		"l2": {
			"chain_id": 111
		},
 		"auth": {
			"secret": "prover secret key",
			"challenge_expire_duration_sec": 3600,
			"login_expire_duration_sec": 3600
  		}
	}`

	t.Run("Success Case", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "example")
		assert.NoError(t, err)
		defer func() {
			assert.NoError(t, tmpFile.Close())
			assert.NoError(t, os.Remove(tmpFile.Name()))
		}()
		_, err = tmpFile.WriteString(configTemplate)
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

		assert.NoError(t, os.WriteFile(tmpJSON, data, 0644))

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
}
