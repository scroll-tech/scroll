package bridgeabi

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
		BlockNumber: big.NewInt(0),
		MerkleProof: make([]byte, 0),
	}
	_, err = l1MessengerABI.Pack("relayMessageWithProof", common.Address{}, common.Address{}, big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), make([]byte, 0), proof)
	assert.NoError(err)
}

func TestPackCommitBlock(t *testing.T) {
	assert := assert.New(t)

	l1RollupABI, err := bridge_abi.RollupMetaData.GetAbi()
	assert.NoError(err)

	header := bridge_abi.IZKRollupBlockHeader{
		BlockHash:   common.Hash{},
		ParentHash:  common.Hash{},
		BaseFee:     big.NewInt(0),
		StateRoot:   common.Hash{},
		BlockHeight: 0,
		GasUsed:     0,
		Timestamp:   0,
		ExtraData:   make([]byte, 0),
	}
	txns := make([]bridge_abi.IZKRollupLayer2Transaction, 5)
	for i := 0; i < 5; i++ {
		txns[i] = bridge_abi.IZKRollupLayer2Transaction{
			Caller:   common.Address{},
			Target:   common.Address{},
			Nonce:    0,
			Gas:      0,
			GasPrice: big.NewInt(0),
			Value:    big.NewInt(0),
			Data:     make([]byte, 0),
		}
	}
	_, err = l1RollupABI.Pack("commitBlock", header, txns)
	assert.NoError(err)
}

func TestPackFinalizeBlockWithProof(t *testing.T) {
	assert := assert.New(t)

	l1RollupABI, err := bridge_abi.RollupMetaData.GetAbi()
	assert.NoError(err)

	proof := make([]*big.Int, 10)
	instance := make([]*big.Int, 10)
	for i := 0; i < 10; i++ {
		proof[i] = big.NewInt(0)
		instance[i] = big.NewInt(0)
	}

	_, err = l1RollupABI.Pack("finalizeBlockWithProof", common.Hash{}, proof, instance)
	assert.NoError(err)
}

func TestPackRelayMessage(t *testing.T) {
	assert := assert.New(t)

	l2MessengerABI, err := bridge_abi.L2MessengerMetaData.GetAbi()
	assert.NoError(err)

	_, err = l2MessengerABI.Pack("relayMessage", common.Address{}, common.Address{}, big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), make([]byte, 0))
	assert.NoError(err)
}
