package verifier_test

import (
	"fmt"
	"scroll-tech/common/version"
	"testing"

	"scroll-tech/roller/core"

	"scroll-tech/common/message"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
)

// skipped due to verifier upgrade
/*
// This test assumes the IPC Verifier is running.
func TestIPCComm(t *testing.T) {
	if os.Getenv("TEST_IPC") != "true" {
		return
	}

	assert := assert.New(t)
	verifier, err := verifier.NewVerifier("/tmp/Verifier.sock")
	assert.NoError(err)

	// Retrieve pre-generated proofs
	evmProof, err := ioutil.ReadFile("../assets/evm_proof")
	assert.NoError(err)
	stateProof, err := ioutil.ReadFile("../assets/state_proof")
	assert.NoError(err)

	proof := &message.ZkProof{
		ID:         1,
		EvmProof:   evmProof,
		StateProof: stateProof,
	}

	traces := &types.BlockResult{}

	verified, err := verifier.VerifyProof(traces, proof)
	assert.NoError(err)
	assert.True(verified)
}
*/

func TestVerifier(t *testing.T) {
	privkey, err := crypto.HexToECDSA("dcf2cbdd171a21c480aa7f53d77f31bb102282b3ff099c78e3118b37348c72f7")
	assert.NoError(t, err)

	pubkey := crypto.CompressPubkey(&privkey.PublicKey)

	msg := &message.Identity{
		Name:      "scroll_roller",
		Timestamp: 1649663001,
		PublicKey: common.Bytes2Hex(pubkey),
		Version:   fmt.Sprintf("%s-%s", version.Version, core.ZK_VERSION),
	}
	hash, err := msg.Hash()
	assert.NoError(t, err)

	sig, err := crypto.Sign(hash, privkey)
	assert.NoError(t, err)

	ok := crypto.VerifySignature(pubkey, hash, sig[:64])
	assert.Equal(t, true, ok)
}
