package message

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
)

func TestAuthMessageSignAndVerify(t *testing.T) {
	privkey, err := crypto.GenerateKey()
	assert.NoError(t, err)

	authMsg := &AuthMsg{
		Identity: &Identity{
			Name:      "testName",
			Timestamp: uint32(time.Now().Unix()),
			Version:   "testVersion",
			Token:     "testToken",
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
		Name:       "testName",
		RollerType: ProofTypeChunk,
		Timestamp:  uint32(1622428800),
		Version:    "testVersion",
		Token:      "testToken",
	}
	hash, err := identity.Hash()
	assert.NoError(t, err)

	expectedHash := "b3f152958dc881446fc131a250526139d909710c6b91b4d3281ceded28ce2e32"
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
			Proof: &AggProof{
				Proof:      []byte("testProof"),
				Instance:   []byte("testInstance"),
				FinalPair:  []byte("testFinalPair"),
				Vk:         []byte("testVk"),
				BlockCount: 1,
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
		Proof: &AggProof{
			Proof:      []byte("testProof"),
			Instance:   []byte("testInstance"),
			FinalPair:  []byte("testFinalPair"),
			Vk:         []byte("testVk"),
			BlockCount: 1,
		},
		Error: "testError",
	}
	hash, err := proofDetail.Hash()
	assert.NoError(t, err)
	expectedHash := "fdfaae752d6fd72a7fdd2ad034ef504d3acda9e691a799323cfa6e371684ba2b"
	assert.Equal(t, expectedHash, hex.EncodeToString(hash))
}

func TestProveTypeString(t *testing.T) {
	proofTypeChunk := ProofType(0)
	assert.Equal(t, "Proof Type Chunk", proofTypeChunk.String())

	proofTypeBatch := ProofType(1)
	assert.Equal(t, "Proof Type Batch", proofTypeBatch.String())

	illegalProof := ProofType(3)
	assert.Equal(t, "Illegal Proof type", illegalProof.String())
}

func TestProofMsgPublicKey(t *testing.T) {
	privkey, err := crypto.GenerateKey()
	assert.NoError(t, err)

	proofMsg := &ProofMsg{
		ProofDetail: &ProofDetail{
			ID:     "testID",
			Type:   ProofTypeChunk,
			Status: StatusOk,
			Proof: &AggProof{
				Proof:      []byte("testProof"),
				Instance:   []byte("testInstance"),
				FinalPair:  []byte("testFinalPair"),
				Vk:         []byte("testVk"),
				BlockCount: 1,
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
