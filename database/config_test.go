package database

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
		"dsn": "postgres://postgres:123456@localhost:5444/test?sslmode=disable",
		"driver_name": "postgres",
		"maxOpenNum": %d,
		"maxIdleNum": %d
	}`

	t.Run("Success Case", func(t *testing.T) {
		tmpfile, _ := ioutil.TempFile("", "example")
		config := fmt.Sprintf(configTemplate, 200, 20)
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
}
