package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"scroll-tech/bridge/config"
)

func TestConfig(t *testing.T) {
	cfg, err := config.NewConfig("../config.json")
	assert.True(t, assert.NoError(t, err), "failed to load config")
	assert.True(t, len(cfg.L2Config.SkippedOpcodes) == 2)
	assert.True(t, cfg.L2Config.ProofGenerationFreq == 1)
	assert.True(t, len(cfg.L1Config.RelayerConfig.MessageSenderPrivateKeys) > 0)
	assert.True(t, len(cfg.L2Config.RelayerConfig.MessageSenderPrivateKeys) > 0)
	assert.True(t, len(cfg.L2Config.RelayerConfig.RollupSenderPrivateKeys) > 0)
}
