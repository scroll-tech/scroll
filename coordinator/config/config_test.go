package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	configTemplate := `{
		"roller_manager_config": {
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
			"order_session": "%s"
		},
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
		tmpfile, _ := ioutil.TempFile("", "example")
		config := fmt.Sprintf(configTemplate, defaultNumberOfSessionRetryAttempts, defaultNumberOfVerifierWorkers, "ASC")
		_, _ = tmpfile.WriteString(config)
		defer os.Remove(tmpfile.Name())

		cfg, err := NewConfig(tmpfile.Name())
		assert.NoError(t, err)

		data, _ := json.Marshal(cfg)
		tmpJSON := fmt.Sprintf("/tmp/%d_config.json", time.Now().Nanosecond())
		defer os.Remove(tmpJSON)

		os.WriteFile(tmpJSON, data, 0644)

		cfg2, err := NewConfig(tmpJSON)
		assert.NoError(t, err)

		assert.Equal(t, cfg, cfg2)
	})

	t.Run("File Not Found", func(t *testing.T) {
		_, err := NewConfig("non_existent_file.json")
		assert.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("Invalid JSON Content", func(t *testing.T) {
		tempFile, _ := os.CreateTemp("", "invalid_json_config.json")
		defer os.Remove(tempFile.Name())

		tempFile.WriteString("{ invalid_json: ")

		_, err := NewConfig(tempFile.Name())
		assert.Error(t, err)
	})

	t.Run("Invalid Order Session", func(t *testing.T) {
		tmpfile, _ := ioutil.TempFile("", "example")
		config := fmt.Sprintf(configTemplate, defaultNumberOfSessionRetryAttempts, defaultNumberOfVerifierWorkers, "INVALID")
		_, _ = tmpfile.WriteString(config)
		defer os.Remove(tmpfile.Name())

		_, err := NewConfig(tmpfile.Name())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "roller config's order session is invalid")
	})

	t.Run("Default MaxVerifierWorkers", func(t *testing.T) {
		tmpfile, _ := ioutil.TempFile("", "example")
		config := fmt.Sprintf(configTemplate, defaultNumberOfSessionRetryAttempts, 0, "ASC")
		_, _ = tmpfile.WriteString(config)
		defer os.Remove(tmpfile.Name())

		cfg, err := NewConfig(tmpfile.Name())
		assert.NoError(t, err)
		assert.Equal(t, defaultNumberOfVerifierWorkers, cfg.RollerManagerConfig.MaxVerifierWorkers)
	})

	t.Run("Default SessionAttempts", func(t *testing.T) {
		tmpfile, _ := ioutil.TempFile("", "example")
		config := fmt.Sprintf(configTemplate, 0, defaultNumberOfVerifierWorkers, "ASC")
		_, _ = tmpfile.WriteString(config)
		defer os.Remove(tmpfile.Name())

		cfg, err := NewConfig(tmpfile.Name())
		assert.NoError(t, err)
		assert.Equal(t, uint8(defaultNumberOfSessionRetryAttempts), cfg.RollerManagerConfig.SessionAttempts)
	})
}
