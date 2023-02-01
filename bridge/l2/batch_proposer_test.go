package l2

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

	"scroll-tech/database"
	"scroll-tech/database/migrate"

	"scroll-tech/bridge/config"

	"scroll-tech/common/utils"
)

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

	id := utils.ComputeBatchID(trace3.Header.Hash(), trace2.Header.ParentHash, big.NewInt(0))

	proposer := newBatchProposer(&config.BatchProposerConfig{
		ProofGenerationFreq: 1,
		BatchGasThreshold:   3000000,
		BatchTxNumThreshold: 135,
		BatchTimeSec:        1,
		BatchBlocksLimit:    100,
	}, db)
	proposer.tryProposeBatch()

	infos, err := db.GetUnbatchedBlocks(map[string]interface{}{},
		fmt.Sprintf("order by number ASC LIMIT %d", 100))
	assert.NoError(t, err)
	assert.Equal(t, true, len(infos) == 0)

	exist, err := db.BatchRecordExist(id)
	assert.NoError(t, err)
	assert.Equal(t, true, exist)
}
