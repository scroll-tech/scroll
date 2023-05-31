package types

import (
	"math/big"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

	abi "scroll-tech/bridge/abi"
)

func TestBatchHash(t *testing.T) {
	txBytes := common.Hex2Bytes("02f8710582fd14808506e38dccc9825208944d496ccc28058b1d74b7a19541663e21154f9c848801561db11e24a43380c080a0d890606d7a35b2ab0f9b866d62c092d5b163f3e6a55537ae1485aac08c3f8ff7a023997be2d32f53e146b160fff0ba81e81dbb4491c865ab174d15c5b3d28c41ae")
	tx := new(gethTypes.Transaction)
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
	tx = new(gethTypes.Transaction)
	if err := tx.UnmarshalBinary(txBytes); err != nil {
		t.Fatalf("invalid tx hex string: %s", err)
	}
	batchData.TxHashes[0] = tx.Hash()

	batchData.hash = nil // clear the cache
	assert.Equal(t, *batchData.Hash(), common.HexToHash("0x398cb22bbfa1665c1b342b813267538a4c933d7f92d8bd9184aba0dd1122987b"))
}

func TestNewGenesisBatch(t *testing.T) {
	genesisBlock := &gethTypes.Header{
		UncleHash:   common.HexToHash("0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347"),
		Root:        common.HexToHash("0x1b186a7a90ec3b41a2417062fe44dce8ce82ae76bfbb09eae786a4f1be1895f5"),
		TxHash:      common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"),
		ReceiptHash: common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"),
		Difficulty:  big.NewInt(1),
		Number:      big.NewInt(0),
		GasLimit:    940000000,
		GasUsed:     0,
		Time:        1639724192,
		Extra:       common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000000214f8d488aa9ebf83e30bad45fb8f9c8ee2509f5511caff794753d07e9dfb218cfc233bb62d2c57022783094e1a7edb6f069f8424bb68496a0926b130000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"),
		BaseFee:     big.NewInt(1000000000),
	}
	assert.Equal(
		t,
		genesisBlock.Hash().Hex(),
		"0x92826bd3aad2ef70d8061dc4e25150b305d1233d9cd7579433a77d6eb01dae1c",
		"wrong genesis block header",
	)

	blockTrace := &WrappedBlock{genesisBlock, nil, common.Hash{}}
	batchData := NewGenesisBatchData(blockTrace)
	t.Log(batchData.Batch.Blocks[0])
	batchData.piCfg = &PublicInputHashConfig{
		MaxTxNum:      25,
		PaddingTxHash: common.HexToHash("0xb5baa665b2664c3bfed7eb46e00ebc110ecf2ebd257854a9bf2b9dbc9b2c08f6"),
	}
	assert.Equal(
		t,
		batchData.Hash().Hex(),
		"0x65cf210e30f75cf8fd198df124255f73bc08d6324759e828a784fa938e7ac43d",
		"wrong genesis batch hash",
	)
}
