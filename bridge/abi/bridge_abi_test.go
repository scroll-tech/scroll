package bridgeabi_test

import (
	"math/big"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/stretchr/testify/assert"

	bridge_abi "scroll-tech/bridge/abi"
)

func TestPackRelayMessageWithProof(t *testing.T) {
	assert := assert.New(t)
	l1MessengerABI, err := bridge_abi.L1MessengerMetaData.GetAbi()
	assert.NoError(err)

	proof := bridge_abi.IL1ScrollMessengerL2MessageProof{
		BatchHash:   common.Hash{},
		MerkleProof: make([]byte, 0),
	}
	_, err = l1MessengerABI.Pack("relayMessageWithProof", common.Address{}, common.Address{}, big.NewInt(0), big.NewInt(0), make([]byte, 0), proof)
	assert.NoError(err)
}

func TestPackCommitBatch(t *testing.T) {
	assert := assert.New(t)

	scrollchainABI, err := bridge_abi.ScrollchainMetaData.GetAbi()
	assert.NoError(err)

	header := bridge_abi.IScrollChainBlockContext{
		BlockHash:       common.Hash{},
		ParentHash:      common.Hash{},
		BlockNumber:     0,
		Timestamp:       0,
		BaseFee:         big.NewInt(0),
		GasLimit:        0,
		NumTransactions: 0,
		NumL1Messages:   0,
	}

	batch := bridge_abi.IScrollChainBatch{
		Blocks:           []bridge_abi.IScrollChainBlockContext{header},
		PrevStateRoot:    common.Hash{},
		NewStateRoot:     common.Hash{},
		WithdrawTrieRoot: common.Hash{},
		BatchIndex:       0,
		L2Transactions:   make([]byte, 0),
	}

	_, err = scrollchainABI.Pack("commitBatch", batch)
	assert.NoError(err)
}

func TestPackFinalizeBatchWithProof(t *testing.T) {
	assert := assert.New(t)

	l1RollupABI, err := bridge_abi.ScrollchainMetaData.GetAbi()
	assert.NoError(err)

	proof := make([]*big.Int, 10)
	instance := make([]*big.Int, 10)
	for i := 0; i < 10; i++ {
		proof[i] = big.NewInt(0)
		instance[i] = big.NewInt(0)
	}

	_, err = l1RollupABI.Pack("finalizeBatchWithProof", common.Hash{}, proof, instance)
	assert.NoError(err)
}

func TestPackRelayMessage(t *testing.T) {
	assert := assert.New(t)

	l2MessengerABI, err := bridge_abi.L2MessengerMetaData.GetAbi()
	assert.NoError(err)

	_, err = l2MessengerABI.Pack("relayMessage", common.Address{}, common.Address{}, big.NewInt(0), big.NewInt(0), make([]byte, 0))
	assert.NoError(err)
}
