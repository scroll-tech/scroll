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

	l2gethImg = docker.NewTestL2Docker(t)
	cfg.L1Config.RelayerConfig.SenderConfig.Endpoint = l2gethImg.Endpoint()
}

func TestSender(t *testing.T) {
	// Setup
	setupEnv(t)

	t.Run("testSenderMsg", testSenderMsg)
	t.Run("test 1 account sender", func(t *testing.T) { testBatchSender(t, 1) })
	t.Run("test 3 account sender", func(t *testing.T) { testBatchSender(t, 3) })
	t.Run("test 8 account sender", func(t *testing.T) { testBatchSender(t, 8) })

	// Teardown
	t.Cleanup(func() {
		assert.NoError(t, l2gethImg.Stop())
	})
}

type confirmMsg struct {
	TxType string
	ID     string
}

func testSenderMsg(t *testing.T) {
	newSender, err := sender.NewSender(context.Background(), cfg.L1Config.RelayerConfig.SenderConfig, privateKeys)
	if err != nil {
		t.Fatal(err)
	}

	toAddr := common.HexToAddress("0x4592d8f8d7b001e72cb26a73e4fa1806a51ac79d")

	msg0 := &confirmMsg{ID: "1", TxType: "1"}
	_, err = newSender.SendTransaction(msg0, &toAddr, nil, nil)
	assert.NoError(t, err)

	_, err = newSender.SendTransaction(msg0, &toAddr, nil, nil)
	assert.Error(t, err)
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
			_, exist := idCache.Pop(cmsg.TxMeta.(string))
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
