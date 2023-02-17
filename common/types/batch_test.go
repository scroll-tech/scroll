package types

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	geth_types "github.com/scroll-tech/go-ethereum/core/types"
	"gotest.tools/assert"

	abi "scroll-tech/bridge/abi"
)

func HexToTransaction(s string) (*geth_types.Transaction, error) {
	buf, err := hex.DecodeString(s)
	//fmt.Printf("%0 x\n", b)

	if err != nil {
		panic(fmt.Sprintf("invalid hex string: %q", s))
	}

	tx := new(geth_types.Transaction)
	err = tx.UnmarshalBinary(buf)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func MockBatchData() *BatchData {
	var err error

	txHex := "02f8710582fd14808506e38dccc9825208944d496ccc28058b1d74b7a19541663e21154f9c848801561db11e24a43380c080a0d890606d7a35b2ab0f9b866d62c092d5b163f3e6a55537ae1485aac08c3f8ff7a023997be2d32f53e146b160fff0ba81e81dbb4491c865ab174d15c5b3d28c41ae"
	tx, err := HexToTransaction(txHex)
	if err != nil {
		panic(fmt.Sprintf("invalid tx hex string: %s", err))
	}

	batchData := new(BatchData)
	batch := &batchData.Batch
	batch.PrevStateRoot = common.HexToHash("0x000000000000000000000000000000000000000000000000000000000000cafe")
	batch.L2Transactions = common.Hex2Bytes("0000007402f8710582fd14808506e38dccc9825208944d496ccc28058b1d74b7a19541663e21154f9c848801561db11e24a43380c080a0d890606d7a35b2ab0f9b866d62c092d5b163f3e6a55537ae1485aac08c3f8ff7a023997be2d32f53e146b160fff0ba81e81dbb4491c865ab174d15c5b3d28c41ae")

	block := abi.IScrollChainBlockContext{
		BlockNumber:     51966,
		Timestamp:       123456789,
		BaseFee:         new(big.Int).SetUint64(0),
		GasLimit:        10000000000000000,
		NumTransactions: 1,
		NumL1Messages:   0,
	}
	batch.Blocks = append(batch.Blocks, block)
	batchData.TxHashes = append(batchData.TxHashes, tx.Hash())

	return batchData
}

func TestBatchHash(t *testing.T) {
	batchData := MockBatchData()
	hash := batchData.Hash(4, common.HexToHash("0xb5baa665b2664c3bfed7eb46e00ebc110ecf2ebd257854a9bf2b9dbc9b2c08f6"))
	assert.Equal(t, *hash, common.HexToHash("0xa9f2ca3175794f91226a410ba1e60fff07a405c957562675c4149b77e659d805"))
}
