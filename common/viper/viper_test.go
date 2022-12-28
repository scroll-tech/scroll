package viper

import (
	"testing"
	"time"

	originVP "github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestViper(t *testing.T) {
	SetConfigFile("../../bridge/config.json")
	assert.NoError(t, ReadInConfig())

	sb := Sub("l2_config.relayer_config.sender_config")
	assert.Equal(t, 10, sb.GetInt("check_pending_time"))
	assert.Equal(t, "DynamicFeeTx", sb.GetString("tx_type"))

	sb.Set("confirmations", 20)
	assert.Equal(t, 20, sb.GetInt("confirmations"))

	relayer := Sub("l2_config.relayer_config")
	assert.Equal(t, "0x0000000000000000000000000000000000000000", relayer.GetString("rollup_contract_address"))

	relayer.Set("sender_config.confirmations", 14)
	assert.Equal(t, 14, sb.GetInt("confirmations"))

	sender := relayer.Sub("sender_config")
	assert.Equal(t, 14, sender.GetInt("confirmations"))

	sender.Set("confirmations", 33)
	assert.Equal(t, 33, sender.GetInt("confirmations"))

	Set("l2_config.relayer_config.sender_config.confirmations", 15)
	assert.Equal(t, 15, sb.GetInt("confirmations"))
	assert.Equal(t, 15, sender.GetInt("confirmations"))
}

func TestFlush(t *testing.T) {
	origin := originVP.New()
	origin.SetConfigFile("../../bridge/config.json")
	origin.WatchConfig()
	assert.NoError(t, origin.ReadInConfig())

	SetConfigFile("../../bridge/config.json")
	assert.NoError(t, ReadInConfig())

	l2Sender := Sub("l2_config.relayer_config.sender_config")
	l2relayer := Sub("l2_config.relayer_config")
	for i := 0; i < 20; i++ {
		Flush(origin)
		t.Log("tx type: ", l2Sender.GetString("tx_type"))
		t.Log("confirmations: ", l2Sender.GetInt("confirmations"))
		t.Log("rollup contract address: ", l2relayer.GetString("rollup_contract_address"))
		<-time.After(time.Second * 3)
	}
}
