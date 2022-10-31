package l2_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

	"scroll-tech/bridge/l2"

	"scroll-tech/database"
	"scroll-tech/database/migrate"
	"scroll-tech/database/orm"
)

var (
	templateLayer2Message = []*orm.Layer2Message{
		{
			Nonce:      1,
			Height:     1,
			Sender:     "0x596a746661dbed76a84556111c2872249b070e15",
			Value:      "100",
			Fee:        "100",
			GasLimit:   11529940,
			Deadline:   uint64(time.Now().Unix()),
			Target:     "0x2c73620b223808297ea734d946813f0dd78eb8f7",
			Calldata:   "testdata",
			Layer2Hash: "hash0",
		},
	}
)

func testCreateNewRelayer(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))

	skippedOpcodes := make(map[string]struct{}, len(cfg.L2Config.SkippedOpcodes))
	for _, op := range cfg.L2Config.SkippedOpcodes {
		skippedOpcodes[op] = struct{}{}
	}

	relayer, err := l2.NewLayer2Relayer(context.Background(), l2Cli, cfg.L2Config.ProofGenerationFreq, skippedOpcodes, int64(cfg.L2Config.Confirmations), db, cfg.L2Config.RelayerConfig)
	assert.NoError(t, err)

	relayer.Start()

	defer relayer.Stop()
}

func testL2RelayerProcessSaveEvents(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer assert.NoError(t, db.Close())

	skippedOpcodes := make(map[string]struct{}, len(cfg.L2Config.SkippedOpcodes))
	for _, op := range cfg.L2Config.SkippedOpcodes {
		skippedOpcodes[op] = struct{}{}
	}
	relayer, err := l2.NewLayer2Relayer(context.Background(), l2Cli, cfg.L2Config.ProofGenerationFreq, skippedOpcodes, int64(cfg.L2Config.Confirmations), db, cfg.L2Config.RelayerConfig)
	assert.NoError(t, err)
	defer relayer.Stop()

	err = db.SaveLayer2Messages(context.Background(), templateLayer2Message)
	assert.NoError(t, err)
	blocks := []*orm.RollupResult{
		{
			Number:         3,
			Status:         orm.RollupFinalized,
			RollupTxHash:   "Rollup Test Hash",
			FinalizeTxHash: "Finalized Hash",
		},
	}
	err = db.InsertPendingBlocks(context.Background(), []uint64{uint64(blocks[0].Number)})
	assert.NoError(t, err)
	err = db.UpdateRollupStatus(context.Background(), uint64(blocks[0].Number), orm.RollupFinalized)
	assert.NoError(t, err)
	relayer.ProcessSavedEvents()

	msg, err := db.GetLayer2MessageByNonce(templateLayer2Message[0].Nonce)
	assert.NoError(t, err)
	assert.Equal(t, orm.MsgSubmitted, msg.Status)
}

func testL2RelayerProcessPendingBlocks(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))

	skippedOpcodes := make(map[string]struct{}, len(cfg.L2Config.SkippedOpcodes))
	for _, op := range cfg.L2Config.SkippedOpcodes {
		skippedOpcodes[op] = struct{}{}
	}
	relayer, err := l2.NewLayer2Relayer(context.Background(), l2Cli, cfg.L2Config.ProofGenerationFreq, skippedOpcodes, int64(cfg.L2Config.Confirmations), db, cfg.L2Config.RelayerConfig)
	assert.NoError(t, err)

	// this blockresult has number of 0x4, need to change it to match the testcase
	// In this testcase scenario, db will store two blocks with height 0x4 and 0x3
	var results []*types.BlockResult

	templateBlockResult, err := os.ReadFile("../../common/testdata/blockResult_relayer_parent.json")
	assert.NoError(t, err)
	blockResult := &types.BlockResult{}
	err = json.Unmarshal(templateBlockResult, blockResult)
	assert.NoError(t, err)
	results = append(results, blockResult)
	templateBlockResult, err = os.ReadFile("../../common/testdata/blockResult_relayer.json")
	assert.NoError(t, err)
	blockResult = &types.BlockResult{}
	err = json.Unmarshal(templateBlockResult, blockResult)
	assert.NoError(t, err)
	results = append(results, blockResult)

	err = db.InsertBlockResultsWithStatus(context.Background(), results, orm.BlockUnassigned)
	assert.NoError(t, err)

	blocks := []*orm.RollupResult{
		{
			Number:         4,
			Status:         1,
			RollupTxHash:   "Rollup Test Hash",
			FinalizeTxHash: "Finalized Hash",
		},
	}
	err = db.InsertPendingBlocks(context.Background(), []uint64{uint64(blocks[0].Number)})
	assert.NoError(t, err)
	err = db.UpdateRollupStatus(context.Background(), uint64(blocks[0].Number), orm.RollupPending)
	assert.NoError(t, err)

	relayer.ProcessPendingBlocks()

	// Check if Rollup Result is changed successfully
	status, err := db.GetRollupStatus(uint64(blocks[0].Number))
	assert.NoError(t, err)
	assert.Equal(t, orm.RollupCommitting, status)
}

func testL2RelayerProcessCommittedBlocks(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer assert.NoError(t, db.Close())

	skippedOpcodes := make(map[string]struct{}, len(cfg.L2Config.SkippedOpcodes))
	for _, op := range cfg.L2Config.SkippedOpcodes {
		skippedOpcodes[op] = struct{}{}
	}
	relayer, err := l2.NewLayer2Relayer(context.Background(), l2Cli, cfg.L2Config.ProofGenerationFreq, skippedOpcodes, int64(cfg.L2Config.Confirmations), db, cfg.L2Config.RelayerConfig)
	assert.NoError(t, err)
	defer relayer.Stop()

	templateBlockResult, err := os.ReadFile("../../common/testdata/blockResult_relayer.json")
	assert.NoError(t, err)
	blockResult := &types.BlockResult{}
	err = json.Unmarshal(templateBlockResult, blockResult)
	assert.NoError(t, err)
	err = db.InsertBlockResultsWithStatus(context.Background(), []*types.BlockResult{blockResult}, orm.BlockVerified)
	assert.NoError(t, err)

	blocks := []*orm.RollupResult{
		{
			Number:         4,
			Status:         1,
			RollupTxHash:   "Rollup Test Hash",
			FinalizeTxHash: "Finalized Hash",
		},
	}
	err = db.InsertPendingBlocks(context.Background(), []uint64{uint64(blocks[0].Number)})
	assert.NoError(t, err)
	err = db.UpdateRollupStatus(context.Background(), uint64(blocks[0].Number), orm.RollupCommitted)
	assert.NoError(t, err)
	tProof := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}
	tStateProof := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}

	err = db.UpdateProofByNumber(context.Background(), uint64(blocks[0].Number), tProof, tStateProof, 100)
	assert.NoError(t, err)
	relayer.ProcessCommittedBlocks()

	status, err := db.GetRollupStatus(uint64(blocks[0].Number))
	assert.NoError(t, err)
	assert.Equal(t, orm.RollupFinalizing, status)
}
