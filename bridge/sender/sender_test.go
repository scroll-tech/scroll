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
	"github.com/scroll-tech/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/docker"

	"scroll-tech/bridge/config"
	"scroll-tech/bridge/sender"
)

const TXBatch = 50

var (
	privateKeys []*ecdsa.PrivateKey
	cfg         *config.Config
	base        *docker.App
)

func TestMain(m *testing.M) {
	base = docker.NewDockerApp()

	m.Run()

	base.Free()
}

func setupEnv(t *testing.T) {
	var err error
	cfg, err = config.NewConfig("../config.json")
	assert.NoError(t, err)
	base.RunImages(t)
	priv, err := crypto.HexToECDSA("1212121212121212121212121212121212121212121212121212121212121212")
	assert.NoError(t, err)
	// Load default private key.
	privateKeys = []*ecdsa.PrivateKey{priv}

	cfg.L1Config.RelayerConfig.SenderConfig.Endpoint = base.L2GethEndpoint()
}

func TestSender(t *testing.T) {
	// Setup
	setupEnv(t)

	t.Run("testLoadOrSendTx", testLoadOrSendTx)
	t.Run("test 1 account sender", func(t *testing.T) { testBatchSender(t, 1) })
	t.Run("test 3 account sender", func(t *testing.T) { testBatchSender(t, 3) })
	t.Run("test 8 account sender", func(t *testing.T) { testBatchSender(t, 8) })
}

func testLoadOrSendTx(t *testing.T) {
	senderCfg := cfg.L1Config.RelayerConfig.SenderConfig
	senderCfg.Confirmations = 0
	newSender, err := sender.NewSender(context.Background(), senderCfg, privateKeys)
	if err != nil {
		t.Fatal(err)
	}

	newSender2, err := sender.NewSender(context.Background(), senderCfg, privateKeys)
	if err != nil {
		t.Fatal(err)
	}

	toAddr := common.HexToAddress("0x4592d8f8d7b001e72cb26a73e4fa1806a51ac79d")
	id := "aaa"

	hash, err := newSender.SendTransaction(id, &toAddr, big.NewInt(0), nil)
	assert.NoError(t, err)

	err = newSender2.LoadOrSendTx(hash, id, &toAddr, big.NewInt(0), nil)
	assert.NoError(t, err)

	select {
	case cfm := <-newSender2.ConfirmChan():
		assert.Equal(t, true, cfm.IsSuccessful)
		assert.Equal(t, hash, cfm.TxHash)
		assert.Equal(t, id, cfm.ID)
	case <-time.After(time.Second * 10):
		t.Error("testLoadOrSendTx test failed because of timeout")
	}
}

func testBatchSender(t *testing.T, batchSize int) {
	for len(privateKeys) < batchSize {
		priv, err := crypto.GenerateKey()
		if err != nil {
			t.Fatal(err)
		}
		privateKeys = append(privateKeys, priv)
	}

	senderCfg := cfg.L1Config.RelayerConfig.SenderConfig
	senderCfg.Confirmations = rpc.LatestBlockNumber
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
			for i := 0; i < TXBatch; i++ {
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
	t.Logf("successful send batch txs, batch size: %d, total count: %d", newSender.NumberOfAccounts(), TXBatch*newSender.NumberOfAccounts())

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
