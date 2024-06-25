package types

import (
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
)

func TestAuthMessageSignAndVerify(t *testing.T) {
	privateKey, err := crypto.GenerateKey()
	assert.NoError(t, err)
	publicKeyHex := common.Bytes2Hex(crypto.CompressPubkey(&privateKey.PublicKey))

	var authMsg LoginParameter
	t.Run("sign", func(t *testing.T) {
		authMsg = LoginParameter{
			Message: Message{
				ProverName:    "test1",
				ProverVersion: "v0.0.1",
				Challenge:     "abcdef",
				VKs:           []string{"vk1", "vk2"},
			},
			PublicKey: publicKeyHex,
		}

		err = authMsg.SignWithKey(privateKey)
		assert.NoError(t, err)
	})

	t.Run("valid verify", func(t *testing.T) {
		ok, verifyErr := authMsg.Verify()
		assert.True(t, ok)
		assert.NoError(t, verifyErr)
	})

	t.Run("invalid verify", func(t *testing.T) {
		authMsg.Message.Challenge = "abcdefgh"
		ok, verifyErr := authMsg.Verify()
		assert.False(t, ok)
		assert.NoError(t, verifyErr)
	})
}
