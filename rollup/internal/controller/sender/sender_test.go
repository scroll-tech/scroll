package sender

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/consensys/gnark-crypto/ecc/bls12-381/fr"
	gokzg4844 "github.com/crate-crypto/go-kzg-4844"
	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/crypto/kzg4844"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"scroll-tech/common/database"
	"scroll-tech/common/docker"
	dockercompose "scroll-tech/common/docker-compose/l1"
	"scroll-tech/common/types"
	"scroll-tech/database/migrate"

	bridgeAbi "scroll-tech/rollup/abi"
	"scroll-tech/rollup/internal/config"
	"scroll-tech/rollup/mock_bridge"
)

const TXBatch = 50

var (
	privateKey             *ecdsa.PrivateKey
	blobSenderPrivateKey   *ecdsa.PrivateKey
	cfg                    *config.Config
	base                   *docker.App
	posL1TestEnv           *dockercompose.PoSL1TestEnv
	txTypes                = []string{"LegacyTx", "DynamicFeeTx"}
	txUint8Types           = []uint8{0, 2}
	db                     *gorm.DB
	mockL1ContractsAddress common.Address
)

func TestMain(m *testing.M) {
	base = docker.NewDockerApp()
	defer base.Free()

	posL1TestEnv = dockercompose.NewPoSL1TestEnv()
	if err := posL1TestEnv.Start(); err != nil {
		fmt.Printf("Failed to start PoS L1 test environment: %v\n", err)
		os.Exit(1)
	}

	defer posL1TestEnv.Stop()

	m.Run()
}

func setupEnv(t *testing.T) {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.LogfmtFormat()))
	glogger.Verbosity(log.LvlInfo)
	log.Root().SetHandler(glogger)

	var err error
	cfg, err = config.NewConfig("../../../conf/config.json")
	assert.NoError(t, err)
	priv, err := crypto.HexToECDSA("1212121212121212121212121212121212121212121212121212121212121212")
	assert.NoError(t, err)
	privateKey = priv

	priv, err = crypto.HexToECDSA("2121212121212121212121212121212121212121212121212121212121212121")
	assert.NoError(t, err)
	blobSenderPrivateKey = priv

	cfg.L1Config.RelayerConfig.SenderConfig.Endpoint = posL1TestEnv.Endpoint()

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

	l1Client, err := posL1TestEnv.L1Client()
	assert.NoError(t, err)

	chainID, err := l1Client.ChainID(context.Background())
	assert.NoError(t, err)

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	assert.NoError(t, err)

	_, tx, _, err := mock_bridge.DeployMockBridgeL1(auth, l1Client)
	assert.NoError(t, err)

	mockL1ContractsAddress, err = bind.WaitDeployed(context.Background(), l1Client, tx)
	assert.NoError(t, err)
}

func TestSender(t *testing.T) {
	// Setup
	setupEnv(t)

	t.Run("test new sender", testNewSender)
	t.Run("test fallback gas limit", testFallbackGasLimit)
	t.Run("test send and retrieve transaction", testSendAndRetrieveTransaction)
	t.Run("test access list transaction gas limit", testAccessListTransactionGasLimit)
	t.Run("test resubmit zero gas price transaction", testResubmitZeroGasPriceTransaction)
	t.Run("test resubmit non-zero gas price transaction", testResubmitNonZeroGasPriceTransaction)
	t.Run("test resubmit under priced transaction", testResubmitUnderpricedTransaction)
	t.Run("test resubmit transaction with rising base fee", testResubmitTransactionWithRisingBaseFee)
	t.Run("test check pending transaction tx confirmed", testCheckPendingTransactionTxConfirmed)
	t.Run("test check pending transaction resubmit tx confirmed", testCheckPendingTransactionResubmitTxConfirmed)
	t.Run("test check pending transaction replaced tx confirmed", testCheckPendingTransactionReplacedTxConfirmed)
	t.Run("test check pending transaction multiple times with only one transaction pending", testCheckPendingTransactionTxMultipleTimesWithOnlyOneTxPending)

	// blob transactions
	t.Run("test new blob sender", testNewBlobSender)
	t.Run("test send and retrieve blob transaction", testSendAndRetrieveBlobTransaction)
}

func testNewSender(t *testing.T) {
	for _, txType := range txTypes {
		sqlDB, err := db.DB()
		assert.NoError(t, err)
		assert.NoError(t, migrate.ResetDB(sqlDB))

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

func testSendAndRetrieveTransaction(t *testing.T) {
	for i, txType := range txTypes {
		sqlDB, err := db.DB()
		assert.NoError(t, err)
		assert.NoError(t, migrate.ResetDB(sqlDB))

		cfgCopy := *cfg.L1Config.RelayerConfig.SenderConfig
		cfgCopy.TxType = txType
		s, err := NewSender(context.Background(), &cfgCopy, privateKey, "test", "test", types.SenderTypeUnknown, db, nil)
		assert.NoError(t, err)

		hash, err := s.SendTransaction("0", &common.Address{}, nil, nil, 0)
		assert.NoError(t, err)
		txs, err := s.pendingTransactionOrm.GetPendingOrReplacedTransactionsBySenderType(context.Background(), s.senderType, 1)
		assert.NoError(t, err)
		assert.Len(t, txs, 1)
		assert.Equal(t, "0", txs[0].ContextID)
		assert.Equal(t, hash.String(), txs[0].Hash)
		assert.Equal(t, txUint8Types[i], txs[0].Type)
		assert.Equal(t, types.TxStatusPending, txs[0].Status)
		assert.Equal(t, "0x1C5A77d9FA7eF466951B2F01F724BCa3A5820b63", txs[0].SenderAddress)
		assert.Equal(t, types.SenderTypeUnknown, txs[0].SenderType)
		assert.Equal(t, "test", txs[0].SenderService)
		assert.Equal(t, "test", txs[0].SenderName)
		s.Stop()
	}
}

func testFallbackGasLimit(t *testing.T) {
	for _, txType := range txTypes {
		sqlDB, err := db.DB()
		assert.NoError(t, err)
		assert.NoError(t, migrate.ResetDB(sqlDB))

		cfgCopy := *cfg.L1Config.RelayerConfig.SenderConfig
		cfgCopy.TxType = txType
		cfgCopy.Confirmations = rpc.LatestBlockNumber
		s, err := NewSender(context.Background(), &cfgCopy, privateKey, "test", "test", types.SenderTypeUnknown, db, nil)
		assert.NoError(t, err)

		client, err := ethclient.Dial(cfgCopy.Endpoint)
		assert.NoError(t, err)

		// FallbackGasLimit = 0
		txHash0, err := s.SendTransaction("0", &common.Address{}, nil, nil, 0)
		assert.NoError(t, err)
		tx0, _, err := client.TransactionByHash(context.Background(), txHash0)
		assert.NoError(t, err)
		assert.Greater(t, tx0.Gas(), uint64(0))

		// FallbackGasLimit = 100000
		patchGuard := gomonkey.ApplyPrivateMethod(s, "estimateGasLimit",
			func(contract *common.Address, data []byte, sidecar *gethTypes.BlobTxSidecar, gasPrice, gasTipCap, gasFeeCap, blobGasFeeCap *big.Int) (uint64, *gethTypes.AccessList, error) {
				return 0, nil, errors.New("estimateGasLimit error")
			},
		)

		txHash1, err := s.SendTransaction("1", &common.Address{}, nil, nil, 100000)
		assert.NoError(t, err)
		tx1, _, err := client.TransactionByHash(context.Background(), txHash1)
		assert.NoError(t, err)
		assert.Equal(t, uint64(100000), tx1.Gas())

		s.Stop()
		patchGuard.Reset()
	}
}

func testResubmitZeroGasPriceTransaction(t *testing.T) {
	for _, txType := range txTypes {
		sqlDB, err := db.DB()
		assert.NoError(t, err)
		assert.NoError(t, migrate.ResetDB(sqlDB))

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
		tx, err := s.createAndSendTx(feeData, &common.Address{}, nil, nil, nil)
		assert.NoError(t, err)
		assert.NotNil(t, tx)
		// Increase at least 1 wei in gas price, gas tip cap and gas fee cap.
		_, err = s.resubmitTransaction(tx, 0, 0)
		assert.NoError(t, err)
		s.Stop()
	}
}

func testAccessListTransactionGasLimit(t *testing.T) {
	for _, txType := range txTypes {
		sqlDB, err := db.DB()
		assert.NoError(t, err)
		assert.NoError(t, migrate.ResetDB(sqlDB))

		cfgCopy := *cfg.L1Config.RelayerConfig.SenderConfig
		cfgCopy.TxType = txType
		s, err := NewSender(context.Background(), &cfgCopy, privateKey, "test", "test", types.SenderTypeUnknown, db, nil)
		assert.NoError(t, err)

		l2GasOracleABI, err := bridgeAbi.L2GasPriceOracleMetaData.GetAbi()
		assert.NoError(t, err)

		data, err := l2GasOracleABI.Pack("setL2BaseFee", big.NewInt(2333))
		assert.NoError(t, err)

		gasLimit, accessList, err := s.estimateGasLimit(&mockL1ContractsAddress, data, nil, nil, big.NewInt(100000000000), big.NewInt(100000000000), big.NewInt(0))
		assert.NoError(t, err)

		if txType == LegacyTxType {
			assert.Equal(t, uint64(43949), gasLimit)
			assert.Nil(t, accessList)
		} else {
			assert.Equal(t, uint64(43472), gasLimit)
			assert.NotNil(t, accessList)
		}

		s.Stop()
	}
}

func testResubmitNonZeroGasPriceTransaction(t *testing.T) {
	for _, txType := range txTypes {
		sqlDB, err := db.DB()
		assert.NoError(t, err)
		assert.NoError(t, migrate.ResetDB(sqlDB))

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
		tx, err := s.createAndSendTx(feeData, &common.Address{}, nil, nil, nil)
		assert.NoError(t, err)
		assert.NotNil(t, tx)
		_, err = s.resubmitTransaction(tx, 0, 0)
		assert.NoError(t, err)
		s.Stop()
	}
}

func testResubmitUnderpricedTransaction(t *testing.T) {
	for _, txType := range txTypes {
		sqlDB, err := db.DB()
		assert.NoError(t, err)
		assert.NoError(t, migrate.ResetDB(sqlDB))

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
		tx, err := s.createAndSendTx(feeData, &common.Address{}, nil, nil, nil)
		assert.NoError(t, err)
		assert.NotNil(t, tx)
		_, err = s.resubmitTransaction(tx, 0, 0)
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
	baseFeePerGas := uint64(1000)
	// bump the basefee by 10x
	baseFeePerGas *= 10
	// resubmit and check that the gas fee has been adjusted accordingly
	newTx, err := s.resubmitTransaction(tx, baseFeePerGas, 0)
	assert.NoError(t, err)

	maxGasPrice := new(big.Int).SetUint64(s.config.MaxGasPrice)
	expectedGasFeeCap := getGasFeeCap(new(big.Int).SetUint64(baseFeePerGas), tx.GasTipCap())
	if expectedGasFeeCap.Cmp(maxGasPrice) > 0 {
		expectedGasFeeCap = maxGasPrice
	}

	assert.Equal(t, expectedGasFeeCap.Int64(), newTx.GasFeeCap().Int64())
	s.Stop()
}

func testCheckPendingTransactionTxConfirmed(t *testing.T) {
	for _, txType := range txTypes {
		sqlDB, err := db.DB()
		assert.NoError(t, err)
		assert.NoError(t, migrate.ResetDB(sqlDB))

		cfgCopy := *cfg.L1Config.RelayerConfig.SenderConfig
		cfgCopy.TxType = txType
		s, err := NewSender(context.Background(), &cfgCopy, privateKey, "test", "test", types.SenderTypeCommitBatch, db, nil)
		assert.NoError(t, err)

		_, err = s.SendTransaction("test", &common.Address{}, nil, nil, 0)
		assert.NoError(t, err)

		txs, err := s.pendingTransactionOrm.GetPendingOrReplacedTransactionsBySenderType(context.Background(), s.senderType, 1)
		assert.NoError(t, err)
		assert.Len(t, txs, 1)
		assert.Equal(t, types.TxStatusPending, txs[0].Status)
		assert.Equal(t, types.SenderTypeCommitBatch, txs[0].SenderType)

		patchGuard := gomonkey.ApplyMethodFunc(s.client, "TransactionReceipt", func(_ context.Context, hash common.Hash) (*gethTypes.Receipt, error) {
			return &gethTypes.Receipt{TxHash: hash, BlockNumber: big.NewInt(0), Status: gethTypes.ReceiptStatusSuccessful}, nil
		})

		s.checkPendingTransaction()
		assert.NoError(t, err)

		txs, err = s.pendingTransactionOrm.GetPendingOrReplacedTransactionsBySenderType(context.Background(), s.senderType, 1)
		assert.NoError(t, err)
		assert.Len(t, txs, 0)

		s.Stop()
		patchGuard.Reset()
	}
}

func testCheckPendingTransactionResubmitTxConfirmed(t *testing.T) {
	for _, txType := range txTypes {
		sqlDB, err := db.DB()
		assert.NoError(t, err)
		assert.NoError(t, migrate.ResetDB(sqlDB))

		cfgCopy := *cfg.L1Config.RelayerConfig.SenderConfig
		cfgCopy.TxType = txType
		cfgCopy.EscalateBlocks = 0
		s, err := NewSender(context.Background(), &cfgCopy, privateKey, "test", "test", types.SenderTypeFinalizeBatch, db, nil)
		assert.NoError(t, err)

		originTxHash, err := s.SendTransaction("test", &common.Address{}, nil, nil, 0)
		assert.NoError(t, err)

		txs, err := s.pendingTransactionOrm.GetPendingOrReplacedTransactionsBySenderType(context.Background(), s.senderType, 1)
		assert.NoError(t, err)
		assert.Len(t, txs, 1)
		assert.Equal(t, types.TxStatusPending, txs[0].Status)
		assert.Equal(t, types.SenderTypeFinalizeBatch, txs[0].SenderType)

		patchGuard := gomonkey.ApplyMethodFunc(s.client, "TransactionReceipt", func(_ context.Context, hash common.Hash) (*gethTypes.Receipt, error) {
			if hash == originTxHash {
				return nil, fmt.Errorf("simulated transaction receipt error")
			}
			return &gethTypes.Receipt{TxHash: hash, BlockNumber: big.NewInt(0), Status: gethTypes.ReceiptStatusSuccessful}, nil
		})

		// Attempt to resubmit the transaction.
		s.checkPendingTransaction()
		assert.NoError(t, err)

		status, err := s.pendingTransactionOrm.GetTxStatusByTxHash(context.Background(), originTxHash)
		assert.NoError(t, err)
		assert.Equal(t, types.TxStatusReplaced, status)

		txs, err = s.pendingTransactionOrm.GetPendingOrReplacedTransactionsBySenderType(context.Background(), s.senderType, 2)
		assert.NoError(t, err)
		assert.Len(t, txs, 2)
		assert.Equal(t, types.TxStatusReplaced, txs[0].Status)
		assert.Equal(t, types.TxStatusPending, txs[1].Status)

		// Check the pending transactions again after attempting to resubmit.
		s.checkPendingTransaction()
		assert.NoError(t, err)

		txs, err = s.pendingTransactionOrm.GetPendingOrReplacedTransactionsBySenderType(context.Background(), s.senderType, 1)
		assert.NoError(t, err)
		assert.Len(t, txs, 0)

		s.Stop()
		patchGuard.Reset()
	}
}

func testCheckPendingTransactionReplacedTxConfirmed(t *testing.T) {
	for _, txType := range txTypes {
		sqlDB, err := db.DB()
		assert.NoError(t, err)
		assert.NoError(t, migrate.ResetDB(sqlDB))

		cfgCopy := *cfg.L1Config.RelayerConfig.SenderConfig
		cfgCopy.TxType = txType
		cfgCopy.EscalateBlocks = 0
		s, err := NewSender(context.Background(), &cfgCopy, privateKey, "test", "test", types.SenderTypeL1GasOracle, db, nil)
		assert.NoError(t, err)

		txHash, err := s.SendTransaction("test", &common.Address{}, nil, nil, 0)
		assert.NoError(t, err)

		txs, err := s.pendingTransactionOrm.GetPendingOrReplacedTransactionsBySenderType(context.Background(), s.senderType, 1)
		assert.NoError(t, err)
		assert.Len(t, txs, 1)
		assert.Equal(t, types.TxStatusPending, txs[0].Status)
		assert.Equal(t, types.SenderTypeL1GasOracle, txs[0].SenderType)

		patchGuard := gomonkey.ApplyMethodFunc(s.client, "TransactionReceipt", func(_ context.Context, hash common.Hash) (*gethTypes.Receipt, error) {
			var status types.TxStatus
			status, err = s.pendingTransactionOrm.GetTxStatusByTxHash(context.Background(), hash)
			if err != nil {
				return nil, fmt.Errorf("failed to get transaction status, hash: %s, err: %w", hash.String(), err)
			}
			// If the transaction status is 'replaced', return a successful receipt.
			if status == types.TxStatusReplaced {
				return &gethTypes.Receipt{
					TxHash:      hash,
					BlockNumber: big.NewInt(0),
					Status:      gethTypes.ReceiptStatusSuccessful,
				}, nil
			}
			return nil, fmt.Errorf("simulated transaction receipt error")
		})

		// Attempt to resubmit the transaction.
		s.checkPendingTransaction()
		assert.NoError(t, err)

		status, err := s.pendingTransactionOrm.GetTxStatusByTxHash(context.Background(), txHash)
		assert.NoError(t, err)
		assert.Equal(t, types.TxStatusReplaced, status)

		txs, err = s.pendingTransactionOrm.GetPendingOrReplacedTransactionsBySenderType(context.Background(), s.senderType, 2)
		assert.NoError(t, err)
		assert.Len(t, txs, 2)
		assert.Equal(t, types.TxStatusReplaced, txs[0].Status)
		assert.Equal(t, types.TxStatusPending, txs[1].Status)

		// Check the pending transactions again after attempting to resubmit.
		s.checkPendingTransaction()
		assert.NoError(t, err)

		txs, err = s.pendingTransactionOrm.GetPendingOrReplacedTransactionsBySenderType(context.Background(), s.senderType, 1)
		assert.NoError(t, err)
		assert.Len(t, txs, 0)

		s.Stop()
		patchGuard.Reset()
	}
}

func testCheckPendingTransactionTxMultipleTimesWithOnlyOneTxPending(t *testing.T) {
	for _, txType := range txTypes {
		sqlDB, err := db.DB()
		assert.NoError(t, err)
		assert.NoError(t, migrate.ResetDB(sqlDB))

		cfgCopy := *cfg.L1Config.RelayerConfig.SenderConfig
		cfgCopy.TxType = txType
		cfgCopy.EscalateBlocks = 0
		s, err := NewSender(context.Background(), &cfgCopy, privateKey, "test", "test", types.SenderTypeCommitBatch, db, nil)
		assert.NoError(t, err)

		_, err = s.SendTransaction("test", &common.Address{}, nil, nil, 0)
		assert.NoError(t, err)

		txs, err := s.pendingTransactionOrm.GetPendingOrReplacedTransactionsBySenderType(context.Background(), s.senderType, 1)
		assert.NoError(t, err)
		assert.Len(t, txs, 1)
		assert.Equal(t, types.TxStatusPending, txs[0].Status)
		assert.Equal(t, types.SenderTypeCommitBatch, txs[0].SenderType)

		patchGuard := gomonkey.ApplyMethodFunc(s.client, "TransactionReceipt", func(_ context.Context, hash common.Hash) (*gethTypes.Receipt, error) {
			return nil, fmt.Errorf("simulated transaction receipt error")
		})

		for i := 1; i <= 6; i++ {
			s.checkPendingTransaction()
			assert.NoError(t, err)

			txs, err = s.pendingTransactionOrm.GetPendingOrReplacedTransactionsBySenderType(context.Background(), s.senderType, 100)
			assert.NoError(t, err)
			assert.Len(t, txs, i+1)
			for j := 0; j < i; j++ {
				assert.Equal(t, types.TxStatusReplaced, txs[j].Status)
			}
			assert.Equal(t, types.TxStatusPending, txs[i].Status)
		}

		s.Stop()
		patchGuard.Reset()
	}
}

func testNewBlobSender(t *testing.T) {
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	// exit by Stop()
	cfgCopy1 := *cfg.L1Config.RelayerConfig.SenderConfig
	cfgCopy1.TxType = BlobTxType
	newSender1, err := NewSender(context.Background(), &cfgCopy1, blobSenderPrivateKey, "test", "test", types.SenderTypeUnknown, db, nil)
	assert.NoError(t, err)
	newSender1.Stop()

	// exit by ctx.Done()
	cfgCopy2 := *cfg.L1Config.RelayerConfig.SenderConfig
	cfgCopy2.TxType = BlobTxType
	subCtx, cancel := context.WithCancel(context.Background())
	_, err = NewSender(subCtx, &cfgCopy2, blobSenderPrivateKey, "test", "test", types.SenderTypeUnknown, db, nil)
	assert.NoError(t, err)
	cancel()
}

func testSendAndRetrieveBlobTransaction(t *testing.T) {
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	cfgCopy := *cfg.L1Config.RelayerConfig.SenderConfig
	cfgCopy.TxType = BlobTxType
	s, err := NewSender(context.Background(), &cfgCopy, blobSenderPrivateKey, "test", "test", types.SenderTypeUnknown, db, nil)
	assert.NoError(t, err)

	hash, err := s.SendTransaction("0", &common.Address{}, nil, randBlob(), 0)
	assert.NoError(t, err)
	txs, err := s.pendingTransactionOrm.GetPendingOrReplacedTransactionsBySenderType(context.Background(), s.senderType, 1)
	assert.NoError(t, err)
	assert.Len(t, txs, 1)
	assert.Equal(t, "0", txs[0].ContextID)
	assert.Equal(t, hash.String(), txs[0].Hash)
	assert.Equal(t, uint8(3), txs[0].Type)
	assert.Equal(t, types.TxStatusPending, txs[0].Status)
	assert.Equal(t, "0x2BD0C9FE079c8FcA0E3352eb3D02839c371E5c41", txs[0].SenderAddress)
	assert.Equal(t, types.SenderTypeUnknown, txs[0].SenderType)
	assert.Equal(t, "test", txs[0].SenderService)
	assert.Equal(t, "test", txs[0].SenderName)
	s.Stop()
}

func randBlob() *kzg4844.Blob {
	var blob kzg4844.Blob
	for i := 0; i < len(blob); i += gokzg4844.SerializedScalarSize {
		fieldElementBytes := randFieldElement()
		copy(blob[i:i+gokzg4844.SerializedScalarSize], fieldElementBytes[:])
	}
	return &blob
}

func randFieldElement() [32]byte {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		panic("failed to get random field element")
	}
	var r fr.Element
	r.SetBytes(bytes)

	return gokzg4844.SerializeScalar(r)
}
