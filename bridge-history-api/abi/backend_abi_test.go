package backendabi

import (
	"testing"

	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
)

func TestEventSignatures(t *testing.T) {
	assert.Equal(t, crypto.Keccak256Hash([]byte("RevertBatch(uint256,bytes32)")), L1RevertBatchV0EventSig)
	assert.Equal(t, crypto.Keccak256Hash([]byte("RevertBatch(uint256,uint256)")), L1RevertBatchV7EventSig)
}
