package database

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
		"dsn": "postgres://postgres:123456@localhost:5444/test?sslmode=disable",
		"driver_name": "postgres",
		"maxOpenNum": %d,
		"maxIdleNum": %d,
		"maxLifetime": %d,
		"maxIdleTime": %d
	}`

	t.Run("Success Case", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "example")
		assert.NoError(t, err)
		defer func() {
			assert.NoError(t, tmpFile.Close())
			assert.NoError(t, os.Remove(tmpFile.Name()))
		}()
		config := fmt.Sprintf(configTemplate, 200, 20)
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
