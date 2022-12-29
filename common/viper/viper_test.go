package viper_test

import (
	"math/big"
	"testing"
	"time"

	originVP "github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/viper"
)

func TestViper(t *testing.T) {
	vp, err := viper.NewViper("../../bridge/config.json")
	assert.NoError(t, err)

	sb := vp.Sub("l2_config.relayer_config.sender_config")
	assert.Equal(t, 10, sb.GetInt("check_pending_time"))
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

func TestFlush(t *testing.T) {
	origin := originVP.New()
	origin.SetConfigFile("../../bridge/config.json")
	origin.WatchConfig()
	assert.NoError(t, origin.ReadInConfig())

	vp, err := viper.NewViper("../../bridge/config.json")
	assert.NoError(t, err)

	l2Sender := vp.Sub("l2_config.relayer_config.sender_config")
	l2relayer := vp.Sub("l2_config.relayer_config")
	for i := 0; i < 3; i++ {
		vp.Flush(origin)
		t.Log("tx type: ", l2Sender.GetString("tx_type"))
		t.Log("confirmations: ", l2Sender.GetInt("confirmations"))
		t.Log("rollup contract address: ", l2relayer.GetString("rollup_contract_address"))
		<-time.After(time.Second * 3)
	}
}
