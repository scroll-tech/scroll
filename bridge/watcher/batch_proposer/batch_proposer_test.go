package batchProposer

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

	"scroll-tech/database"
	"scroll-tech/database/migrate"

	"scroll-tech/bridge/config"

	"scroll-tech/common/docker"
	"scroll-tech/common/utils"
)

var (
	// config
	cfg *config.Config

	// docker consider handler.
	l1gethImg docker.ImgInstance
	l2gethImg docker.ImgInstance
	dbImg     docker.ImgInstance

	// l2geth client
	l2Cli *ethclient.Client
)

func setupEnv(t *testing.T) (err error) {
	// Load config.
	cfg, err = config.NewConfig("../config.json")
	assert.NoError(t, err)

	// Create l1geth container.
	l1gethImg = docker.NewTestL1Docker(t)
	cfg.L2Config.RelayerConfig.SenderConfig.Endpoint = l1gethImg.Endpoint()
	cfg.L1Config.Endpoint = l1gethImg.Endpoint()

	// Create l2geth container.
	l2gethImg = docker.NewTestL2Docker(t)
	cfg.L1Config.RelayerConfig.SenderConfig.Endpoint = l2gethImg.Endpoint()
	cfg.L2Config.Endpoint = l2gethImg.Endpoint()

	// Create db container.
	dbImg = docker.NewTestDBDocker(t, cfg.DBConfig.DriverName)
	cfg.DBConfig.DSN = dbImg.Endpoint()

	// Create l2geth client.
	l2Cli, err = ethclient.Dial(cfg.L2Config.Endpoint)
	assert.NoError(t, err)

	return err
}

func free(t *testing.T) {
	if dbImg != nil {
		assert.NoError(t, dbImg.Stop())
	}
	if l1gethImg != nil {
		assert.NoError(t, l1gethImg.Stop())
	}
	if l2gethImg != nil {
		assert.NoError(t, l2gethImg.Stop())
	}
}

func TestFunction(t *testing.T) {
	if err := setupEnv(t); err != nil {
		t.Fatal(err)
	}

	t.Run("testBatchProposer", testBatchProposer)

	t.Cleanup(func() {
		free(t)
	})
}

func testBatchProposer(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	trace2 := &types.BlockTrace{}
	trace3 := &types.BlockTrace{}

	data, err := os.ReadFile("../../common/testdata/blockTrace_02.json")
	assert.NoError(t, err)
	err = json.Unmarshal(data, trace2)
	assert.NoError(t, err)

	data, err = os.ReadFile("../../common/testdata/blockTrace_03.json")
	assert.NoError(t, err)
	err = json.Unmarshal(data, trace3)
	assert.NoError(t, err)
	// Insert traces into db.
	assert.NoError(t, db.InsertBlockTraces([]*types.BlockTrace{trace2, trace3}))

	id := utils.ComputeBatchID(trace3.Header.Hash(), trace2.Header.ParentHash, big.NewInt(1))

	proposer := NewBatchProposer(&config.BatchProposerConfig{
		ProofGenerationFreq: 1,
		BatchGasThreshold:   3000000,
		BatchTxNumThreshold: 135,
		BatchTimeSec:        1,
		BatchBlocksLimit:    100,
	}, db)
	proposer.TryProposeBatch()

	infos, err := db.GetUnbatchedBlocks(map[string]interface{}{},
		fmt.Sprintf("order by number ASC LIMIT %d", 100))
	assert.NoError(t, err)
	assert.Equal(t, true, len(infos) == 0)

	exist, err := db.BatchRecordExist(id)
	assert.NoError(t, err)
	assert.Equal(t, true, exist)
}
