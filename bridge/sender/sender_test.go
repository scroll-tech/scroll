package sender_test

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	cmap "github.com/orcaman/concurrent-map"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"

	"scroll-tech/internal/docker"
	"scroll-tech/internal/mock"

	"scroll-tech/bridge/config"
	. "scroll-tech/bridge/sender"
)

const TX_BATCH = 100

var (
	TestConfig = &mock.TestConfig{
		L1GethTestConfig: mock.L1GethTestConfig{
			HPort: 0,
			WPort: 8575,
		},
	}

	l1gethImg docker.ImgInstance
	private   *ecdsa.PrivateKey
)

func setupEnv(t *testing.T) {
	prv, err := crypto.HexToECDSA("ad29c7c341a23f04851b6c8602c7c74b98e3fc9488582791bda60e0e261f9cbb")
	assert.NoError(t, err)
	private = prv
	l1gethImg = mock.NewTestL1Docker(t, TestConfig)
}

func TestFunction(t *testing.T) {
	// Setup
	setupEnv(t)
	t.Run("test Run sender", func(t *testing.T) {
		// set config
		cfg, err := config.NewConfig("../../config.json")
		assert.NoError(t, err)
		cfg.L2Config.RelayerConfig.SenderConfig.Endpoint = l1gethImg.Endpoint()

		// create sender
		sender, err := NewSender(context.Background(), cfg.L2Config.RelayerConfig.SenderConfig, private)
		assert.NoError(t, err)
		defer sender.Stop()

		// create geth client
		client, err := ethclient.Dial(l1gethImg.Endpoint())
		assert.NoError(t, err)

		// subscribe new header
		headerCh := make(chan *types.Header, 4)
		subs, err := client.SubscribeNewHead(context.Background(), headerCh)
		assert.NoError(t, err)
		defer subs.Unsubscribe()

		// send transactions
		idCache := cmap.New()
		txFinished := uint64(0)
		go func() {
			for i := 0; i < TX_BATCH; i++ {
				toAddr := common.BigToAddress(big.NewInt(int64(i + 1000)))
				id := strconv.Itoa(i + 1000)
				txHash, err := sender.SendTransaction(id, &toAddr, big.NewInt(1), nil)
				if err != nil {
					t.Error("failed to send tx", "err", err)
					continue
				}
				t.Log("successful send a tx", "ID", id, "tx hash", txHash.String())
				idCache.Set(id, struct{}{})
			}
			atomic.StoreUint64(&txFinished, 1)
		}()

		// confirm tx
		confirmCh := sender.ConfirmChan()
		for {
			select {
			case head := <-headerCh:
				block, err := client.BlockByNumber(context.Background(), head.Number)
				assert.NoError(t, err)
				sender.CheckPendingTransaction(block)
			case cmsg := <-confirmCh:
				_, exist := idCache.Pop(cmsg.ID)
				assert.Equal(t, true, exist)
			case <-time.Tick(3 * time.Second):
				if atomic.LoadUint64(&txFinished) == 1 && idCache.Count() == 0 {
					return
				}
			case <-time.After(30 * time.Second):
				assert.Equal(t, 0, idCache.Count())
				return
			}
		}
	})

	// Teardown
	t.Cleanup(func() {
		assert.NoError(t, l1gethImg.Stop())
	})
}
