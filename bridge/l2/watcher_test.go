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

	"scroll-tech/common/docker"
	"scroll-tech/common/utils"

	"scroll-tech/database"

	db_config "scroll-tech/database"

	"scroll-tech/bridge/config"
	"scroll-tech/bridge/mock"
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
		DbPort: 5438,
		DB_CONFIG: &db_config.DBConfig{
			DriverName: utils.GetEnvWithDefault("TEST_DB_DRIVER", "postgres"),
			DSN:        utils.GetEnvWithDefault("TEST_DB_DSN", "postgres://postgres:123456@localhost:5438/testwatcher_db?sslmode=disable"),
		},
	},
}

var (
	// previousHeight store previous chain height
	previousHeight uint64
	l1gethImg      docker.ImgInstance
	l2gethImg      docker.ImgInstance
	dbImg          docker.ImgInstance
	l2Backend      *l2.Backend
)

func setenv(t *testing.T) {
	cfg, err := config.NewConfig("../config.json")
	assert.NoError(t, err)
	l1gethImg = mock.NewL1Docker(t, TEST_CONFIG)
	cfg.L2Config.RelayerConfig.SenderConfig.Endpoint = l1gethImg.Endpoint()
	l2Backend, l2gethImg, dbImg = mock.L2gethDocker(t, cfg, TEST_CONFIG)
}

func TestWatcherFunction(t *testing.T) {
	setenv(t)
	t.Run("TestL2Backend", func(t *testing.T) {
		err := l2Backend.Start()
		assert.NoError(t, err)
		l2Backend.Stop()
	})
	t.Run("TestCreateNewWatcherAndStop", func(t *testing.T) {
		cfg, err := config.NewConfig("../config.json")
		assert.NoError(t, err)

		cfg.L2Config.Endpoint = l2gethImg.Endpoint()
		client, err := ethclient.Dial(cfg.L2Config.Endpoint)
		assert.NoError(t, err)
		db := mock.ResetDB(t, TEST_CONFIG.DB_CONFIG)
		messengerABI, err := bridge_abi.L2MessengerMetaData.GetAbi()
		assert.NoError(t, err)

		skippedOpcodes := make(map[string]struct{}, len(cfg.L2Config.SkippedOpcodes))
		for _, op := range cfg.L2Config.SkippedOpcodes {
			skippedOpcodes[op] = struct{}{}
		}
		proofGenerationFreq := cfg.L2Config.ProofGenerationFreq
		if proofGenerationFreq == 0 {
			proofGenerationFreq = 1
		}
		rc := l2.NewL2WatcherClient(context.Background(), client, cfg.L2Config.Confirmations, proofGenerationFreq, skippedOpcodes, cfg.L2Config.L2MessengerAddress, messengerABI, db)
		rc.Start()

		// Create several transactions and commit to block
		numTransactions := 3

		for i := 0; i < numTransactions; i++ {
			mock.SendTxToL2Client(t, client, cfg.L2Config.RelayerConfig.PrivateKey)
		}

		<-time.After(10 * time.Second)
		blockNum, err := client.BlockNumber(context.Background())
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, blockNum, uint64(numTransactions))

		rc.Stop()
		db.Close()
	})

	t.Run("TestMonitorBridgeContract", func(t *testing.T) {
		cfg, err := config.NewConfig("../config.json")
		assert.NoError(t, err)
		t.Log("confirmations:", cfg.L2Config.Confirmations)

		cfg.L2Config.Endpoint = l2gethImg.Endpoint()
		client, err := ethclient.Dial(cfg.L2Config.Endpoint)
		assert.NoError(t, err)

		db := mock.ResetDB(t, TEST_CONFIG.DB_CONFIG)
		previousHeight, err = client.BlockNumber(context.Background())
		assert.NoError(t, err)

		auth := prepareAuth(t, client, cfg.L2Config.RelayerConfig.PrivateKey)

		// deploy mock bridge
		_, tx, instance, err := mock_bridge.DeployMockBridge(auth, client)
		assert.NoError(t, err)
		address, err := bind.WaitDeployed(context.Background(), client, tx)
		assert.NoError(t, err)

		rc := prepareRelayerClient(client, db, address)
		rc.Start()

		// Call mock_bridge instance sendMessage to trigger emit events
		addr := common.HexToAddress("0x1c5a77d9fa7ef466951b2f01f724bca3a5820b63")
		nonce, err := client.PendingNonceAt(context.Background(), addr)
		assert.NoError(t, err)
		auth.Nonce = big.NewInt(int64(nonce))
		toAddress := common.HexToAddress("0x4592d8f8d7b001e72cb26a73e4fa1806a51ac79d")
		message := []byte("testbridgecontract")
		tx, err = instance.SendMessage(auth, toAddress, message, auth.GasPrice)
		assert.NoError(t, err)
		receipt, err := bind.WaitMined(context.Background(), client, tx)
		if receipt.Status != types.ReceiptStatusSuccessful || err != nil {
			t.Fatalf("Call failed")
		}

		//extra block mined
		addr = common.HexToAddress("0x1c5a77d9fa7ef466951b2f01f724bca3a5820b63")
		nonce, nounceErr := client.PendingNonceAt(context.Background(), addr)
		assert.NoError(t, nounceErr)
		auth.Nonce = big.NewInt(int64(nonce))
		toAddress = common.HexToAddress("0x4592d8f8d7b001e72cb26a73e4fa1806a51ac79d")
		message = []byte("testbridgecontract")
		tx, err = instance.SendMessage(auth, toAddress, message, auth.GasPrice)
		assert.NoError(t, err)
		receipt, err = bind.WaitMined(context.Background(), client, tx)
		if receipt.Status != types.ReceiptStatusSuccessful || err != nil {
			t.Fatalf("Call failed")
		}

		// wait for dealing time
		<-time.After(6 * time.Second)

		var latestHeight uint64
		latestHeight, err = client.BlockNumber(context.Background())
		assert.NoError(t, err)
		t.Log("Latest height is", latestHeight)

		// check if we successfully stored events
		height, err := db.GetLayer2LatestWatchedHeight()
		assert.NoError(t, err)
		t.Log("Height in DB is", height)
		assert.Greater(t, height, int64(previousHeight))
		msgs, err := db.GetL2UnprocessedMessages()
		assert.NoError(t, err)
		assert.Equal(t, 2, len(msgs))

		rc.Stop()
		db.Close()
	})

	t.Run("TestFetchMultipleSentMessageInOneBlock", func(t *testing.T) {
		cfg, err := config.NewConfig("../config.json")
		assert.NoError(t, err)

		cfg.L2Config.Endpoint = l2gethImg.Endpoint()
		client, err := ethclient.Dial(cfg.L2Config.Endpoint)
		assert.NoError(t, err)

		db := mock.ResetDB(t, TEST_CONFIG.DB_CONFIG)

		previousHeight, err := client.BlockNumber(context.Background()) // shallow the global previousHeight
		assert.NoError(t, err)

		auth := prepareAuth(t, client, cfg.L2Config.RelayerConfig.PrivateKey)

		_, trx, instance, err := mock_bridge.DeployMockBridge(auth, client)
		assert.NoError(t, err)
		address, err := bind.WaitDeployed(context.Background(), client, trx)
		assert.NoError(t, err)

		rc := prepareRelayerClient(client, db, address)
		rc.Start()

		// Call mock_bridge instance sendMessage to trigger emit events multiple times
		numTransactions := 4
		var tx *types.Transaction

		for i := 0; i < numTransactions; i++ {
			addr := common.HexToAddress("0x1c5a77d9fa7ef466951b2f01f724bca3a5820b63")
			nonce, nounceErr := client.PendingNonceAt(context.Background(), addr)
			assert.NoError(t, nounceErr)
			auth.Nonce = big.NewInt(int64(nonce))
			toAddress := common.HexToAddress("0x4592d8f8d7b001e72cb26a73e4fa1806a51ac79d")
			message := []byte("testbridgecontract")
			tx, err = instance.SendMessage(auth, toAddress, message, auth.GasPrice)
			assert.NoError(t, err)
		}

		receipt, err := bind.WaitMined(context.Background(), client, tx)
		if receipt.Status != types.ReceiptStatusSuccessful || err != nil {
			t.Fatalf("Call failed")
		}

		// extra block mined
		addr := common.HexToAddress("0x1c5a77d9fa7ef466951b2f01f724bca3a5820b63")
		nonce, nounceErr := client.PendingNonceAt(context.Background(), addr)
		assert.NoError(t, nounceErr)
		auth.Nonce = big.NewInt(int64(nonce))
		toAddress := common.HexToAddress("0x4592d8f8d7b001e72cb26a73e4fa1806a51ac79d")
		message := []byte("testbridgecontract")
		tx, err = instance.SendMessage(auth, toAddress, message, auth.GasPrice)
		assert.NoError(t, err)
		receipt, err = bind.WaitMined(context.Background(), client, tx)
		if receipt.Status != types.ReceiptStatusSuccessful || err != nil {
			t.Fatalf("Call failed")
		}

		// wait for dealing time
		<-time.After(6 * time.Second)

		// check if we successfully stored events
		height, err := db.GetLayer2LatestWatchedHeight()
		assert.NoError(t, err)
		t.Log("LatestHeight is", height)
		assert.Greater(t, height, int64(previousHeight)) // height must be greater than previousHeight because confirmations is 0
		msgs, err := db.GetL2UnprocessedMessages()
		assert.NoError(t, err)
		assert.Equal(t, 5, len(msgs))

		rc.Stop()
		db.Close()
	})

	// Teardown
	t.Cleanup(func() {
		assert.NoError(t, l1gethImg.Stop())
		assert.NoError(t, l2gethImg.Stop())
		assert.NoError(t, dbImg.Stop())
	})

}

func TestTraceHasUnsupportedOpcodes(t *testing.T) {
	cfg, err := config.NewConfig("../config.json")
	assert.NoError(t, err)

	delegateTrace, err := os.ReadFile("../../common/testdata/blockResult_delegate.json")
	assert.NoError(t, err)

	trace := &types.BlockResult{}
	assert.NoError(t, json.Unmarshal(delegateTrace, &trace))

	unsupportedOpcodes := make(map[string]struct{})
	for _, val := range cfg.L2Config.SkippedOpcodes {
		unsupportedOpcodes[val] = struct{}{}
	}
}

func prepareRelayerClient(client *ethclient.Client, db database.OrmFactory, contractAddr common.Address) *l2.WatcherClient {
	messengerABI, _ := bridge_abi.L1MessengerMetaData.GetAbi()
	return l2.NewL2WatcherClient(context.Background(), client, 0, 1, map[string]struct{}{}, contractAddr, messengerABI, db)
}

func prepareAuth(t *testing.T, client *ethclient.Client, private string) *bind.TransactOpts {
	privateKey, err := crypto.HexToECDSA(private)
	assert.NoError(t, err)
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	assert.True(t, ok)
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	assert.NoError(t, err)
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(53077))
	assert.NoError(t, err)
	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = big.NewInt(0)       // in wei
	auth.GasLimit = uint64(30000000) // in units
	auth.GasPrice, err = client.SuggestGasPrice(context.Background())
	assert.NoError(t, err)
	return auth
}
