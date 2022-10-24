package sender_test

import (
	"context"
	"crypto/ecdsa"
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

const TX_BATCH = 100

var (
	TestConfig = &mock.TestConfig{
		L1GethTestConfig: mock.L1GethTestConfig{
			HPort: 0,
			WPort: 8576,
		},
	}

	l1gethImg docker.ImgInstance
	private   *ecdsa.PrivateKey
)

func setupEnv(t *testing.T) {
	cfg, err := config.NewConfig("../config.json")
	assert.NoError(t, err)
	prv, err := crypto.HexToECDSA(cfg.L2Config.RelayerConfig.PrivateKey)
	assert.NoError(t, err)
	private = prv
	l1gethImg = mock.NewL1Docker(t, TestConfig)
}

func TestFunction(t *testing.T) {
	// Setup
	setupEnv(t)
	t.Run("test Run sender", func(t *testing.T) {
		// set config
		cfg, err := config.NewConfig("../config.json")
		assert.NoError(t, err)
		cfg.L2Config.RelayerConfig.SenderConfig.Endpoint = l1gethImg.Endpoint()

		// create newSender
		newSender, err := sender.NewSender(context.Background(), cfg.L2Config.RelayerConfig.SenderConfig, private)
		assert.NoError(t, err)
		defer newSender.Stop()

		assert.NoError(t, err)

		// send transactions
		idCache := cmap.New()
		confirmCh := newSender.ConfirmChan()
		var (
			eg    errgroup.Group
			errCh chan error
		)
		go func() {
			for i := 0; i < TX_BATCH; i++ {
				//toAddr := common.HexToAddress("0x4592d8f8d7b001e72cb26a73e4fa1806a51ac79d")
				toAddr := common.BigToAddress(big.NewInt(int64(i + 1000)))
				id := strconv.Itoa(i + 1000)
				eg.Go(func() error {
					txHash, err := newSender.SendTransaction(id, &toAddr, big.NewInt(1), nil)
					if err != nil {
						t.Error("failed to send tx", "err", err)
						return err
					}
					t.Log("successful send a tx", "ID", id, "tx hash", txHash.String())
					idCache.Set(id, struct{}{})
					return nil
				})
			}
			errCh <- eg.Wait()
		}()

		// avoid 10 mins cause testcase panic
		after := time.After(60 * time.Second)
		for {
			select {
			case cmsg := <-confirmCh:
				t.Log("get confirmations of", "ID: ", cmsg.ID, "status: ", cmsg.IsSuccessful)
				assert.Equal(t, true, cmsg.IsSuccessful)
				_, exist := idCache.Pop(cmsg.ID)
				assert.Equal(t, true, exist)
				// Receive all confirmed txs.
				if idCache.Count() == 0 {
					return
				}
			case err := <-errCh:
				if err != nil {
					t.Errorf("failed to send tx, err: %v", err)
					return
				}
				assert.NoError(t, err)
			case <-after:
				t.Logf("newSender test failed because timeout")
				t.FailNow()
			}
		}
	})
	// Teardown
	t.Cleanup(func() {
		assert.NoError(t, l1gethImg.Stop())
	})
}
