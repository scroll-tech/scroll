package config_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"scroll-tech/common/viper"
)

func TestConfig(t *testing.T) {
	vp, err := viper.NewViper("../config.json")
	assert.NoError(t, err)

	l1MessageSenderPrivateKeys := vp.Sub("l1_config.relayer_config").GetECDSAKeys("message_sender_private_keys")
	assert.True(t, len(l1MessageSenderPrivateKeys) > 0)

	l2MessageSenderPrivateKeys := vp.Sub("l2_config.relayer_config").GetECDSAKeys("message_sender_private_keys")
	assert.True(t, len(l2MessageSenderPrivateKeys) > 0)

	rollupSenderPrivateKeys := vp.Sub("l2_config.relayer_config").GetECDSAKeys("rollup_sender_private_keys")
	assert.True(t, len(rollupSenderPrivateKeys) > 0)

	l1MinBalance := vp.Sub("l1_config.relayer_config.sender_config").GetBigInt("min_balance")
	assert.True(t, l1MinBalance.Cmp(big.NewInt(0)) > 0)

	l2MinBalance := vp.Sub("l2_config.relayer_config.sender_config").GetBigInt("min_balance")
	assert.True(t, l2MinBalance.Cmp(big.NewInt(0)) > 0)
}
