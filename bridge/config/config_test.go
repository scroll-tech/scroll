package config_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"scroll-tech/bridge/config"
	"scroll-tech/common/viper"
)

func TestConfig(t *testing.T) {
	vp, err := viper.NewViper("../config.json")
	assert.NoError(t, err)

	l1MessageSenderPrivateKeys, err := config.UnmarshalPrivateKeys(vp.Sub("l1_config.relayer_config").GetStringSlice("message_sender_private_keys"))
	assert.NoError(t, err)
	assert.True(t, len(l1MessageSenderPrivateKeys) > 0)

	l2MessageSenderPrivateKeys, err := config.UnmarshalPrivateKeys(vp.Sub("l2_config.relayer_config").GetStringSlice("message_sender_private_keys"))
	assert.NoError(t, err)
	assert.True(t, len(l2MessageSenderPrivateKeys) > 0)

	rollupSenderPrivateKeys, err := config.UnmarshalPrivateKeys(vp.Sub("l2_config.relayer_config").GetStringSlice("rollup_sender_private_keys"))
	assert.NoError(t, err)
	assert.True(t, len(rollupSenderPrivateKeys) > 0)

	l1MinBalance, err := config.UnmarshalMinBalance(vp.Sub("l1_config.relayer_config.sender_config").GetString("min_balance"))
	assert.NoError(t, err)
	assert.True(t, l1MinBalance.Cmp(big.NewInt(0)) > 0)

	l2MinBalance, err := config.UnmarshalMinBalance(vp.Sub("l2_config.relayer_config.sender_config").GetString("min_balance"))
	assert.NoError(t, err)
	assert.True(t, l2MinBalance.Cmp(big.NewInt(0)) > 0)
}
