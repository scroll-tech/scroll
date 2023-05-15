package types

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/core/types"
	geth_types "github.com/scroll-tech/go-ethereum/core/types"

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

func TestBatchHashWithL1Messages(t *testing.T) {
	// TODO: store an actual trace with L1 messages in `common/testdata`

	zeroAddress := common.HexToAddress("0x0000000000000000000000000000000000000000")

	hexToBig := func(hex string) *hexutil.Big {
		big, success := new(big.Int).SetString(hex, 16)
		if !success {
			panic(fmt.Sprintf("Failed to convert hex string to big int: %s", hex))
		}
		return (*hexutil.Big)(big)
	}

	parentBatch := BlockBatch{
		Hash:      "0x0000000000000000000000000000000000000000000000000000000000000001",
		Index:     2,
		StateRoot: "0x0000000000000000000000000000000000000000000000000000000000000003",
		// other fields are not used here
	}

	header := geth_types.Header{
		UncleHash:   common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000004"),
		Root:        common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000005"),
		TxHash:      common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000006"),
		ReceiptHash: common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000007"),
		Difficulty:  big.NewInt(8),
		Number:      big.NewInt(9),
		GasLimit:    0xa,
		GasUsed:     0xb,
		Time:        0xc,
		Extra:       common.Hex2Bytes("0x000000000000000000000000000000000000000000000000000000000000000d"),
		BaseFee:     big.NewInt(0xe),
	}

	tx0 := types.TransactionData{
		Type:   geth_types.L1MessageTxType,
		Nonce:  0,
		From:   zeroAddress,
		To:     &zeroAddress,
		Value:  (*hexutil.Big)(big.NewInt(0)),
		Data:   "0x1",
		TxHash: "0x518e760abcf3f88f286153d9a68250640c0cd76a054267a6f4967eae3b17a63e",
	}

	tx1 := types.TransactionData{
		Type:   geth_types.L1MessageTxType,
		Nonce:  1,
		From:   zeroAddress,
		To:     &zeroAddress,
		Value:  (*hexutil.Big)(big.NewInt(0)),
		Data:   "0x",
		TxHash: "0x2a618fedf9cb0996adfa285f418658c4b7b97e02daac67a459e9a9848b233179",
	}

	to := common.HexToAddress("0x6d79aa2e4fbf80cf8543ad97e294861853fb0649")

	tx2 := types.TransactionData{
		Type:     geth_types.LegacyTxType,
		Nonce:    0x2,
		To:       &to,
		Value:    hexToBig("e23c37ec2147400"),
		Gas:      0x435db,
		GasPrice: hexToBig("11490c80"),
		Data:     "0xc7cdea370000000000000000000000000000000000000000000000000de0b6b3a76400000000000000000000000000000000000000000000000000000000000000027100",
		V:        hexToBig("104ec5"),
		R:        hexToBig("546dba2b8b8bd64d2d9891bf33da3d0e0b8da2d0b0f278904a27d243fc84589c"),
		S:        hexToBig("7f30a2e4d5927c658f80d5de5dfc8f198ff690dd1aaa4343a446a8d57cbf0ba2"),
		TxHash:   "0x41cd05dea245041442a5dd8b5c66da7e744d47fba4be8acbd5b9852c6eb24411",
	}

	block := WrappedBlock{
		Header:           &header,
		Transactions:     []*types.TransactionData{&tx0, &tx1, &tx2},
		WithdrawTrieRoot: common.HexToHash("0x000000000000000000000000000000000000000000000000000000000000000f"),
	}

	blocks := []*WrappedBlock{&block}

	piCfg := &PublicInputHashConfig{
		MaxTxNum:      4,
		PaddingTxHash: common.HexToHash("0xb5baa665b2664c3bfed7eb46e00ebc110ecf2ebd257854a9bf2b9dbc9b2c08f6"),
	}

	// encode batch data
	batchData := NewBatchData(&parentBatch, blocks, piCfg)

	// tx counts are correct
	assert.Equal(t, batchData.TotalTxNum, uint64(3))
	assert.Equal(t, batchData.TotalL1TxNum, uint64(2))
	assert.Equal(t, len(batchData.Batch.Blocks), 1)
	assert.Equal(t, batchData.Batch.Blocks[0].NumTransactions, uint16(3))
	assert.Equal(t, batchData.Batch.Blocks[0].NumL1Messages, uint16(2))

	// TxHashes contains both L1 and L2 hashes
	expectedHashes := []common.Hash{common.HexToHash(tx0.TxHash), common.HexToHash(tx1.TxHash), common.HexToHash(tx2.TxHash)}
	for ii := 0; ii < int(batchData.TotalTxNum); ii++ {
		assert.Equal(t, batchData.TxHashes[ii], expectedHashes[ii])
	}

	// L2Transactions only contains L2 transaction data
	assert.Equal(t, common.Bytes2Hex(batchData.Batch.L2Transactions), "000000b6f8b4028411490c80830435db946d79aa2e4fbf80cf8543ad97e294861853fb0649880e23c37ec2147400b844c7cdea370000000000000000000000000000000000000000000000000de0b6b3a7640000000000000000000000000000000000000000000000000000000000000002710083104ec5a0546dba2b8b8bd64d2d9891bf33da3d0e0b8da2d0b0f278904a27d243fc84589ca07f30a2e4d5927c658f80d5de5dfc8f198ff690dd1aaa4343a446a8d57cbf0ba2")

	// batch hash is correct
	assert.Equal(t, *batchData.Hash(), common.HexToHash("0xf18020568915f64ab24e92195915d1f940a5100fc3499196430a546909b9eca2"))
}

func TestNewGenesisBatch(t *testing.T) {
	genesisBlock := &geth_types.Header{
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

func TestNewBatchData(t *testing.T) {
	templateBlockTrace, err := os.ReadFile("../testdata/blockTrace_02.json")
	assert.NoError(t, err)

	wrappedBlock := &WrappedBlock{}
	assert.NoError(t, json.Unmarshal(templateBlockTrace, wrappedBlock))

	parentBatch := &BlockBatch{
		Index:     1,
		Hash:      "0x0000000000000000000000000000000000000000",
		StateRoot: "0x0000000000000000000000000000000000000000",
	}
	batchData1 := NewBatchData(parentBatch, []*WrappedBlock{wrappedBlock}, nil)
	assert.NotNil(t, batchData1)
	assert.NotNil(t, batchData1.Batch)
	assert.Equal(t, "0xac4487c0d8f429dafda3c68cbb8983ac08af83c03c83c365d7df02864f80af37", batchData1.Hash().Hex())

	templateBlockTrace, err = os.ReadFile("../testdata/blockTrace_03.json")
	assert.NoError(t, err)

	wrappedBlock2 := &WrappedBlock{}
	assert.NoError(t, json.Unmarshal(templateBlockTrace, wrappedBlock2))

	parentBatch2 := &BlockBatch{
		Index:     batchData1.Batch.BatchIndex,
		Hash:      batchData1.Hash().Hex(),
		StateRoot: batchData1.Batch.NewStateRoot.Hex(),
	}
	batchData2 := NewBatchData(parentBatch2, []*WrappedBlock{wrappedBlock2}, nil)
	assert.NotNil(t, batchData2)
	assert.NotNil(t, batchData2.Batch)
	assert.Equal(t, "0x8f1447573740b3e75b979879866b8ad02eecf88e1946275eb8cf14ab95876efc", batchData2.Hash().Hex())
}

func TestBatchDataTimestamp(t *testing.T) {
	// Test case 1: when the batch data contains no blocks.
	assert.Equal(t, uint64(0), (&BatchData{}).Timestamp())

	// Test case 2: when the batch data contains blocks.
	batchData := &BatchData{
		Batch: abi.IScrollChainBatch{
			Blocks: []abi.IScrollChainBlockContext{
				{Timestamp: 123456789},
				{Timestamp: 234567891},
			},
		},
	}
	assert.Equal(t, uint64(123456789), batchData.Timestamp())
}
