package sender

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"scroll-tech/common/database"
	"scroll-tech/common/docker"
	"scroll-tech/common/types"
	"scroll-tech/database/migrate"
	"scroll-tech/rollup/internal/config"
)

const TXBatch = 50

var (
	privateKey *ecdsa.PrivateKey
	cfg        *config.Config
	base       *docker.App
	txTypes    = []string{"LegacyTx", "AccessListTx", "DynamicFeeTx"}
	db         *gorm.DB
)

func TestMain(m *testing.M) {
	base = docker.NewDockerApp()

	m.Run()

	base.Free()
}

func setupEnv(t *testing.T) {
	var err error
	cfg, err = config.NewConfig("../../../conf/config.json")
	assert.NoError(t, err)
	priv, err := crypto.HexToECDSA("1212121212121212121212121212121212121212121212121212121212121212")
	assert.NoError(t, err)
	// Load default private key.
	privateKey = priv

	base.RunL1Geth(t)
	cfg.L1Config.RelayerConfig.SenderConfig.Endpoint = base.L1gethImg.Endpoint()

	base.RunDBImage(t)
	db, err = database.InitDB(
		&database.Config{
			DSN:        base.DBConfig.DSN,
			DriverName: base.DBConfig.DriverName,
			MaxOpenNum: base.DBConfig.MaxOpenNum,
			MaxIdleNum: base.DBConfig.MaxIdleNum,
		},
	)
	assert.NoError(t, err)
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))
}

func TestSender(t *testing.T) {
	// Setup
	setupEnv(t)

	t.Run("test new sender", testNewSender)
	t.Run("test fallback gas limit", testFallbackGasLimit)
	t.Run("test resubmit zero gas price transaction", testResubmitZeroGasPriceTransaction)
	t.Run("test resubmit non-zero gas price transaction", testResubmitNonZeroGasPriceTransaction)
	t.Run("test resubmit under priced transaction", testResubmitUnderpricedTransaction)
	t.Run("test resubmit transaction with rising base fee", testResubmitTransactionWithRisingBaseFee)
}

func testNewSender(t *testing.T) {
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))
	for _, txType := range txTypes {
		// exit by Stop()
		cfgCopy1 := *cfg.L1Config.RelayerConfig.SenderConfig
		cfgCopy1.TxType = txType
		newSender1, err := NewSender(context.Background(), &cfgCopy1, privateKey, "test", "test", types.SenderTypeUnknown, db, nil)
		assert.NoError(t, err)
		newSender1.Stop()

		// exit by ctx.Done()
		cfgCopy2 := *cfg.L1Config.RelayerConfig.SenderConfig
		cfgCopy2.TxType = txType
		subCtx, cancel := context.WithCancel(context.Background())
		_, err = NewSender(subCtx, &cfgCopy2, privateKey, "test", "test", types.SenderTypeUnknown, db, nil)
		assert.NoError(t, err)
		cancel()
	}
}

func testFallbackGasLimit(t *testing.T) {
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))
	for _, txType := range txTypes {
		cfgCopy := *cfg.L1Config.RelayerConfig.SenderConfig
		cfgCopy.TxType = txType
		cfgCopy.Confirmations = rpc.LatestBlockNumber
		s, err := NewSender(context.Background(), &cfgCopy, privateKey, "test", "test", types.SenderTypeUnknown, db, nil)
		assert.NoError(t, err)

		client, err := ethclient.Dial(cfgCopy.Endpoint)
		assert.NoError(t, err)

		// FallbackGasLimit = 0
		txHash0, err := s.SendTransaction("0", &common.Address{}, big.NewInt(1), nil, 0)
		assert.NoError(t, err)
		tx0, _, err := client.TransactionByHash(context.Background(), txHash0)
		assert.NoError(t, err)
		assert.Greater(t, tx0.Gas(), uint64(0))

		// FallbackGasLimit = 100000
		patchGuard := gomonkey.ApplyPrivateMethod(s, "estimateGasLimit",
			func(opts *bind.TransactOpts, contract *common.Address, input []byte, gasPrice, gasTipCap, gasFeeCap, value *big.Int) (uint64, error) {
				return 0, errors.New("estimateGasLimit error")
			},
		)

		txHash1, err := s.SendTransaction("1", &common.Address{}, big.NewInt(1), nil, 100000)
		assert.NoError(t, err)
		tx1, _, err := client.TransactionByHash(context.Background(), txHash1)
		assert.NoError(t, err)
		assert.Equal(t, uint64(100000), tx1.Gas())

		s.Stop()
		patchGuard.Reset()
	}
}

func testResubmitZeroGasPriceTransaction(t *testing.T) {
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))
	for _, txType := range txTypes {
		cfgCopy := *cfg.L1Config.RelayerConfig.SenderConfig
		cfgCopy.TxType = txType
		s, err := NewSender(context.Background(), &cfgCopy, privateKey, "test", "test", types.SenderTypeUnknown, db, nil)
		assert.NoError(t, err)
		feeData := &FeeData{
			gasPrice:  big.NewInt(0),
			gasTipCap: big.NewInt(0),
			gasFeeCap: big.NewInt(0),
			gasLimit:  50000,
		}
		tx, err := s.createAndSendTx(s.auth, feeData, &common.Address{}, big.NewInt(0), nil, nil)
		assert.NoError(t, err)
		assert.NotNil(t, tx)
		// Increase at least 1 wei in gas price, gas tip cap and gas fee cap.
		_, err = s.resubmitTransaction(s.auth, tx)
		assert.NoError(t, err)
		s.Stop()
	}
}

func testResubmitNonZeroGasPriceTransaction(t *testing.T) {
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))
	for _, txType := range txTypes {
		cfgCopy := *cfg.L1Config.RelayerConfig.SenderConfig
		// Bump gas price, gas tip cap and gas fee cap just touch the minimum threshold of 10% (default config of geth).
		cfgCopy.EscalateMultipleNum = 110
		cfgCopy.EscalateMultipleDen = 100
		cfgCopy.TxType = txType
		s, err := NewSender(context.Background(), &cfgCopy, privateKey, "test", "test", types.SenderTypeUnknown, db, nil)
		assert.NoError(t, err)
		feeData := &FeeData{
			gasPrice:  big.NewInt(100000),
			gasTipCap: big.NewInt(100000),
			gasFeeCap: big.NewInt(100000),
			gasLimit:  50000,
		}
		tx, err := s.createAndSendTx(s.auth, feeData, &common.Address{}, big.NewInt(0), nil, nil)
		assert.NoError(t, err)
		assert.NotNil(t, tx)
		_, err = s.resubmitTransaction(s.auth, tx)
		assert.NoError(t, err)
		s.Stop()
	}
}

func testResubmitUnderpricedTransaction(t *testing.T) {
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))
	for _, txType := range txTypes {
		cfgCopy := *cfg.L1Config.RelayerConfig.SenderConfig
		// Bump gas price, gas tip cap and gas fee cap less than 10% (default config of geth).
		cfgCopy.EscalateMultipleNum = 109
		cfgCopy.EscalateMultipleDen = 100
		cfgCopy.TxType = txType
		s, err := NewSender(context.Background(), &cfgCopy, privateKey, "test", "test", types.SenderTypeUnknown, db, nil)
		assert.NoError(t, err)
		feeData := &FeeData{
			gasPrice:  big.NewInt(100000),
			gasTipCap: big.NewInt(100000),
			gasFeeCap: big.NewInt(100000),
			gasLimit:  50000,
		}
		tx, err := s.createAndSendTx(s.auth, feeData, &common.Address{}, big.NewInt(0), nil, nil)
		assert.NoError(t, err)
		assert.NotNil(t, tx)
		_, err = s.resubmitTransaction(s.auth, tx)
		assert.Error(t, err, "replacement transaction underpriced")
		s.Stop()
	}
}

func testResubmitTransactionWithRisingBaseFee(t *testing.T) {
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	txType := "DynamicFeeTx"
	cfgCopy := *cfg.L1Config.RelayerConfig.SenderConfig
	cfgCopy.TxType = txType
	s, err := NewSender(context.Background(), &cfgCopy, privateKey, "test", "test", types.SenderTypeUnknown, db, nil)
	assert.NoError(t, err)
	tx := gethTypes.NewTransaction(s.auth.Nonce.Uint64(), common.Address{}, big.NewInt(0), 21000, big.NewInt(0), nil)
	s.baseFeePerGas = 1000
	// bump the basefee by 10x
	s.baseFeePerGas *= 10
	// resubmit and check that the gas fee has been adjusted accordingly
	newTx, err := s.resubmitTransaction(s.auth, tx)
	assert.NoError(t, err)

	escalateMultipleNum := new(big.Int).SetUint64(s.config.EscalateMultipleNum)
	escalateMultipleDen := new(big.Int).SetUint64(s.config.EscalateMultipleDen)
	maxGasPrice := new(big.Int).SetUint64(s.config.MaxGasPrice)

	adjBaseFee := new(big.Int)
	adjBaseFee.SetUint64(s.baseFeePerGas)
	adjBaseFee = adjBaseFee.Mul(adjBaseFee, escalateMultipleNum)
	adjBaseFee = adjBaseFee.Div(adjBaseFee, escalateMultipleDen)

	expectedGasFeeCap := new(big.Int).Add(tx.GasTipCap(), adjBaseFee)
	if expectedGasFeeCap.Cmp(maxGasPrice) > 0 {
		expectedGasFeeCap = maxGasPrice
	}

	assert.Equal(t, expectedGasFeeCap.Int64(), newTx.GasFeeCap().Int64())
	s.Stop()
}
