package relayer

import (
	"context"
	"errors"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto/kzg4844"
	"github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"scroll-tech/common/database"
	"scroll-tech/common/types"
	"scroll-tech/common/utils"

	"scroll-tech/database/migrate"

	"scroll-tech/rollup/internal/controller/sender"
	"scroll-tech/rollup/internal/orm"
)

func setupL1RelayerDB(t *testing.T) *gorm.DB {
	db, err := database.InitDB(cfg.DBConfig)
	assert.NoError(t, err)
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))
	return db
}

// testCreateNewL1Relayer test create new relayer instance and stop
func testCreateNewL1Relayer(t *testing.T) {
	db := setupL1RelayerDB(t)
	defer database.CloseDB(db)
	relayer, err := NewLayer1Relayer(context.Background(), db, cfg.L2Config.RelayerConfig, ServiceTypeL1GasOracle, nil)
	assert.NoError(t, err)
	assert.NotNil(t, relayer)
	defer relayer.StopSenders()
}

func testL1RelayerGasOracleConfirm(t *testing.T) {
	db := setupL1RelayerDB(t)
	defer database.CloseDB(db)
	l1BlockOrm := orm.NewL1Block(db)

	l1Block := []orm.L1Block{
		{Hash: "gas-oracle-1", Number: 0, GasOracleStatus: int16(types.GasOraclePending)},
		{Hash: "gas-oracle-2", Number: 1, GasOracleStatus: int16(types.GasOraclePending)},
	}
	// Insert test data.
	assert.NoError(t, l1BlockOrm.InsertL1Blocks(context.Background(), l1Block))

	// Create and set up the Layer2 Relayer.
	l1Cfg := cfg.L1Config
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l1Relayer, err := NewLayer1Relayer(ctx, db, l1Cfg.RelayerConfig, ServiceTypeL1GasOracle, nil)
	assert.NoError(t, err)
	defer l1Relayer.StopSenders()

	// Simulate message confirmations.
	l1Relayer.gasOracleSender.SendConfirmation(&sender.Confirmation{
		ContextID:    "gas-oracle-1",
		IsSuccessful: true,
		SenderType:   types.SenderTypeL1GasOracle,
	})
	l1Relayer.gasOracleSender.SendConfirmation(&sender.Confirmation{
		ContextID:    "gas-oracle-2",
		IsSuccessful: false,
		SenderType:   types.SenderTypeL1GasOracle,
	})

	// Check the database for the updated status using TryTimes.
	ok := utils.TryTimes(5, func() bool {
		msg1, err1 := l1BlockOrm.GetL1Blocks(ctx, map[string]interface{}{"hash": "gas-oracle-1"})
		msg2, err2 := l1BlockOrm.GetL1Blocks(ctx, map[string]interface{}{"hash": "gas-oracle-2"})
		return err1 == nil && len(msg1) == 1 && types.GasOracleStatus(msg1[0].GasOracleStatus) == types.GasOracleImported &&
			err2 == nil && len(msg2) == 1 && types.GasOracleStatus(msg2[0].GasOracleStatus) == types.GasOracleImportedFailed
	})
	assert.True(t, ok)
}

func testL1RelayerProcessGasPriceOracle(t *testing.T) {
	db := setupL1RelayerDB(t)
	defer database.CloseDB(db)

	l1Cfg := cfg.L1Config
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l1Relayer, err := NewLayer1Relayer(ctx, db, l1Cfg.RelayerConfig, ServiceTypeL1GasOracle, nil)
	assert.NoError(t, err)
	assert.NotNil(t, l1Relayer)
	defer l1Relayer.StopSenders()

	var l1BlockOrm *orm.L1Block
	convey.Convey("GetLatestL1BlockHeight failure", t, func() {
		targetErr := errors.New("GetLatestL1BlockHeight error")
		patchGuard := gomonkey.ApplyMethodFunc(l1BlockOrm, "GetLatestL1BlockHeight", func(ctx context.Context) (uint64, error) {
			return 0, targetErr
		})
		defer patchGuard.Reset()
		l1Relayer.ProcessGasPriceOracle()
	})

	patchGuard := gomonkey.ApplyMethodFunc(l1BlockOrm, "GetLatestL1BlockHeight", func(ctx context.Context) (uint64, error) {
		return 100, nil
	})
	defer patchGuard.Reset()

	convey.Convey("GetL1Blocks failure", t, func() {
		targetErr := errors.New("GetL1Blocks error")
		patchGuard.ApplyMethodFunc(l1BlockOrm, "GetL1Blocks", func(ctx context.Context, fields map[string]interface{}) ([]orm.L1Block, error) {
			return nil, targetErr
		})
		l1Relayer.ProcessGasPriceOracle()
	})

	convey.Convey("Block not exist", t, func() {
		patchGuard.ApplyMethodFunc(l1BlockOrm, "GetL1Blocks", func(ctx context.Context, fields map[string]interface{}) ([]orm.L1Block, error) {
			tmpInfo := []orm.L1Block{
				{Hash: "gas-oracle-1", Number: 0},
				{Hash: "gas-oracle-2", Number: 1},
			}
			return tmpInfo, nil
		})
		l1Relayer.ProcessGasPriceOracle()
	})

	patchGuard.ApplyMethodFunc(l1BlockOrm, "GetL1Blocks", func(ctx context.Context, fields map[string]interface{}) ([]orm.L1Block, error) {
		tmpInfo := []orm.L1Block{
			{
				Hash:            "gas-oracle-1",
				Number:          0,
				GasOracleStatus: int16(types.GasOraclePending),
			},
		}
		return tmpInfo, nil
	})

	patchGuard.ApplyMethodFunc(l1Relayer.l1GasOracleABI, "Pack", func(name string, args ...interface{}) ([]byte, error) {
		return []byte("for test"), nil
	})

	convey.Convey("send transaction failure", t, func() {
		targetErr := errors.New("send transaction failure")
		patchGuard.ApplyMethodFunc(l1Relayer.gasOracleSender, "SendTransaction", func(string, *common.Address, []byte, *kzg4844.Blob, uint64) (hash common.Hash, err error) {
			return common.Hash{}, targetErr
		})
		l1Relayer.ProcessGasPriceOracle()
	})

	patchGuard.ApplyMethodFunc(l1Relayer.gasOracleSender, "SendTransaction", func(string, *common.Address, []byte, *kzg4844.Blob, uint64) (hash common.Hash, err error) {
		return common.Hash{}, nil
	})

	convey.Convey("UpdateL1GasOracleStatusAndOracleTxHash failure", t, func() {
		targetErr := errors.New("UpdateL1GasOracleStatusAndOracleTxHash failure")
		patchGuard.ApplyMethodFunc(l1BlockOrm, "UpdateL1GasOracleStatusAndOracleTxHash", func(context.Context, string, types.GasOracleStatus, string) error {
			return targetErr
		})
		l1Relayer.ProcessGasPriceOracle()
	})

	patchGuard.ApplyMethodFunc(l1BlockOrm, "UpdateL1GasOracleStatusAndOracleTxHash", func(context.Context, string, types.GasOracleStatus, string) error {
		return nil
	})

	l1Relayer.ProcessGasPriceOracle()
}
