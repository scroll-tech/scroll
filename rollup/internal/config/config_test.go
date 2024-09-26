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
		os.Setenv("SCROLL_ROLLUP_L1_CONFIG_RELAYER_CONFIG_GAS_ORACLE_SENDER_SIGNER_CONFIG_SIGNER_ADDRESS", "0x03a1Bba60B5Aa37094cf16123AdD674c01589489")
		os.Setenv("SCROLL_ROLLUP_L2_CONFIG_RELAYER_CONFIG_GAS_ORACLE_SENDER_SIGNER_CONFIG_SIGNER_ADDRESS", "0x03a1Bba60B5Aa37094cf16123AdD674c01589480")
		os.Setenv("SCROLL_ROLLUP_L2_CONFIG_RELAYER_CONFIG_COMMIT_SENDER_SIGNER_CONFIG_PRIVATE_KEY", "1818181818181818181818181818181818181818181818181818181818181818")
		os.Setenv("SCROLL_ROLLUP_L2_CONFIG_RELAYER_CONFIG_FINALIZE_SENDER_SIGNER_CONFIG_SIGNER_ADDRESS", "0x33e0F539E31B35170FAaA062af703b76a8282bf8")

		cfg2, err := NewConfig("../../conf/config.json")
		assert.NoError(t, err)

		assert.NotEqual(t, cfg.DBConfig.DSN, cfg2.DBConfig.DSN)
		assert.NotEqual(t, cfg.L1Config.RelayerConfig.GasOracleSenderSignerConfig, cfg2.L1Config.RelayerConfig.GasOracleSenderSignerConfig)
		assert.NotEqual(t, cfg.L2Config.RelayerConfig.GasOracleSenderSignerConfig, cfg2.L2Config.RelayerConfig.GasOracleSenderSignerConfig)
		assert.NotEqual(t, cfg.L2Config.RelayerConfig.CommitSenderSignerConfig, cfg2.L2Config.RelayerConfig.CommitSenderSignerConfig)
		assert.NotEqual(t, cfg.L2Config.RelayerConfig.FinalizeSenderSignerConfig, cfg2.L2Config.RelayerConfig.FinalizeSenderSignerConfig)

		assert.Equal(t, cfg2.DBConfig.DSN, "postgres://test:test@postgresql:5432/scroll?sslmode=disable")
		assert.Equal(t, "0x03a1Bba60B5Aa37094cf16123AdD674c01589489", cfg2.L1Config.RelayerConfig.GasOracleSenderSignerConfig.SignerAddress)
		assert.Equal(t, "0x03a1Bba60B5Aa37094cf16123AdD674c01589480", cfg2.L2Config.RelayerConfig.GasOracleSenderSignerConfig.SignerAddress)
		assert.Equal(t, "1818181818181818181818181818181818181818181818181818181818181818", cfg2.L2Config.RelayerConfig.CommitSenderSignerConfig.PrivateKey)
		assert.Equal(t, "0x33e0F539E31B35170FAaA062af703b76a8282bf8", cfg2.L2Config.RelayerConfig.FinalizeSenderSignerConfig.SignerAddress)
	})
}
