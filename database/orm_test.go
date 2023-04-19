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
	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	etypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"

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
	wrappedBlock *types.WrappedBlock
	batchData1   *types.BatchData
	batchData2   *types.BatchData

	base       *docker.App
	ormBlock   orm.BlockTraceOrm
	ormLayer1  orm.L1MessageOrm
	ormLayer2  orm.L2MessageOrm
	ormBatch   orm.BlockBatchOrm
	ormSession orm.SessionInfoOrm
	ormTx      orm.ScrollTxOrm

	auth *bind.TransactOpts
)

func setupEnv(t *testing.T) error {
	// Start postgres docker container.
	base.RunDBImage(t)

	// Create db handler and reset db.
	factory, err := database.NewOrmFactory(base.DBConfig)
	assert.NoError(t, err)
	db := factory.GetDB()
	assert.NoError(t, migrate.ResetDB(db.DB))

	// Init several orm handles.
	ormBlock = orm.NewBlockTraceOrm(db)
	ormLayer1 = orm.NewL1MessageOrm(db)
	ormLayer2 = orm.NewL2MessageOrm(db)
	ormBatch = orm.NewBlockBatchOrm(db)
	ormSession = orm.NewSessionInfoOrm(db)
	ormTx = orm.NewScrollTxOrm(db)

	templateBlockTrace, err := os.ReadFile("../common/testdata/blockTrace_02.json")
	if err != nil {
		return err
	}
	// unmarshal blockTrace
	wrappedBlock = &types.WrappedBlock{}
	if err = json.Unmarshal(templateBlockTrace, wrappedBlock); err != nil {
		return err
	}

	parentBatch := &types.BlockBatch{
		Index: 1,
		Hash:  "0x0000000000000000000000000000000000000000",
	}
	batchData1 = types.NewBatchData(parentBatch, []*types.WrappedBlock{wrappedBlock}, nil)

	templateBlockTrace, err = os.ReadFile("../common/testdata/blockTrace_03.json")
	if err != nil {
		return err
	}
	// unmarshal blockTrace
	wrappedBlock2 := &types.WrappedBlock{}
	if err = json.Unmarshal(templateBlockTrace, wrappedBlock2); err != nil {
		return err
	}
	parentBatch2 := &types.BlockBatch{
		Index: batchData1.Batch.BatchIndex,
		Hash:  batchData1.Hash().Hex(),
	}
	batchData2 = types.NewBatchData(parentBatch2, []*types.WrappedBlock{wrappedBlock2}, nil)

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

	sk, err := crypto.ToECDSA(common.FromHex("1212121212121212121212121212121212121212121212121212121212121212"))
	assert.NoError(t, err)
	auth, err = bind.NewKeyedTransactorWithChainID(sk, big.NewInt(1))
	assert.NoError(t, err)

	fmt.Printf("batchhash1 = %x\n", batchData1.Hash())
	fmt.Printf("batchhash2 = %x\n", batchData2.Hash())
	return nil
}

// TestOrmFactory run several test cases.
func TestOrmFactory(t *testing.T) {
	base = docker.NewDockerApp()
	defer func() {
		base.Free()
	}()
	if err := setupEnv(t); err != nil {
		t.Fatal(err)
	}

	t.Run("testOrmBlockTraces", testOrmBlockTraces)

	t.Run("testOrmL1Message", testOrmL1Message)

	t.Run("testOrmL2Message", testOrmL2Message)

	t.Run("testOrmBlockBatch", testOrmBlockBatch)

	t.Run("testOrmSessionInfo", testOrmSessionInfo)

	// test OrmTx interface.
	t.Run("testTxOrm", testTxOrmSaveTxAndGetTxByHash)
	t.Run("testTxOrmGetL1TxMessages", testTxOrmGetL1TxMessages)
	t.Run("testTxOrmGetL2TxMessages", testTxOrmGetL2TxMessages)
	t.Run("testTxOrmGetBlockBatchTxMessages", testTxOrmGetBlockBatchTxMessages)
}

func testOrmBlockTraces(t *testing.T) {
	// Create db handler and reset db.
	factory, err := database.NewOrmFactory(base.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(factory.GetDB().DB))

	res, err := ormBlock.GetL2WrappedBlocks(map[string]interface{}{})
	assert.NoError(t, err)
	assert.Equal(t, true, len(res) == 0)

	exist, err := ormBlock.IsL2BlockExists(wrappedBlock.Header.Number.Uint64())
	assert.NoError(t, err)
	assert.Equal(t, false, exist)

	// Insert into db
	assert.NoError(t, ormBlock.InsertWrappedBlocks([]*types.WrappedBlock{wrappedBlock}))

	res2, err := ormBlock.GetUnbatchedL2Blocks(map[string]interface{}{})
	assert.NoError(t, err)
	assert.Equal(t, true, len(res2) == 1)

	exist, err = ormBlock.IsL2BlockExists(wrappedBlock.Header.Number.Uint64())
	assert.NoError(t, err)
	assert.Equal(t, true, exist)

	res, err = ormBlock.GetL2WrappedBlocks(map[string]interface{}{
		"hash": wrappedBlock.Header.Hash().String(),
	})
	assert.NoError(t, err)
	assert.Equal(t, true, len(res) == 1)

	// Compare trace
	data1, err := json.Marshal(res[0])
	assert.NoError(t, err)
	data2, err := json.Marshal(wrappedBlock)
	assert.NoError(t, err)
	// check trace
	assert.Equal(t, true, string(data1) == string(data2))
}

func testOrmL1Message(t *testing.T) {
	// Create db handler and reset db.
	factory, err := database.NewOrmFactory(base.DBConfig)
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
	factory, err := database.NewOrmFactory(base.DBConfig)
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
	factory, err := database.NewOrmFactory(base.DBConfig)
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
	factory, err := database.NewOrmFactory(base.DBConfig)
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

func mockTx(auth *bind.TransactOpts) (*etypes.Transaction, error) {
	if auth.Nonce == nil {
		auth.Nonce = big.NewInt(0)
	} else {
		auth.Nonce.Add(auth.Nonce, big.NewInt(1))
	}

	tx := etypes.NewTx(&etypes.LegacyTx{
		Nonce:    auth.Nonce.Uint64(),
		To:       &auth.From,
		Value:    big.NewInt(0),
		Gas:      500000,
		GasPrice: big.NewInt(500000),
		Data:     common.Hex2Bytes("1212121212121212121212121212121212121212121212121212121212121212"),
	})

	return auth.Signer(auth.From, tx)
}

func testTxOrmSaveTxAndGetTxByHash(t *testing.T) {
	// Create db handler and reset db.
	factory, err := database.NewOrmFactory(base.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(factory.GetDB().DB))

	tx, err := mockTx(auth)
	assert.NoError(t, err)

	signedTx, err := auth.Signer(auth.From, tx)
	assert.NoError(t, err)

	err = ormTx.SaveScrollTx("1", auth.From.String(), types.L1toL2MessageTx, signedTx, "")
	assert.Nil(t, err)

	// Update tx message by id.
	err = ormTx.SetScrollTxConfirmedByID("1", signedTx.Hash().String())
	assert.NoError(t, err)

	savedTx, err := ormTx.GetTxByID("1")
	assert.NoError(t, err)

	assert.Equal(t, signedTx.Hash().String(), savedTx.TxHash.String)
	assert.Equal(t, auth.From.String(), savedTx.Sender.String)
	assert.Equal(t, auth.Nonce.Int64(), savedTx.Nonce.Int64)
	assert.Equal(t, []byte{}, savedTx.Data)
}

func testTxOrmGetL1TxMessages(t *testing.T) {
	// Create db handler and reset db.
	factory, err := database.NewOrmFactory(base.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(factory.GetDB().DB))

	signedTx, err := mockTx(auth)
	assert.NoError(t, err)
	err = ormTx.SaveScrollTx(templateL1Message[0].MsgHash, auth.From.String(), types.L1toL2MessageTx, signedTx, "")
	assert.Nil(t, err)

	signedTx, err = mockTx(auth)
	assert.NoError(t, err)
	err = ormTx.SaveScrollTx("3", auth.From.String(), types.L1toL2MessageTx, signedTx, "")
	assert.Nil(t, err)

	// Insert into db
	err = ormLayer1.SaveL1Messages(context.Background(), templateL1Message)
	assert.NoError(t, err)

	for _, msg := range templateL1Message {
		err = ormLayer1.UpdateLayer1Status(context.Background(), msg.MsgHash, types.MsgSubmitted)
		assert.NoError(t, err)
	}

	txMsgs, err := ormTx.GetL1TxMessages(
		map[string]interface{}{"status": types.MsgSubmitted},
		fmt.Sprintf("AND queue_index > %d", 0),
		fmt.Sprintf("ORDER BY queue_index ASC LIMIT %d", 10),
	)
	assert.NoError(t, err)
	assert.Equal(t, len(templateL1Message), len(txMsgs))
	// The first field is full.
	assert.Equal(t, templateL1Message[0].MsgHash, txMsgs[0].ID)
	// The second field is empty.
	assert.Equal(t, false, txMsgs[1].TxHash.Valid)
	assert.Equal(t, templateL1Message[1].MsgHash, txMsgs[1].ID)
}

func testTxOrmGetL2TxMessages(t *testing.T) {
	// Create db handler and reset db.
	factory, err := database.NewOrmFactory(base.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(factory.GetDB().DB))

	signedTx, err := mockTx(auth)
	assert.NoError(t, err)
	err = ormTx.SaveScrollTx(templateL1Message[0].MsgHash, auth.From.String(), types.L2toL1MessageTx, signedTx, "")
	assert.Nil(t, err)

	// Insert into db
	err = ormLayer2.SaveL2Messages(context.Background(), templateL2Message)
	assert.NoError(t, err)

	for _, msg := range templateL2Message {
		err = ormLayer2.UpdateLayer2Status(context.Background(), msg.MsgHash, types.MsgSubmitted)
		assert.NoError(t, err)
	}

	txMsgs, err := ormTx.GetL2TxMessages(
		map[string]interface{}{"status": types.MsgSubmitted},
		fmt.Sprintf("AND nonce > %d", 0),
		fmt.Sprintf("ORDER BY nonce ASC LIMIT %d", 10),
	)
	assert.NoError(t, err)
	assert.Equal(t, len(templateL2Message), len(txMsgs))
	assert.Equal(t, templateL2Message[0].MsgHash, txMsgs[0].ID)
	assert.Equal(t, false, txMsgs[1].TxHash.Valid)
	assert.Equal(t, templateL2Message[1].MsgHash, txMsgs[1].ID)
}

func testTxOrmGetBlockBatchTxMessages(t *testing.T) {
	// Create db handler and reset db.
	factory, err := database.NewOrmFactory(base.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(factory.GetDB().DB))

	dbTx, err := factory.Beginx()
	assert.NoError(t, err)
	for _, batch := range []*types.BatchData{batchData1, batchData2} {
		err = ormBatch.NewBatchInDBTx(dbTx, batch)
		assert.NoError(t, err)
	}
	assert.NoError(t, dbTx.Commit())

	signedTx, err := mockTx(auth)
	assert.NoError(t, err)
	extraData := "extra data"
	err = ormTx.SaveScrollTx(batchData1.Hash().String(), auth.From.String(), types.RollUpCommitTx, signedTx, extraData)
	assert.Nil(t, err)

	txMsgs, err := ormTx.GetBlockBatchTxMessages(
		map[string]interface{}{"rollup_status": types.RollupPending},
		fmt.Sprintf("AND index > %d", 0),
		fmt.Sprintf("ORDER BY index ASC LIMIT %d", 10),
	)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(txMsgs))
	assert.Equal(t, batchData1.Hash().String(), txMsgs[0].ID)
	assert.Equal(t, false, txMsgs[1].TxHash.Valid)
	assert.Equal(t, batchData2.Hash().String(), txMsgs[1].ID)
	assert.Equal(t, extraData, txMsgs[0].Note.String)
}
