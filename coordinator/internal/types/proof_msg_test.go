package types

import (
	"encoding/hex"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
)

func TestProofMessageSignAndVerify(t *testing.T) {
	privateKey, err := crypto.GenerateKey()
	assert.NoError(t, err)
	publicKeyHex := common.Bytes2Hex(crypto.CompressPubkey(&privateKey.PublicKey))

	var proofMsg ProofMsg
	t.Run("sign", func(t *testing.T) {
		proofMsg = ProofMsg{
			ProofDetail: &ProofDetail{
				ID:     "testID",
				Type:   ProofTypeChunk,
				Status: StatusOk,
				ChunkProof: &ChunkProof{
					StorageTrace: []byte("testStorageTrace"),
					Protocol:     []byte("testProtocol"),
					Proof:        []byte("testProof"),
					Instances:    []byte("testInstance"),
					Vk:           []byte("testVk"),
					ChunkInfo:    nil,
				},
				Error: "testError",
			},
			publicKey: publicKeyHex,
		}
		err = proofMsg.Sign(privateKey)
		assert.NoError(t, err)
	})

	t.Run("valid verify", func(t *testing.T) {
		ok, verifyErr := proofMsg.Verify()
		assert.True(t, ok)
		assert.NoError(t, verifyErr)
	})

	t.Run("invalid verify", func(t *testing.T) {
		proofMsg.ProofDetail.ChunkProof.Proof = []byte("new proof")
		ok, verifyErr := proofMsg.Verify()
		assert.False(t, ok)
		assert.NoError(t, verifyErr)
	})
}

func TestProofDetailHash(t *testing.T) {
	proofDetail := &ProofDetail{
		ID:     "testID",
		Type:   ProofTypeChunk,
		Status: StatusOk,
		ChunkProof: &ChunkProof{
			StorageTrace: []byte("testStorageTrace"),
			Protocol:     []byte("testProtocol"),
			Proof:        []byte("testProof"),
			Instances:    []byte("testInstance"),
			Vk:           []byte("testVk"),
			ChunkInfo:    nil,
		},
		Error: "testError",
	}
	hash, err := proofDetail.Hash()
	assert.NoError(t, err)
	expectedHash := "01128ea9006601146ba80dbda959c96ebaefca463e78570e473a57d821db5ec1"
	assert.Equal(t, expectedHash, hex.EncodeToString(hash))
}
