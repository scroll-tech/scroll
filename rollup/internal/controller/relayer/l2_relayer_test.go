package relayer

import (
	"context"
	"errors"
	"math/big"
	"net/http"
	"strings"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/gin-gonic/gin"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"scroll-tech/common/database"
	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
	"scroll-tech/common/utils"

	"scroll-tech/database/migrate"

	"scroll-tech/rollup/internal/controller/sender"
	"scroll-tech/rollup/internal/orm"
)

func setupL2RelayerDB(t *testing.T) *gorm.DB {
	db, err := database.InitDB(cfg.DBConfig)
	assert.NoError(t, err)
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))
	return db
}

func testCreateNewRelayer(t *testing.T) {
	db := setupL2RelayerDB(t)
	defer database.CloseDB(db)
	relayer, err := NewLayer2Relayer(context.Background(), l2Cli, db, cfg.L2Config.RelayerConfig, false, nil)
	assert.NoError(t, err)
	assert.NotNil(t, relayer)
}

func testL2RelayerProcessPendingBatches(t *testing.T) {
	db := setupL2RelayerDB(t)
	defer database.CloseDB(db)

	l2Cfg := cfg.L2Config
	relayer, err := NewLayer2Relayer(context.Background(), l2Cli, db, l2Cfg.RelayerConfig, false, nil)
	assert.NoError(t, err)

	l2BlockOrm := orm.NewL2Block(db)
	err = l2BlockOrm.InsertL2Blocks(context.Background(), []*types.WrappedBlock{wrappedBlock1, wrappedBlock2})
	assert.NoError(t, err)
	chunkOrm := orm.NewChunk(db)
	dbChunk1, err := chunkOrm.InsertChunk(context.Background(), chunk1)
	assert.NoError(t, err)
	dbChunk2, err := chunkOrm.InsertChunk(context.Background(), chunk2)
	assert.NoError(t, err)
	batchMeta := &types.BatchMeta{
		StartChunkIndex: 0,
		StartChunkHash:  dbChunk1.Hash,
		EndChunkIndex:   1,
		EndChunkHash:    dbChunk2.Hash,
	}
	batchOrm := orm.NewBatch(db)
	batch, err := batchOrm.InsertBatch(context.Background(), []*types.Chunk{chunk1, chunk2}, batchMeta)
	assert.NoError(t, err)

	relayer.ProcessPendingBatches()

	statuses, err := batchOrm.GetRollupStatusByHashList(context.Background(), []string{batch.Hash})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(statuses))
	assert.Equal(t, types.RollupCommitting, statuses[0])
}

func testL2RelayerProcessCommittedBatches(t *testing.T) {
	db := setupL2RelayerDB(t)
	defer database.CloseDB(db)

	l2Cfg := cfg.L2Config
	relayer, err := NewLayer2Relayer(context.Background(), l2Cli, db, l2Cfg.RelayerConfig, false, nil)
	assert.NoError(t, err)
	batchMeta := &types.BatchMeta{
		StartChunkIndex: 0,
		StartChunkHash:  chunkHash1.Hex(),
		EndChunkIndex:   1,
		EndChunkHash:    chunkHash2.Hex(),
	}
	batchOrm := orm.NewBatch(db)
	batch, err := batchOrm.InsertBatch(context.Background(), []*types.Chunk{chunk1, chunk2}, batchMeta)
	assert.NoError(t, err)

	err = batchOrm.UpdateRollupStatus(context.Background(), batch.Hash, types.RollupCommitted)
	assert.NoError(t, err)

	err = batchOrm.UpdateProvingStatus(context.Background(), batch.Hash, types.ProvingTaskVerified)
	assert.NoError(t, err)

	relayer.ProcessCommittedBatches()

	statuses, err := batchOrm.GetRollupStatusByHashList(context.Background(), []string{batch.Hash})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(statuses))
	// no valid proof, rollup status remains the same
	assert.Equal(t, types.RollupCommitted, statuses[0])

	proof := &message.BatchProof{
		Proof: []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
	}
	err = batchOrm.UpdateProofByHash(context.Background(), batch.Hash, proof, 100)
	assert.NoError(t, err)

	relayer.ProcessCommittedBatches()
	statuses, err = batchOrm.GetRollupStatusByHashList(context.Background(), []string{batch.Hash})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(statuses))
	assert.Equal(t, types.RollupFinalizing, statuses[0])
}

func testL2RelayerCommitConfirm(t *testing.T) {
	db := setupL2RelayerDB(t)
	defer database.CloseDB(db)

	// Create and set up the Layer2 Relayer.
	l2Cfg := cfg.L2Config
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l2Relayer, err := NewLayer2Relayer(ctx, l2Cli, db, l2Cfg.RelayerConfig, false, nil)
	assert.NoError(t, err)

	// Simulate message confirmations.
	processingKeys := []string{"committed-1", "committed-2"}
	isSuccessful := []bool{true, false}

	batchOrm := orm.NewBatch(db)
	batchHashes := make([]string, len(processingKeys))
	for i := range batchHashes {
		batchMeta := &types.BatchMeta{
			StartChunkIndex: 0,
			StartChunkHash:  chunkHash1.Hex(),
			EndChunkIndex:   1,
			EndChunkHash:    chunkHash2.Hex(),
		}
		batch, err := batchOrm.InsertBatch(context.Background(), []*types.Chunk{chunk1, chunk2}, batchMeta)
		assert.NoError(t, err)
		batchHashes[i] = batch.Hash
	}

	for i, key := range processingKeys {
		l2Relayer.processingCommitment.Store(key, batchHashes[i])
		l2Relayer.commitSender.SendConfirmation(&sender.Confirmation{
			ID:           key,
			IsSuccessful: isSuccessful[i],
			TxHash:       common.HexToHash("0x123456789abcdef"),
		})
	}

	// Check the database for the updated status using TryTimes.
	ok := utils.TryTimes(5, func() bool {
		expectedStatuses := []types.RollupStatus{
			types.RollupCommitted,
			types.RollupCommitFailed,
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

func testL2RelayerFinalizeConfirm(t *testing.T) {
	db := setupL2RelayerDB(t)
	defer database.CloseDB(db)

	// Create and set up the Layer2 Relayer.
	l2Cfg := cfg.L2Config
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l2Relayer, err := NewLayer2Relayer(ctx, l2Cli, db, l2Cfg.RelayerConfig, false, nil)
	assert.NoError(t, err)

	// Simulate message confirmations.
	processingKeys := []string{"finalized-1", "finalized-2"}
	isSuccessful := []bool{true, false}

	batchOrm := orm.NewBatch(db)
	batchHashes := make([]string, len(processingKeys))
	for i := range batchHashes {
		batchMeta := &types.BatchMeta{
			StartChunkIndex: 0,
			StartChunkHash:  chunkHash1.Hex(),
			EndChunkIndex:   1,
			EndChunkHash:    chunkHash2.Hex(),
		}
		batch, err := batchOrm.InsertBatch(context.Background(), []*types.Chunk{chunk1, chunk2}, batchMeta)
		assert.NoError(t, err)
		batchHashes[i] = batch.Hash
	}

	for i, key := range processingKeys {
		l2Relayer.processingFinalization.Store(key, batchHashes[i])
		l2Relayer.finalizeSender.SendConfirmation(&sender.Confirmation{
			ID:           key,
			IsSuccessful: isSuccessful[i],
			TxHash:       common.HexToHash("0x123456789abcdef"),
		})
	}

	// Check the database for the updated status using TryTimes.
	ok := utils.TryTimes(5, func() bool {
		expectedStatuses := []types.RollupStatus{
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
	defer database.CloseDB(db)

	batchMeta1 := &types.BatchMeta{
		StartChunkIndex: 0,
		StartChunkHash:  chunkHash1.Hex(),
		EndChunkIndex:   0,
		EndChunkHash:    chunkHash1.Hex(),
	}
	batchOrm := orm.NewBatch(db)
	batch1, err := batchOrm.InsertBatch(context.Background(), []*types.Chunk{chunk1}, batchMeta1)
	assert.NoError(t, err)

	batchMeta2 := &types.BatchMeta{
		StartChunkIndex: 1,
		StartChunkHash:  chunkHash2.Hex(),
		EndChunkIndex:   1,
		EndChunkHash:    chunkHash2.Hex(),
	}
	batch2, err := batchOrm.InsertBatch(context.Background(), []*types.Chunk{chunk2}, batchMeta2)
	assert.NoError(t, err)

	// Create and set up the Layer2 Relayer.
	l2Cfg := cfg.L2Config
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l2Relayer, err := NewLayer2Relayer(ctx, l2Cli, db, l2Cfg.RelayerConfig, false, nil)
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
	defer database.CloseDB(db)

	relayer, err := NewLayer2Relayer(context.Background(), l2Cli, db, cfg.L2Config.RelayerConfig, false, nil)
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
		patchGuard.ApplyMethodFunc(relayer.gasOracleSender, "SendTransaction", func(ID string, target *common.Address, value *big.Int, data []byte, fallbackGasLimit uint64) (hash common.Hash, err error) {
			return common.Hash{}, targetErr
		})
		relayer.ProcessGasPriceOracle()
	})

	patchGuard.ApplyMethodFunc(relayer.gasOracleSender, "SendTransaction", func(ID string, target *common.Address, value *big.Int, data []byte, fallbackGasLimit uint64) (hash common.Hash, err error) {
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

func mockChainMonitorServer(baseURL string) (*http.Server, error) {
	router := gin.New()
	r := router.Group("/v1")
	r.GET("/batch_status", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, struct {
			ErrCode int    `json:"errcode"`
			ErrMsg  string `json:"errmsg"`
			Data    bool   `json:"data"`
		}{
			ErrCode: 0,
			ErrMsg:  "",
			Data:    true,
		})
	})
	return utils.StartHTTPServer(strings.Split(baseURL, "//")[1], router)
}

func testGetBatchStatusByIndex(t *testing.T) {
	db := setupL2RelayerDB(t)
	defer database.CloseDB(db)

	cfg.L2Config.RelayerConfig.ChainMonitor.EnableChainMonitor = true
	relayer, err := NewLayer2Relayer(context.Background(), l2Cli, db, cfg.L2Config.RelayerConfig, false, nil)
	assert.NoError(t, err)
	assert.NotNil(t, relayer)

	status, err := relayer.getBatchStatusByIndex(1)
	assert.NoError(t, err)
	assert.Equal(t, true, status)
}
