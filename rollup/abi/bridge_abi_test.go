package bridgeabi

import (
	"math/big"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

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

func TestPackSetL1BaseFeeAndBlobBaseFee(t *testing.T) {
	assert := assert.New(t)

	l1GasOracleABI, err := L1GasPriceOracleMetaData.GetAbi()
	assert.NoError(err)

	baseFee := big.NewInt(2333)
	blobBaseFee := big.NewInt(1)
	_, err = l1GasOracleABI.Pack("setL1BaseFeeAndBlobBaseFee", baseFee, blobBaseFee)
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
