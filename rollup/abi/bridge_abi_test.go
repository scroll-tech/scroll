package bridgeabi

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/assert"
)

func TestPackCommitBatches(t *testing.T) {
	scrollChainABI, err := ScrollChainMetaData.GetAbi()
	assert.NoError(t, err)

	version := uint8(7)
	var parentBatchHash common.Hash
	var lastBatchHash common.Hash

	_, err = scrollChainABI.Pack("commitBatches", version, parentBatchHash, lastBatchHash)
	assert.NoError(t, err)
}

func TestPackFinalizeBundlePostEuclidV2(t *testing.T) {
	l1RollupABI, err := ScrollChainMetaData.GetAbi()
	assert.NoError(t, err)

	var batchHeader []byte
	totalL1MessagesPoppedOverall := big.NewInt(0)
	var postStateRoot common.Hash
	var withdrawRoot common.Hash
	var aggrProof []byte

	_, err = l1RollupABI.Pack("finalizeBundlePostEuclidV2", batchHeader, totalL1MessagesPoppedOverall, postStateRoot, withdrawRoot, aggrProof)
	assert.NoError(t, err)
}

func TestPackCommitAndFinalizeBatch(t *testing.T) {
	l1RollupABI, err := ScrollChainMetaData.GetAbi()
	assert.NoError(t, err)

	version := uint8(7)
	var parentBatchHash common.Hash
	// Create the FinalizeStruct tuple as an abi-compatible struct
	finalizeStruct := struct {
		BatchHeader                  []byte
		TotalL1MessagesPoppedOverall *big.Int
		PostStateRoot                common.Hash
		WithdrawRoot                 common.Hash
		ZkProof                      []byte
	}{
		TotalL1MessagesPoppedOverall: big.NewInt(0),
	}

	_, err = l1RollupABI.Pack("commitAndFinalizeBatch", version, parentBatchHash, finalizeStruct)
	assert.NoError(t, err)
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

func TestPrintABISignatures(t *testing.T) {
	// print all error signatures of ABI
	abi, err := ScrollChainMetaData.GetAbi()
	if err != nil {
		t.Fatal(err)
	}

	for _, method := range abi.Methods {
		fmt.Println(hexutil.Encode(method.ID[:4]), method.Sig, method.Name)
	}

	fmt.Println("------------------------------")
	for _, errors := range abi.Errors {
		fmt.Println(hexutil.Encode(errors.ID[:4]), errors.Sig, errors.Name)
	}
}
