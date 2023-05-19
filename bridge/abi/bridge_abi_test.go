package bridgeabi

import (
	"math/big"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestEventSignature(t *testing.T) {
	assert := assert.New(t)

	assert.Equal(L1SentMessageEventSignature, common.HexToHash("104371f3b442861a2a7b82a070afbbaab748bb13757bf47769e170e37809ec1e"))
	assert.Equal(L1RelayedMessageEventSignature, common.HexToHash("4641df4a962071e12719d8c8c8e5ac7fc4d97b927346a3d7a335b1f7517e133c"))
	assert.Equal(L1FailedRelayedMessageEventSignature, common.HexToHash("99d0e048484baa1b1540b1367cb128acd7ab2946d1ed91ec10e3c85e4bf51b8f"))

	assert.Equal(L1CommitBatchEventSignature, common.HexToHash("2cdc615c74452778c0fb6184735e014c13aad2b62774fe0b09bd1dcc2cc14a62"))
	assert.Equal(L1FinalizeBatchEventSignature, common.HexToHash("6be443154c959a7a1645b4392b6fa97d8e8ab6e8fd853d7085e8867083737d79"))

	assert.Equal(L1QueueTransactionEventSignature, common.HexToHash("bdcc7517f8fe3db6506dfd910942d0bbecaf3d6a506dadea65b0d988e75b9439"))

	assert.Equal(L2SentMessageEventSignature, common.HexToHash("104371f3b442861a2a7b82a070afbbaab748bb13757bf47769e170e37809ec1e"))
	assert.Equal(L2RelayedMessageEventSignature, common.HexToHash("4641df4a962071e12719d8c8c8e5ac7fc4d97b927346a3d7a335b1f7517e133c"))
	assert.Equal(L2FailedRelayedMessageEventSignature, common.HexToHash("99d0e048484baa1b1540b1367cb128acd7ab2946d1ed91ec10e3c85e4bf51b8f"))

	assert.Equal(L2ImportBlockEventSignature, common.HexToHash("a7823f45e1ee21f9530b77959b57507ad515a14fa9fa24d262ee80e79b2b5745"))

	assert.Equal(L2AppendMessageEventSignature, common.HexToHash("faa617c2d8ce12c62637dbce76efcc18dae60574aa95709bdcedce7e76071693"))
}

func TestPackRelayL2MessageWithProof(t *testing.T) {
	assert := assert.New(t)
	l1MessengerABI, err := L1ScrollMessengerMetaData.GetAbi()
	assert.NoError(err)

	proof := IL1ScrollMessengerL2MessageProof{
		BatchHash:   common.Hash{},
		MerkleProof: make([]byte, 0),
	}
	_, err = l1MessengerABI.Pack("relayMessageWithProof", common.Address{}, common.Address{}, big.NewInt(0), big.NewInt(0), make([]byte, 0), proof)
	assert.NoError(err)
}

func TestPackCommitBatch(t *testing.T) {
	assert := assert.New(t)

	scrollChainABI, err := ScrollChainMetaData.GetAbi()
	assert.NoError(err)

	header := IScrollChainBlockContext{
		BlockHash:       common.Hash{},
		ParentHash:      common.Hash{},
		BlockNumber:     0,
		Timestamp:       0,
		BaseFee:         big.NewInt(0),
		GasLimit:        0,
		NumTransactions: 0,
		NumL1Messages:   0,
	}

	batch := IScrollChainBatch{
		Blocks:           []IScrollChainBlockContext{header},
		PrevStateRoot:    common.Hash{},
		NewStateRoot:     common.Hash{},
		WithdrawTrieRoot: common.Hash{},
		BatchIndex:       0,
		L2Transactions:   make([]byte, 0),
	}

	_, err = scrollChainABI.Pack("commitBatch", batch)
	assert.NoError(err)
}

func TestPackFinalizeBatchWithProof(t *testing.T) {
	assert := assert.New(t)

	l1RollupABI, err := ScrollChainMetaData.GetAbi()
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

func TestPackRelayL1Message(t *testing.T) {
	assert := assert.New(t)

	l2MessengerABI, err := L2ScrollMessengerMetaData.GetAbi()
	assert.NoError(err)

	_, err = l2MessengerABI.Pack("relayMessage", common.Address{}, common.Address{}, big.NewInt(0), big.NewInt(0), make([]byte, 0))
	assert.NoError(err)
}

func TestPackSetL1BaseFee(t *testing.T) {
	assert := assert.New(t)

	l1GasOracleABI, err := L1GasPriceOracleMetaData.GetAbi()
	assert.NoError(err)

	baseFee := big.NewInt(2333)
	_, err = l1GasOracleABI.Pack("setL1BaseFee", baseFee)
	assert.NoError(err)
}

func TestPackSetL2BaseFee(t *testing.T) {
	assert := assert.New(t)

	l2GasOracleABI, err := L2GasPriceOracleMetaData.GetAbi()
	assert.NoError(err)

	baseFee := big.NewInt(2333)
	_, err = l2GasOracleABI.Pack("setL2BaseFee", baseFee)
	assert.NoError(err)
}

func TestPackImportBlock(t *testing.T) {
	assert := assert.New(t)

	l1BlockContainerABI := L1BlockContainerABI

	_, err := l1BlockContainerABI.Pack("importBlockHeader", common.Hash{}, make([]byte, 0), false)
	assert.NoError(err)
}
