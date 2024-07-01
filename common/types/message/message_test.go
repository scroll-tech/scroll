package message

import (
	"encoding/hex"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
)

func TestAuthMessageSignAndVerify(t *testing.T) {
	privkey, err := crypto.GenerateKey()
	assert.NoError(t, err)

	authMsg := &AuthMsg{
		Identity: &Identity{
			Challenge:     "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2OTEwMzgxNzUsIm9yaWdfaWF0IjoxNjkxMDM0NTc1fQ.HybBMsEJFhyZqtIa2iVcHUP7CEFttf708jmTMAImAWA",
			ProverName:    "test",
			ProverVersion: "v1.0.0",
		},
	}
	assert.NoError(t, authMsg.SignWithKey(privkey))

	// Check public key.
	pk, err := authMsg.PublicKey()
	assert.NoError(t, err)
	assert.Equal(t, common.Bytes2Hex(crypto.CompressPubkey(&privkey.PublicKey)), pk)

	ok, err := authMsg.Verify()
	assert.NoError(t, err)
	assert.Equal(t, true, ok)

	// Check public key is ok.
	pub, err := authMsg.PublicKey()
	assert.NoError(t, err)
	pubkey := crypto.CompressPubkey(&privkey.PublicKey)
	assert.Equal(t, pub, common.Bytes2Hex(pubkey))
}

func TestGenerateToken(t *testing.T) {
	token, err := GenerateToken()
	assert.NoError(t, err)
	assert.Equal(t, 32, len(token))
}

func TestIdentityHash(t *testing.T) {
	identity := &Identity{
		Challenge:     "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2OTEwMzM0MTksIm9yaWdfaWF0IjoxNjkxMDI5ODE5fQ.EhkLZsj__rNPVC3ZDYBtvdh0nB8mmM_Hl82hObaIWOs",
		ProverName:    "test",
		ProverVersion: "v1.0.0",
	}

	hash, err := identity.Hash()
	assert.NoError(t, err)

	expectedHash := "9b8b00f5655411ec1d68ba1666261281c5414aedbda932e5b6a9f7f1b114fdf2"
	assert.Equal(t, expectedHash, hex.EncodeToString(hash))
}

func TestProofMessageSignVerifyPublicKey(t *testing.T) {
	privkey, err := crypto.GenerateKey()
	assert.NoError(t, err)

	proofMsg := &ProofMsg{
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
	}
	assert.NoError(t, proofMsg.Sign(privkey))

	// Test when publicKey is not set.
	ok, err := proofMsg.Verify()
	assert.NoError(t, err)
	assert.Equal(t, true, ok)

	// Test when publicKey is already set.
	ok, err = proofMsg.Verify()
	assert.NoError(t, err)
	assert.Equal(t, true, ok)
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
	expectedHash := "4c291e7582ee773add1c145270a6e704e00ba193b6118ee2c5fd646112bc867c"
	assert.Equal(t, expectedHash, hex.EncodeToString(hash))
}

func TestProveTypeString(t *testing.T) {
	proofTypeChunk := ProofType(1)
	assert.Equal(t, "proof type chunk", proofTypeChunk.String())

	proofTypeBatch := ProofType(2)
	assert.Equal(t, "proof type batch", proofTypeBatch.String())

	proofTypeBundle := ProofType(3)
	assert.Equal(t, "proof type bundle", proofTypeBundle.String())

	illegalProof := ProofType(4)
	assert.Equal(t, "illegal proof type: 4", illegalProof.String())
}

func TestProofMsgPublicKey(t *testing.T) {
	privkey, err := crypto.GenerateKey()
	assert.NoError(t, err)

	proofMsg := &ProofMsg{
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
	}
	assert.NoError(t, proofMsg.Sign(privkey))

	// Test when publicKey is not set.
	pk, err := proofMsg.PublicKey()
	assert.NoError(t, err)
	assert.Equal(t, common.Bytes2Hex(crypto.CompressPubkey(&privkey.PublicKey)), pk)

	// Test when publicKey is already set.
	proofMsg.publicKey = common.Bytes2Hex(crypto.CompressPubkey(&privkey.PublicKey))
	pk, err = proofMsg.PublicKey()
	assert.NoError(t, err)
	assert.Equal(t, common.Bytes2Hex(crypto.CompressPubkey(&privkey.PublicKey)), pk)
}
