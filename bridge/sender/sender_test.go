package sender_test

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"
	"strconv"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"

	cmap "github.com/orcaman/concurrent-map"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/docker"

	"scroll-tech/bridge/config"
	"scroll-tech/bridge/mock"
	"scroll-tech/bridge/sender"
)

const TX_BATCH = 50

var (
	TestConfig = &mock.TestConfig{
		L1GethTestConfig: mock.L1GethTestConfig{
			HPort: 0,
			WPort: 8576,
		},
		L2GethTestConfig: mock.L2GethTestConfig{
			HPort: 0,
			WPort: 8676,
		},
	}

	privateKeys []*ecdsa.PrivateKey
	cfg         *config.Config
	l2gethImg   docker.ImgInstance
)

func setupEnv(t *testing.T) {
	var err error
	cfg, err = config.NewConfig("../config.json")
	assert.NoError(t, err)

	priv, err := crypto.HexToECDSA("1212121212121212121212121212121212121212121212121212121212121212")
	assert.NoError(t, err)
	// Load default private key.
	privateKeys = []*ecdsa.PrivateKey{priv}

	l2gethImg = mock.NewTestL2Docker(t, TestConfig)
	cfg.L1Config.RelayerConfig.SenderConfig.Endpoint = l2gethImg.Endpoint()
}

func TestSender(t *testing.T) {
	// Setup
	setupEnv(t)

	t.Run("test 1 account sender", func(t *testing.T) { testBatchSender(t, 1) })
	t.Run("test 3 account sender", func(t *testing.T) { testBatchSender(t, 3) })
	t.Run("test 8 account sender", func(t *testing.T) { testBatchSender(t, 8) })

	// Teardown
	t.Cleanup(func() {
		assert.NoError(t, l2gethImg.Stop())
	})
}

func testBatchSender(t *testing.T, batchSize int) {
	for i := 0; i < batchSize; i++ {
		priv, err := crypto.GenerateKey()
		if err != nil {
			t.Fatal(err)
		}
		privateKeys = append(privateKeys, priv)
	}

	senderCfg := cfg.L1Config.RelayerConfig.SenderConfig
	senderCfg.Confirmations = 0
	newSender, err := sender.NewSender(context.Background(), senderCfg, privateKeys)
	if err != nil {
		t.Fatal(err)
	}
	defer newSender.Stop()

	// send transactions
	var (
		eg        errgroup.Group
		idCache   = cmap.New()
		confirmCh = newSender.ConfirmChan()
	)
	for idx := 0; idx < newSender.NumberOfAccounts(); idx++ {
		index := idx
		eg.Go(func() error {
			for i := 0; i < TX_BATCH; i++ {
				toAddr := common.HexToAddress("0x4592d8f8d7b001e72cb26a73e4fa1806a51ac79d")
				id := strconv.Itoa(i + index*1000)
				_, err := newSender.SendTransaction(id, &toAddr, big.NewInt(1), nil)
				if errors.Is(err, sender.ErrNoAvailableAccount) {
					<-time.After(time.Second)
					continue
				}
				if err != nil {
					return err
				}
				idCache.Set(id, struct{}{})
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		t.Error(err)
	}
	t.Logf("successful send batch txs, batch size: %d, total count: %d", newSender.NumberOfAccounts(), TX_BATCH*newSender.NumberOfAccounts())

	// avoid 10 mins cause testcase panic
	after := time.After(80 * time.Second)
	for {
		select {
		case cmsg := <-confirmCh:
			assert.Equal(t, true, cmsg.IsSuccessful)
			_, exist := idCache.Pop(cmsg.ID)
			assert.Equal(t, true, exist)
			// Receive all confirmed txs.
			if idCache.Count() == 0 {
				return
			}
		case <-after:
			t.Error("newSender test failed because of timeout")
			return
		}
	}
}
