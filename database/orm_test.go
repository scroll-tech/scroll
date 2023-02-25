package database_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/scroll-tech/go-ethereum/common"
	geth_types "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

	"scroll-tech/database/cache"

	"scroll-tech/common/docker"
	"scroll-tech/common/types"

	abi "scroll-tech/bridge/abi"

	"scroll-tech/database"
	"scroll-tech/database/migrate"
	"scroll-tech/database/orm"
)

var (
	templateL1Message = []*types.L1Message{
		{
			QueueIndex: 1,
			MsgHash:    "msg_hash1",
			Height:     1,
			Sender:     "0x596a746661dbed76a84556111c2872249b070e15",
			Value:      "0x19ece",
			GasLimit:   11529940,
			Target:     "0x2c73620b223808297ea734d946813f0dd78eb8f7",
			Calldata:   "testdata",
			Layer1Hash: "hash0",
		},
		{
			QueueIndex: 2,
			MsgHash:    "msg_hash2",
			Height:     2,
			Sender:     "0x596a746661dbed76a84556111c2872249b070e15",
			Value:      "0x19ece",
			GasLimit:   11529940,
			Target:     "0x2c73620b223808297ea734d946813f0dd78eb8f7",
			Calldata:   "testdata",
			Layer1Hash: "hash1",
		},
	}
	templateL2Message = []*types.L2Message{
		{
			Nonce:      1,
			MsgHash:    "msg_hash1",
			Height:     1,
			Sender:     "0x596a746661dbed76a84556111c2872249b070e15",
			Value:      "0x19ece",
			Target:     "0x2c73620b223808297ea734d946813f0dd78eb8f7",
			Calldata:   "testdata",
			Layer2Hash: "hash0",
		},
		{
			Nonce:      2,
			MsgHash:    "msg_hash2",
			Height:     2,
			Sender:     "0x596a746661dbed76a84556111c2872249b070e15",
			Value:      "0x19ece",
			Target:     "0x2c73620b223808297ea734d946813f0dd78eb8f7",
			Calldata:   "testdata",
			Layer2Hash: "hash1",
		},
	}
	blockTrace *geth_types.BlockTrace
	batchData1 *types.BatchData
	batchData2 *types.BatchData

	dbConfig   *database.DBConfig
	dbImg      docker.ImgInstance
	redisImg   docker.ImgInstance
	ormBlock   orm.BlockTraceOrm
	ormLayer1  orm.L1MessageOrm
	ormLayer2  orm.L2MessageOrm
	ormBatch   orm.BlockBatchOrm
	ormSession orm.SessionInfoOrm
)

func setupEnv(t *testing.T) error {
	// Init db config and start db container.
	dbImg = docker.NewTestDBDocker(t, "postgres")
	redisImg = docker.NewTestRedisDocker(t)
	dbConfig = &database.DBConfig{
		Persistence: &database.PersistenceConfig{
			DriverName: "postgres",
			DSN:        dbImg.Endpoint(),
		},
		Redis: &cache.RedisConfig{
			URL:         redisImg.Endpoint(),
			Expirations: map[string]int64{"trace": 30},
		},
	}

	// Create db handler and reset db.
	factory, err := database.NewOrmFactory(dbConfig)
	assert.NoError(t, err)
	db := factory.GetDB()
	assert.NoError(t, migrate.ResetDB(db.DB))

	// Init several orm handles.
	ormBlock = orm.BlockTraceOrm(factory)
	ormLayer1 = orm.L1MessageOrm(factory)
	ormLayer2 = orm.L2MessageOrm(factory)
	ormBatch = orm.BlockBatchOrm(factory)
	ormSession = orm.SessionInfoOrm(factory)

	templateBlockTrace, err := os.ReadFile("../common/testdata/blockTrace_02.json")
	if err != nil {
		return err
	}
	// unmarshal blockTrace
	blockTrace = &geth_types.BlockTrace{}
	if err = json.Unmarshal(templateBlockTrace, blockTrace); err != nil {
		return err
	}

	parentBatch := &types.BlockBatch{
		Index: 1,
		Hash:  "0x0000000000000000000000000000000000000000",
	}
	batchData1 = types.NewBatchData(parentBatch, []*geth_types.BlockTrace{blockTrace}, nil)

	templateBlockTrace, err = os.ReadFile("../common/testdata/blockTrace_03.json")
	if err != nil {
		return err
	}
	// unmarshal blockTrace
	blockTrace2 := &geth_types.BlockTrace{}
	if err = json.Unmarshal(templateBlockTrace, blockTrace2); err != nil {
		return err
	}
	parentBatch2 := &types.BlockBatch{
		Index: batchData1.Batch.BatchIndex,
		Hash:  batchData1.Hash().Hex(),
	}
	batchData2 = types.NewBatchData(parentBatch2, []*geth_types.BlockTrace{blockTrace2}, nil)

	// insert a fake empty block to batchData2
	fakeBlockContext := abi.IScrollChainBlockContext{
		BlockHash:       common.HexToHash("0x000000000000000000000000000000000000000000000000000000000000dead"),
		ParentHash:      batchData2.Batch.Blocks[0].BlockHash,
		BlockNumber:     batchData2.Batch.Blocks[0].BlockNumber + 1,
		BaseFee:         new(big.Int).SetUint64(0),
		Timestamp:       123456789,
		GasLimit:        10000000000000000,
		NumTransactions: 0,
		NumL1Messages:   0,
	}
	batchData2.Batch.Blocks = append(batchData2.Batch.Blocks, fakeBlockContext)
	batchData2.Batch.NewStateRoot = common.HexToHash("0x000000000000000000000000000000000000000000000000000000000000cafe")

	fmt.Printf("batchhash1 = %x\n", batchData1.Hash())
	fmt.Printf("batchhash2 = %x\n", batchData2.Hash())
	return nil
}

func freeDB(t *testing.T) {
	if dbImg != nil {
		assert.NoError(t, dbImg.Stop())
	}
	if redisImg != nil {
		assert.NoError(t, redisImg.Stop())
	}
}

// TestOrmFactory run several test cases.
func TestOrmFactory(t *testing.T) {
	if err := setupEnv(t); err != nil {
		t.Fatal(err)
	}
	defer freeDB(t)

	t.Run("testOrmBlockTraces", testOrmBlockTraces)

	t.Run("testOrmL1Message", testOrmL1Message)

	t.Run("testOrmL2Message", testOrmL2Message)

	t.Run("testOrmBlockBatch", testOrmBlockBatch)

	t.Run("testOrmSessionInfo", testOrmSessionInfo)
}

func testOrmBlockTraces(t *testing.T) {
	// Create db handler and reset db.
	factory, err := database.NewOrmFactory(dbConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(factory.GetDB().DB))

	res, err := ormBlock.GetL2BlockTraces(map[string]interface{}{})
	assert.NoError(t, err)
	assert.Equal(t, true, len(res) == 0)

	exist, err := ormBlock.IsL2BlockExists(blockTrace.Header.Number.Uint64())
	assert.NoError(t, err)
	assert.Equal(t, false, exist)

	// Insert into db
	err = ormBlock.InsertL2BlockTraces([]*geth_types.BlockTrace{blockTrace})
	assert.NoError(t, err)

	res2, err := ormBlock.GetUnbatchedL2Blocks(map[string]interface{}{})
	assert.NoError(t, err)
	assert.Equal(t, true, len(res2) == 1)

	exist, err = ormBlock.IsL2BlockExists(blockTrace.Header.Number.Uint64())
	assert.NoError(t, err)
	assert.Equal(t, true, exist)

	res, err = ormBlock.GetL2BlockTraces(map[string]interface{}{
		"hash": blockTrace.Header.Hash().String(),
	})
	assert.NoError(t, err)
	assert.Equal(t, true, len(res) == 1)

	// Compare trace
	data1, err := json.Marshal(res[0])
	assert.NoError(t, err)
	data2, err := json.Marshal(blockTrace)
	assert.NoError(t, err)
	// check trace
	assert.Equal(t, true, string(data1) == string(data2))
}

func testOrmL1Message(t *testing.T) {
	// Create db handler and reset db.
	factory, err := database.NewOrmFactory(dbConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(factory.GetDB().DB))

	expected := "expect hash"

	// Insert into db
	err = ormLayer1.SaveL1Messages(context.Background(), templateL1Message)
	assert.NoError(t, err)

	err = ormLayer1.UpdateLayer1Status(context.Background(), "msg_hash1", types.MsgConfirmed)
	assert.NoError(t, err)

	err = ormLayer1.UpdateLayer1Status(context.Background(), "msg_hash2", types.MsgSubmitted)
	assert.NoError(t, err)

	err = ormLayer1.UpdateLayer2Hash(context.Background(), "msg_hash2", expected)
	assert.NoError(t, err)

	result, err := ormLayer1.GetL1ProcessedQueueIndex()
	assert.NoError(t, err)
	assert.Equal(t, int64(1), result)

	height, err := ormLayer1.GetLayer1LatestWatchedHeight()
	assert.NoError(t, err)
	assert.Equal(t, int64(2), height)

	msg, err := ormLayer1.GetL1MessageByMsgHash("msg_hash2")
	assert.NoError(t, err)
	assert.Equal(t, types.MsgSubmitted, msg.Status)
}

func testOrmL2Message(t *testing.T) {
	// Create db handler and reset db.
	factory, err := database.NewOrmFactory(dbConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(factory.GetDB().DB))

	expected := "expect hash"

	// Insert into db
	err = ormLayer2.SaveL2Messages(context.Background(), templateL2Message)
	assert.NoError(t, err)

	err = ormLayer2.UpdateLayer2Status(context.Background(), "msg_hash1", types.MsgConfirmed)
	assert.NoError(t, err)

	err = ormLayer2.UpdateLayer2Status(context.Background(), "msg_hash2", types.MsgSubmitted)
	assert.NoError(t, err)

	err = ormLayer2.UpdateLayer1Hash(context.Background(), "msg_hash2", expected)
	assert.NoError(t, err)

	result, err := ormLayer2.GetL2ProcessedNonce()
	assert.NoError(t, err)
	assert.Equal(t, int64(1), result)

	height, err := ormLayer2.GetLayer2LatestWatchedHeight()
	assert.NoError(t, err)
	assert.Equal(t, int64(2), height)

	msg, err := ormLayer2.GetL2MessageByMsgHash("msg_hash2")
	assert.NoError(t, err)
	assert.Equal(t, types.MsgSubmitted, msg.Status)
	assert.Equal(t, msg.MsgHash, "msg_hash2")
}

// testOrmBlockBatch test rollup result table functions
func testOrmBlockBatch(t *testing.T) {
	// Create db handler and reset db.
	factory, err := database.NewOrmFactory(dbConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(factory.GetDB().DB))

	dbTx, err := factory.Beginx()
	assert.NoError(t, err)
	err = ormBatch.NewBatchInDBTx(dbTx, batchData1)
	assert.NoError(t, err)
	batchHash1 := batchData1.Hash().Hex()
	err = ormBlock.SetBatchHashForL2BlocksInDBTx(dbTx, []uint64{
		batchData1.Batch.Blocks[0].BlockNumber}, batchHash1)
	assert.NoError(t, err)
	err = ormBatch.NewBatchInDBTx(dbTx, batchData2)
	assert.NoError(t, err)
	batchHash2 := batchData2.Hash().Hex()
	err = ormBlock.SetBatchHashForL2BlocksInDBTx(dbTx, []uint64{
		batchData2.Batch.Blocks[0].BlockNumber,
		batchData2.Batch.Blocks[1].BlockNumber}, batchHash2)
	assert.NoError(t, err)
	err = dbTx.Commit()
	assert.NoError(t, err)

	batches, err := ormBatch.GetBlockBatches(map[string]interface{}{})
	assert.NoError(t, err)
	assert.Equal(t, int(2), len(batches))

	batcheHashes, err := ormBatch.GetPendingBatches(10)
	assert.NoError(t, err)
	assert.Equal(t, int(2), len(batcheHashes))
	assert.Equal(t, batchHash1, batcheHashes[0])
	assert.Equal(t, batchHash2, batcheHashes[1])

	err = ormBatch.UpdateCommitTxHashAndRollupStatus(context.Background(), batchHash1, "commit_tx_1", types.RollupCommitted)
	assert.NoError(t, err)

	batcheHashes, err = ormBatch.GetPendingBatches(10)
	assert.NoError(t, err)
	assert.Equal(t, int(1), len(batcheHashes))
	assert.Equal(t, batchHash2, batcheHashes[0])

	provingStatus, err := ormBatch.GetProvingStatusByHash(batchHash1)
	assert.NoError(t, err)
	assert.Equal(t, types.ProvingTaskUnassigned, provingStatus)
	err = ormBatch.UpdateProofByHash(context.Background(), batchHash1, []byte{1}, []byte{2}, 1200)
	assert.NoError(t, err)
	err = ormBatch.UpdateProvingStatus(batchHash1, types.ProvingTaskVerified)
	assert.NoError(t, err)
	provingStatus, err = ormBatch.GetProvingStatusByHash(batchHash1)
	assert.NoError(t, err)
	assert.Equal(t, types.ProvingTaskVerified, provingStatus)

	rollupStatus, err := ormBatch.GetRollupStatus(batchHash1)
	assert.NoError(t, err)
	assert.Equal(t, types.RollupCommitted, rollupStatus)
	err = ormBatch.UpdateFinalizeTxHashAndRollupStatus(context.Background(), batchHash1, "finalize_tx_1", types.RollupFinalized)
	assert.NoError(t, err)
	rollupStatus, err = ormBatch.GetRollupStatus(batchHash1)
	assert.NoError(t, err)
	assert.Equal(t, types.RollupFinalized, rollupStatus)
	result, err := ormBatch.GetLatestFinalizedBatch()
	assert.NoError(t, err)
	assert.Equal(t, batchHash1, result.Hash)

	status1, err := ormBatch.GetRollupStatus(batchHash1)
	assert.NoError(t, err)
	status2, err := ormBatch.GetRollupStatus(batchHash2)
	assert.NoError(t, err)
	assert.NotEqual(t, status1, status2)
	statues, err := ormBatch.GetRollupStatusByHashList([]string{batchHash1, batchHash2, batchHash1, batchHash2})
	assert.NoError(t, err)
	assert.Equal(t, statues[0], status1)
	assert.Equal(t, statues[1], status2)
	assert.Equal(t, statues[2], status1)
	assert.Equal(t, statues[3], status2)
	statues, err = ormBatch.GetRollupStatusByHashList([]string{batchHash2, batchHash1, batchHash2, batchHash1})
	assert.NoError(t, err)
	assert.Equal(t, statues[0], status2)
	assert.Equal(t, statues[1], status1)
	assert.Equal(t, statues[2], status2)
	assert.Equal(t, statues[3], status1)
}

// testOrmSessionInfo test rollup result table functions
func testOrmSessionInfo(t *testing.T) {
	// Create db handler and reset db.
	factory, err := database.NewOrmFactory(dbConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(factory.GetDB().DB))
	dbTx, err := factory.Beginx()
	assert.NoError(t, err)
	err = ormBatch.NewBatchInDBTx(dbTx, batchData1)
	batchHash := batchData1.Hash().Hex()
	assert.NoError(t, err)
	assert.NoError(t, ormBlock.SetBatchHashForL2BlocksInDBTx(dbTx, []uint64{
		batchData1.Batch.Blocks[0].BlockNumber}, batchHash))
	assert.NoError(t, dbTx.Commit())
	assert.NoError(t, ormBatch.UpdateProvingStatus(batchHash, types.ProvingTaskAssigned))

	// empty
	hashes, err := ormBatch.GetAssignedBatchHashes()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(hashes))
	sessionInfos, err := ormSession.GetSessionInfosByHashes(hashes)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(sessionInfos))

	sessionInfo := types.SessionInfo{
		ID: batchHash,
		Rollers: map[string]*types.RollerStatus{
			"0": {
				PublicKey: "0",
				Name:      "roller-0",
				Status:    types.RollerAssigned,
			},
		},
		StartTimestamp: time.Now().Unix()}

	// insert
	assert.NoError(t, ormSession.SetSessionInfo(&sessionInfo))
	sessionInfos, err = ormSession.GetSessionInfosByHashes(hashes)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(sessionInfos))
	assert.Equal(t, sessionInfo, *sessionInfos[0])

	// update
	sessionInfo.Rollers["0"].Status = types.RollerProofValid
	assert.NoError(t, ormSession.SetSessionInfo(&sessionInfo))
	sessionInfos, err = ormSession.GetSessionInfosByHashes(hashes)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(sessionInfos))
	assert.Equal(t, sessionInfo, *sessionInfos[0])

	// delete
	assert.NoError(t, ormBatch.UpdateProvingStatus(batchHash, types.ProvingTaskVerified))
	hashes, err = ormBatch.GetAssignedBatchHashes()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(hashes))
	sessionInfos, err = ormSession.GetSessionInfosByHashes(hashes)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(sessionInfos))
}
