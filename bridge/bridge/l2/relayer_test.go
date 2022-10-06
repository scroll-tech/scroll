package l2_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"

	"scroll-tech/bridge/bridge/l2"
	"scroll-tech/bridge/config"
	"scroll-tech/internal/mock"
	"scroll-tech/store"
	"scroll-tech/store/orm"
)

var templateLayer2Message = []*orm.Layer2Message{
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

// TestCreateNewRelayer test create new relayer instance and stop
func TestCreateNewRelayer(t *testing.T) {
	assert := assert.New(t)
	cfg, err := config.NewConfig("../../config.json")
	assert.NoError(err)
	_, l2Docker, dbDocker := mock.Mockl2gethDocker(t, cfg, TEST_CONFIG)
	defer l2Docker.Stop()
	defer dbDocker.Stop()
	client, err := ethclient.Dial(l2Docker.Endpoint())
	assert.NoError(err)

	db, err := store.NewOrmFactory(TEST_CONFIG.DB_CONFIG)
	assert.NoError(err)

	skippedOpcodes := make(map[string]struct{}, len(cfg.L2Config.SkippedOpcodes))
	for _, op := range cfg.L2Config.SkippedOpcodes {
		skippedOpcodes[op] = struct{}{}
	}

	relayer, err := l2.NewLayer2Relayer(context.Background(), client, cfg.L2Config.ProofGenerationFreq, skippedOpcodes, db, cfg.L2Config.RelayerConfig)
	assert.NoError(err)

	relayer.Start()

	defer relayer.Stop()

}

// TestL2RelayerProcessSaveEvents will test l2relayer process db stored block events to sender
func TestL2RelayerProcessSaveEvents(t *testing.T) {
	assert := assert.New(t)
	cfg, err := config.NewConfig("../../config.json")
	assert.NoError(err)

	l1docker := mock.NewTestL1Docker(t, TEST_CONFIG)
	l1docker.Start()
	defer l1docker.Stop()
	cfg.L2Config.RelayerConfig.SenderConfig.Endpoint = l1docker.Endpoint()
	_, l2Docker, dbDocker := mock.Mockl2gethDocker(t, cfg, TEST_CONFIG)
	defer l2Docker.Stop()
	defer dbDocker.Stop()
	client, err := ethclient.Dial(l2Docker.Endpoint())
	assert.NoError(err)

	mock.MockClearDB(assert, TEST_CONFIG.DB_CONFIG)
	db := mock.MockPrepareDB(assert, TEST_CONFIG.DB_CONFIG)

	skippedOpcodes := make(map[string]struct{}, len(cfg.L2Config.SkippedOpcodes))
	for _, op := range cfg.L2Config.SkippedOpcodes {
		skippedOpcodes[op] = struct{}{}
	}
	relayer, err := l2.NewLayer2Relayer(context.Background(), client, cfg.L2Config.ProofGenerationFreq, skippedOpcodes, db, cfg.L2Config.RelayerConfig)
	assert.NoError(err)

	err = db.SaveLayer2Messages(context.Background(), templateLayer2Message)
	assert.NoError(err)
	blocks := []*orm.RollupResult{
		{
			Number:         3,
			Status:         orm.RollupFinalized,
			RollupTxHash:   "Rollup Test Hash",
			FinalizeTxHash: "Finalized Hash",
		},
	}
	err = db.InsertPendingBlocks(context.Background(), []uint64{uint64(blocks[0].Number)})
	assert.NoError(err)
	err = db.UpdateRollupStatus(context.Background(), uint64(blocks[0].Number), orm.RollupFinalized)
	assert.NoError(err)
	relayer.ProcessSavedEvents()

	msg, err := db.GetLayer2MessageByNonce(templateLayer2Message[0].Nonce)
	assert.NoError(err)
	assert.Equal(orm.MsgSubmitted, msg.Status)
}

// TestL2RelayerProcessPendingBlocks will test reayer process pending blocks event
func TestL2RelayerProcessPendingBlocks(t *testing.T) {
	assert := assert.New(t)
	cfg, err := config.NewConfig("../../config.json")
	assert.NoError(err)

	_, l2Docker, dbDocker := mock.Mockl2gethDocker(t, cfg, TEST_CONFIG)
	defer l2Docker.Stop()
	defer dbDocker.Stop()
	client, err := ethclient.Dial(l2Docker.Endpoint())
	assert.NoError(err)

	mock.MockClearDB(assert, TEST_CONFIG.DB_CONFIG)
	db := mock.MockPrepareDB(assert, TEST_CONFIG.DB_CONFIG)

	skippedOpcodes := make(map[string]struct{}, len(cfg.L2Config.SkippedOpcodes))
	for _, op := range cfg.L2Config.SkippedOpcodes {
		skippedOpcodes[op] = struct{}{}
	}
	relayer, err := l2.NewLayer2Relayer(context.Background(), client, cfg.L2Config.ProofGenerationFreq, skippedOpcodes, db, cfg.L2Config.RelayerConfig)
	assert.NoError(err)

	// this blockresult has number of 0x4, need to change it to match the testcase
	// In this testcase scenario, db will store two blocks with height 0x4 and 0x3
	var results []*types.BlockResult

	templateBlockResult, err := os.ReadFile("../../../internal/testdata/blockResult_relayer_parent.json")
	assert.NoError(err)
	blockResult := &types.BlockResult{}
	err = json.Unmarshal(templateBlockResult, blockResult)
	assert.NoError(err)
	results = append(results, blockResult)
	templateBlockResult, err = os.ReadFile("../../../internal/testdata/blockResult_relayer.json")
	assert.NoError(err)
	blockResult = &types.BlockResult{}
	err = json.Unmarshal(templateBlockResult, blockResult)
	assert.NoError(err)
	results = append(results, blockResult)

	err = db.InsertBlockResultsWithStatus(context.Background(), results, orm.BlockUnassigned)
	assert.NoError(err)

	blocks := []*orm.RollupResult{
		{
			Number:         4,
			Status:         1,
			RollupTxHash:   "Rollup Test Hash",
			FinalizeTxHash: "Finalized Hash",
		},
	}
	err = db.InsertPendingBlocks(context.Background(), []uint64{uint64(blocks[0].Number)})
	assert.NoError(err)
	err = db.UpdateRollupStatus(context.Background(), uint64(blocks[0].Number), orm.RollupPending)
	assert.NoError(err)

	relayer.ProcessPendingBlocks()

	// Check if Rollup Result is changed successfully
	status, err := db.GetRollupStatus(uint64(blocks[0].Number))
	assert.NoError(err)
	assert.Equal(orm.RollupCommitting, status)
}

// TestL2RelayerProcessCommittedBlocks test relayer process db stored blocks
func TestL2RelayerProcessCommittedBlocks(t *testing.T) {
	assert := assert.New(t)
	cfg, err := config.NewConfig("../../config.json")
	assert.NoError(err)

	_, l2Docker, dbDocker := mock.Mockl2gethDocker(t, cfg, TEST_CONFIG)
	defer l2Docker.Stop()
	defer dbDocker.Stop()
	client, err := ethclient.Dial(l2Docker.Endpoint())
	assert.NoError(err)

	mock.MockClearDB(assert, TEST_CONFIG.DB_CONFIG)
	db := mock.MockPrepareDB(assert, TEST_CONFIG.DB_CONFIG)

	skippedOpcodes := make(map[string]struct{}, len(cfg.L2Config.SkippedOpcodes))
	for _, op := range cfg.L2Config.SkippedOpcodes {
		skippedOpcodes[op] = struct{}{}
	}
	relayer, err := l2.NewLayer2Relayer(context.Background(), client, cfg.L2Config.ProofGenerationFreq, skippedOpcodes, db, cfg.L2Config.RelayerConfig)
	assert.NoError(err)

	templateBlockResult, err := os.ReadFile("../../../internal/testdata/blockResult_relayer.json")
	assert.NoError(err)
	blockResult := &types.BlockResult{}
	err = json.Unmarshal(templateBlockResult, blockResult)
	assert.NoError(err)
	err = db.InsertBlockResultsWithStatus(context.Background(), []*types.BlockResult{blockResult}, orm.BlockVerified)
	assert.NoError(err)

	blocks := []*orm.RollupResult{
		{
			Number:         4,
			Status:         1,
			RollupTxHash:   "Rollup Test Hash",
			FinalizeTxHash: "Finalized Hash",
		},
	}
	err = db.InsertPendingBlocks(context.Background(), []uint64{uint64(blocks[0].Number)})
	assert.NoError(err)
	err = db.UpdateRollupStatus(context.Background(), uint64(blocks[0].Number), orm.RollupCommitted)
	assert.NoError(err)
	tProof := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}
	tStateProof := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}

	err = db.UpdateProofByNumber(context.Background(), uint64(blocks[0].Number), tProof, tStateProof, 100)
	assert.NoError(err)
	relayer.ProcessCommittedBlocks()

	status, err := db.GetRollupStatus(uint64(blocks[0].Number))
	assert.NoError(err)
	assert.Equal(orm.RollupFinalizing, status)
}
