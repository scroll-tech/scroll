package sender

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/consensys/gnark-crypto/ecc/bls12-381/fr"
	gokzg4844 "github.com/crate-crypto/go-kzg-4844"
	"github.com/holiman/uint256"
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

	"scroll-tech/common/testcontainers"
	"scroll-tech/common/types"
	"scroll-tech/database/migrate"

	bridgeAbi "scroll-tech/rollup/abi"
	"scroll-tech/rollup/internal/config"
	"scroll-tech/rollup/internal/orm"
	"scroll-tech/rollup/mock_bridge"
)

var (
	privateKey           *ecdsa.PrivateKey
	cfg                  *config.Config
	testApps             *testcontainers.TestcontainerApps
	txTypes              = []string{"LegacyTx", "DynamicFeeTx", "DynamicFeeTx"}
	txBlob               = []*kzg4844.Blob{nil, nil, randBlob()}
	txUint8Types         = []uint8{0, 2, 3}
	db                   *gorm.DB
	testContractsAddress common.Address
)

func TestMain(m *testing.M) {
	defer func() {
		if testApps != nil {
			testApps.Free()
		}
	}()
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

	testApps = testcontainers.NewTestcontainerApps()
	assert.NoError(t, testApps.StartPostgresContainer())
	assert.NoError(t, testApps.StartL2GethContainer())
	assert.NoError(t, testApps.StartPoSL1Container())

	cfg.L2Config.RelayerConfig.SenderConfig.Endpoint, err = testApps.GetPoSL1EndPoint()
	assert.NoError(t, err)

	db, err = testApps.GetGormDBClient()
	assert.NoError(t, err)
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	l1Client, err := testApps.GetPoSL1Client()
	assert.NoError(t, err)

	chainID, err := l1Client.ChainID(context.Background())
	assert.NoError(t, err)

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	assert.NoError(t, err)

	nonce, err := l1Client.PendingNonceAt(context.Background(), auth.From)
	assert.NoError(t, err)

	testContractsAddress = crypto.CreateAddress(auth.From, nonce)

	tx := gethTypes.NewContractCreation(nonce, big.NewInt(0), 10000000, big.NewInt(10000000000), common.FromHex(mock_bridge.MockBridgeMetaData.Bin))
	signedTx, err := auth.Signer(auth.From, tx)
	assert.NoError(t, err)
	err = l1Client.SendTransaction(context.Background(), signedTx)
	assert.NoError(t, err)

	assert.Eventually(t, func() bool {
		_, isPending, err := l1Client.TransactionByHash(context.Background(), signedTx.Hash())
		return err == nil && !isPending
	}, 30*time.Second, time.Second)

	assert.Eventually(t, func() bool {
		receipt, err := l1Client.TransactionReceipt(context.Background(), signedTx.Hash())
		return err == nil && receipt.Status == gethTypes.ReceiptStatusSuccessful
	}, 30*time.Second, time.Second)

	assert.Eventually(t, func() bool {
		code, err := l1Client.CodeAt(context.Background(), testContractsAddress, nil)
		return err == nil && len(code) > 0
	}, 30*time.Second, time.Second)
}

func TestSender(t *testing.T) {
	setupEnv(t)
	t.Run("test new sender", testNewSender)
	t.Run("test send and retrieve transaction", testSendAndRetrieveTransaction)
	t.Run("test fallback gas limit", testFallbackGasLimit)
	t.Run("test access list transaction gas limit", testAccessListTransactionGasLimit)
	t.Run("test resubmit zero gas price transaction", testResubmitZeroGasPriceTransaction)
	t.Run("test resubmit non-zero gas price transaction", testResubmitNonZeroGasPriceTransaction)
	t.Run("test resubmit under priced transaction", testResubmitUnderpricedTransaction)
	t.Run("test resubmit dynamic fee transaction with rising base fee", testResubmitDynamicFeeTransactionWithRisingBaseFee)
	t.Run("test resubmit blob transaction with rising base fee and blob base fee", testResubmitBlobTransactionWithRisingBaseFeeAndBlobBaseFee)
	t.Run("test resubmit nonce gapped transaction", testResubmitNonceGappedTransaction)
	t.Run("test check pending transaction tx confirmed", testCheckPendingTransactionTxConfirmed)
	t.Run("test check pending transaction resubmit tx confirmed", testCheckPendingTransactionResubmitTxConfirmed)
	t.Run("test check pending transaction replaced tx confirmed", testCheckPendingTransactionReplacedTxConfirmed)
	t.Run("test check pending transaction multiple times with only one transaction pending", testCheckPendingTransactionTxMultipleTimesWithOnlyOneTxPending)
	t.Run("test blob transaction with blobhash op contract call", testBlobTransactionWithBlobhashOpContractCall)
	t.Run("test test send blob-carrying tx over limit", testSendBlobCarryingTxOverLimit)
}

func testNewSender(t *testing.T) {
	for _, txType := range txTypes {
		sqlDB, err := db.DB()
		assert.NoError(t, err)
		assert.NoError(t, migrate.ResetDB(sqlDB))

		// exit by Stop()
		cfgCopy1 := *cfg.L2Config.RelayerConfig.SenderConfig
		cfgCopy1.TxType = txType
		newSender1, err := NewSender(context.Background(), &cfgCopy1, privateKey, "test", "test", types.SenderTypeUnknown, db, nil)
		assert.NoError(t, err)
		newSender1.Stop()

		// exit by ctx.Done()
		cfgCopy2 := *cfg.L2Config.RelayerConfig.SenderConfig
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

		cfgCopy := *cfg.L2Config.RelayerConfig.SenderConfig
		cfgCopy.TxType = txType
		s, err := NewSender(context.Background(), &cfgCopy, privateKey, "test", "test", types.SenderTypeUnknown, db, nil)
		assert.NoError(t, err)

		hash, err := s.SendTransaction("0", &common.Address{}, nil, txBlob[i], 0)
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

		assert.Eventually(t, func() bool {
			txs, err = s.pendingTransactionOrm.GetPendingOrReplacedTransactionsBySenderType(context.Background(), s.senderType, 100)
			assert.NoError(t, err)
			return len(txs) == 0
		}, 30*time.Second, time.Second)

		s.Stop()
	}
}

func testFallbackGasLimit(t *testing.T) {
	for i, txType := range txTypes {
		sqlDB, err := db.DB()
		assert.NoError(t, err)
		assert.NoError(t, migrate.ResetDB(sqlDB))

		cfgCopy := *cfg.L2Config.RelayerConfig.SenderConfig
		cfgCopy.TxType = txType
		cfgCopy.Confirmations = rpc.LatestBlockNumber
		s, err := NewSender(context.Background(), &cfgCopy, privateKey, "test", "test", types.SenderTypeUnknown, db, nil)
		assert.NoError(t, err)

		client, err := ethclient.Dial(cfgCopy.Endpoint)
		assert.NoError(t, err)

		// FallbackGasLimit = 0
		txHash0, err := s.SendTransaction("0", &common.Address{}, nil, txBlob[i], 0)
		assert.NoError(t, err)
		tx0, _, err := client.TransactionByHash(context.Background(), txHash0)
		assert.NoError(t, err)
		assert.Greater(t, tx0.Gas(), uint64(0))

		assert.Eventually(t, func() bool {
			var txs []orm.PendingTransaction
			txs, err = s.pendingTransactionOrm.GetPendingOrReplacedTransactionsBySenderType(context.Background(), s.senderType, 100)
			assert.NoError(t, err)
			return len(txs) == 0
		}, 30*time.Second, time.Second)

		// FallbackGasLimit = 100000
		patchGuard := gomonkey.ApplyPrivateMethod(s, "estimateGasLimit",
			func(contract *common.Address, data []byte, sidecar *gethTypes.BlobTxSidecar, gasPrice, gasTipCap, gasFeeCap, blobGasFeeCap *big.Int) (uint64, *gethTypes.AccessList, error) {
				return 0, nil, errors.New("estimateGasLimit error")
			},
		)

		txHash1, err := s.SendTransaction("1", &common.Address{}, nil, txBlob[i], 100000)
		assert.NoError(t, err)
		tx1, _, err := client.TransactionByHash(context.Background(), txHash1)
		assert.NoError(t, err)
		assert.Equal(t, uint64(100000), tx1.Gas())

		assert.Eventually(t, func() bool {
			txs, err := s.pendingTransactionOrm.GetPendingOrReplacedTransactionsBySenderType(context.Background(), s.senderType, 100)
			assert.NoError(t, err)
			return len(txs) == 0
		}, 30*time.Second, time.Second)

		s.Stop()
		patchGuard.Reset()
	}
}

func testResubmitZeroGasPriceTransaction(t *testing.T) {
	for i, txType := range txTypes {
		if txBlob[i] != nil {
			continue
		}

		sqlDB, err := db.DB()
		assert.NoError(t, err)
		assert.NoError(t, migrate.ResetDB(sqlDB))

		cfgCopy := *cfg.L2Config.RelayerConfig.SenderConfig
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
		// Bumping the fees enough times to let the transaction be included in a block.
		for i := 0; i < 30; i++ {
			tx, err = s.resubmitTransaction(tx, 0, 0)
			assert.NoError(t, err)
		}

		assert.Eventually(t, func() bool {
			_, isPending, err := s.client.TransactionByHash(context.Background(), tx.Hash())
			return err == nil && !isPending
		}, 30*time.Second, time.Second)

		assert.Eventually(t, func() bool {
			receipt, err := s.client.TransactionReceipt(context.Background(), tx.Hash())
			return err == nil && receipt != nil
		}, 30*time.Second, time.Second)

		s.Stop()
	}
}

func testAccessListTransactionGasLimit(t *testing.T) {
	for i, txType := range txTypes {
		sqlDB, err := db.DB()
		assert.NoError(t, err)
		assert.NoError(t, migrate.ResetDB(sqlDB))

		cfgCopy := *cfg.L2Config.RelayerConfig.SenderConfig
		cfgCopy.TxType = txType
		s, err := NewSender(context.Background(), &cfgCopy, privateKey, "test", "test", types.SenderTypeUnknown, db, nil)
		assert.NoError(t, err)

		l2GasOracleABI, err := bridgeAbi.L2GasPriceOracleMetaData.GetAbi()
		assert.NoError(t, err)

		data, err := l2GasOracleABI.Pack("setL2BaseFee", big.NewInt(int64(i+1)))
		assert.NoError(t, err)

		var sidecar *gethTypes.BlobTxSidecar
		if txBlob[i] != nil {
			sidecar, err = makeSidecar(txBlob[i])
			assert.NoError(t, err)
		}

		gasLimit, accessList, err := s.estimateGasLimit(&testContractsAddress, data, sidecar, nil, big.NewInt(1000000000), big.NewInt(1000000000), big.NewInt(1000000000))
		assert.NoError(t, err)

		if txType == LegacyTxType { // Legacy transactions can not have an access list.
			assert.Equal(t, uint64(43935), gasLimit)
			assert.Nil(t, accessList)
		} else { // Dynamic fee and blob transactions can have an access list.
			assert.Equal(t, uint64(43458), gasLimit)
			assert.NotNil(t, accessList)
		}

		s.Stop()
	}
}

func testResubmitNonZeroGasPriceTransaction(t *testing.T) {
	for i, txType := range txTypes {
		sqlDB, err := db.DB()
		assert.NoError(t, err)
		assert.NoError(t, migrate.ResetDB(sqlDB))

		cfgCopy := *cfg.L2Config.RelayerConfig.SenderConfig
		// Bump gas price, gas tip cap and gas fee cap just touch the minimum threshold of 10% (default config of geth).
		cfgCopy.EscalateMultipleNum = 110
		cfgCopy.EscalateMultipleDen = 100
		cfgCopy.TxType = txType
		s, err := NewSender(context.Background(), &cfgCopy, privateKey, "test", "test", types.SenderTypeUnknown, db, nil)
		assert.NoError(t, err)
		feeData := &FeeData{
			gasPrice:      big.NewInt(1000000000),
			gasTipCap:     big.NewInt(1000000000),
			gasFeeCap:     big.NewInt(1000000000),
			blobGasFeeCap: big.NewInt(1000000000),
			gasLimit:      50000,
		}
		var sidecar *gethTypes.BlobTxSidecar
		if txBlob[i] != nil {
			sidecar, err = makeSidecar(txBlob[i])
			assert.NoError(t, err)
		}
		tx, err := s.createAndSendTx(feeData, &common.Address{}, nil, sidecar, nil)
		assert.NoError(t, err)
		assert.NotNil(t, tx)
		resubmittedTx, err := s.resubmitTransaction(tx, 0, 0)
		assert.NoError(t, err)

		assert.Eventually(t, func() bool {
			_, isPending, err := s.client.TransactionByHash(context.Background(), resubmittedTx.Hash())
			return err == nil && !isPending
		}, 30*time.Second, time.Second)

		assert.Eventually(t, func() bool {
			receipt, err := s.client.TransactionReceipt(context.Background(), resubmittedTx.Hash())
			return err == nil && receipt != nil
		}, 30*time.Second, time.Second)

		s.Stop()
	}
}

func testResubmitUnderpricedTransaction(t *testing.T) {
	for i, txType := range txTypes {
		if txBlob[i] != nil {
			continue
		}

		sqlDB, err := db.DB()
		assert.NoError(t, err)
		assert.NoError(t, migrate.ResetDB(sqlDB))

		cfgCopy := *cfg.L2Config.RelayerConfig.SenderConfig
		// Bump gas price, gas tip cap and gas fee cap less than 10% (default config of geth).
		cfgCopy.EscalateMultipleNum = 109
		cfgCopy.EscalateMultipleDen = 100
		cfgCopy.TxType = txType
		s, err := NewSender(context.Background(), &cfgCopy, privateKey, "test", "test", types.SenderTypeUnknown, db, nil)
		assert.NoError(t, err)
		feeData := &FeeData{
			gasPrice:  big.NewInt(1000000000),
			gasTipCap: big.NewInt(1000000000),
			gasFeeCap: big.NewInt(1000000000),
			gasLimit:  50000,
		}
		tx, err := s.createAndSendTx(feeData, &common.Address{}, nil, nil, nil)
		assert.NoError(t, err)
		assert.NotNil(t, tx)
		_, err = s.resubmitTransaction(tx, 0, 0)
		assert.Error(t, err, "replacement transaction underpriced")

		assert.Eventually(t, func() bool {
			_, isPending, err := s.client.TransactionByHash(context.Background(), tx.Hash())
			return err == nil && !isPending
		}, 30*time.Second, time.Second)

		assert.Eventually(t, func() bool {
			receipt, err := s.client.TransactionReceipt(context.Background(), tx.Hash())
			return err == nil && receipt != nil
		}, 30*time.Second, time.Second)

		s.Stop()
	}
}

func testResubmitDynamicFeeTransactionWithRisingBaseFee(t *testing.T) {
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	txType := "DynamicFeeTx"
	cfgCopy := *cfg.L2Config.RelayerConfig.SenderConfig
	cfgCopy.TxType = txType

	s, err := NewSender(context.Background(), &cfgCopy, privateKey, "test", "test", types.SenderTypeUnknown, db, nil)
	assert.NoError(t, err)

	patchGuard := gomonkey.ApplyMethodFunc(s.client, "SendTransaction", func(_ context.Context, _ *gethTypes.Transaction) error {
		return nil
	})
	defer patchGuard.Reset()

	tx := gethTypes.NewTx(&gethTypes.DynamicFeeTx{
		Nonce:     s.auth.Nonce.Uint64(),
		To:        &common.Address{},
		Data:      nil,
		Gas:       21000,
		ChainID:   s.chainID,
		GasTipCap: big.NewInt(0),
		GasFeeCap: big.NewInt(0),
	})
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

	assert.Equal(t, expectedGasFeeCap.Uint64(), newTx.GasFeeCap().Uint64())
	s.Stop()
}

func testResubmitBlobTransactionWithRisingBaseFeeAndBlobBaseFee(t *testing.T) {
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	cfgCopy := *cfg.L2Config.RelayerConfig.SenderConfig
	cfgCopy.TxType = DynamicFeeTxType

	s, err := NewSender(context.Background(), &cfgCopy, privateKey, "test", "test", types.SenderTypeUnknown, db, nil)
	assert.NoError(t, err)

	patchGuard := gomonkey.ApplyMethodFunc(s.client, "SendTransaction", func(_ context.Context, _ *gethTypes.Transaction) error {
		return nil
	})
	defer patchGuard.Reset()

	sidecar, err := makeSidecar(randBlob())
	assert.NoError(t, err)
	tx := gethTypes.NewTx(&gethTypes.BlobTx{
		ChainID:    uint256.MustFromBig(s.chainID),
		Nonce:      s.auth.Nonce.Uint64(),
		GasTipCap:  uint256.MustFromBig(big.NewInt(0)),
		GasFeeCap:  uint256.MustFromBig(big.NewInt(0)),
		Gas:        21000,
		To:         common.Address{},
		Data:       nil,
		BlobFeeCap: uint256.MustFromBig(big.NewInt(1)),
		BlobHashes: sidecar.BlobHashes(),
		Sidecar:    sidecar,
	})
	baseFeePerGas := uint64(1000)
	blobBaseFeePerGas := uint64(10000000000000) // bounded by max blob base fee.
	// bump the basefee and blobbasefee by 10x
	baseFeePerGas *= 10
	blobBaseFeePerGas *= 10
	// resubmit and check that the gas fee has been adjusted accordingly
	newTx, err := s.resubmitTransaction(tx, baseFeePerGas, blobBaseFeePerGas)
	assert.NoError(t, err)

	maxGasPrice := new(big.Int).SetUint64(s.config.MaxGasPrice)
	expectedGasFeeCap := getGasFeeCap(new(big.Int).SetUint64(baseFeePerGas), tx.GasTipCap())
	if expectedGasFeeCap.Cmp(maxGasPrice) > 0 {
		expectedGasFeeCap = maxGasPrice
	}

	maxBlobGasPrice := new(big.Int).SetUint64(s.config.MaxBlobGasPrice)
	expectedBlobGasFeeCap := getBlobGasFeeCap(new(big.Int).SetUint64(blobBaseFeePerGas))
	if expectedBlobGasFeeCap.Cmp(maxBlobGasPrice) > 0 {
		expectedBlobGasFeeCap = maxBlobGasPrice
	}

	assert.Equal(t, expectedGasFeeCap.Uint64(), newTx.GasFeeCap().Uint64())
	assert.Equal(t, expectedBlobGasFeeCap.Uint64(), newTx.BlobGasFeeCap().Uint64())
	s.Stop()
}

func testResubmitNonceGappedTransaction(t *testing.T) {
	for i, txType := range txTypes {
		sqlDB, err := db.DB()
		assert.NoError(t, err)
		assert.NoError(t, migrate.ResetDB(sqlDB))

		cfgCopy := *cfg.L2Config.RelayerConfig.SenderConfig

		// Bump gas price, gas tip cap and gas fee cap just touch the minimum threshold of 10% (default config of geth).
		cfgCopy.EscalateMultipleNum = 110
		cfgCopy.EscalateMultipleDen = 100
		cfgCopy.TxType = txType

		// resubmit immediately if not nonce gapped
		cfgCopy.Confirmations = rpc.LatestBlockNumber
		cfgCopy.EscalateBlocks = 0

		// stop background check pending transaction
		cfgCopy.CheckPendingTime = math.MaxUint32

		s, err := NewSender(context.Background(), &cfgCopy, privateKey, "test", "test", types.SenderTypeUnknown, db, nil)
		assert.NoError(t, err)

		patchGuard1 := gomonkey.ApplyMethodFunc(s.client, "SendTransaction", func(_ context.Context, _ *gethTypes.Transaction) error {
			return nil
		})

		// simulating not confirmed transaction
		patchGuard2 := gomonkey.ApplyMethodFunc(s.client, "TransactionReceipt", func(_ context.Context, hash common.Hash) (*gethTypes.Receipt, error) {
			return nil, errors.New("simulated transaction receipt error")
		})

		_, err = s.SendTransaction("test-1", &common.Address{}, nil, txBlob[i], 0)
		assert.NoError(t, err)

		_, err = s.SendTransaction("test-2", &common.Address{}, nil, txBlob[i], 0)
		assert.NoError(t, err)

		s.checkPendingTransaction()

		txs, err := s.pendingTransactionOrm.GetPendingOrReplacedTransactionsBySenderType(context.Background(), s.senderType, 10)
		assert.NoError(t, err)
		assert.Len(t, txs, 3)

		assert.Equal(t, txs[0].Nonce, txs[1].Nonce)
		assert.Equal(t, txs[0].Nonce+1, txs[2].Nonce)

		// the first 2 transactions have the same nonce, with one replaced and another pending
		assert.Equal(t, types.TxStatusReplaced, txs[0].Status)
		assert.Equal(t, types.TxStatusPending, txs[1].Status)

		// the third transaction has nonce + 1, which will not be replaced due to the nonce gap,
		// thus the status should be pending
		assert.Equal(t, types.TxStatusPending, txs[2].Status)

		s.Stop()
		patchGuard1.Reset()
		patchGuard2.Reset()
	}
}

func testCheckPendingTransactionTxConfirmed(t *testing.T) {
	for _, txType := range txTypes {
		sqlDB, err := db.DB()
		assert.NoError(t, err)
		assert.NoError(t, migrate.ResetDB(sqlDB))

		cfgCopy := *cfg.L2Config.RelayerConfig.SenderConfig
		cfgCopy.TxType = txType
		s, err := NewSender(context.Background(), &cfgCopy, privateKey, "test", "test", types.SenderTypeCommitBatch, db, nil)
		assert.NoError(t, err)

		patchGuard1 := gomonkey.ApplyMethodFunc(s.client, "SendTransaction", func(_ context.Context, _ *gethTypes.Transaction) error {
			return nil
		})

		_, err = s.SendTransaction("test", &common.Address{}, nil, randBlob(), 0)
		assert.NoError(t, err)

		txs, err := s.pendingTransactionOrm.GetPendingOrReplacedTransactionsBySenderType(context.Background(), s.senderType, 1)
		assert.NoError(t, err)
		assert.Len(t, txs, 1)
		assert.Equal(t, types.TxStatusPending, txs[0].Status)
		assert.Equal(t, types.SenderTypeCommitBatch, txs[0].SenderType)

		patchGuard2 := gomonkey.ApplyMethodFunc(s.client, "TransactionReceipt", func(_ context.Context, hash common.Hash) (*gethTypes.Receipt, error) {
			return &gethTypes.Receipt{TxHash: hash, BlockNumber: big.NewInt(0), Status: gethTypes.ReceiptStatusSuccessful}, nil
		})

		s.checkPendingTransaction()
		assert.NoError(t, err)

		txs, err = s.pendingTransactionOrm.GetPendingOrReplacedTransactionsBySenderType(context.Background(), s.senderType, 1)
		assert.NoError(t, err)
		assert.Len(t, txs, 0)

		s.Stop()
		patchGuard1.Reset()
		patchGuard2.Reset()
	}
}

func testCheckPendingTransactionResubmitTxConfirmed(t *testing.T) {
	for _, txType := range txTypes {
		sqlDB, err := db.DB()
		assert.NoError(t, err)
		assert.NoError(t, migrate.ResetDB(sqlDB))

		cfgCopy := *cfg.L2Config.RelayerConfig.SenderConfig
		cfgCopy.TxType = txType
		cfgCopy.EscalateBlocks = 0
		s, err := NewSender(context.Background(), &cfgCopy, privateKey, "test", "test", types.SenderTypeFinalizeBatch, db, nil)
		assert.NoError(t, err)

		patchGuard1 := gomonkey.ApplyMethodFunc(s.client, "SendTransaction", func(_ context.Context, _ *gethTypes.Transaction) error {
			return nil
		})

		originTxHash, err := s.SendTransaction("test", &common.Address{}, nil, randBlob(), 0)
		assert.NoError(t, err)

		txs, err := s.pendingTransactionOrm.GetPendingOrReplacedTransactionsBySenderType(context.Background(), s.senderType, 1)
		assert.NoError(t, err)
		assert.Len(t, txs, 1)
		assert.Equal(t, types.TxStatusPending, txs[0].Status)
		assert.Equal(t, types.SenderTypeFinalizeBatch, txs[0].SenderType)

		patchGuard2 := gomonkey.ApplyMethodFunc(s.client, "TransactionReceipt", func(_ context.Context, hash common.Hash) (*gethTypes.Receipt, error) {
			if hash == originTxHash {
				return nil, errors.New("simulated transaction receipt error")
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
		patchGuard1.Reset()
		patchGuard2.Reset()
	}
}

func testCheckPendingTransactionReplacedTxConfirmed(t *testing.T) {
	for _, txType := range txTypes {
		sqlDB, err := db.DB()
		assert.NoError(t, err)
		assert.NoError(t, migrate.ResetDB(sqlDB))

		cfgCopy := *cfg.L2Config.RelayerConfig.SenderConfig
		cfgCopy.TxType = txType
		cfgCopy.EscalateBlocks = 0
		s, err := NewSender(context.Background(), &cfgCopy, privateKey, "test", "test", types.SenderTypeL1GasOracle, db, nil)
		assert.NoError(t, err)

		patchGuard1 := gomonkey.ApplyMethodFunc(s.client, "SendTransaction", func(_ context.Context, _ *gethTypes.Transaction) error {
			return nil
		})

		txHash, err := s.SendTransaction("test", &common.Address{}, nil, randBlob(), 0)
		assert.NoError(t, err)

		txs, err := s.pendingTransactionOrm.GetPendingOrReplacedTransactionsBySenderType(context.Background(), s.senderType, 1)
		assert.NoError(t, err)
		assert.Len(t, txs, 1)
		assert.Equal(t, types.TxStatusPending, txs[0].Status)
		assert.Equal(t, types.SenderTypeL1GasOracle, txs[0].SenderType)

		patchGuard2 := gomonkey.ApplyMethodFunc(s.client, "TransactionReceipt", func(_ context.Context, hash common.Hash) (*gethTypes.Receipt, error) {
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
			return nil, errors.New("simulated transaction receipt error")
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
		patchGuard1.Reset()
		patchGuard2.Reset()
	}
}

func testCheckPendingTransactionTxMultipleTimesWithOnlyOneTxPending(t *testing.T) {
	for _, txType := range txTypes {
		sqlDB, err := db.DB()
		assert.NoError(t, err)
		assert.NoError(t, migrate.ResetDB(sqlDB))

		cfgCopy := *cfg.L2Config.RelayerConfig.SenderConfig
		cfgCopy.TxType = txType
		cfgCopy.EscalateBlocks = 0
		s, err := NewSender(context.Background(), &cfgCopy, privateKey, "test", "test", types.SenderTypeCommitBatch, db, nil)
		assert.NoError(t, err)

		patchGuard1 := gomonkey.ApplyMethodFunc(s.client, "SendTransaction", func(_ context.Context, _ *gethTypes.Transaction) error {
			return nil
		})

		_, err = s.SendTransaction("test", &common.Address{}, nil, randBlob(), 0)
		assert.NoError(t, err)

		txs, err := s.pendingTransactionOrm.GetPendingOrReplacedTransactionsBySenderType(context.Background(), s.senderType, 1)
		assert.NoError(t, err)
		assert.Len(t, txs, 1)
		assert.Equal(t, types.TxStatusPending, txs[0].Status)
		assert.Equal(t, types.SenderTypeCommitBatch, txs[0].SenderType)

		patchGuard2 := gomonkey.ApplyMethodFunc(s.client, "TransactionReceipt", func(_ context.Context, hash common.Hash) (*gethTypes.Receipt, error) {
			return nil, errors.New("simulated transaction receipt error")
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
		patchGuard1.Reset()
		patchGuard2.Reset()
	}
}

func testBlobTransactionWithBlobhashOpContractCall(t *testing.T) {
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	blob := randBlob()
	sideCar, err := makeSidecar(blob)
	assert.NoError(t, err)
	versionedHash := sideCar.BlobHashes()[0]
	blsModulo, ok := new(big.Int).SetString("52435875175126190479447740508185965837690552500527637822603658699938581184513", 10)
	assert.True(t, ok)
	pointHash := crypto.Keccak256Hash(versionedHash.Bytes())
	pointBigInt := new(big.Int).SetBytes(pointHash.Bytes())
	pointBytes := new(big.Int).Mod(pointBigInt, blsModulo).Bytes()
	start := 32 - len(pointBytes)
	var point kzg4844.Point
	copy(point[start:], pointBytes)
	commitment := sideCar.Commitments[0]
	proof, claim, err := kzg4844.ComputeProof(blob, point)
	assert.NoError(t, err)

	var claimArray [32]byte
	copy(claimArray[:], claim[:])

	demoContractMetaData := &bind.MetaData{ABI: "[{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"claim\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"commitment\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"proof\",\"type\":\"bytes\"}],\"name\":\"verifyProof\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"}
	demoContractABI, err := demoContractMetaData.GetAbi()
	assert.NoError(t, err)

	data, err := demoContractABI.Pack(
		"verifyProof",
		claimArray,
		commitment[:],
		proof[:],
	)
	assert.NoError(t, err)

	cfgCopy := *cfg.L2Config.RelayerConfig.SenderConfig
	cfgCopy.TxType = DynamicFeeTxType
	s, err := NewSender(context.Background(), &cfgCopy, privateKey, "test", "test", types.SenderTypeL1GasOracle, db, nil)
	assert.NoError(t, err)
	defer s.Stop()

	_, err = s.SendTransaction("0", &testContractsAddress, data, blob, 0)
	assert.NoError(t, err)

	var txHash common.Hash
	assert.Eventually(t, func() bool {
		txs, err := s.pendingTransactionOrm.GetConfirmedTransactionsBySenderType(context.Background(), s.senderType, 100)
		assert.NoError(t, err)
		if len(txs) == 1 {
			txHash = common.HexToHash(txs[0].Hash)
			return true
		}
		return false
	}, 30*time.Second, time.Second)

	assert.Eventually(t, func() bool {
		receipt, err := s.client.TransactionReceipt(context.Background(), txHash)
		return err == nil && receipt.Status == gethTypes.ReceiptStatusSuccessful
	}, 30*time.Second, time.Second)
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

func testSendBlobCarryingTxOverLimit(t *testing.T) {
	cfgCopy := *cfg.L2Config.RelayerConfig.SenderConfig
	cfgCopy.TxType = "DynamicFeeTx"

	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))
	s, err := NewSender(context.Background(), &cfgCopy, privateKey, "test", "test", types.SenderTypeCommitBatch, db, nil)
	assert.NoError(t, err)

	for i := 0; i < int(cfgCopy.MaxPendingBlobTxs); i++ {
		_, err = s.SendTransaction("0", &common.Address{}, nil, randBlob(), 0)
		assert.NoError(t, err)
	}
	_, err = s.SendTransaction("0", &common.Address{}, nil, randBlob(), 0)
	assert.ErrorIs(t, err, ErrTooManyPendingBlobTxs)
	s.Stop()
}
