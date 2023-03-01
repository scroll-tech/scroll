package l2

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	geth_types "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

	"scroll-tech/database"
	"scroll-tech/database/migrate"

	"scroll-tech/bridge/config"

	"scroll-tech/common/types"
)

func testBatchProposerProposeBatch(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	// Insert traces into db.
	assert.NoError(t, db.InsertL2BlockTraces([]*geth_types.BlockTrace{blockTrace1}))

	l2cfg := cfg.L2Config
	wc := NewL2WatcherClient(context.Background(), l2Cli, l2cfg.Confirmations, l2cfg.L2MessengerAddress, l2cfg.L2MessageQueueAddress, db)
	wc.Start()
	defer wc.Stop()

	relayer, err := NewLayer2Relayer(context.Background(), l2Cli, db, cfg.L2Config.RelayerConfig)
	assert.NoError(t, err)

	proposer := NewBatchProposer(context.Background(), &config.BatchProposerConfig{
		ProofGenerationFreq: 1,
		BatchGasThreshold:   3000000,
		BatchTxNumThreshold: 135,
		BatchTimeSec:        1,
		BatchBlocksLimit:    100,
	}, relayer, db)
	proposer.tryProposeBatch()

	infos, err := db.GetUnbatchedL2Blocks(map[string]interface{}{},
		fmt.Sprintf("order by number ASC LIMIT %d", 100))
	assert.NoError(t, err)
	assert.Equal(t, 0, len(infos))

	exist, err := db.BatchRecordExist(batchData1.Hash().Hex())
	assert.NoError(t, err)
	assert.Equal(t, true, exist)
}

func testBatchProposerGracefulRestart(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	relayer, err := NewLayer2Relayer(context.Background(), l2Cli, db, cfg.L2Config.RelayerConfig)
	assert.NoError(t, err)

	// Insert traces into db.
	assert.NoError(t, db.InsertL2BlockTraces([]*geth_types.BlockTrace{blockTrace2}))

	// Insert block batch into db.
	dbTx, err := db.Beginx()
	assert.NoError(t, err)
	assert.NoError(t, db.NewBatchInDBTx(dbTx, batchData1))
	assert.NoError(t, db.NewBatchInDBTx(dbTx, batchData2))
	assert.NoError(t, db.SetBatchHashForL2BlocksInDBTx(dbTx, []uint64{
		batchData1.Batch.Blocks[0].BlockNumber}, batchData1.Hash().Hex()))
	assert.NoError(t, db.SetBatchHashForL2BlocksInDBTx(dbTx, []uint64{
		batchData2.Batch.Blocks[0].BlockNumber}, batchData2.Hash().Hex()))
	assert.NoError(t, dbTx.Commit())

	assert.NoError(t, db.UpdateRollupStatus(context.Background(), batchData1.Hash().Hex(), types.RollupFinalized))

	batchHashes, err := db.GetPendingBatches(math.MaxInt32)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(batchHashes))
	assert.Equal(t, batchData2.Hash().Hex(), batchHashes[0])
	// test p.recoverBatchDataBuffer().
	_ = NewBatchProposer(context.Background(), &config.BatchProposerConfig{
		ProofGenerationFreq: 1,
		BatchGasThreshold:   3000000,
		BatchTxNumThreshold: 135,
		BatchTimeSec:        1,
		BatchBlocksLimit:    100,
	}, relayer, db)

	batchHashes, err = db.GetPendingBatches(math.MaxInt32)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(batchHashes))

	exist, err := db.BatchRecordExist(batchData2.Hash().Hex())
	assert.NoError(t, err)
	assert.Equal(t, true, exist)
}

func testProposeBatchWithMessages(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	// create proposer
	relayer, err := NewLayer2Relayer(context.Background(), l2Cli, db, cfg.L2Config.RelayerConfig)
	assert.NoError(t, err)

	proposer := NewBatchProposer(context.Background(), &config.BatchProposerConfig{
		ProofGenerationFreq: 1,
		BatchGasThreshold:   3000000,
		BatchTxNumThreshold: 135,
		BatchTimeSec:        1,
		BatchBlocksLimit:    100,
	}, relayer, db)

	trie := NewWithdrawTrie()

	// fake 11 blocks and 10 messages
	var msgs []*types.L2Message
	var traces []*geth_types.BlockTrace
	for i := 0; i < 11; i++ {
		parentHash := common.Hash{}
		withdrawTrieRoot := common.Hash{}
		if i > 0 {
			parentHash = traces[i-1].Header.Hash()
			msgs = append(msgs, &types.L2Message{
				Nonce:   uint64(i - 1),
				MsgHash: common.BigToHash(big.NewInt(int64(i))).String(),
				Height:  uint64(i),
			})
			trie.AppendMessages([]common.Hash{common.HexToHash(msgs[i-1].MsgHash)})
			withdrawTrieRoot = trie.MessageRoot()
		}
		traces = append(traces, &geth_types.BlockTrace{
			Header: &geth_types.Header{
				ParentHash: parentHash,
				Difficulty: big.NewInt(0),
				Number:     big.NewInt(int64(i)),
			},
			StorageTrace: &geth_types.StorageTrace{
				RootBefore: common.BigToHash(big.NewInt(int64(i))),
				RootAfter:  common.BigToHash(big.NewInt(int64(i + 1))),
			},
			WithdrawTrieRoot: withdrawTrieRoot,
		})
	}

	// insert blocks and message
	err = db.InsertL2BlockTraces(traces)
	assert.NoError(t, err)
	err = db.SaveL2Messages(context.Background(), msgs)
	assert.NoError(t, err)

	// insert genesis batch
	genssisBatchData := types.NewGenesisBatchData(traces[0])
	err = AddBatchInfoToDB(db, genssisBatchData, make([]*types.L2Message, 0), make([][]byte, 0))
	assert.NoError(t, err)
	batchHash := genssisBatchData.Hash().Hex()
	err = db.UpdateProvingStatus(batchHash, types.ProvingTaskProved)
	assert.NoError(t, err)
	err = db.UpdateRollupStatus(context.Background(), batchHash, types.RollupFinalized)
	assert.NoError(t, err)

	var blocks []*types.BlockInfo
	var proof sql.NullString
	// propose batch with 1 block
	blocks, err = db.GetUnbatchedL2Blocks(map[string]interface{}{}, "order by number ASC LIMIT 1")
	assert.NoError(t, err)
	proposer.proposeBatch(blocks)
	for i := 0; i < 10; i++ {
		proof, err = db.GetL2MessageProofByNonce(msgs[i].Nonce)
		assert.NoError(t, err)
		assert.Equal(t, proof.Valid, i < 1)
	}
	// propose batch with 2 block
	blocks, err = db.GetUnbatchedL2Blocks(map[string]interface{}{}, "order by number ASC LIMIT 2")
	assert.NoError(t, err)
	proposer.proposeBatch(blocks)
	for i := 0; i < 10; i++ {
		proof, err = db.GetL2MessageProofByNonce(msgs[i].Nonce)
		assert.NoError(t, err)
		assert.Equal(t, proof.Valid, i < 3)
	}
	// propose batch with 3 block
	blocks, err = db.GetUnbatchedL2Blocks(map[string]interface{}{}, "order by number ASC LIMIT 3")
	assert.NoError(t, err)
	proposer.proposeBatch(blocks)
	for i := 0; i < 10; i++ {
		proof, err = db.GetL2MessageProofByNonce(msgs[i].Nonce)
		assert.NoError(t, err)
		assert.Equal(t, proof.Valid, i < 6)
	}
	// propose batch with 4 block
	blocks, err = db.GetUnbatchedL2Blocks(map[string]interface{}{}, "order by number ASC LIMIT 4")
	assert.NoError(t, err)
	proposer.proposeBatch(blocks)
	for i := 0; i < 10; i++ {
		proof, err = db.GetL2MessageProofByNonce(msgs[i].Nonce)
		assert.NoError(t, err)
		assert.Equal(t, proof.Valid, true)
	}
}

func testInitializeMissingMessageProof(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	trie := NewWithdrawTrie()

	// fake 11 blocks and 10 messages
	var msgs []*types.L2Message
	var traces []*geth_types.BlockTrace
	for i := 0; i < 11; i++ {
		parentHash := common.Hash{}
		withdrawTrieRoot := common.Hash{}
		if i > 0 {
			parentHash = traces[i-1].Header.Hash()
			msgs = append(msgs, &types.L2Message{
				Nonce:   uint64(i - 1),
				MsgHash: common.BigToHash(big.NewInt(int64(i))).String(),
				Height:  uint64(i),
			})
			trie.AppendMessages([]common.Hash{common.HexToHash(msgs[i-1].MsgHash)})
			withdrawTrieRoot = trie.MessageRoot()
		}
		if i <= 6 {
			withdrawTrieRoot = common.Hash{}
		}
		traces = append(traces, &geth_types.BlockTrace{
			Header: &geth_types.Header{
				ParentHash: parentHash,
				Difficulty: big.NewInt(0),
				Number:     big.NewInt(int64(i)),
			},
			StorageTrace: &geth_types.StorageTrace{
				RootBefore: common.BigToHash(big.NewInt(int64(i))),
				RootAfter:  common.BigToHash(big.NewInt(int64(i + 1))),
			},
			WithdrawTrieRoot: withdrawTrieRoot,
		})
	}

	// insert blocks
	err = db.InsertL2BlockTraces(traces)
	assert.NoError(t, err)

	// insert genesis batch
	genssisBatchData := types.NewGenesisBatchData(traces[0])
	err = AddBatchInfoToDB(db, genssisBatchData, make([]*types.L2Message, 0), make([][]byte, 0))
	assert.NoError(t, err)
	batchHash := genssisBatchData.Hash().Hex()
	err = db.UpdateProvingStatus(batchHash, types.ProvingTaskProved)
	assert.NoError(t, err)
	err = db.UpdateRollupStatus(context.Background(), batchHash, types.RollupFinalized)
	assert.NoError(t, err)

	// create proposer
	relayer, err := NewLayer2Relayer(context.Background(), l2Cli, db, cfg.L2Config.RelayerConfig)
	assert.NoError(t, err)

	proposer := NewBatchProposer(context.Background(), &config.BatchProposerConfig{
		ProofGenerationFreq: 1,
		BatchGasThreshold:   3000000,
		BatchTxNumThreshold: 135,
		BatchTimeSec:        1,
		BatchBlocksLimit:    100,
	}, relayer, db)

	var blocks []*types.BlockInfo
	var proof sql.NullString
	// propose batch with 1 block
	blocks, err = db.GetUnbatchedL2Blocks(map[string]interface{}{}, "order by number ASC LIMIT 1")
	assert.NoError(t, err)
	proposer.proposeBatch(blocks)
	// propose batch with 2 block
	blocks, err = db.GetUnbatchedL2Blocks(map[string]interface{}{}, "order by number ASC LIMIT 2")
	assert.NoError(t, err)
	proposer.proposeBatch(blocks)
	// propose batch with 3 block
	blocks, err = db.GetUnbatchedL2Blocks(map[string]interface{}{}, "order by number ASC LIMIT 3")
	assert.NoError(t, err)
	proposer.proposeBatch(blocks)

	// save messages
	err = db.SaveL2Messages(context.Background(), msgs)
	assert.NoError(t, err)

	for i := 0; i < 10; i++ {
		proof, err = db.GetL2MessageProofByNonce(msgs[i].Nonce)
		assert.NoError(t, err)
		assert.Equal(t, proof.Valid, false)
	}

	// initialize missing proof
	err = proposer.initializeMissingMessageProof()
	assert.NoError(t, err)

	for i := 0; i < 6; i++ {
		proof, err = db.GetL2MessageProofByNonce(msgs[i].Nonce)
		assert.NoError(t, err)
		assert.Equal(t, proof.Valid, true)
	}

	// propose batch with 4 block
	blocks, err = db.GetUnbatchedL2Blocks(map[string]interface{}{}, "order by number ASC LIMIT 4")
	assert.NoError(t, err)
	proposer.proposeBatch(blocks)
	for i := 0; i < 10; i++ {
		proof, err = db.GetL2MessageProofByNonce(msgs[i].Nonce)
		assert.NoError(t, err)
		assert.Equal(t, proof.Valid, true)
	}
}
