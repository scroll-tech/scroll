package types

import (
	"math/big"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	geth_types "github.com/scroll-tech/go-ethereum/core/types"
	"gotest.tools/assert"

	abi "scroll-tech/bridge/abi"
)

func TestBatchHash(t *testing.T) {
	txBytes := common.Hex2Bytes("02f8710582fd14808506e38dccc9825208944d496ccc28058b1d74b7a19541663e21154f9c848801561db11e24a43380c080a0d890606d7a35b2ab0f9b866d62c092d5b163f3e6a55537ae1485aac08c3f8ff7a023997be2d32f53e146b160fff0ba81e81dbb4491c865ab174d15c5b3d28c41ae")
	tx := new(geth_types.Transaction)
	if err := tx.UnmarshalBinary(txBytes); err != nil {
		t.Fatalf("invalid tx hex string: %s", err)
	}

	batchData := new(BatchData)
	batchData.TxHashes = append(batchData.TxHashes, tx.Hash())
	batchData.piCfg = &PublicInputHashConfig{
		MaxTxNum:      4,
		PaddingTxHash: common.HexToHash("0xb5baa665b2664c3bfed7eb46e00ebc110ecf2ebd257854a9bf2b9dbc9b2c08f6"),
	}

	batch := &batchData.Batch
	batch.PrevStateRoot = common.HexToHash("0x000000000000000000000000000000000000000000000000000000000000cafe")

	block := abi.IScrollChainBlockContext{
		BlockNumber:     51966,
		Timestamp:       123456789,
		BaseFee:         new(big.Int).SetUint64(0),
		GasLimit:        10000000000000000,
		NumTransactions: 1,
		NumL1Messages:   0,
	}
	batch.Blocks = append(batch.Blocks, block)

	hash := batchData.Hash()
	assert.Equal(t, *hash, common.HexToHash("0xa9f2ca3175794f91226a410ba1e60fff07a405c957562675c4149b77e659d805"))

	// use a different tx hash
	txBytes = common.Hex2Bytes("f8628001830f424094000000000000000000000000000000000000bbbb8080820a97a064e07cd8f939e2117724bdcbadc80dda421381cbc2a1f4e0d093d9cc5c5cf68ea03e264227f80852d88743cd9e43998f2746b619180366a87e4531debf9c3fa5dc")
	tx = new(geth_types.Transaction)
	if err := tx.UnmarshalBinary(txBytes); err != nil {
		t.Fatalf("invalid tx hex string: %s", err)
	}
	batchData.TxHashes[0] = tx.Hash()

	batchData.hash = nil // clear the cache
	assert.Equal(t, *batchData.Hash(), common.HexToHash("0x398cb22bbfa1665c1b342b813267538a4c933d7f92d8bd9184aba0dd1122987b"))
}
