package config_test

import (
	"math/big"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"scroll-tech/bridge/config"
)

func TestConfig(t *testing.T) {
	assert.NoError(t, config.NewConfig("../config.json"))

	skippedOpcodes := viper.GetStringSlice("l2_config.batch_proposer_config.skipped_opcodes")
	assert.True(t, len(skippedOpcodes) > 0)

	l1MessageSenderPrivateKeys, err := config.UnmarshalPrivateKeys(viper.GetStringSlice("l1_config.relayer_config.message_sender_private_keys"))
	assert.NoError(t, err)
	assert.True(t, len(l1MessageSenderPrivateKeys) > 0)

	l2MessageSenderPrivateKeys, err := config.UnmarshalPrivateKeys(viper.GetStringSlice("l2_config.relayer_config.message_sender_private_keys"))
	assert.NoError(t, err)
	assert.True(t, len(l2MessageSenderPrivateKeys) > 0)

	rollupSenderPrivateKeys, err := config.UnmarshalPrivateKeys(viper.GetStringSlice("l2_config.relayer_config.rollup_sender_private_keys"))
	assert.NoError(t, err)
	assert.True(t, len(rollupSenderPrivateKeys) > 0)

	l1MinBalance, err := config.UnmarshalMinBalance(viper.GetString("l1_config.relayer_config.sender_config.min_balance"))
	assert.NoError(t, err)
	assert.True(t, l1MinBalance.Cmp(big.NewInt(0)) > 0)

	l2MinBalance, err := config.UnmarshalMinBalance(viper.GetString("l2_config.relayer_config.sender_config.min_balance"))
	assert.NoError(t, err)
	assert.True(t, l2MinBalance.Cmp(big.NewInt(0)) > 0)
}
