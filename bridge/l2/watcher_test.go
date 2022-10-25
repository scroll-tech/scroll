package l2_test

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"math/big"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"

	bridge_abi "scroll-tech/bridge/abi"
	"scroll-tech/bridge/l2"
	"scroll-tech/bridge/mock_bridge"
	"scroll-tech/bridge/sender"

	"scroll-tech/database"
	"scroll-tech/database/migrate"
)

func testCreateNewWatcherAndStop(t *testing.T) {
	// Create db handler and reset db.
	l2db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(l2db.GetDB().DB))
	defer assert.NoError(t, l2db.Close())

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
	rc := l2.NewL2WatcherClient(context.Background(), l2Cli, cfg.L2Config.Confirmations, proofGenerationFreq, skippedOpcodes, cfg.L2Config.L2MessengerAddress, messengerABI, l2db)
	rc.Start()
	defer rc.Stop()

	cfg.L1Config.RelayerConfig.SenderConfig.Confirmations = 0
	newSender, err := sender.NewSender(context.Background(), cfg.L1Config.RelayerConfig.SenderConfig, privkey)
	assert.NoError(t, err)

	// Create several transactions and commit to block
	numTransactions := 3
	toAddress := common.HexToAddress("0x4592d8f8d7b001e72cb26a73e4fa1806a51ac79d")
	for i := 0; i < numTransactions; i++ {
		_, err = newSender.SendTransaction(strconv.Itoa(1000+i), &toAddress, big.NewInt(1000000000), nil)
		assert.NoError(t, err)
		<-newSender.ConfirmChan()
	}

	//<-time.After(10 * time.Second)
	blockNum, err := l2Cli.BlockNumber(context.Background())
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, blockNum, uint64(numTransactions))
}

func testMonitorBridgeContract(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))

	previousHeight, err := l2Cli.BlockNumber(context.Background())
	assert.NoError(t, err)

	auth := prepareAuth(t, l2Cli, cfg.L2Config.RelayerConfig.PrivateKey)

	// deploy mock bridge
	_, tx, instance, err := mock_bridge.DeployMockBridge(auth, l2Cli)
	assert.NoError(t, err)
	address, err := bind.WaitDeployed(context.Background(), l2Cli, tx)
	assert.NoError(t, err)

	rc := prepareRelayerClient(l2Cli, db, address)
	rc.Start()

	// Call mock_bridge instance sendMessage to trigger emit events
	addr := common.HexToAddress("0x1c5a77d9fa7ef466951b2f01f724bca3a5820b63")
	nonce, err := l2Cli.PendingNonceAt(context.Background(), addr)
	assert.NoError(t, err)
	auth.Nonce = big.NewInt(int64(nonce))
	toAddress := common.HexToAddress("0x4592d8f8d7b001e72cb26a73e4fa1806a51ac79d")
	message := []byte("testbridgecontract")
	tx, err = instance.SendMessage(auth, toAddress, message, auth.GasPrice)
	assert.NoError(t, err)
	receipt, err := bind.WaitMined(context.Background(), l2Cli, tx)
	if receipt.Status != types.ReceiptStatusSuccessful || err != nil {
		t.Fatalf("Call failed")
	}

	//extra block mined
	addr = common.HexToAddress("0x1c5a77d9fa7ef466951b2f01f724bca3a5820b63")
	nonce, nounceErr := l2Cli.PendingNonceAt(context.Background(), addr)
	assert.NoError(t, nounceErr)
	auth.Nonce = big.NewInt(int64(nonce))
	toAddress = common.HexToAddress("0x4592d8f8d7b001e72cb26a73e4fa1806a51ac79d")
	message = []byte("testbridgecontract")
	tx, err = instance.SendMessage(auth, toAddress, message, auth.GasPrice)
	assert.NoError(t, err)
	receipt, err = bind.WaitMined(context.Background(), l2Cli, tx)
	if receipt.Status != types.ReceiptStatusSuccessful || err != nil {
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
	assert.Greater(t, uint64(height), previousHeight)
	msgs, err := db.GetL2UnprocessedMessages()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(msgs))

	rc.Stop()
	db.Close()
}

func testFetchMultipleSentMessageInOneBlock(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))

	previousHeight, err := l2Cli.BlockNumber(context.Background()) // shallow the global previousHeight
	assert.NoError(t, err)

	auth := prepareAuth(t, l2Cli, cfg.L2Config.RelayerConfig.PrivateKey)

	_, trx, instance, err := mock_bridge.DeployMockBridge(auth, l2Cli)
	assert.NoError(t, err)
	address, err := bind.WaitDeployed(context.Background(), l2Cli, trx)
	assert.NoError(t, err)

	rc := prepareRelayerClient(l2Cli, db, address)
	rc.Start()

	// Call mock_bridge instance sendMessage to trigger emit events multiple times
	numTransactions := 4
	var tx *types.Transaction

	for i := 0; i < numTransactions; i++ {
		addr := common.HexToAddress("0x1c5a77d9fa7ef466951b2f01f724bca3a5820b63")
		nonce, nounceErr := l2Cli.PendingNonceAt(context.Background(), addr)
		assert.NoError(t, nounceErr)
		auth.Nonce = big.NewInt(int64(nonce))
		toAddress := common.HexToAddress("0x4592d8f8d7b001e72cb26a73e4fa1806a51ac79d")
		message := []byte("testbridgecontract")
		tx, err = instance.SendMessage(auth, toAddress, message, auth.GasPrice)
		assert.NoError(t, err)
	}

	receipt, err := bind.WaitMined(context.Background(), l2Cli, tx)
	if receipt.Status != types.ReceiptStatusSuccessful || err != nil {
		t.Fatalf("Call failed")
	}

	// extra block mined
	addr := common.HexToAddress("0x1c5a77d9fa7ef466951b2f01f724bca3a5820b63")
	nonce, nounceErr := l2Cli.PendingNonceAt(context.Background(), addr)
	assert.NoError(t, nounceErr)
	auth.Nonce = big.NewInt(int64(nonce))
	toAddress := common.HexToAddress("0x4592d8f8d7b001e72cb26a73e4fa1806a51ac79d")
	message := []byte("testbridgecontract")
	tx, err = instance.SendMessage(auth, toAddress, message, auth.GasPrice)
	assert.NoError(t, err)
	receipt, err = bind.WaitMined(context.Background(), l2Cli, tx)
	if receipt.Status != types.ReceiptStatusSuccessful || err != nil {
		t.Fatalf("Call failed")
	}

	// wait for dealing time
	<-time.After(6 * time.Second)

	// check if we successfully stored events
	height, err := db.GetLayer2LatestWatchedHeight()
	assert.NoError(t, err)
	t.Log("LatestHeight is", height)
	assert.Greater(t, uint64(height), previousHeight) // height must be greater than previousHeight because confirmations is 0
	msgs, err := db.GetL2UnprocessedMessages()
	assert.NoError(t, err)
	assert.Equal(t, 5, len(msgs))

	rc.Stop()
	db.Close()
}

func testTraceHasUnsupportedOpcodes(t *testing.T) {
	delegateTrace, err := os.ReadFile("../../common/testdata/blockResult_delegate.json")
	assert.NoError(t, err)

	trace := &types.BlockResult{}
	assert.NoError(t, json.Unmarshal(delegateTrace, &trace))

	unsupportedOpcodes := make(map[string]struct{})
	for _, val := range cfg.L2Config.SkippedOpcodes {
		unsupportedOpcodes[val] = struct{}{}
	}
}

func prepareRelayerClient(l2Cli *ethclient.Client, db database.OrmFactory, contractAddr common.Address) *l2.WatcherClient {
	messengerABI, _ := bridge_abi.L1MessengerMetaData.GetAbi()
	return l2.NewL2WatcherClient(context.Background(), l2Cli, 0, 1, map[string]struct{}{}, contractAddr, messengerABI, db)
}

func prepareAuth(t *testing.T, l2Cli *ethclient.Client, private string) *bind.TransactOpts {
	privateKey, err := crypto.HexToECDSA(private)
	assert.NoError(t, err)
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	assert.True(t, ok)
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := l2Cli.PendingNonceAt(context.Background(), fromAddress)
	assert.NoError(t, err)
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(53077))
	assert.NoError(t, err)
	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = big.NewInt(0)       // in wei
	auth.GasLimit = uint64(30000000) // in units
	auth.GasPrice, err = l2Cli.SuggestGasPrice(context.Background())
	assert.NoError(t, err)
	return auth
}
