package l2

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	geth_types "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

	"scroll-tech/database"
	"scroll-tech/database/migrate"

	"scroll-tech/bridge/config"
	"scroll-tech/bridge/l1"

	"scroll-tech/common/types"
)

func testBatchProposer(t *testing.T) {
	// prepare trace.
	templateBlockTrace, err := os.ReadFile("../../common/testdata/blockTrace_02.json")
	assert.NoError(t, err)
	// unmarshal blockTrace
	blockTrace := &geth_types.BlockTrace{}
	assert.NoError(t, json.Unmarshal(templateBlockTrace, blockTrace))

	parentBatch := &types.BlockBatch{
		Index: 1,
		Hash:  "0x0000000000000000000000000000000000000000",
	}
	batchData := types.NewBatchData(parentBatch, []*geth_types.BlockTrace{blockTrace}, nil)

	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	// Insert traces into db.
	assert.NoError(t, db.InsertBlockTraces([]*geth_types.BlockTrace{blockTrace}))

	l2cfg := cfg.L2Config
	rc := NewL2WatcherClient(context.Background(), l2Cli, l2cfg.Confirmations, l2cfg.BatchProposerConfig, l2cfg.L2MessengerAddress, l2cfg.L2MessageQueueAddress, nil, db)
	rc.Start()
	defer rc.Stop()

	relayer, err := l1.NewLayer1Relayer(context.Background(), db, cfg.L2Config.RelayerConfig)
	assert.NoError(t, err)
	defer relayer.Stop()
	relayer.Start()

	proposer := newBatchProposer(&config.BatchProposerConfig{
		ProofGenerationFreq: 1,
		BatchGasThreshold:   3000000,
		BatchTxNumThreshold: 135,
		BatchTimeSec:        1,
		BatchBlocksLimit:    100,
	}, nil, db)
	proposer.tryProposeBatch()

	infos, err := db.GetUnbatchedBlocks(map[string]interface{}{},
		fmt.Sprintf("order by number ASC LIMIT %d", 100))
	assert.NoError(t, err)
	assert.Equal(t, 0, len(infos))

	exist, err := db.BatchRecordExist(batchData.Hash().Hex())
	assert.NoError(t, err)
	assert.Equal(t, true, exist)
}
