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

		data, err := json.Marshal(cfg)
		assert.NoError(t, err)

		tmpJSON := fmt.Sprintf("/tmp/%d_rollup_config.json", time.Now().Nanosecond())
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

	t.Run("Override config value", func(t *testing.T) {
		cfg, err := NewConfig("../../conf/config.json")
		assert.NoError(t, err)

		os.Setenv("SCROLL_ROLLUP_DB_CONFIG_DSN", "postgres://test:test@postgresql:5432/scroll?sslmode=disable")
		os.Setenv("SCROLL_ROLLUP_L1_CONFIG_RELAYER_CONFIG_GAS_ORACLE_SENDER_PRIVATE_KEY", "1616161616161616161616161616161616161616161616161616161616161616")
		os.Setenv("SCROLL_ROLLUP_L2_CONFIG_RELAYER_CONFIG_GAS_ORACLE_SENDER_PRIVATE_KEY", "1717171717171717171717171717171717171717171717171717171717171717")
		os.Setenv("SCROLL_ROLLUP_L2_CONFIG_RELAYER_CONFIG_COMMIT_SENDER_PRIVATE_KEY", "1818181818181818181818181818181818181818181818181818181818181818")
		os.Setenv("SCROLL_ROLLUP_L2_CONFIG_RELAYER_CONFIG_FINALIZE_SENDER_PRIVATE_KEY", "1919191919191919191919191919191919191919191919191919191919191919")

		cfg2, err := NewConfig("../../conf/config.json")
		assert.NoError(t, err)

		assert.NotEqual(t, cfg.DBConfig.DSN, cfg2.DBConfig.DSN)
		assert.NotEqual(t, cfg.L1Config.RelayerConfig.GasOracleSenderPrivateKey, cfg2.L1Config.RelayerConfig.GasOracleSenderPrivateKey)
		assert.NotEqual(t, cfg.L2Config.RelayerConfig.GasOracleSenderPrivateKey, cfg2.L2Config.RelayerConfig.GasOracleSenderPrivateKey)
		assert.NotEqual(t, cfg.L2Config.RelayerConfig.CommitSenderPrivateKey, cfg2.L2Config.RelayerConfig.CommitSenderPrivateKey)
		assert.NotEqual(t, cfg.L2Config.RelayerConfig.FinalizeSenderPrivateKey, cfg2.L2Config.RelayerConfig.FinalizeSenderPrivateKey)

		assert.Equal(t, cfg2.DBConfig.DSN, "postgres://test:test@postgresql:5432/scroll?sslmode=disable")
		assert.Equal(t, "1616161616161616161616161616161616161616161616161616161616161616", cfg2.L1Config.RelayerConfig.GasOracleSenderPrivateKey)
		assert.Equal(t, "1717171717171717171717171717171717171717171717171717171717171717", cfg2.L2Config.RelayerConfig.GasOracleSenderPrivateKey)
		assert.Equal(t, "1818181818181818181818181818181818181818181818181818181818181818", cfg2.L2Config.RelayerConfig.CommitSenderPrivateKey)
		assert.Equal(t, "1919191919191919191919191919191919191919191919191919191919191919", cfg2.L2Config.RelayerConfig.FinalizeSenderPrivateKey)
	})
}
