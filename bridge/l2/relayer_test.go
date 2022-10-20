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

	"scroll-tech/common/docker"
	db_config "scroll-tech/database"

	"scroll-tech/database"
	"scroll-tech/database/orm"

	"scroll-tech/bridge/config"

	"scroll-tech/bridge/l2"
	"scroll-tech/bridge/mock"
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
	l1Docker docker.ImgInstance
	l2Docker docker.ImgInstance
	dbDocker docker.ImgInstance
)

func setupEnv(t *testing.T) {
	l1Docker = mock.NewL1Docker(t, TEST_CONFIG)
	l2Docker = mock.NewL2Docker(t, TEST_CONFIG)
	dbDocker = mock.NewDBDocker(t, TEST_CONFIG)
}

func TestRelayerFunction(t *testing.T) {
	// Setup
	setupEnv(t)

	t.Run("TestCreateNewRelayer", func(t *testing.T) {
		cfg, err := config.NewConfig("../config.json")
		assert.NoError(t, err)
		cfg.L2Config.RelayerConfig.SenderConfig.Endpoint = l1Docker.Endpoint()

		client, err := ethclient.Dial(l2Docker.Endpoint())
		assert.NoError(t, err)

		db, err := database.NewOrmFactory(TEST_CONFIG.DB_CONFIG)
		assert.NoError(t, err)

		skippedOpcodes := make(map[string]struct{}, len(cfg.L2Config.SkippedOpcodes))
		for _, op := range cfg.L2Config.SkippedOpcodes {
			skippedOpcodes[op] = struct{}{}
		}

		relayer, err := l2.NewLayer2Relayer(context.Background(), client, cfg.L2Config.ProofGenerationFreq, skippedOpcodes, int64(cfg.L2Config.Confirmations), db, cfg.L2Config.RelayerConfig)
		assert.NoError(t, err)

		relayer.Start()

		defer relayer.Stop()
	})

	t.Run("TestL2RelayerProcessSaveEvents", func(t *testing.T) {
		cfg, err := config.NewConfig("../config.json")
		assert.NoError(t, err)
		cfg.L2Config.RelayerConfig.SenderConfig.Endpoint = l1Docker.Endpoint()

		client, err := ethclient.Dial(l2Docker.Endpoint())
		assert.NoError(t, err)

		db, err := database.NewOrmFactory(&db_config.DBConfig{
			DriverName: "postgres",
			DSN:        dbDocker.Endpoint(),
		})
		assert.NoError(t, err)

		skippedOpcodes := make(map[string]struct{}, len(cfg.L2Config.SkippedOpcodes))
		for _, op := range cfg.L2Config.SkippedOpcodes {
			skippedOpcodes[op] = struct{}{}
		}
		relayer, err := l2.NewLayer2Relayer(context.Background(), client, cfg.L2Config.ProofGenerationFreq, skippedOpcodes, int64(cfg.L2Config.Confirmations), db, cfg.L2Config.RelayerConfig)
		assert.NoError(t, err)

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

		var (
			tick     = time.Tick(time.Millisecond * 500)
			tickStop = time.Tick(time.Second * 3)
		)
		for {
			select {
			case <-tick:
				msg, err := db.GetLayer2MessageByNonce(templateLayer2Message[0].Nonce)
				if err == nil && orm.MsgSubmitted == msg.Status {
					return
				}
			case <-tickStop:
				t.Error("wait l2 message MsgSubmitted status timeout")
			}
		}

	})

	t.Run("TestL2RelayerProcessPendingBlocks", func(t *testing.T) {
		cfg, err := config.NewConfig("../config.json")
		assert.NoError(t, err)
		cfg.L2Config.RelayerConfig.SenderConfig.Endpoint = l1Docker.Endpoint()

		client, err := ethclient.Dial(l2Docker.Endpoint())
		assert.NoError(t, err)

		db, err := database.NewOrmFactory(&db_config.DBConfig{
			DriverName: "postgres",
			DSN:        dbDocker.Endpoint(),
		})
		assert.NoError(t, err)

		skippedOpcodes := make(map[string]struct{}, len(cfg.L2Config.SkippedOpcodes))
		for _, op := range cfg.L2Config.SkippedOpcodes {
			skippedOpcodes[op] = struct{}{}
		}
		relayer, err := l2.NewLayer2Relayer(context.Background(), client, cfg.L2Config.ProofGenerationFreq, skippedOpcodes, int64(cfg.L2Config.Confirmations), db, cfg.L2Config.RelayerConfig)
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

		var (
			tick     = time.Tick(time.Millisecond * 500)
			tickStop = time.Tick(time.Second * 3)
		)
		for {
			select {
			case <-tick:
				// Check if Rollup Result is changed successfully
				status, err := db.GetRollupStatus(uint64(blocks[0].Number))
				if err == nil && orm.RollupCommitting == status {
					return
				}
			case <-tickStop:
				t.Error("wait rollup RollupCommitting status timeout")
			}
		}
	})

	t.Run("TestL2RelayerProcessCommittedBlocks", func(t *testing.T) {
		cfg, err := config.NewConfig("../config.json")
		assert.NoError(t, err)
		cfg.L2Config.RelayerConfig.SenderConfig.Endpoint = l1Docker.Endpoint()

		client, err := ethclient.Dial(l2Docker.Endpoint())
		assert.NoError(t, err)

		db, err := database.NewOrmFactory(&db_config.DBConfig{
			DriverName: "postgres",
			DSN:        dbDocker.Endpoint(),
		})
		assert.NoError(t, err)

		skippedOpcodes := make(map[string]struct{}, len(cfg.L2Config.SkippedOpcodes))
		for _, op := range cfg.L2Config.SkippedOpcodes {
			skippedOpcodes[op] = struct{}{}
		}
		relayer, err := l2.NewLayer2Relayer(context.Background(), client, cfg.L2Config.ProofGenerationFreq, skippedOpcodes, int64(cfg.L2Config.Confirmations), db, cfg.L2Config.RelayerConfig)
		assert.NoError(t, err)

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
		assert.NoError(t, relayer.ProcessCommittedBlocks())

		<-time.After(time.Millisecond * 500)

		status, err := db.GetRollupStatus(uint64(blocks[0].Number))
		assert.NoError(t, err)
		assert.Equal(t, orm.RollupFinalizing, status)
	})

	// Teardown
	t.Cleanup(func() {
		assert.NoError(t, l1Docker.Stop())
		assert.NoError(t, l2Docker.Stop())
		assert.NoError(t, dbDocker.Stop())
	})
}
