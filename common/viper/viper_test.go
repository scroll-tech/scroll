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
	vp := GetViper()
	t.Log(vp.AllKeys())

	sb := vp.Sub("l2_config.relayer_config.sender_config")
	t.Log(sb.GetInt("check_pending_time"), sb.GetString("tx_type"))

	/*viper.Set("l2_config.relayer_config.sender_config.confirmations", 20)
	t.Log(sb.GetInt("confirmations"))*/

	sb.Set("confirmations", 20)
	t.Log(sb.GetInt("confirmations"))

	relayer := vp.Sub("l2_config.relayer_config")
	t.Log(relayer.GetString("rollup_contract_address"))
	sender := relayer.Sub("sender_config")
	t.Log(sender.GetInt("confirmations"))
	sender.Set("confirmations", 33)
	t.Log(sender.GetInt("confirmations"))

	Set("l2_config.relayer_config.sender_config.confirmations", 15)
	t.Log(sb.GetInt("confirmations"))
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
