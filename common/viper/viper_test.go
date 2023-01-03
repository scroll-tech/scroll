package viper_test

import (
	"bytes"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	config "scroll-tech/common/apollo"
	"scroll-tech/common/viper"
)

func TestViper(t *testing.T) {
	vp := viper.New()
	vp.SetConfigFile("../../bridge/config.json")
	assert.NoError(t, vp.ReadInFile())

	sb := vp.Sub("l2_config.relayer_config.sender_config")
	assert.Equal(t, 10, sb.GetInt("check_pending_time_sec"))
	assert.Equal(t, "DynamicFeeTx", sb.GetString("tx_type"))

	sb.Set("confirmations", 20)
	assert.Equal(t, 20, sb.GetInt("confirmations"))

	relayer := vp.Sub("l2_config.relayer_config")
	assert.Equal(t, "0x0000000000000000000000000000000000000000", relayer.GetString("rollup_contract_address"))

	relayer.Set("sender_config.confirmations", 14)
	assert.Equal(t, 14, sb.GetInt("confirmations"))

	sender := relayer.Sub("sender_config")
	assert.Equal(t, 14, sender.GetInt("confirmations"))

	sender.Set("confirmations", 33)
	assert.Equal(t, 33, sender.GetInt("confirmations"))

	vp.Set("l2_config.relayer_config.sender_config.confirmations", 15)
	assert.Equal(t, 15, sb.GetInt("confirmations"))
	assert.Equal(t, 15, sender.GetInt("confirmations"))

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

func TestApolloFlush(t *testing.T) {
	agolloClient := config.MustInitApollo()

	vp := viper.New()
	vp.SetConfigFile("../../bridge/config.json")
	assert.NoError(t, vp.ReadInFile())
	l2Sender := vp.Sub("l2_config.relayer_config.sender_config")
	l2Relayer := vp.Sub("l2_config.relayer_config")

	for i := 0; i < 3; i++ {
		t.Log("tx type: ", l2Sender.GetString("tx_type"))
		t.Log("confirmations: ", l2Sender.GetInt("confirmations"))
		t.Log("rollup contract address: ", l2Relayer.GetString("rollup_contract_address"))

		cfgStr := agolloClient.GetStringValue("bridge_config", "")
		assert.NoError(t, vp.ReadConfig(bytes.NewReader([]byte(cfgStr))))
		<-time.After(time.Second * 3)
	}
}
