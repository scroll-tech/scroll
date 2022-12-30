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

	"scroll-tech/common/utils"
	"scroll-tech/common/viper"
)

func testBatchProposer(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(vp.Sub("db_config"))
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

	tmpVP := viper.NewEmptyViper()
	tmpVP.Set("proof_generation_freq", 1)
	tmpVP.Set("batch_gas_threshold", 3000000)
	tmpVP.Set("batch_tx_num_threshold", 135)
	tmpVP.Set("batch_time_sec", 1)
	tmpVP.Set("batch_blocks_limit", 100)
	proposer := newBatchProposer(db, tmpVP)
	assert.NoError(t, proposer.tryProposeBatch())

	infos, err := db.GetUnbatchedBlocks(map[string]interface{}{},
		fmt.Sprintf("order by number ASC LIMIT %d", 100))
	assert.NoError(t, err)
	assert.Equal(t, true, len(infos) == 0)

	exist, err := db.BatchRecordExist(id)
	assert.NoError(t, err)
	assert.Equal(t, true, exist)
}
