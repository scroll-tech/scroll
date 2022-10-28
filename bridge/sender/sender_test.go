package sender_test

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"github.com/scroll-tech/go-ethereum/crypto"
	"math/big"
	"runtime"
	"strconv"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"

	cmap "github.com/orcaman/concurrent-map"
	"github.com/scroll-tech/go-ethereum/common"
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
		L2GethTestConfig: mock.L2GethTestConfig{
			HPort: 0,
			WPort: 8676,
		},
	}

	privateKeys []*ecdsa.PrivateKey
	cfg         *config.Config
	l1gethImg   docker.ImgInstance
	l2gethImg   docker.ImgInstance
)

func setupEnv(t *testing.T) {
	var err error
	cfg, err = config.NewConfig("../config.json")
	assert.NoError(t, err)

	// Load default private key.
	privateKeys = append(cfg.L1Config.RelayerConfig.PrivateKeyList, cfg.L2Config.RelayerConfig.PrivateKeyList...)

	l1gethImg = mock.NewTestL1Docker(t, TestConfig)
	cfg.L2Config.RelayerConfig.SenderConfig.Endpoint = l1gethImg.Endpoint()
	l2gethImg = mock.NewTestL2Docker(t, TestConfig)
	cfg.L1Config.RelayerConfig.SenderConfig.Endpoint = l2gethImg.Endpoint()
}

func TestFunction(t *testing.T) {
	// Setup
	setupEnv(t)

	t.Run("TestL1Sender", testL1Sender)

	t.Run("TestL2Sender", testL2Sender)

	// Teardown
	t.Cleanup(func() {
		assert.NoError(t, l1gethImg.Stop())
		assert.NoError(t, l2gethImg.Stop())
	})
}

func testL1Sender(t *testing.T) {
	// create newSender
	newSender, err := sender.NewSender(context.Background(), cfg.L2Config.RelayerConfig.SenderConfig, privateKeys)
	assert.NoError(t, err)
	defer newSender.Stop()

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
}

func testL2Sender(t *testing.T) {
	for i := 0; i < 10; i++ {
		priv, err := crypto.GenerateKey()
		if err != nil {
			t.Fatal(err)
		}
		privateKeys = append(privateKeys, priv)
	}

	newSender, err := sender.NewSender(context.Background(), cfg.L1Config.RelayerConfig.SenderConfig, privateKeys)
	if err != nil {
		t.Fatal(err)
	}

	// send transactions
	var (
		idCache   = cmap.New()
		confirmCh = newSender.ConfirmChan()
		threads   = runtime.GOMAXPROCS(-1)
		eg        errgroup.Group
	)
	for th := 0; th < threads; th++ {
		index := th
		eg.Go(func() error {
			for i := 0; i < TX_BATCH/2; i++ {
				//toAddr := common.HexToAddress("0x4592d8f8d7b001e72cb26a73e4fa1806a51ac79d")
				id := strconv.Itoa(i + index*1000)
				toAddr := common.BigToAddress(big.NewInt(int64(i + index*1000)))
				txHash, err := newSender.SendTransaction(id, &toAddr, big.NewInt(1), nil)
				if errors.Is(err, sender.ErrEmptyAccount) {
					<-time.After(time.Second)
					continue
				}
				if err != nil {
					t.Error("failed to send tx", "err", err)
					return err
				}
				t.Log("successful send a tx", "ID", id, "tx hash", txHash.String())
				idCache.Set(id, struct{}{})
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		t.Fatal(err)
	}

	// avoid 10 mins cause testcase panic
	after := time.After(80 * time.Second)
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
		case <-after:
			t.Logf("newSender test failed because timeout")
			t.FailNow()
		}
	}
}
