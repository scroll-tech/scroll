package l2_test

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/ethclient"

	bridge_abi "scroll-tech/bridge/abi"
	"scroll-tech/bridge/l2"

	"scroll-tech/store"
	db_config "scroll-tech/store/config"

	"scroll-tech/internal/mock"

	"scroll-tech/bridge/config"
	"scroll-tech/bridge/mock_bridge"
)

const TEST_BUFFER = 500

var TEST_CONFIG = &mock.TestConfig{
	L1GethTestConfig: mock.L1GethTestConfig{
		HPort: 0,
		WPort: 8571,
	},
	L2GethTestConfig: mock.L2GethTestConfig{
		HPort: 0,
		WPort: 8567,
	},
	DbTestconfig: mock.DbTestconfig{
		DbName: "testwatcher_db",
		DbPort: 5437,
		DB_CONFIG: &db_config.DBConfig{
			DriverName: db_config.GetEnvWithDefault("TEST_DB_DRIVER", "postgres"),
			DSN:        db_config.GetEnvWithDefault("TEST_DB_DSN", "postgres://postgres:123456@localhost:5437/testwatcher_db?sslmode=disable"),
		},
	},
}

var (
	// previousHeight store previous chain height
	previousHeight uint64
)

func TestTraceHasUnsupportedOpcodes(t *testing.T) {
	cfg, err := config.NewConfig("../config.json")
	assert.NoError(t, err)

	delegateTrace, err := os.ReadFile("../../internal/testdata/blockResult_delegate.json")
	assert.NoError(t, err)

	trace := &types.BlockResult{}
	assert.NoError(t, json.Unmarshal(delegateTrace, &trace))

	unsupportedOpcodes := make(map[string]struct{})
	for _, val := range cfg.L2Config.SkippedOpcodes {
		unsupportedOpcodes[val] = struct{}{}
	}
	assert.Equal(t, true, l2.TraceHasUnsupportedOpcodes(unsupportedOpcodes, trace))
}

// test start l2 backend and shut it down gracefully
func TestL2Backend(t *testing.T) {
	assert := assert.New(t)
	cfg, err := config.NewConfig("../config.json")
	assert.NoError(err)

	// Set up mock l2 geth
	l2_backend, img_geth, img_db := mock.Mockl2gethDocker(t, cfg, TEST_CONFIG)
	defer img_geth.Stop()
	defer img_db.Stop()

	err = l2_backend.Start()
	assert.NoError(err)
	l2_backend.Stop()
}

// TestCreateNewRelayerAndStop will test creating a new instance of watcher client client and stoping it
func TestCreateNewWatcherAndStop(t *testing.T) {
	assert := assert.New(t)
	cfg, err := config.NewConfig("../config.json")
	assert.NoError(err)

	_, img_geth, img_db := mock.Mockl2gethDocker(t, cfg, TEST_CONFIG)
	defer img_geth.Stop()
	defer img_db.Stop()

	cfg.L2Config.Endpoint = img_geth.Endpoint()
	client, err := ethclient.Dial(cfg.L2Config.Endpoint)
	assert.NoError(err)
	mock.MockClearDB(assert, TEST_CONFIG.DB_CONFIG)

	messengerABI, err := bridge_abi.L2MessengerMetaData.GetAbi()
	assert.NoError(err)

	l2db := mock.MockPrepareDB(assert, TEST_CONFIG.DB_CONFIG)

	skippedOpcodes := make(map[string]struct{}, len(cfg.L2Config.SkippedOpcodes))
	for _, op := range cfg.L2Config.SkippedOpcodes {
		skippedOpcodes[op] = struct{}{}
	}
	proofGenerationFreq := cfg.L2Config.ProofGenerationFreq
	if proofGenerationFreq == 0 {
		proofGenerationFreq = 1
	}
	rc := l2.NewL2WatcherClient(context.Background(), client, cfg.L2Config.Confirmations, proofGenerationFreq, skippedOpcodes, cfg.L2Config.L2MessengerAddress, messengerABI, l2db)
	rc.Start()

	// Create several transactions and commit to block
	numTransactions := 3

	for i := 0; i < numTransactions; i++ {
		mock.MockSendTxToL2Client(assert, client)
		// wait for 10 seconds for mining
		<-time.After(10 * time.Second)
	}

	<-time.After(10 * time.Second)
	blockNum, err := client.BlockNumber(context.Background())
	assert.NoError(err)
	assert.GreaterOrEqual(blockNum, uint64(numTransactions))

	rc.Stop()
	l2db.Close()
}

// TestMonitorBridgeContract will depoly contract and test if monitor can fetch it and store into db
func TestMonitorBridgeContract(t *testing.T) {
	assert := assert.New(t)

	cfg, err := config.NewConfig("../config.json")
	assert.NoError(err)
	t.Log("confirmations:", cfg.L2Config.Confirmations)

	_, img_geth, img_db := mock.Mockl2gethDocker(t, cfg, TEST_CONFIG)
	defer img_geth.Stop()
	defer img_db.Stop()

	cfg.L2Config.Endpoint = img_geth.Endpoint()
	client, err := ethclient.Dial(cfg.L2Config.Endpoint)
	assert.NoError(err)

	mock.MockClearDB(assert, TEST_CONFIG.DB_CONFIG)
	previousHeight, err = client.BlockNumber(context.Background())
	assert.NoError(err)

	auth := prepareAuth(assert, client)

	// deploy mock bridge
	_, tx, instance, err := mock_bridge.DeployMockBridge(auth, client)
	assert.NoError(err)
	address, err := bind.WaitDeployed(context.Background(), client, tx)
	assert.NoError(err)

	db := mock.MockPrepareDB(assert, TEST_CONFIG.DB_CONFIG)
	rc := prepareRelayerClient(client, db, address)
	rc.Start()

	// Call mock_bridge instance sendMessage to trigger emit events
	addr := common.HexToAddress("0x4cb1ab63af5d8931ce09673ebd8ae2ce16fd6571")
	nonce, err := client.PendingNonceAt(context.Background(), addr)
	assert.NoError(err)
	auth.Nonce = big.NewInt(int64(nonce))
	toAddress := common.HexToAddress("0x4592d8f8d7b001e72cb26a73e4fa1806a51ac79d")
	message := []byte("testbridgecontract")
	tx, err = instance.SendMessage(auth, toAddress, message, auth.GasPrice)
	assert.NoError(err)
	receipt, err := bind.WaitMined(context.Background(), client, tx)
	if receipt.Status != types.ReceiptStatusSuccessful || err != nil {
		t.Fatalf("Call failed")
	}

	//extra block mined
	addr = common.HexToAddress("0x4cb1ab63af5d8931ce09673ebd8ae2ce16fd6571")
	nonce, nounceErr := client.PendingNonceAt(context.Background(), addr)
	assert.NoError(nounceErr)
	auth.Nonce = big.NewInt(int64(nonce))
	toAddress = common.HexToAddress("0x4592d8f8d7b001e72cb26a73e4fa1806a51ac79d")
	message = []byte("testbridgecontract")
	tx, err = instance.SendMessage(auth, toAddress, message, auth.GasPrice)
	assert.NoError(err)
	receipt, err = bind.WaitMined(context.Background(), client, tx)
	if receipt.Status != types.ReceiptStatusSuccessful || err != nil {
		t.Fatalf("Call failed")
	}

	// wait for dealing time
	<-time.After(15 * time.Second)

	var latestHeight uint64
	latestHeight, err = client.BlockNumber(context.Background())
	assert.NoError(err)
	t.Log("Latest height is", latestHeight)

	// check if we successfully stored events
	height, err := db.GetLayer2LatestWatchedHeight()
	assert.NoError(err)
	t.Log("Height in DB is", height)
	assert.Greater(uint64(height), previousHeight)
	msgs, err := db.GetL2UnprocessedMessages()
	assert.NoError(err)
	assert.Equal(2, len(msgs))

	rc.Stop()
	db.Close()
}

// TestFetchMultipleSentMessageInOneBlock test monitor when one block has multiple sentMessage called
func TestFetchMultipleSentMessageInOneBlock(t *testing.T) {
	assert := assert.New(t)

	cfg, err := config.NewConfig("../config.json")
	assert.NoError(err)

	_, img_geth, img_db := mock.Mockl2gethDocker(t, cfg, TEST_CONFIG)
	defer img_geth.Stop()
	defer img_db.Stop()

	cfg.L2Config.Endpoint = img_geth.Endpoint()
	client, err := ethclient.Dial(cfg.L2Config.Endpoint)
	assert.NoError(err)

	mock.MockClearDB(assert, TEST_CONFIG.DB_CONFIG)

	previousHeight, err := client.BlockNumber(context.Background()) // shallow the global previousHeight
	assert.NoError(err)

	auth := prepareAuth(assert, client)

	_, trx, instance, err := mock_bridge.DeployMockBridge(auth, client)
	assert.NoError(err)
	address, err := bind.WaitDeployed(context.Background(), client, trx)
	assert.NoError(err)

	db := mock.MockPrepareDB(assert, TEST_CONFIG.DB_CONFIG)
	rc := prepareRelayerClient(client, db, address)
	rc.Start()

	// Call mock_bridge instance sendMessage to trigger emit events multiple times
	numTransactions := 4
	var tx *types.Transaction

	for i := 0; i < numTransactions; i++ {
		addr := common.HexToAddress("0x4cb1ab63af5d8931ce09673ebd8ae2ce16fd6571")
		nonce, nounceErr := client.PendingNonceAt(context.Background(), addr)
		assert.NoError(nounceErr)
		auth.Nonce = big.NewInt(int64(nonce))
		toAddress := common.HexToAddress("0x4592d8f8d7b001e72cb26a73e4fa1806a51ac79d")
		message := []byte("testbridgecontract")
		tx, err = instance.SendMessage(auth, toAddress, message, auth.GasPrice)
		assert.NoError(err)
	}

	receipt, err := bind.WaitMined(context.Background(), client, tx)
	if receipt.Status != types.ReceiptStatusSuccessful || err != nil {
		t.Fatalf("Call failed")
	}

	// extra block mined
	addr := common.HexToAddress("0x4cb1ab63af5d8931ce09673ebd8ae2ce16fd6571")
	nonce, nounceErr := client.PendingNonceAt(context.Background(), addr)
	assert.NoError(nounceErr)
	auth.Nonce = big.NewInt(int64(nonce))
	toAddress := common.HexToAddress("0x4592d8f8d7b001e72cb26a73e4fa1806a51ac79d")
	message := []byte("testbridgecontract")
	tx, err = instance.SendMessage(auth, toAddress, message, auth.GasPrice)
	assert.NoError(err)
	receipt, err = bind.WaitMined(context.Background(), client, tx)
	if receipt.Status != types.ReceiptStatusSuccessful || err != nil {
		t.Fatalf("Call failed")
	}

	// wait for dealing time
	<-time.After(10 * time.Second)

	// check if we successfully stored events
	height, err := db.GetLayer2LatestWatchedHeight()
	assert.NoError(err)
	t.Log("LatestHeight is", height)
	assert.Greater(uint64(height), previousHeight) // height must be greater than previousHeight because confirmations is 0
	msgs, err := db.GetL2UnprocessedMessages()
	assert.NoError(err)
	assert.Equal(5, len(msgs))

	rc.Stop()
	db.Close()
}

func prepareRelayerClient(client *ethclient.Client, db store.OrmFactory, contractAddr common.Address) *l2.WatcherClient {
	messengerABI, _ := bridge_abi.L1MessengerMetaData.GetAbi()
	return l2.NewL2WatcherClient(context.Background(), client, 0, 1, map[string]struct{}{}, contractAddr, messengerABI, db)
}

func prepareAuth(assert *assert.Assertions, client *ethclient.Client) *bind.TransactOpts {
	privateKey, err := crypto.HexToECDSA("ad29c7c341a23f04851b6c8602c7c74b98e3fc9488582791bda60e0e261f9cbb")
	assert.NoError(err)
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	assert.True(ok)
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	assert.NoError(err)
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(53077))
	assert.NoError(err)
	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = big.NewInt(0)       // in wei
	auth.GasLimit = uint64(30000000) // in units
	auth.GasPrice, err = client.SuggestGasPrice(context.Background())
	assert.NoError(err)
	return auth
}
