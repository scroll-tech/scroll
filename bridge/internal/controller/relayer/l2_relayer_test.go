package relayer

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"os"
	"strconv"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
	"scroll-tech/common/utils"

	"scroll-tech/bridge/internal/controller/sender"
	"scroll-tech/bridge/internal/orm"
	"scroll-tech/bridge/internal/orm/migrate"
	bridgeTypes "scroll-tech/bridge/internal/types"
	bridgeUtils "scroll-tech/bridge/internal/utils"
)

var (
	templateL2Message = []orm.L2Message{
		{
			Nonce:      1,
			Height:     1,
			Sender:     "0x596a746661dbed76a84556111c2872249b070e15",
			Value:      "100",
			Target:     "0x2c73620b223808297ea734d946813f0dd78eb8f7",
			Calldata:   "testdata",
			Layer2Hash: "hash0",
		},
	}
)

func setupL2RelayerDB(t *testing.T) *gorm.DB {
	db, err := bridgeUtils.InitDB(cfg.DBConfig)
	assert.NoError(t, err)
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))
	return db
}

func testCreateNewRelayer(t *testing.T) {
	db := setupL2RelayerDB(t)
	defer bridgeUtils.CloseDB(db)
	relayer, err := NewLayer2Relayer(context.Background(), l2Cli, db, cfg.L2Config.RelayerConfig)
	assert.NoError(t, err)
	assert.NotNil(t, relayer)
}

func testL2RelayerProcessSaveEvents(t *testing.T) {
	db := setupL2RelayerDB(t)
	defer bridgeUtils.CloseDB(db)
	l2Cfg := cfg.L2Config
	relayer, err := NewLayer2Relayer(context.Background(), l2Cli, db, l2Cfg.RelayerConfig)
	assert.NoError(t, err)

	l2MessageOrm := orm.NewL2Message(db)
	err = l2MessageOrm.SaveL2Messages(context.Background(), templateL2Message)
	assert.NoError(t, err)

	traces := []*bridgeTypes.WrappedBlock{
		{
			Header: &gethTypes.Header{
				Number: big.NewInt(int64(templateL2Message[0].Height)),
			},
			Transactions:     nil,
			WithdrawTrieRoot: common.Hash{},
		},
		{
			Header: &gethTypes.Header{
				Number: big.NewInt(int64(templateL2Message[0].Height + 1)),
			},
			Transactions:     nil,
			WithdrawTrieRoot: common.Hash{},
		},
	}

	blockTraceOrm := orm.NewBlockTrace(db)
	assert.NoError(t, blockTraceOrm.InsertWrappedBlocks(traces))
	blockBatchOrm := orm.NewBlockBatch(db)
	parentBatch1 := &bridgeTypes.BatchInfo{
		Index:     0,
		Hash:      common.Hash{}.Hex(),
		StateRoot: common.Hash{}.Hex(),
	}
	batchData1 := bridgeTypes.NewBatchData(parentBatch1, []*bridgeTypes.WrappedBlock{wrappedBlock1}, nil)
	batchHash := batchData1.Hash().Hex()
	err = db.Transaction(func(tx *gorm.DB) error {
		rowsAffected, dbTxErr := blockBatchOrm.InsertBlockBatchByBatchData(tx, batchData1)
		if dbTxErr != nil {
			return dbTxErr
		}
		if rowsAffected != 1 {
			dbTxErr = errors.New("the InsertBlockBatchByBatchData affected row is not 1")
			return dbTxErr
		}
		dbTxErr = blockTraceOrm.UpdateChunkHashInClosedRange(tx, []uint64{1}, batchHash)
		if dbTxErr != nil {
			return dbTxErr
		}
		return nil
	})
	assert.NoError(t, err)

	err = blockBatchOrm.UpdateRollupStatus(context.Background(), batchHash, types.RollupFinalized)
	assert.NoError(t, err)

	relayer.ProcessSavedEvents()

	msg, err := l2MessageOrm.GetL2MessageByNonce(templateL2Message[0].Nonce)
	assert.NoError(t, err)
	assert.Equal(t, types.MsgSubmitted, types.MsgStatus(msg.Status))
}

func testL2RelayerProcessCommittedBatches(t *testing.T) {
	db := setupL2RelayerDB(t)
	defer bridgeUtils.CloseDB(db)

	l2Cfg := cfg.L2Config
	relayer, err := NewLayer2Relayer(context.Background(), l2Cli, db, l2Cfg.RelayerConfig)
	assert.NoError(t, err)

	parentBatch1 := &bridgeTypes.BatchInfo{
		Index:     0,
		Hash:      common.Hash{}.Hex(),
		StateRoot: common.Hash{}.Hex(),
	}

	blockBatchOrm := orm.NewBlockBatch(db)
	batchData1 := bridgeTypes.NewBatchData(parentBatch1, []*bridgeTypes.WrappedBlock{wrappedBlock1}, nil)
	batchHash := batchData1.Hash().Hex()
	err = db.Transaction(func(tx *gorm.DB) error {
		rowsAffected, dbTxErr := blockBatchOrm.InsertBlockBatchByBatchData(tx, batchData1)
		if dbTxErr != nil {
			return dbTxErr
		}
		if rowsAffected != 1 {
			dbTxErr = errors.New("the InsertBlockBatchByBatchData affected row is not 1")
			return dbTxErr
		}
		return nil
	})
	assert.NoError(t, err)

	err = blockBatchOrm.UpdateRollupStatus(context.Background(), batchHash, types.RollupCommitted)
	assert.NoError(t, err)

	proof := &message.AggProof{
		Proof:     []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
		FinalPair: []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
	}
	err = blockBatchOrm.UpdateProofByHash(context.Background(), batchHash, proof, 100)
	assert.NoError(t, err)
	err = blockBatchOrm.UpdateProvingStatus(batchHash, types.ProvingTaskVerified)
	assert.NoError(t, err)

	relayer.ProcessCommittedBatches()

	statuses, err := blockBatchOrm.GetRollupStatusByHashList([]string{batchHash})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(statuses))
	assert.Equal(t, types.RollupFinalizing, statuses[0])
}

func testL2RelayerSkipBatches(t *testing.T) {
	db := setupL2RelayerDB(t)
	defer bridgeUtils.CloseDB(db)

	l2Cfg := cfg.L2Config
	relayer, err := NewLayer2Relayer(context.Background(), l2Cli, db, l2Cfg.RelayerConfig)
	assert.NoError(t, err)

	blockBatchOrm := orm.NewBlockBatch(db)
	createBatch := func(rollupStatus types.RollupStatus, provingStatus types.ProvingStatus, index uint64) string {
		batchData := genBatchData(t, index)
		err = db.Transaction(func(tx *gorm.DB) error {
			rowsAffected, dbTxErr := blockBatchOrm.InsertBlockBatchByBatchData(tx, batchData)
			if dbTxErr != nil {
				return dbTxErr
			}
			if rowsAffected != 1 {
				dbTxErr = errors.New("the InsertBlockBatchByBatchData affected row is not 1")
				return dbTxErr
			}
			return nil
		})
		assert.NoError(t, err)

		batchHash := batchData.Hash().Hex()
		err = blockBatchOrm.UpdateRollupStatus(context.Background(), batchHash, rollupStatus)
		assert.NoError(t, err)

		proof := &message.AggProof{
			Proof:     []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
			FinalPair: []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
		}
		err = blockBatchOrm.UpdateProofByHash(context.Background(), batchHash, proof, 100)
		assert.NoError(t, err)
		err = blockBatchOrm.UpdateProvingStatus(batchHash, provingStatus)
		assert.NoError(t, err)
		return batchHash
	}

	skipped := []string{
		createBatch(types.RollupCommitted, types.ProvingTaskSkipped, 1),
		createBatch(types.RollupCommitted, types.ProvingTaskFailed, 2),
	}

	notSkipped := []string{
		createBatch(types.RollupPending, types.ProvingTaskSkipped, 3),
		createBatch(types.RollupCommitting, types.ProvingTaskSkipped, 4),
		createBatch(types.RollupFinalizing, types.ProvingTaskSkipped, 5),
		createBatch(types.RollupFinalized, types.ProvingTaskSkipped, 6),
		createBatch(types.RollupPending, types.ProvingTaskFailed, 7),
		createBatch(types.RollupCommitting, types.ProvingTaskFailed, 8),
		createBatch(types.RollupFinalizing, types.ProvingTaskFailed, 9),
		createBatch(types.RollupFinalized, types.ProvingTaskFailed, 10),
		createBatch(types.RollupCommitted, types.ProvingTaskVerified, 11),
	}

	relayer.ProcessCommittedBatches()

	for _, id := range skipped {
		statuses, err := blockBatchOrm.GetRollupStatusByHashList([]string{id})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(statuses))
		assert.Equal(t, types.RollupFinalizationSkipped, statuses[0])
	}

	for _, id := range notSkipped {
		statuses, err := blockBatchOrm.GetRollupStatusByHashList([]string{id})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(statuses))
		assert.NotEqual(t, types.RollupFinalizationSkipped, statuses[0])
	}
}

func testL2RelayerMsgConfirm(t *testing.T) {
	db := setupL2RelayerDB(t)
	defer bridgeUtils.CloseDB(db)
	l2MessageOrm := orm.NewL2Message(db)
	insertL2Messages := []orm.L2Message{
		{MsgHash: "msg-1", Nonce: 0},
		{MsgHash: "msg-2", Nonce: 1},
	}
	err := l2MessageOrm.SaveL2Messages(context.Background(), insertL2Messages)
	assert.NoError(t, err)

	// Create and set up the Layer2 Relayer.
	l2Cfg := cfg.L2Config
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l2Relayer, err := NewLayer2Relayer(ctx, l2Cli, db, l2Cfg.RelayerConfig)
	assert.NoError(t, err)

	// Simulate message confirmations.
	l2Relayer.processingMessage.Store("msg-1", "msg-1")
	l2Relayer.messageSender.SendConfirmation(&sender.Confirmation{
		ID:           "msg-1",
		IsSuccessful: true,
	})
	l2Relayer.processingMessage.Store("msg-2", "msg-2")
	l2Relayer.messageSender.SendConfirmation(&sender.Confirmation{
		ID:           "msg-2",
		IsSuccessful: false,
	})

	// Check the database for the updated status using TryTimes.
	assert.True(t, utils.TryTimes(5, func() bool {
		fields1 := map[string]interface{}{"msg_hash": "msg-1"}
		msg1, err1 := l2MessageOrm.GetL2Messages(fields1, nil, 0)
		if len(msg1) != 1 {
			return false
		}
		fields2 := map[string]interface{}{"msg_hash": "msg-2"}
		msg2, err2 := l2MessageOrm.GetL2Messages(fields2, nil, 0)
		if len(msg2) != 1 {
			return false
		}
		return err1 == nil && types.MsgStatus(msg1[0].Status) == types.MsgConfirmed &&
			err2 == nil && types.MsgStatus(msg2[0].Status) == types.MsgRelayFailed
	}))
}

func testL2RelayerRollupConfirm(t *testing.T) {
	db := setupL2RelayerDB(t)
	defer bridgeUtils.CloseDB(db)

	// Insert test data.
	batches := make([]*bridgeTypes.BatchData, 6)
	for i := 0; i < 6; i++ {
		batches[i] = genBatchData(t, uint64(i))
	}

	blockBatchOrm := orm.NewBlockBatch(db)
	err := db.Transaction(func(tx *gorm.DB) error {
		for _, batch := range batches {
			rowsAffected, dbTxErr := blockBatchOrm.InsertBlockBatchByBatchData(tx, batch)
			if dbTxErr != nil {
				return dbTxErr
			}
			if rowsAffected != 1 {
				dbTxErr = errors.New("the InsertBlockBatchByBatchData affected row is not 1")
				return dbTxErr
			}
		}
		return nil
	})
	assert.NoError(t, err)

	// Create and set up the Layer2 Relayer.
	l2Cfg := cfg.L2Config
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l2Relayer, err := NewLayer2Relayer(ctx, l2Cli, db, l2Cfg.RelayerConfig)
	assert.NoError(t, err)

	// Simulate message confirmations.
	processingKeys := []string{"committed-1", "committed-2", "finalized-1", "finalized-2"}
	isSuccessful := []bool{true, false, true, false}

	for i, key := range processingKeys[:2] {
		batchHashes := []string{batches[i*2].Hash().Hex(), batches[i*2+1].Hash().Hex()}
		l2Relayer.processingBatchCommitment.Store(key, batchHashes)
		l2Relayer.messageSender.SendConfirmation(&sender.Confirmation{
			ID:           key,
			IsSuccessful: isSuccessful[i],
		})
	}

	for i, key := range processingKeys[2:] {
		batchHash := batches[i+4].Hash().Hex()
		l2Relayer.processingFinalization.Store(key, batchHash)
		l2Relayer.rollupSender.SendConfirmation(&sender.Confirmation{
			ID:           key,
			IsSuccessful: isSuccessful[i+2],
			TxHash:       common.HexToHash("0x56789abcdef1234"),
		})
	}

	// Check the database for the updated status using TryTimes.
	ok := utils.TryTimes(5, func() bool {
		expectedStatuses := []types.RollupStatus{
			types.RollupCommitted,
			types.RollupCommitted,
			types.RollupCommitFailed,
			types.RollupCommitFailed,
			types.RollupFinalized,
			types.RollupFinalizeFailed,
		}

		for i, batch := range batches[:6] {
			batchInDB, err := blockBatchOrm.GetBlockBatches(map[string]interface{}{"hash": batch.Hash().Hex()}, nil, 0)
			if err != nil || len(batchInDB) != 1 || types.RollupStatus(batchInDB[0].RollupStatus) != expectedStatuses[i] {
				return false
			}
		}
		return true
	})
	assert.True(t, ok)
}

func testL2RelayerGasOracleConfirm(t *testing.T) {
	db := setupL2RelayerDB(t)
	defer bridgeUtils.CloseDB(db)

	// Insert test data.
	batches := make([]*bridgeTypes.BatchData, 2)
	for i := 0; i < 2; i++ {
		batches[i] = genBatchData(t, uint64(i))
	}

	blockBatchOrm := orm.NewBlockBatch(db)
	err := db.Transaction(func(tx *gorm.DB) error {
		for _, batch := range batches {
			rowsAffected, dbTxErr := blockBatchOrm.InsertBlockBatchByBatchData(tx, batch)
			if dbTxErr != nil {
				return dbTxErr
			}
			if rowsAffected != 1 {
				dbTxErr = errors.New("the InsertBlockBatchByBatchData affected row is not 1")
				return dbTxErr
			}
		}
		return nil
	})
	assert.NoError(t, err)

	// Create and set up the Layer2 Relayer.
	l2Cfg := cfg.L2Config
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l2Relayer, err := NewLayer2Relayer(ctx, l2Cli, db, l2Cfg.RelayerConfig)
	assert.NoError(t, err)

	// Simulate message confirmations.
	isSuccessful := []bool{true, false}
	for i, batch := range batches {
		l2Relayer.gasOracleSender.SendConfirmation(&sender.Confirmation{
			ID:           batch.Hash().Hex(),
			IsSuccessful: isSuccessful[i],
		})
	}

	// Check the database for the updated status using TryTimes.
	ok := utils.TryTimes(5, func() bool {
		expectedStatuses := []types.GasOracleStatus{types.GasOracleImported, types.GasOracleFailed}
		for i, batch := range batches {
			gasOracle, err := blockBatchOrm.GetBlockBatches(map[string]interface{}{"hash": batch.Hash().Hex()}, nil, 0)
			if err != nil || len(gasOracle) != 1 || types.GasOracleStatus(gasOracle[0].OracleStatus) != expectedStatuses[i] {
				return false
			}
		}
		return true
	})
	assert.True(t, ok)
}

func genBatchData(t *testing.T, index uint64) *bridgeTypes.BatchData {
	templateBlockTrace, err := os.ReadFile("../../../testdata/blockTrace_02.json")
	assert.NoError(t, err)
	// unmarshal blockTrace
	wrappedBlock := &bridgeTypes.WrappedBlock{}
	err = json.Unmarshal(templateBlockTrace, wrappedBlock)
	assert.NoError(t, err)
	wrappedBlock.Header.ParentHash = common.HexToHash("0x" + strconv.FormatUint(index+1, 16))
	parentBatch := &bridgeTypes.BatchInfo{
		Index: index,
		Hash:  "0x0000000000000000000000000000000000000000",
	}
	return bridgeTypes.NewBatchData(parentBatch, []*bridgeTypes.WrappedBlock{wrappedBlock}, nil)
}

func testLayer2RelayerProcessGasPriceOracle(t *testing.T) {
	db := setupL2RelayerDB(t)
	defer bridgeUtils.CloseDB(db)

	relayer, err := NewLayer2Relayer(context.Background(), l2Cli, db, cfg.L2Config.RelayerConfig)
	assert.NoError(t, err)
	assert.NotNil(t, relayer)

	var blockBatchOrm *orm.BlockBatch
	convey.Convey("Failed to GetLatestBatch", t, func() {
		targetErr := errors.New("GetLatestBatch error")
		patchGuard := gomonkey.ApplyMethodFunc(blockBatchOrm, "GetLatestBatch", func() (*orm.BlockBatch, error) {
			return nil, targetErr
		})
		defer patchGuard.Reset()
		relayer.ProcessGasPriceOracle()
	})

	patchGuard := gomonkey.ApplyMethodFunc(blockBatchOrm, "GetLatestBatch", func() (*orm.BlockBatch, error) {
		batch := orm.BlockBatch{
			OracleStatus: int(types.GasOraclePending),
			Hash:         "0x0000000000000000000000000000000000000000",
		}
		return &batch, nil
	})
	defer patchGuard.Reset()

	convey.Convey("Failed to fetch SuggestGasPrice from l2geth", t, func() {
		targetErr := errors.New("SuggestGasPrice error")
		patchGuard.ApplyMethodFunc(relayer.l2Client, "SuggestGasPrice", func(ctx context.Context) (*big.Int, error) {
			return nil, targetErr
		})
		relayer.ProcessGasPriceOracle()
	})

	patchGuard.ApplyMethodFunc(relayer.l2Client, "SuggestGasPrice", func(ctx context.Context) (*big.Int, error) {
		return big.NewInt(100), nil
	})

	convey.Convey("Failed to pack setL2BaseFee", t, func() {
		targetErr := errors.New("setL2BaseFee error")
		patchGuard.ApplyMethodFunc(relayer.l2GasOracleABI, "Pack", func(name string, args ...interface{}) ([]byte, error) {
			return nil, targetErr
		})
		relayer.ProcessGasPriceOracle()
	})

	patchGuard.ApplyMethodFunc(relayer.l2GasOracleABI, "Pack", func(name string, args ...interface{}) ([]byte, error) {
		return nil, nil
	})

	convey.Convey("Failed to send setL2BaseFee tx to layer2", t, func() {
		targetErr := errors.New("failed to send setL2BaseFee tx to layer2 error")
		patchGuard.ApplyMethodFunc(relayer.gasOracleSender, "SendTransaction", func(ID string, target *common.Address, value *big.Int, data []byte, minGasLimit uint64) (hash common.Hash, err error) {
			return common.Hash{}, targetErr
		})
		relayer.ProcessGasPriceOracle()
	})

	patchGuard.ApplyMethodFunc(relayer.gasOracleSender, "SendTransaction", func(ID string, target *common.Address, value *big.Int, data []byte, minGasLimit uint64) (hash common.Hash, err error) {
		return common.HexToHash("0x56789abcdef1234"), nil
	})

	convey.Convey("UpdateGasOracleStatusAndOracleTxHash failed", t, func() {
		targetErr := errors.New("UpdateL2GasOracleStatusAndOracleTxHash error")
		patchGuard.ApplyMethodFunc(blockBatchOrm, "UpdateL2GasOracleStatusAndOracleTxHash", func(ctx context.Context, hash string, status types.GasOracleStatus, txHash string) error {
			return targetErr
		})
		relayer.ProcessGasPriceOracle()
	})

	patchGuard.ApplyMethodFunc(blockBatchOrm, "UpdateL2GasOracleStatusAndOracleTxHash", func(ctx context.Context, hash string, status types.GasOracleStatus, txHash string) error {
		return nil
	})
	relayer.ProcessGasPriceOracle()
}

func testLayer2RelayerSendCommitTx(t *testing.T) {
	db := setupL2RelayerDB(t)
	defer bridgeUtils.CloseDB(db)

	relayer, err := NewLayer2Relayer(context.Background(), l2Cli, db, cfg.L2Config.RelayerConfig)
	assert.NoError(t, err)
	assert.NotNil(t, relayer)

	var batchDataList []*bridgeTypes.BatchData
	convey.Convey("SendCommitTx receives empty batch", t, func() {
		err = relayer.SendCommitTx(batchDataList)
		assert.NoError(t, err)
	})

	parentBatch := &bridgeTypes.BatchInfo{
		Index: 0,
		Hash:  "0x0000000000000000000000000000000000000000",
	}

	traces := []*bridgeTypes.WrappedBlock{
		{
			Header: &gethTypes.Header{
				Number:     big.NewInt(1000),
				ParentHash: common.Hash{},
				Difficulty: big.NewInt(0),
				BaseFee:    big.NewInt(0),
			},
			Transactions:     nil,
			WithdrawTrieRoot: common.Hash{},
		},
	}

	blocks := []*bridgeTypes.WrappedBlock{traces[0]}
	tmpBatchData := bridgeTypes.NewBatchData(parentBatch, blocks, cfg.L2Config.BatchProposerConfig.PublicInputConfig)
	batchDataList = append(batchDataList, tmpBatchData)

	var s abi.ABI
	convey.Convey("Failed to pack commitBatches", t, func() {
		targetErr := errors.New("commitBatches error")
		patchGuard := gomonkey.ApplyMethodFunc(s, "Pack", func(name string, args ...interface{}) ([]byte, error) {
			return nil, targetErr
		})
		defer patchGuard.Reset()

		err = relayer.SendCommitTx(batchDataList)
		assert.EqualError(t, err, targetErr.Error())
	})

	patchGuard := gomonkey.ApplyMethodFunc(s, "Pack", func(name string, args ...interface{}) ([]byte, error) {
		return nil, nil
	})
	defer patchGuard.Reset()

	convey.Convey("Failed to send commitBatches tx to layer1", t, func() {
		targetErr := errors.New("SendTransaction failure")
		patchGuard.ApplyMethodFunc(relayer.rollupSender, "SendTransaction", func(ID string, target *common.Address, value *big.Int, data []byte, minGasLimit uint64) (hash common.Hash, err error) {
			return common.Hash{}, targetErr
		})
		err = relayer.SendCommitTx(batchDataList)
		assert.EqualError(t, err, targetErr.Error())
	})

	patchGuard.ApplyMethodFunc(relayer.rollupSender, "SendTransaction", func(ID string, target *common.Address, value *big.Int, data []byte, minGasLimit uint64) (hash common.Hash, err error) {
		return common.HexToHash("0x56789abcdef1234"), nil
	})

	var blockBatchOrm *orm.BlockBatch
	convey.Convey("UpdateCommitTxHashAndRollupStatus failed", t, func() {
		targetErr := errors.New("UpdateCommitTxHashAndRollupStatus failure")
		patchGuard.ApplyMethodFunc(blockBatchOrm, "UpdateCommitTxHashAndRollupStatus", func(ctx context.Context, hash string, commitTxHash string, status types.RollupStatus) error {
			return targetErr
		})
		err = relayer.SendCommitTx(batchDataList)
		assert.NoError(t, err)
	})

	patchGuard.ApplyMethodFunc(blockBatchOrm, "UpdateCommitTxHashAndRollupStatus", func(ctx context.Context, hash string, commitTxHash string, status types.RollupStatus) error {
		return nil
	})
	err = relayer.SendCommitTx(batchDataList)
	assert.NoError(t, err)
}
