package relayer

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/scroll-tech/go-ethereum/common"
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

func testL2RelayerProcessPendingBatches(t *testing.T) {
	db := setupL2RelayerDB(t)
	defer bridgeUtils.CloseDB(db)

	l2Cfg := cfg.L2Config
	relayer, err := NewLayer2Relayer(context.Background(), l2Cli, db, l2Cfg.RelayerConfig)
	assert.NoError(t, err)

	l2BlockOrm := orm.NewL2Block(db)
	err = l2BlockOrm.InsertL2Blocks(context.Background(), []*bridgeTypes.WrappedBlock{wrappedBlock1, wrappedBlock2})
	assert.NoError(t, err)
	chunkOrm := orm.NewChunk(db)
	dbChunk1, err := chunkOrm.InsertChunk(context.Background(), chunk1)
	assert.NoError(t, err)
	dbChunk2, err := chunkOrm.InsertChunk(context.Background(), chunk2)
	assert.NoError(t, err)
	batchOrm := orm.NewBatch(db)
	batch, err := batchOrm.InsertBatch(context.Background(), 0, 1, dbChunk1.Hash, dbChunk2.Hash, []*bridgeTypes.Chunk{chunk1, chunk2})
	assert.NoError(t, err)

	relayer.ProcessPendingBatches()

	statuses, err := batchOrm.GetRollupStatusByHashList(context.Background(), []string{batch.Hash})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(statuses))
	assert.Equal(t, types.RollupCommitting, statuses[0])
}

func testL2RelayerProcessCommittedBatches(t *testing.T) {
	db := setupL2RelayerDB(t)
	defer bridgeUtils.CloseDB(db)

	l2Cfg := cfg.L2Config
	relayer, err := NewLayer2Relayer(context.Background(), l2Cli, db, l2Cfg.RelayerConfig)
	assert.NoError(t, err)
	batchOrm := orm.NewBatch(db)
	batch, err := batchOrm.InsertBatch(context.Background(), 0, 1, chunkHash1.Hex(), chunkHash2.Hex(), []*bridgeTypes.Chunk{chunk1, chunk2})
	assert.NoError(t, err)

	err = batchOrm.UpdateRollupStatus(context.Background(), batch.Hash, types.RollupCommitted)
	assert.NoError(t, err)

	err = batchOrm.UpdateProvingStatus(context.Background(), batch.Hash, types.ProvingTaskVerified)
	assert.NoError(t, err)

	relayer.ProcessCommittedBatches()

	statuses, err := batchOrm.GetRollupStatusByHashList(context.Background(), []string{batch.Hash})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(statuses))
	assert.Equal(t, types.RollupFinalizationSkipped, statuses[0])

	err = batchOrm.UpdateRollupStatus(context.Background(), batch.Hash, types.RollupCommitted)
	assert.NoError(t, err)
	proof := &message.AggProof{
		Proof:     []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
		FinalPair: []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
	}
	err = batchOrm.UpdateProofByHash(context.Background(), batch.Hash, proof, 100)
	assert.NoError(t, err)

	relayer.ProcessCommittedBatches()
	statuses, err = batchOrm.GetRollupStatusByHashList(context.Background(), []string{batch.Hash})
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

	batchOrm := orm.NewBatch(db)
	createBatch := func(rollupStatus types.RollupStatus, provingStatus types.ProvingStatus) string {
		batch, err := batchOrm.InsertBatch(context.Background(), 0, 1, chunkHash1.Hex(), chunkHash2.Hex(), []*bridgeTypes.Chunk{chunk1, chunk2})
		assert.NoError(t, err)

		err = batchOrm.UpdateRollupStatus(context.Background(), batch.Hash, rollupStatus)
		assert.NoError(t, err)

		proof := &message.AggProof{
			Proof:     []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
			FinalPair: []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
		}
		err = batchOrm.UpdateProofByHash(context.Background(), batch.Hash, proof, 100)
		assert.NoError(t, err)
		err = batchOrm.UpdateProvingStatus(context.Background(), batch.Hash, provingStatus)
		assert.NoError(t, err)
		return batch.Hash
	}

	skipped := []string{
		createBatch(types.RollupCommitted, types.ProvingTaskSkipped),
		createBatch(types.RollupCommitted, types.ProvingTaskFailed),
	}

	notSkipped := []string{
		createBatch(types.RollupPending, types.ProvingTaskSkipped),
		createBatch(types.RollupCommitting, types.ProvingTaskSkipped),
		createBatch(types.RollupFinalizing, types.ProvingTaskSkipped),
		createBatch(types.RollupFinalized, types.ProvingTaskSkipped),
		createBatch(types.RollupPending, types.ProvingTaskFailed),
		createBatch(types.RollupCommitting, types.ProvingTaskFailed),
		createBatch(types.RollupFinalizing, types.ProvingTaskFailed),
		createBatch(types.RollupFinalized, types.ProvingTaskFailed),
		createBatch(types.RollupCommitted, types.ProvingTaskVerified),
	}

	relayer.ProcessCommittedBatches()

	for _, id := range skipped {
		statuses, err := batchOrm.GetRollupStatusByHashList(context.Background(), []string{id})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(statuses))
		assert.Equal(t, types.RollupFinalizationSkipped, statuses[0])
	}

	for _, id := range notSkipped {
		statuses, err := batchOrm.GetRollupStatusByHashList(context.Background(), []string{id})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(statuses))
		assert.NotEqual(t, types.RollupFinalizationSkipped, statuses[0])
	}
}

func testL2RelayerRollupConfirm(t *testing.T) {
	db := setupL2RelayerDB(t)
	defer bridgeUtils.CloseDB(db)

	// Create and set up the Layer2 Relayer.
	l2Cfg := cfg.L2Config
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l2Relayer, err := NewLayer2Relayer(ctx, l2Cli, db, l2Cfg.RelayerConfig)
	assert.NoError(t, err)

	// Simulate message confirmations.
	processingKeys := []string{"committed-1", "committed-2", "finalized-1", "finalized-2"}
	isSuccessful := []bool{true, false, true, false}

	batchOrm := orm.NewBatch(db)
	batchHashes := make([]string, len(processingKeys))
	for i := range batchHashes {
		batch, err := batchOrm.InsertBatch(context.Background(), 0, 1, chunkHash1.Hex(), chunkHash2.Hex(), []*bridgeTypes.Chunk{chunk1, chunk2})
		assert.NoError(t, err)
		batchHashes[i] = batch.Hash
	}

	for i, key := range processingKeys[:2] {
		l2Relayer.processingCommitment.Store(key, batchHashes[i])
		l2Relayer.rollupSender.SendConfirmation(&sender.Confirmation{
			ID:           key,
			IsSuccessful: isSuccessful[i],
			TxHash:       common.HexToHash("0x123456789abcdef"),
		})
	}

	for i, key := range processingKeys[2:] {
		l2Relayer.processingFinalization.Store(key, batchHashes[i+2])
		l2Relayer.rollupSender.SendConfirmation(&sender.Confirmation{
			ID:           key,
			IsSuccessful: isSuccessful[i+2],
			TxHash:       common.HexToHash("0x123456789abcdef"),
		})
	}

	// Check the database for the updated status using TryTimes.
	ok := utils.TryTimes(5, func() bool {
		expectedStatuses := []types.RollupStatus{
			types.RollupCommitted,
			types.RollupCommitFailed,
			types.RollupFinalized,
			types.RollupFinalizeFailed,
		}

		for i, batchHash := range batchHashes {
			batchInDB, err := batchOrm.GetBatches(context.Background(), map[string]interface{}{"hash": batchHash}, nil, 0)
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

	batchOrm := orm.NewBatch(db)
	batch1, err := batchOrm.InsertBatch(context.Background(), 0, 0, chunkHash1.Hex(), chunkHash1.Hex(), []*bridgeTypes.Chunk{chunk1})
	assert.NoError(t, err)

	batch2, err := batchOrm.InsertBatch(context.Background(), 1, 1, chunkHash2.Hex(), chunkHash2.Hex(), []*bridgeTypes.Chunk{chunk2})
	assert.NoError(t, err)

	// Create and set up the Layer2 Relayer.
	l2Cfg := cfg.L2Config
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l2Relayer, err := NewLayer2Relayer(ctx, l2Cli, db, l2Cfg.RelayerConfig)
	assert.NoError(t, err)

	// Simulate message confirmations.
	type BatchConfirmation struct {
		batchHash    string
		isSuccessful bool
	}

	confirmations := []BatchConfirmation{
		{batchHash: batch1.Hash, isSuccessful: true},
		{batchHash: batch2.Hash, isSuccessful: false},
	}

	for _, confirmation := range confirmations {
		l2Relayer.gasOracleSender.SendConfirmation(&sender.Confirmation{
			ID:           confirmation.batchHash,
			IsSuccessful: confirmation.isSuccessful,
		})
	}
	// Check the database for the updated status using TryTimes.
	ok := utils.TryTimes(5, func() bool {
		expectedStatuses := []types.GasOracleStatus{types.GasOracleImported, types.GasOracleFailed}
		for i, confirmation := range confirmations {
			gasOracle, err := batchOrm.GetBatches(context.Background(), map[string]interface{}{"hash": confirmation.batchHash}, nil, 0)
			if err != nil || len(gasOracle) != 1 || types.GasOracleStatus(gasOracle[0].OracleStatus) != expectedStatuses[i] {
				return false
			}
		}
		return true
	})
	assert.True(t, ok)
}

func testLayer2RelayerProcessGasPriceOracle(t *testing.T) {
	db := setupL2RelayerDB(t)
	defer bridgeUtils.CloseDB(db)

	relayer, err := NewLayer2Relayer(context.Background(), l2Cli, db, cfg.L2Config.RelayerConfig)
	assert.NoError(t, err)
	assert.NotNil(t, relayer)

	var batchOrm *orm.Batch
	convey.Convey("Failed to GetLatestBatch", t, func() {
		targetErr := errors.New("GetLatestBatch error")
		patchGuard := gomonkey.ApplyMethodFunc(batchOrm, "GetLatestBatch", func(context.Context) (*orm.Batch, error) {
			return nil, targetErr
		})
		defer patchGuard.Reset()
		relayer.ProcessGasPriceOracle()
	})

	patchGuard := gomonkey.ApplyMethodFunc(batchOrm, "GetLatestBatch", func(context.Context) (*orm.Batch, error) {
		batch := orm.Batch{
			OracleStatus: int16(types.GasOraclePending),
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
		patchGuard.ApplyMethodFunc(batchOrm, "UpdateL2GasOracleStatusAndOracleTxHash", func(ctx context.Context, hash string, status types.GasOracleStatus, txHash string) error {
			return targetErr
		})
		relayer.ProcessGasPriceOracle()
	})

	patchGuard.ApplyMethodFunc(batchOrm, "UpdateL2GasOracleStatusAndOracleTxHash", func(ctx context.Context, hash string, status types.GasOracleStatus, txHash string) error {
		return nil
	})
	relayer.ProcessGasPriceOracle()
}
