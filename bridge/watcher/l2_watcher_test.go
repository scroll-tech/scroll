package watcher

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"
	"strconv"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	geth_types "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/rpc"
	"github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/types"

	bridge_abi "scroll-tech/bridge/abi"
	"scroll-tech/bridge/mock_bridge"
	"scroll-tech/bridge/sender"
	"scroll-tech/bridge/utils"

	cutils "scroll-tech/common/utils"

	"scroll-tech/database"
	"scroll-tech/database/migrate"
)

func setupL2Watcher(t *testing.T) *L2WatcherClient {
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	l2cfg := cfg.L2Config
	watcher := NewL2WatcherClient(context.Background(), l2Cli, l2cfg.Confirmations, l2cfg.L2MessengerAddress, l2cfg.L2MessageQueueAddress, l2cfg.WithdrawTrieRootSlot, db)
	return watcher
}

func testCreateNewWatcherAndStop(t *testing.T) {
	// Create db handler and reset db.
	l2db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(l2db.GetDB().DB))
	ctx := context.Background()
	subCtx, cancel := context.WithCancel(ctx)
	defer func() {
		cancel()
		l2db.Close()
	}()

	l2cfg := cfg.L2Config
	wc := NewL2WatcherClient(context.Background(), l2Cli, l2cfg.Confirmations, l2cfg.L2MessengerAddress, l2cfg.L2MessageQueueAddress, l2cfg.WithdrawTrieRootSlot, l2db)
	loopToFetchEvent(subCtx, wc)

	l1cfg := cfg.L1Config
	l1cfg.RelayerConfig.SenderConfig.Confirmations = rpc.LatestBlockNumber
	newSender, err := sender.NewSender(context.Background(), l1cfg.RelayerConfig.SenderConfig, l1cfg.RelayerConfig.MessageSenderPrivateKeys)
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

func testMonitorBridgeContract(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	ctx := context.Background()
	subCtx, cancel := context.WithCancel(ctx)

	defer func() {
		cancel()
		db.Close()
	}()

	l2cfg := cfg.L2Config
	wc := NewL2WatcherClient(context.Background(), l2Cli, l2cfg.Confirmations, l2cfg.L2MessengerAddress, l2cfg.L2MessageQueueAddress, l2cfg.WithdrawTrieRootSlot, db)
	loopToFetchEvent(subCtx, wc)

	previousHeight, err := l2Cli.BlockNumber(context.Background())
	assert.NoError(t, err)

	auth := prepareAuth(t, l2Cli, cfg.L2Config.RelayerConfig.MessageSenderPrivateKeys[0])

	// deploy mock bridge
	_, tx, instance, err := mock_bridge.DeployMockBridgeL2(auth, l2Cli)
	assert.NoError(t, err)
	address, err := bind.WaitDeployed(context.Background(), l2Cli, tx)
	assert.NoError(t, err)

	rc := prepareWatcherClient(l2Cli, db, address)
	loopToFetchEvent(subCtx, rc)
	// Call mock_bridge instance sendMessage to trigger emit events
	toAddress := common.HexToAddress("0x4592d8f8d7b001e72cb26a73e4fa1806a51ac79d")
	message := []byte("testbridgecontract")
	fee := big.NewInt(0)
	gasLimit := big.NewInt(1)

	tx, err = instance.SendMessage(auth, toAddress, fee, message, gasLimit)
	assert.NoError(t, err)
	receipt, err := bind.WaitMined(context.Background(), l2Cli, tx)
	if receipt.Status != geth_types.ReceiptStatusSuccessful || err != nil {
		t.Fatalf("Call failed")
	}

	// extra block mined
	toAddress = common.HexToAddress("0x4592d8f8d7b001e72cb26a73e4fa1806a51ac79d")
	message = []byte("testbridgecontract")
	tx, err = instance.SendMessage(auth, toAddress, fee, message, gasLimit)
	assert.NoError(t, err)
	receipt, err = bind.WaitMined(context.Background(), l2Cli, tx)
	if receipt.Status != geth_types.ReceiptStatusSuccessful || err != nil {
		t.Fatalf("Call failed")
	}

	// wait for dealing time
	<-time.After(6 * time.Second)

	var latestHeight uint64
	latestHeight, err = l2Cli.BlockNumber(context.Background())
	assert.NoError(t, err)
	t.Log("Latest height is", latestHeight)

	// check if we successfully stored events
	height, err := db.GetLayer2LatestWatchedHeight()
	assert.NoError(t, err)
	t.Log("Height in DB is", height)
	assert.Greater(t, height, int64(previousHeight))
	msgs, err := db.GetL2Messages(map[string]interface{}{"status": types.MsgPending})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(msgs))
}

func testFetchMultipleSentMessageInOneBlock(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	ctx := context.Background()
	subCtx, cancel := context.WithCancel(ctx)

	defer func() {
		cancel()
		db.Close()
	}()

	previousHeight, err := l2Cli.BlockNumber(context.Background()) // shallow the global previousHeight
	assert.NoError(t, err)

	auth := prepareAuth(t, l2Cli, cfg.L2Config.RelayerConfig.MessageSenderPrivateKeys[0])

	_, trx, instance, err := mock_bridge.DeployMockBridgeL2(auth, l2Cli)
	assert.NoError(t, err)
	address, err := bind.WaitDeployed(context.Background(), l2Cli, trx)
	assert.NoError(t, err)

	wc := prepareWatcherClient(l2Cli, db, address)
	loopToFetchEvent(subCtx, wc)

	// Call mock_bridge instance sendMessage to trigger emit events multiple times
	numTransactions := 4
	var tx *geth_types.Transaction

	for i := 0; i < numTransactions; i++ {
		addr := common.HexToAddress("0x1c5a77d9fa7ef466951b2f01f724bca3a5820b63")
		nonce, nounceErr := l2Cli.PendingNonceAt(context.Background(), addr)
		assert.NoError(t, nounceErr)
		auth.Nonce = big.NewInt(int64(nonce))
		toAddress := common.HexToAddress("0x4592d8f8d7b001e72cb26a73e4fa1806a51ac79d")
		message := []byte("testbridgecontract")
		fee := big.NewInt(0)
		gasLimit := big.NewInt(1)
		tx, err = instance.SendMessage(auth, toAddress, fee, message, gasLimit)
		assert.NoError(t, err)
	}

	receipt, err := bind.WaitMined(context.Background(), l2Cli, tx)
	if receipt.Status != geth_types.ReceiptStatusSuccessful || err != nil {
		t.Fatalf("Call failed")
	}

	// extra block mined
	addr := common.HexToAddress("0x1c5a77d9fa7ef466951b2f01f724bca3a5820b63")
	nonce, nounceErr := l2Cli.PendingNonceAt(context.Background(), addr)
	assert.NoError(t, nounceErr)
	auth.Nonce = big.NewInt(int64(nonce))
	toAddress := common.HexToAddress("0x4592d8f8d7b001e72cb26a73e4fa1806a51ac79d")
	message := []byte("testbridgecontract")
	fee := big.NewInt(0)
	gasLimit := big.NewInt(1)
	tx, err = instance.SendMessage(auth, toAddress, fee, message, gasLimit)
	assert.NoError(t, err)
	receipt, err = bind.WaitMined(context.Background(), l2Cli, tx)
	if receipt.Status != geth_types.ReceiptStatusSuccessful || err != nil {
		t.Fatalf("Call failed")
	}

	// wait for dealing time
	<-time.After(6 * time.Second)

	// check if we successfully stored events
	height, err := db.GetLayer2LatestWatchedHeight()
	assert.NoError(t, err)
	t.Log("LatestHeight is", height)
	assert.Greater(t, height, int64(previousHeight)) // height must be greater than previousHeight because confirmations is 0
	msgs, err := db.GetL2Messages(map[string]interface{}{"status": types.MsgPending})
	assert.NoError(t, err)
	assert.Equal(t, 5, len(msgs))
}

func testFetchRunningMissingBlocks(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	auth := prepareAuth(t, l2Cli, cfg.L2Config.RelayerConfig.MessageSenderPrivateKeys[0])

	// deploy mock bridge
	_, tx, _, err := mock_bridge.DeployMockBridgeL2(auth, l2Cli)
	assert.NoError(t, err)
	address, err := bind.WaitDeployed(context.Background(), l2Cli, tx)
	assert.NoError(t, err)

	// wait for dealing time
	<-time.After(3 * time.Second)

	latestHeight, err := l2Cli.BlockNumber(context.Background())
	assert.NoError(t, err)
	wc := prepareWatcherClient(l2Cli, db, address)
	wc.TryFetchRunningMissingBlocks(context.Background(), latestHeight)
	fetchedHeight, err := db.GetL2BlocksLatestHeight()
	assert.NoError(t, err)
	assert.Equal(t, uint64(fetchedHeight), latestHeight)
}

func prepareWatcherClient(l2Cli *ethclient.Client, db database.OrmFactory, contractAddr common.Address) *L2WatcherClient {
	confirmations := rpc.LatestBlockNumber
	return NewL2WatcherClient(context.Background(), l2Cli, confirmations, contractAddr, contractAddr, common.Hash{}, db)
}

func prepareAuth(t *testing.T, l2Cli *ethclient.Client, privateKey *ecdsa.PrivateKey) *bind.TransactOpts {
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(53077))
	assert.NoError(t, err)
	auth.Value = big.NewInt(0) // in wei
	assert.NoError(t, err)
	auth.GasPrice, err = l2Cli.SuggestGasPrice(context.Background())
	assert.NoError(t, err)
	return auth
}

func loopToFetchEvent(subCtx context.Context, watcher *L2WatcherClient) {
	go cutils.Loop(subCtx, 2*time.Second, watcher.FetchContractEvent)
}

func testParseBridgeEventLogsL2SentMessageEventSignature(t *testing.T) {
	watcher := setupL2Watcher(t)
	logs := []geth_types.Log{
		{
			Topics: []common.Hash{
				bridge_abi.L2SentMessageEventSignature,
			},
			BlockNumber: 100,
			TxHash:      common.HexToHash("0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347"),
		},
	}

	convey.Convey("unpack SentMessage log failure", t, func() {
		targetErr := errors.New("UnpackLog SentMessage failure")
		patchGuard := gomonkey.ApplyFunc(utils.UnpackLog, func(c *abi.ABI, out interface{}, event string, log geth_types.Log) error {
			return targetErr
		})
		defer patchGuard.Reset()

		l2Messages, relayedMessages, err := watcher.parseBridgeEventLogs(logs)
		assert.EqualError(t, err, targetErr.Error())
		assert.Empty(t, l2Messages)
		assert.Empty(t, relayedMessages)
	})

	convey.Convey("L2SentMessageEventSignature success", t, func() {
		tmpSendAddr := common.HexToAddress("0xb4c11951957c6f8f642c4af61cd6b24640fec6dc7fc607ee8206a99e92410d30")
		tmpTargetAddr := common.HexToAddress("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5")
		tmpValue := big.NewInt(1000)
		tmpMessageNonce := big.NewInt(100)
		tmpMessage := []byte("test for L2SentMessageEventSignature")
		patchGuard := gomonkey.ApplyFunc(utils.UnpackLog, func(c *abi.ABI, out interface{}, event string, log geth_types.Log) error {
			tmpOut := out.(*bridge_abi.L2SentMessageEvent)
			tmpOut.Sender = tmpSendAddr
			tmpOut.Value = tmpValue
			tmpOut.Target = tmpTargetAddr
			tmpOut.MessageNonce = tmpMessageNonce
			tmpOut.Message = tmpMessage
			return nil
		})
		defer patchGuard.Reset()

		l2Messages, relayedMessages, err := watcher.parseBridgeEventLogs(logs)
		assert.Error(t, err)
		assert.Empty(t, relayedMessages)
		assert.Empty(t, l2Messages)
	})
}

func testParseBridgeEventLogsL2RelayedMessageEventSignature(t *testing.T) {
	watcher := setupL2Watcher(t)
	logs := []geth_types.Log{
		{
			Topics:      []common.Hash{bridge_abi.L2RelayedMessageEventSignature},
			BlockNumber: 100,
			TxHash:      common.HexToHash("0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347"),
		},
	}

	convey.Convey("unpack RelayedMessage log failure", t, func() {
		targetErr := errors.New("UnpackLog RelayedMessage failure")
		patchGuard := gomonkey.ApplyFunc(utils.UnpackLog, func(c *abi.ABI, out interface{}, event string, log geth_types.Log) error {
			return targetErr
		})
		defer patchGuard.Reset()

		l2Messages, relayedMessages, err := watcher.parseBridgeEventLogs(logs)
		assert.EqualError(t, err, targetErr.Error())
		assert.Empty(t, l2Messages)
		assert.Empty(t, relayedMessages)
	})

	convey.Convey("L2RelayedMessageEventSignature success", t, func() {
		msgHash := common.HexToHash("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5")
		patchGuard := gomonkey.ApplyFunc(utils.UnpackLog, func(c *abi.ABI, out interface{}, event string, log geth_types.Log) error {
			tmpOut := out.(*bridge_abi.L2RelayedMessageEvent)
			tmpOut.MessageHash = msgHash
			return nil
		})
		defer patchGuard.Reset()

		l2Messages, relayedMessages, err := watcher.parseBridgeEventLogs(logs)
		assert.NoError(t, err)
		assert.Empty(t, l2Messages)
		assert.Len(t, relayedMessages, 1)
		assert.Equal(t, relayedMessages[0].msgHash, msgHash)
	})
}

func testParseBridgeEventLogsL2FailedRelayedMessageEventSignature(t *testing.T) {
	watcher := setupL2Watcher(t)
	logs := []geth_types.Log{
		{
			Topics:      []common.Hash{bridge_abi.L2FailedRelayedMessageEventSignature},
			BlockNumber: 100,
			TxHash:      common.HexToHash("0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347"),
		},
	}

	convey.Convey("unpack FailedRelayedMessage log failure", t, func() {
		targetErr := errors.New("UnpackLog FailedRelayedMessage failure")
		patchGuard := gomonkey.ApplyFunc(utils.UnpackLog, func(c *abi.ABI, out interface{}, event string, log geth_types.Log) error {
			return targetErr
		})
		defer patchGuard.Reset()

		l2Messages, relayedMessages, err := watcher.parseBridgeEventLogs(logs)
		assert.EqualError(t, err, targetErr.Error())
		assert.Empty(t, l2Messages)
		assert.Empty(t, relayedMessages)
	})

	convey.Convey("L2FailedRelayedMessageEventSignature success", t, func() {
		msgHash := common.HexToHash("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5")
		patchGuard := gomonkey.ApplyFunc(utils.UnpackLog, func(c *abi.ABI, out interface{}, event string, log geth_types.Log) error {
			tmpOut := out.(*bridge_abi.L2FailedRelayedMessageEvent)
			tmpOut.MessageHash = msgHash
			return nil
		})
		defer patchGuard.Reset()

		l2Messages, relayedMessages, err := watcher.parseBridgeEventLogs(logs)
		assert.NoError(t, err)
		assert.Empty(t, l2Messages)
		assert.Len(t, relayedMessages, 1)
		assert.Equal(t, relayedMessages[0].msgHash, msgHash)
	})
}

func testParseBridgeEventLogsL2AppendMessageEventSignature(t *testing.T) {
	watcher := setupL2Watcher(t)
	logs := []geth_types.Log{
		{
			Topics:      []common.Hash{bridge_abi.L2AppendMessageEventSignature},
			BlockNumber: 100,
			TxHash:      common.HexToHash("0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347"),
		},
	}

	convey.Convey("unpack AppendMessage log failure", t, func() {
		targetErr := errors.New("UnpackLog AppendMessage failure")
		patchGuard := gomonkey.ApplyFunc(utils.UnpackLog, func(c *abi.ABI, out interface{}, event string, log geth_types.Log) error {
			return targetErr
		})
		defer patchGuard.Reset()

		l2Messages, relayedMessages, err := watcher.parseBridgeEventLogs(logs)
		assert.EqualError(t, err, targetErr.Error())
		assert.Empty(t, l2Messages)
		assert.Empty(t, relayedMessages)
	})

	convey.Convey("L2AppendMessageEventSignature success", t, func() {
		msgHash := common.HexToHash("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5")
		patchGuard := gomonkey.ApplyFunc(utils.UnpackLog, func(c *abi.ABI, out interface{}, event string, log geth_types.Log) error {
			tmpOut := out.(*bridge_abi.L2AppendMessageEvent)
			tmpOut.MessageHash = msgHash
			tmpOut.Index = big.NewInt(100)
			return nil
		})
		defer patchGuard.Reset()

		l2Messages, relayedMessages, err := watcher.parseBridgeEventLogs(logs)
		assert.NoError(t, err)
		assert.Empty(t, l2Messages)
		assert.Empty(t, relayedMessages)
	})
}
