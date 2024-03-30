package encoding

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.LogfmtFormat()))
	glogger.Verbosity(log.LvlInfo)
	log.Root().SetHandler(glogger)

	m.Run()
}

func TestUtilFunctions(t *testing.T) {
	block1 := readBlockFromJSON(t, "../../testdata/blockTrace_02.json")
	block2 := readBlockFromJSON(t, "../../testdata/blockTrace_03.json")
	block3 := readBlockFromJSON(t, "../../testdata/blockTrace_04.json")
	block4 := readBlockFromJSON(t, "../../testdata/blockTrace_05.json")
	block5 := readBlockFromJSON(t, "../../testdata/blockTrace_06.json")
	block6 := readBlockFromJSON(t, "../../testdata/blockTrace_07.json")

	chunk1 := &Chunk{Blocks: []*Block{block1, block2}}
	chunk2 := &Chunk{Blocks: []*Block{block3, block4}}
	chunk3 := &Chunk{Blocks: []*Block{block5, block6}}

	batch := &Batch{Chunks: []*Chunk{chunk1, chunk2, chunk3}}

	// Test Block methods
	assert.Equal(t, uint64(0), block1.NumL1Messages(0))
	assert.Equal(t, uint64(2), block1.NumL2Transactions())
	assert.Equal(t, uint64(0), block2.NumL1Messages(0))
	assert.Equal(t, uint64(1), block2.NumL2Transactions())
	assert.Equal(t, uint64(11), block3.NumL1Messages(0))
	assert.Equal(t, uint64(1), block3.NumL2Transactions())
	assert.Equal(t, uint64(42), block4.NumL1Messages(0))
	assert.Equal(t, uint64(0), block4.NumL2Transactions())
	assert.Equal(t, uint64(10), block5.NumL1Messages(0))
	assert.Equal(t, uint64(0), block5.NumL2Transactions())
	assert.Equal(t, uint64(257), block6.NumL1Messages(0))
	assert.Equal(t, uint64(0), block6.NumL2Transactions())

	// Test Chunk methods
	assert.Equal(t, uint64(0), chunk1.NumL1Messages(0))
	assert.Equal(t, uint64(3), chunk1.NumL2Transactions())
	crc1Max, err := chunk1.CrcMax()
	assert.NoError(t, err)
	assert.Equal(t, uint64(11), crc1Max)
	assert.Equal(t, uint64(3), chunk1.NumTransactions())
	assert.Equal(t, uint64(1194994), chunk1.L2GasUsed())

	assert.Equal(t, uint64(42), chunk2.NumL1Messages(0))
	assert.Equal(t, uint64(1), chunk2.NumL2Transactions())
	crc2Max, err := chunk2.CrcMax()
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), crc2Max)
	assert.Equal(t, uint64(7), chunk2.NumTransactions())
	assert.Equal(t, uint64(144000), chunk2.L2GasUsed())

	assert.Equal(t, uint64(257), chunk3.NumL1Messages(0))
	assert.Equal(t, uint64(0), chunk3.NumL2Transactions())
	chunk3.Blocks[0].RowConsumption = nil
	crc3Max, err := chunk3.CrcMax()
	assert.Error(t, err)
	assert.EqualError(t, err, "block (17, 0x003fee335455c0c293dda17ea9365fe0caa94071ed7216baf61f7aeb808e8a28) has nil RowConsumption")
	assert.Equal(t, uint64(0), crc3Max)
	assert.Equal(t, uint64(5), chunk3.NumTransactions())
	assert.Equal(t, uint64(240000), chunk3.L2GasUsed())

	// Test Batch methods
	assert.Equal(t, uint64(3), batch.NumChunks())
	assert.Equal(t, block6.Header.Root, batch.StateRoot())
	assert.Equal(t, block6.WithdrawRoot, batch.WithdrawRoot())
}

func TestConvertTxDataToRLPEncoding(t *testing.T) {
	blocks := []*Block{
		readBlockFromJSON(t, "../../testdata/blockTrace_02.json"),
		readBlockFromJSON(t, "../../testdata/blockTrace_03.json"),
		readBlockFromJSON(t, "../../testdata/blockTrace_04.json"),
		readBlockFromJSON(t, "../../testdata/blockTrace_05.json"),
		readBlockFromJSON(t, "../../testdata/blockTrace_06.json"),
		readBlockFromJSON(t, "../../testdata/blockTrace_07.json"),
	}

	for _, block := range blocks {
		for _, txData := range block.Transactions {
			if txData.Type == types.L1MessageTxType {
				continue
			}

			rlpTxData, err := ConvertTxDataToRLPEncoding(txData)
			assert.NoError(t, err)
			var tx types.Transaction
			err = tx.UnmarshalBinary(rlpTxData)
			assert.NoError(t, err)
			assert.Equal(t, txData.TxHash, tx.Hash().Hex())
		}
	}
}

func TestEmptyBatchRoots(t *testing.T) {
	emptyBatch := &Batch{Chunks: []*Chunk{}}
	assert.Equal(t, common.Hash{}, emptyBatch.StateRoot())
	assert.Equal(t, common.Hash{}, emptyBatch.WithdrawRoot())
}

func readBlockFromJSON(t *testing.T, filename string) *Block {
	data, err := os.ReadFile(filename)
	assert.NoError(t, err)

	block := &Block{}
	assert.NoError(t, json.Unmarshal(data, block))
	return block
}
