package types

import (
	"encoding/hex"
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
				ProverTypes:   []ProverType{ProverTypeBatch},
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

// TestGenerateSignature this unit test isn't for test, just generate the signature for manually test.
func TestGenerateSignature(t *testing.T) {
	privateKeyHex := "8b8df68fddf7ee2724b79ccbd07799909d59b4dd4f4df3f6ecdc4fb8d56bdf4c"
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	assert.Nil(t, err)
	privateKey, err := crypto.ToECDSA(privateKeyBytes)
	assert.NoError(t, err)
	assert.NoError(t, err)
	publicKeyHex := common.Bytes2Hex(crypto.CompressPubkey(&privateKey.PublicKey))

	t.Log("publicKey: ", publicKeyHex)

	authMsg := LoginParameter{
		Message: Message{
			ProverName:    "test",
			ProverVersion: "v4.4.32-37af5ef5-38a68e2-1c5093c",
			Challenge:     "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MjEzMjc5MTIsIm9yaWdfaWF0IjoxNzIxMzI0MzEyLCJyYW5kb20iOiJWMVFlT19yNEV5eGRmYUtDalprVExEa0ZIemEyNTdQRG93dTV4SnVxYTdZPSJ9.x-B_TnkTUvs8-hiMfJXejxetAP6rXfeRUmyZ3S0uBiM",
			ProverTypes:   []ProverType{ProverTypeBatch},
			VKs: []string{"AAAAGgAAAARX2S0K1wF333B1waOsnG/vcASJmWG9YM6SNWCBy1ywD9jfGkei+f0wNYpkjW7JO12EfU7CjYVBo+PGku3zaQJI64lbn6BwyTBa4RfrPFpV5mP47ix0sXZ+Wt5wklMLRW7OIJb1yfCDm+gkSsp3/Zqrxt4SY4rQ4WtHfynTCQ0KDi78jNuiFvwxO3ub3DkgGVaxMkGxTRP/Vz6E7MCZMUBR5wZFcMzJn+73f0wYjDxfj00krg9O1VrwVxbVV1ycLR6oQLcOgm/l+xwth8io0vDpF9OY21gD5DgJn9GgcYe8KoRVEbEqApLZPdBibpcSMTY9czZI2LnFcqrDDmYvhEwgjhZrsTog2xLXOODoOupZ/is5ekQ9Gi0y871b1mLlCGA=",
				"AAAAGgAAAARX2S0K1wF333B1waOsnG/vcASJmWG9YM6SNWCBy1ywD1DEjW4Kell67H07wazT5DdzrSh4+amh+cmosQHp9p9snFypyoBGt3UHtoJGQBZlywZWDS9ht5pnaEoGBdaKcQk+lFb+WxTiId0KOAa0mafTZTQw8yToy57Jple64qzlRu1dux30tZZGuerLN1CKzg5Xl2iOpMK+l87jCINwVp5cUtF/XrvhBbU7onKh3KBiy99iUqVyA3Y6iiIZhGKWBSuSA4bNgDYIoVkqjHpdL35aEShoRO6pNXt7rDzxFoPzH0JuPI54nE4OhVrzZXwtkAEosxVa/fszcE092FH+HhhtxZBYe/KEzwdISU9TOPdId3UF/UMYC0MiYOlqffVTgAg="},
		},
		PublicKey: publicKeyHex,
	}
	err = authMsg.SignWithKey(privateKey)
	assert.NoError(t, err)
	t.Log("signature: ", authMsg.Signature)

	verify, err := authMsg.Verify()
	assert.NoError(t, err)
	assert.True(t, verify)
}
