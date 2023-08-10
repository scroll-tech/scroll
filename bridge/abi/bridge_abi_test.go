package bridgeabi

import (
	"math/big"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestEventSignature(t *testing.T) {
	assert := assert.New(t)

	assert.Equal(L1CommitBatchEventSignature, common.HexToHash("2c32d4ae151744d0bf0b9464a3e897a1d17ed2f1af71f7c9a75f12ce0d28238f"))
	assert.Equal(L1FinalizeBatchEventSignature, common.HexToHash("26ba82f907317eedc97d0cbef23de76a43dd6edb563bdb6e9407645b950a7a2d"))

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

func TestPackImportGenesisBatch(t *testing.T) {
	assert := assert.New(t)

	l1RollupABI, err := ScrollChainMetaData.GetAbi()
	assert.NoError(err)

	batchHeader := []byte{}
	stateRoot := common.Hash{}

	_, err = l1RollupABI.Pack("importGenesisBatch", batchHeader, stateRoot)
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
