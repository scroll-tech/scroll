package viper_test

import (
	"bytes"
	"math/big"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

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

func TestLoadConfig(t *testing.T) {
	vp := viper.New()
	vp.SetConfigType("json")

	data, err := os.ReadFile("../../bridge/config.json")
	assert.NoError(t, err)
	assert.NoError(t, vp.ReadConfig(bytes.NewReader(data)))

	l1 := vp.Sub("l1_config")
	l1.SetConfigType("json")

	l1Buf := bytes.Buffer{}
	assert.NoError(t, l1.WriteConfig(&l1Buf))
	str1 := l1Buf.String()

	l1Tmp := viper.New()
	l1Tmp.SetConfigType("json")
	assert.NoError(t, l1Tmp.ReadConfig(&l1Buf))

	l1TmpBuf := bytes.Buffer{}
	assert.NoError(t, l1Tmp.WriteConfig(&l1TmpBuf))

	str2 := l1TmpBuf.String()
	assert.Equal(t, str1, str2)
}
