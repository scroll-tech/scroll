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
	assert.Equal(L1FinalizeBatchEventSignature, common.HexToHash("0x9d3058a3cb9739a2527f22dd9a4138065844037d3004254952e2458d808cc364"))

	assert.Equal(L1QueueTransactionEventSignature, common.HexToHash("bdcc7517f8fe3db6506dfd910942d0bbecaf3d6a506dadea65b0d988e75b9439"))

	assert.Equal(L2SentMessageEventSignature, common.HexToHash("104371f3b442861a2a7b82a070afbbaab748bb13757bf47769e170e37809ec1e"))
	assert.Equal(L2RelayedMessageEventSignature, common.HexToHash("4641df4a962071e12719d8c8c8e5ac7fc4d97b927346a3d7a335b1f7517e133c"))
	assert.Equal(L2FailedRelayedMessageEventSignature, common.HexToHash("99d0e048484baa1b1540b1367cb128acd7ab2946d1ed91ec10e3c85e4bf51b8f"))

	assert.Equal(L2AppendMessageEventSignature, common.HexToHash("faa617c2d8ce12c62637dbce76efcc18dae60574aa95709bdcedce7e76071693"))
}

func TestPackRelayL2MessageWithProof(t *testing.T) {
	assert := assert.New(t)
	l1MessengerABI, err := L1ScrollMessengerMetaData.GetAbi()
	assert.NoError(err)

	proof := IL1ScrollMessengerL2MessageProof{
		BatchIndex:  big.NewInt(0),
		MerkleProof: []byte{},
	}
	_, err = l1MessengerABI.Pack("relayMessageWithProof", common.Address{}, common.Address{}, big.NewInt(0), big.NewInt(0), []byte{}, proof)
	assert.NoError(err)
}

func TestPackCommitBatch(t *testing.T) {
	assert := assert.New(t)

	scrollChainABI, err := ScrollChainMetaData.GetAbi()
	assert.NoError(err)

	version := uint8(1)
	var parentBatchHeader []byte
	var chunks [][]byte
	var skippedL1MessageBitmap []byte

	_, err = scrollChainABI.Pack("commitBatch", version, parentBatchHeader, chunks, skippedL1MessageBitmap)
	assert.NoError(err)
}

func TestPackFinalizeBatchWithProof(t *testing.T) {
	assert := assert.New(t)

	l1RollupABI, err := ScrollChainMetaData.GetAbi()
	assert.NoError(err)

	batchHeader := []byte{}
	prevStateRoot := common.Hash{}
	postStateRoot := common.Hash{}
	withdrawRoot := common.Hash{}
	aggrProof := []byte{}

	_, err = l1RollupABI.Pack("finalizeBatchWithProof", batchHeader, prevStateRoot, postStateRoot, withdrawRoot, aggrProof)
	assert.NoError(err)
}

func TestPackRelayL1Message(t *testing.T) {
	assert := assert.New(t)

	l2MessengerABI, err := L2ScrollMessengerMetaData.GetAbi()
	assert.NoError(err)

	_, err = l2MessengerABI.Pack("relayMessage", common.Address{}, common.Address{}, big.NewInt(0), big.NewInt(0), []byte{})
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

	_, err := l1BlockContainerABI.Pack("importBlockHeader", common.Hash{}, []byte{}, false)
	assert.NoError(err)
}
