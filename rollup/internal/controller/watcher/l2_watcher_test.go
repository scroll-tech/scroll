package watcher

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"
	"strconv"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/rpc"
	"github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/database"
	cutils "scroll-tech/common/utils"

	bridgeAbi "scroll-tech/bridge/abi"
	"scroll-tech/bridge/internal/controller/sender"
	"scroll-tech/bridge/internal/orm"
	"scroll-tech/bridge/internal/utils"
	"scroll-tech/bridge/mock_bridge"
)

func setupL2Watcher(t *testing.T) (*L2WatcherClient, *gorm.DB) {
	db := setupDB(t)
	l2cfg := cfg.L2Config
	watcher := NewL2WatcherClient(context.Background(), l2Cli, l2cfg.Confirmations, l2cfg.L2MessengerAddress,
		l2cfg.L2MessageQueueAddress, l2cfg.WithdrawTrieRootSlot, db, nil)
	return watcher, db
}

func testCreateNewWatcherAndStop(t *testing.T) {
	wc, db := setupL2Watcher(t)
	subCtx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		defer database.CloseDB(db)
	}()

	loopToFetchEvent(subCtx, wc)

	l1cfg := cfg.L1Config
	l1cfg.RelayerConfig.SenderConfig.Confirmations = rpc.LatestBlockNumber
	newSender, err := sender.NewSender(context.Background(), l1cfg.RelayerConfig.SenderConfig, l1cfg.RelayerConfig.GasOracleSenderPrivateKey, "test", "test", nil)
	assert.NoError(t, err)

	// Create several transactions and commit to block
	numTransactions := 3
	toAddress := common.HexToAddress("0x4592d8f8d7b001e72cb26a73e4fa1806a51ac79d")
	for i := 0; i < numTransactions; i++ {
		_, err = newSender.SendTransaction(strconv.Itoa(1000+i), &toAddress, big.NewInt(1000000000), nil, 0)
		assert.NoError(t, err)
		<-newSender.ConfirmChan()
	}

	blockNum, err := l2Cli.BlockNumber(context.Background())
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, blockNum, uint64(numTransactions))
}

func testFetchRunningMissingBlocks(t *testing.T) {
	_, db := setupL2Watcher(t)
	defer database.CloseDB(db)

	auth := prepareAuth(t, l2Cli, cfg.L2Config.RelayerConfig.GasOracleSenderPrivateKey)

	// deploy mock bridge
	_, tx, _, err := mock_bridge.DeployMockBridgeL2(auth, l2Cli)
	assert.NoError(t, err)
	address, err := bind.WaitDeployed(context.Background(), l2Cli, tx)
	assert.NoError(t, err)

	l2BlockOrm := orm.NewL2Block(db)
	ok := cutils.TryTimes(10, func() bool {
		latestHeight, err := l2Cli.BlockNumber(context.Background())
		if err != nil {
			return false
		}
		wc := prepareWatcherClient(l2Cli, db, address)
		wc.TryFetchRunningMissingBlocks(latestHeight)
		fetchedHeight, err := l2BlockOrm.GetL2BlocksLatestHeight(context.Background())
		return err == nil && fetchedHeight == latestHeight
	})
	assert.True(t, ok)
}

func prepareWatcherClient(l2Cli *ethclient.Client, db *gorm.DB, contractAddr common.Address) *L2WatcherClient {
	confirmations := rpc.LatestBlockNumber
	return NewL2WatcherClient(context.Background(), l2Cli, confirmations, contractAddr, contractAddr, common.Hash{}, db, nil)
}

func prepareAuth(t *testing.T, l2Cli *ethclient.Client, privateKey *ecdsa.PrivateKey) *bind.TransactOpts {
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(53077))
	assert.NoError(t, err)
	auth.Value = big.NewInt(0) // in wei
	assert.NoError(t, err)
	auth.GasPrice, err = l2Cli.SuggestGasPrice(context.Background())
	assert.NoError(t, err)
	auth.GasLimit = 500000
	return auth
}

func loopToFetchEvent(subCtx context.Context, watcher *L2WatcherClient) {
	go cutils.Loop(subCtx, 2*time.Second, watcher.FetchContractEvent)
}

func testParseBridgeEventLogsL2RelayedMessageEventSignature(t *testing.T) {
	watcher, db := setupL2Watcher(t)
	defer database.CloseDB(db)

	logs := []gethTypes.Log{
		{
			Topics:      []common.Hash{bridgeAbi.L2RelayedMessageEventSignature},
			BlockNumber: 100,
			TxHash:      common.HexToHash("0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347"),
		},
	}

	convey.Convey("unpack RelayedMessage log failure", t, func() {
		targetErr := errors.New("UnpackLog RelayedMessage failure")
		patchGuard := gomonkey.ApplyFunc(utils.UnpackLog, func(c *abi.ABI, out interface{}, event string, log gethTypes.Log) error {
			return targetErr
		})
		defer patchGuard.Reset()

		relayedMessages, err := watcher.parseBridgeEventLogs(logs)
		assert.EqualError(t, err, targetErr.Error())
		assert.Empty(t, relayedMessages)
	})

	convey.Convey("L2RelayedMessageEventSignature success", t, func() {
		msgHash := common.HexToHash("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5")
		patchGuard := gomonkey.ApplyFunc(utils.UnpackLog, func(c *abi.ABI, out interface{}, event string, log gethTypes.Log) error {
			tmpOut := out.(*bridgeAbi.L2RelayedMessageEvent)
			tmpOut.MessageHash = msgHash
			return nil
		})
		defer patchGuard.Reset()

		relayedMessages, err := watcher.parseBridgeEventLogs(logs)
		assert.NoError(t, err)
		assert.Len(t, relayedMessages, 1)
		assert.Equal(t, relayedMessages[0].msgHash, msgHash)
	})
}

func testParseBridgeEventLogsL2FailedRelayedMessageEventSignature(t *testing.T) {
	watcher, db := setupL2Watcher(t)
	defer database.CloseDB(db)

	logs := []gethTypes.Log{
		{
			Topics:      []common.Hash{bridgeAbi.L2FailedRelayedMessageEventSignature},
			BlockNumber: 100,
			TxHash:      common.HexToHash("0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347"),
		},
	}

	convey.Convey("unpack FailedRelayedMessage log failure", t, func() {
		targetErr := errors.New("UnpackLog FailedRelayedMessage failure")
		patchGuard := gomonkey.ApplyFunc(utils.UnpackLog, func(c *abi.ABI, out interface{}, event string, log gethTypes.Log) error {
			return targetErr
		})
		defer patchGuard.Reset()

		relayedMessages, err := watcher.parseBridgeEventLogs(logs)
		assert.EqualError(t, err, targetErr.Error())
		assert.Empty(t, relayedMessages)
	})

	convey.Convey("L2FailedRelayedMessageEventSignature success", t, func() {
		msgHash := common.HexToHash("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5")
		patchGuard := gomonkey.ApplyFunc(utils.UnpackLog, func(c *abi.ABI, out interface{}, event string, log gethTypes.Log) error {
			tmpOut := out.(*bridgeAbi.L2FailedRelayedMessageEvent)
			tmpOut.MessageHash = msgHash
			return nil
		})
		defer patchGuard.Reset()

		relayedMessages, err := watcher.parseBridgeEventLogs(logs)
		assert.NoError(t, err)
		assert.Len(t, relayedMessages, 1)
		assert.Equal(t, relayedMessages[0].msgHash, msgHash)
	})
}
