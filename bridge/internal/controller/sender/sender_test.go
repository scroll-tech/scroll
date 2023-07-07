package sender

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"
	"strconv"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	cmap "github.com/orcaman/concurrent-map"
	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"

	"scroll-tech/common/docker"

	"scroll-tech/bridge/internal/config"
)

const TXBatch = 50

var (
	privateKeys []*ecdsa.PrivateKey
	cfg         *config.Config
	base        *docker.App
	txTypes     = []string{"LegacyTx", "AccessListTx", "DynamicFeeTx"}
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
	privateKeys = []*ecdsa.PrivateKey{priv}

	cfg.L1Config.RelayerConfig.SenderConfig.Endpoint = base.L1gethImg.Endpoint()
	cfg.L1Config.RelayerConfig.SenderConfig.CheckBalanceTime = 1
}

func TestSender(t *testing.T) {
	// Setup
	setupEnv(t)

	t.Run("test new sender", testNewSender)
	t.Run("test pending limit", testPendLimit)

	t.Run("test min gas limit", testMinGasLimit)
	t.Run("test resubmit transaction", testResubmitTransaction)
	t.Run("test resubmit transaction with rising base fee", testResubmitTransactionWithRisingBaseFee)
	t.Run("test check pending transaction", testCheckPendingTransaction)

	t.Run("test 1 account sender", func(t *testing.T) { testBatchSender(t, 1) })
	t.Run("test 3 account sender", func(t *testing.T) { testBatchSender(t, 3) })
	t.Run("test 8 account sender", func(t *testing.T) { testBatchSender(t, 8) })
}

func testNewSender(t *testing.T) {
	for _, txType := range txTypes {
		// exit by Stop()
		cfgCopy1 := *cfg.L1Config.RelayerConfig.SenderConfig
		cfgCopy1.TxType = txType
		newSender1, err := NewSender(context.Background(), &cfgCopy1, privateKeys)
		assert.NoError(t, err)
		newSender1.Stop()

		// exit by ctx.Done()
		cfgCopy2 := *cfg.L1Config.RelayerConfig.SenderConfig
		cfgCopy2.TxType = txType
		subCtx, cancel := context.WithCancel(context.Background())
		_, err = NewSender(subCtx, &cfgCopy2, privateKeys)
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
		newSender, err := NewSender(context.Background(), &cfgCopy, privateKeys)
		assert.NoError(t, err)

		for i := 0; i < 2*newSender.PendingLimit(); i++ {
			_, err = newSender.SendTransaction(strconv.Itoa(i), &common.Address{}, big.NewInt(1), nil, 0)
			assert.True(t, err == nil || (err != nil && err.Error() == "sender's pending pool is full"))
		}
		assert.True(t, newSender.PendingCount() <= newSender.PendingLimit())
		newSender.Stop()
	}
}

func testMinGasLimit(t *testing.T) {
	for _, txType := range txTypes {
		cfgCopy := *cfg.L1Config.RelayerConfig.SenderConfig
		cfgCopy.TxType = txType
		cfgCopy.Confirmations = rpc.LatestBlockNumber
		newSender, err := NewSender(context.Background(), &cfgCopy, privateKeys)
		assert.NoError(t, err)

		client, err := ethclient.Dial(cfgCopy.Endpoint)
		assert.NoError(t, err)

		// MinGasLimit = 0
		txHash0, err := newSender.SendTransaction("0", &common.Address{}, big.NewInt(1), nil, 0)
		assert.NoError(t, err)
		tx0, _, err := client.TransactionByHash(context.Background(), txHash0)
		assert.NoError(t, err)
		assert.Greater(t, tx0.Gas(), uint64(0))

		// MinGasLimit = 100000
		txHash1, err := newSender.SendTransaction("1", &common.Address{}, big.NewInt(1), nil, 100000)
		assert.NoError(t, err)
		tx1, _, err := client.TransactionByHash(context.Background(), txHash1)
		assert.NoError(t, err)
		assert.Equal(t, tx1.Gas(), uint64(150000))

		newSender.Stop()
	}
}

func testResubmitTransaction(t *testing.T) {
	for _, txType := range txTypes {
		cfgCopy := *cfg.L1Config.RelayerConfig.SenderConfig
		cfgCopy.TxType = txType
		s, err := NewSender(context.Background(), &cfgCopy, privateKeys)
		assert.NoError(t, err)
		auth := s.auths.getAccount()
		tx := types.NewTransaction(auth.Nonce.Uint64(), common.Address{}, big.NewInt(0), 0, big.NewInt(0), nil)
		feeData, err := s.getFeeData(auth, &common.Address{}, big.NewInt(0), nil, 0)
		assert.NoError(t, err)
		_, err = s.resubmitTransaction(feeData, auth, tx)
		assert.NoError(t, err)
		s.Stop()
	}
}

func testResubmitTransactionWithRisingBaseFee(t *testing.T) {
	txType := "DynamicFeeTx"

	cfgCopy := *cfg.L1Config.RelayerConfig.SenderConfig
	cfgCopy.TxType = txType
	s, err := NewSender(context.Background(), &cfgCopy, privateKeys)
	assert.NoError(t, err)
	auth := s.auths.getAccount()
	tx := types.NewTransaction(auth.Nonce.Uint64(), common.Address{}, big.NewInt(0), 0, big.NewInt(0), nil)
	feeData, err := s.getFeeData(auth, &common.Address{}, big.NewInt(0), nil, 0)
	assert.NoError(t, err)
	// bump the basefee by 10x
	s.baseFeePerGas *= 10
	// resubmit and check that the gas fee has been adjusted accordingly
	_, err = s.resubmitTransaction(feeData, auth, tx)
	assert.NoError(t, err)

	escalateMultipleNum := new(big.Int).SetUint64(s.config.EscalateMultipleNum)
	escalateMultipleDen := new(big.Int).SetUint64(s.config.EscalateMultipleDen)
	maxGasPrice := new(big.Int).SetUint64(s.config.MaxGasPrice)

	adjBaseFee := big.NewInt(0)
	adjBaseFee.SetUint64(s.baseFeePerGas)
	adjBaseFee = adjBaseFee.Mul(adjBaseFee, escalateMultipleNum)
	adjBaseFee = adjBaseFee.Div(adjBaseFee, escalateMultipleDen)

	expectedGasFeeCap := new(big.Int).Add(
		feeData.gasTipCap,
		adjBaseFee,
	)
	if expectedGasFeeCap.Cmp(maxGasPrice) > 0 {
		expectedGasFeeCap = maxGasPrice
	}

	assert.Equal(t, tx.GasFeeCap().Int64(), expectedGasFeeCap.Int64())

	s.Stop()
}

func testCheckPendingTransaction(t *testing.T) {
	for _, txType := range txTypes {
		cfgCopy := *cfg.L1Config.RelayerConfig.SenderConfig
		cfgCopy.TxType = txType
		s, err := NewSender(context.Background(), &cfgCopy, privateKeys)
		assert.NoError(t, err)

		header := &types.Header{Number: big.NewInt(100), BaseFee: big.NewInt(100)}
		confirmed := uint64(100)
		receipt := &types.Receipt{Status: types.ReceiptStatusSuccessful, BlockNumber: big.NewInt(90)}
		auth := s.auths.getAccount()
		tx := types.NewTransaction(auth.Nonce.Uint64(), common.Address{}, big.NewInt(0), 0, big.NewInt(0), nil)

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

func testBatchSender(t *testing.T, batchSize int) {
	for _, txType := range txTypes {
		for len(privateKeys) < batchSize {
			priv, err := crypto.GenerateKey()
			assert.NoError(t, err)
			privateKeys = append(privateKeys, priv)
		}

		cfgCopy := *cfg.L1Config.RelayerConfig.SenderConfig
		cfgCopy.Confirmations = rpc.LatestBlockNumber
		cfgCopy.PendingLimit = batchSize * TXBatch
		cfgCopy.TxType = txType
		newSender, err := NewSender(context.Background(), &cfgCopy, privateKeys)
		assert.NoError(t, err)

		// send transactions
		var (
			eg        errgroup.Group
			idCache   = cmap.New()
			confirmCh = newSender.ConfirmChan()
		)
		for idx := 0; idx < newSender.NumberOfAccounts(); idx++ {
			index := idx
			eg.Go(func() error {
				for i := 0; i < TXBatch; i++ {
					toAddr := common.HexToAddress("0x4592d8f8d7b001e72cb26a73e4fa1806a51ac79d")
					id := strconv.Itoa(i + index*1000)
					_, err := newSender.SendTransaction(id, &toAddr, big.NewInt(1), nil, 0)
					if errors.Is(err, ErrNoAvailableAccount) || errors.Is(err, ErrFullPending) {
						<-time.After(time.Second)
						continue
					}
					assert.NoError(t, err)
					idCache.Set(id, struct{}{})
				}
				return nil
			})
		}
		assert.NoError(t, eg.Wait())
		t.Logf("successful send batch txs, batch size: %d, total count: %d", newSender.NumberOfAccounts(), TXBatch*newSender.NumberOfAccounts())

		// avoid 10 mins cause testcase panic
		after := time.After(80 * time.Second)
		isDone := false
		for !isDone {
			select {
			case cmsg := <-confirmCh:
				assert.Equal(t, true, cmsg.IsSuccessful)
				_, exist := idCache.Pop(cmsg.ID)
				assert.Equal(t, true, exist)
				// Receive all confirmed txs.
				if idCache.Count() == 0 {
					isDone = true
				}
			case <-after:
				t.Error("newSender test failed because of timeout")
				isDone = true
			}
		}
		newSender.Stop()
	}
}
