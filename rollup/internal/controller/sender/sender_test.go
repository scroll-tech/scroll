package sender

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"
	"strconv"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/docker"

	"scroll-tech/rollup/internal/config"
)

const TXBatch = 50

var (
	privateKey *ecdsa.PrivateKey
	cfg        *config.Config
	base       *docker.App
	txTypes    = []string{"LegacyTx", "AccessListTx", "DynamicFeeTx"}
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
	base.RunImages(t)
	priv, err := crypto.HexToECDSA("1212121212121212121212121212121212121212121212121212121212121212")
	assert.NoError(t, err)
	// Load default private key.
	privateKey = priv

	cfg.L1Config.RelayerConfig.SenderConfig.Endpoint = base.L1gethImg.Endpoint()
	cfg.L1Config.RelayerConfig.SenderConfig.CheckBalanceTime = 1
}

func TestSender(t *testing.T) {
	// Setup
	setupEnv(t)

	t.Run("test new sender", testNewSender)
	t.Run("test pending limit", testPendLimit)
	t.Run("test fallback gas limit", testFallbackGasLimit)
	t.Run("test resubmit transaction", testResubmitTransaction)
	t.Run("test resubmit transaction with rising base fee", testResubmitTransactionWithRisingBaseFee)
	t.Run("test check pending transaction", testCheckPendingTransaction)
}

func testNewSender(t *testing.T) {
	for _, txType := range txTypes {
		// exit by Stop()
		cfgCopy1 := *cfg.L1Config.RelayerConfig.SenderConfig
		cfgCopy1.TxType = txType
		newSender1, err := NewSender(context.Background(), &cfgCopy1, privateKey, "test", "test", nil)
		assert.NoError(t, err)
		newSender1.Stop()

		// exit by ctx.Done()
		cfgCopy2 := *cfg.L1Config.RelayerConfig.SenderConfig
		cfgCopy2.TxType = txType
		subCtx, cancel := context.WithCancel(context.Background())
		_, err = NewSender(subCtx, &cfgCopy2, privateKey, "test", "test", nil)
		assert.NoError(t, err)
		cancel()
	}
}

func testPendLimit(t *testing.T) {
	for _, txType := range txTypes {
		cfgCopy := *cfg.L1Config.RelayerConfig.SenderConfig
		cfgCopy.TxType = txType
		cfgCopy.Confirmations = rpc.LatestBlockNumber
		cfgCopy.PendingLimit = 2
		newSender, err := NewSender(context.Background(), &cfgCopy, privateKey, "test", "test", nil)
		assert.NoError(t, err)

		for i := 0; i < 2*newSender.PendingLimit(); i++ {
			_, err = newSender.SendTransaction(strconv.Itoa(i), &common.Address{}, big.NewInt(1), nil, 0)
			assert.True(t, err == nil || (err != nil && err.Error() == "sender's pending pool is full"))
		}
		assert.True(t, newSender.PendingCount() <= newSender.PendingLimit())
		newSender.Stop()
	}
}

func testFallbackGasLimit(t *testing.T) {
	for _, txType := range txTypes {
		cfgCopy := *cfg.L1Config.RelayerConfig.SenderConfig
		cfgCopy.TxType = txType
		cfgCopy.Confirmations = rpc.LatestBlockNumber
		s, err := NewSender(context.Background(), &cfgCopy, privateKey, "test", "test", nil)
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

func testResubmitTransaction(t *testing.T) {
	for _, txType := range txTypes {
		cfgCopy := *cfg.L1Config.RelayerConfig.SenderConfig
		cfgCopy.TxType = txType
		s, err := NewSender(context.Background(), &cfgCopy, privateKey, "test", "test", nil)
		assert.NoError(t, err)
		tx := types.NewTransaction(s.auth.Nonce.Uint64(), common.Address{}, big.NewInt(0), 0, big.NewInt(0), nil)
		feeData, err := s.getFeeData(s.auth, &common.Address{}, big.NewInt(0), nil, 0)
		assert.NoError(t, err)
		_, err = s.resubmitTransaction(feeData, s.auth, tx)
		assert.NoError(t, err)
		s.Stop()
	}
}

func testResubmitTransactionWithRisingBaseFee(t *testing.T) {
	txType := "DynamicFeeTx"

	cfgCopy := *cfg.L1Config.RelayerConfig.SenderConfig
	cfgCopy.TxType = txType
	s, err := NewSender(context.Background(), &cfgCopy, privateKey, "test", "test", nil)
	assert.NoError(t, err)
	tx := types.NewTransaction(s.auth.Nonce.Uint64(), common.Address{}, big.NewInt(0), 0, big.NewInt(0), nil)
	s.baseFeePerGas = 1000
	feeData, err := s.getFeeData(s.auth, &common.Address{}, big.NewInt(0), nil, 0)
	assert.NoError(t, err)
	// bump the basefee by 10x
	s.baseFeePerGas *= 10
	// resubmit and check that the gas fee has been adjusted accordingly
	newTx, err := s.resubmitTransaction(feeData, s.auth, tx)
	assert.NoError(t, err)

	escalateMultipleNum := new(big.Int).SetUint64(s.config.EscalateMultipleNum)
	escalateMultipleDen := new(big.Int).SetUint64(s.config.EscalateMultipleDen)
	maxGasPrice := new(big.Int).SetUint64(s.config.MaxGasPrice)

	adjBaseFee := new(big.Int)
	adjBaseFee.SetUint64(s.baseFeePerGas)
	adjBaseFee = adjBaseFee.Mul(adjBaseFee, escalateMultipleNum)
	adjBaseFee = adjBaseFee.Div(adjBaseFee, escalateMultipleDen)

	expectedGasFeeCap := new(big.Int).Add(feeData.gasTipCap, adjBaseFee)
	if expectedGasFeeCap.Cmp(maxGasPrice) > 0 {
		expectedGasFeeCap = maxGasPrice
	}

	assert.Equal(t, expectedGasFeeCap.Int64(), newTx.GasFeeCap().Int64())

	s.Stop()
}

func testCheckPendingTransaction(t *testing.T) {
	for _, txType := range txTypes {
		cfgCopy := *cfg.L1Config.RelayerConfig.SenderConfig
		cfgCopy.TxType = txType
		s, err := NewSender(context.Background(), &cfgCopy, privateKey, "test", "test", nil)
		assert.NoError(t, err)

		header := &types.Header{Number: big.NewInt(100), BaseFee: big.NewInt(100)}
		confirmed := uint64(100)
		receipt := &types.Receipt{Status: types.ReceiptStatusSuccessful, BlockNumber: big.NewInt(90)}
		tx := types.NewTransaction(s.auth.Nonce.Uint64(), common.Address{}, big.NewInt(0), 0, big.NewInt(0), nil)

		testCases := []struct {
			name          string
			receipt       *types.Receipt
			receiptErr    error
			resubmitErr   error
			expectedCount int
			expectedFound bool
		}{
			{
				name:          "Normal case, transaction receipt exists and successful",
				receipt:       receipt,
				receiptErr:    nil,
				resubmitErr:   nil,
				expectedCount: 0,
				expectedFound: false,
			},
			{
				name:          "Resubmit case, resubmitTransaction error (not nonce) case",
				receipt:       receipt,
				receiptErr:    errors.New("receipt error"),
				resubmitErr:   errors.New("resubmit error"),
				expectedCount: 1,
				expectedFound: true,
			},
			{
				name:          "Resubmit case, resubmitTransaction success case",
				receipt:       receipt,
				receiptErr:    errors.New("receipt error"),
				resubmitErr:   nil,
				expectedCount: 1,
				expectedFound: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				var c *ethclient.Client
				patchGuard := gomonkey.ApplyMethodFunc(c, "TransactionReceipt", func(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
					return tc.receipt, tc.receiptErr
				})
				patchGuard.ApplyPrivateMethod(s, "resubmitTransaction",
					func(feeData *FeeData, auth *bind.TransactOpts, tx *types.Transaction) (*types.Transaction, error) {
						return tx, tc.resubmitErr
					},
				)

				pendingTx := &PendingTransaction{id: "abc", tx: tx, submitAt: header.Number.Uint64() - s.config.EscalateBlocks - 1}
				s.pendingTxs.Set(pendingTx.id, pendingTx)
				s.checkPendingTransaction(header, confirmed)

				if tc.receiptErr == nil {
					expectedConfirmation := &Confirmation{
						ID:           pendingTx.id,
						IsSuccessful: tc.receipt.Status == types.ReceiptStatusSuccessful,
						TxHash:       pendingTx.tx.Hash(),
					}
					actualConfirmation := <-s.confirmCh
					assert.Equal(t, expectedConfirmation, actualConfirmation)
				}

				if tc.expectedFound && tc.resubmitErr == nil {
					actualPendingTx, found := s.pendingTxs.Get(pendingTx.id)
					assert.Equal(t, true, found)
					assert.Equal(t, header.Number.Uint64(), actualPendingTx.submitAt)
				}

				_, found := s.pendingTxs.Get(pendingTx.id)
				assert.Equal(t, tc.expectedFound, found)
				assert.Equal(t, tc.expectedCount, s.pendingTxs.Count())
				patchGuard.Reset()
			})
		}
		s.Stop()
	}
}
