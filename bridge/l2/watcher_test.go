package l2_test

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"strconv"
	"testing"
	"time"

	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"

	"scroll-tech/bridge/config"
	"scroll-tech/bridge/l2"
	"scroll-tech/bridge/mock_bridge"
	"scroll-tech/bridge/sender"
	"scroll-tech/common/viper"

	"scroll-tech/database"
	"scroll-tech/database/migrate"
	"scroll-tech/database/orm"
)

func testCreateNewWatcherAndStop(t *testing.T) {
	// Create db handler and reset db.
	l2db, err := database.NewOrmFactory(vp.Sub("db_config"))
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(l2db.GetDB().DB))
	defer l2db.Close()

	rc := l2.NewL2WatcherClient(context.Background(), l2Cli, l2db, vp.Sub("l2_config"))
	rc.Start()
	defer rc.Stop()

	messageSenderPrivateKeys, err := config.UnmarshalPrivateKeys(vp.Sub("l2_config.relayer_config").GetStringSlice("message_sender_private_keys"))
	assert.NoError(t, err)

	senderCfg := vp.Sub("l1_config.relayer_config.sender_config")
	senderCfg.Set("confirmations", 0)
	newSender, err := sender.NewSender(context.Background(), senderCfg, messageSenderPrivateKeys)
	assert.NoError(t, err)

	// Create several transactions and commit to block
	numTransactions := 3
	toAddress := common.HexToAddress("0x4592d8f8d7b001e72cb26a73e4fa1806a51ac79d")
	for i := 0; i < numTransactions; i++ {
		_, err = newSender.SendTransaction(strconv.Itoa(1000+i), &toAddress, big.NewInt(1000000000), nil)
		assert.NoError(t, err)
		<-newSender.ConfirmChan()
	}

	blockNum, err := l2Cli.BlockNumber(context.Background())
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, blockNum, uint64(numTransactions))
}

func testMonitorBridgeContract(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(vp.Sub("db_config"))
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	previousHeight, err := l2Cli.BlockNumber(context.Background())
	assert.NoError(t, err)

	messageSenderPrivateKeys, err := config.UnmarshalPrivateKeys(vp.Sub("l2_config.relayer_config").GetStringSlice("message_sender_private_keys"))
	assert.NoError(t, err)
	auth := prepareAuth(t, l2Cli, messageSenderPrivateKeys[0])

	// deploy mock bridge
	_, tx, instance, err := mock_bridge.DeployMockBridge(auth, l2Cli)
	assert.NoError(t, err)
	address, err := bind.WaitDeployed(context.Background(), l2Cli, tx)
	assert.NoError(t, err)

	vp.Set("l2_config.l2_messenger_address", address.String())
	rc := prepareRelayerClient(l2Cli, db, vp.Sub("l2_config"))
	rc.Start()
	defer rc.Stop()

	// Call mock_bridge instance sendMessage to trigger emit events
	toAddress := common.HexToAddress("0x4592d8f8d7b001e72cb26a73e4fa1806a51ac79d")
	message := []byte("testbridgecontract")
	tx, err = instance.SendMessage(auth, toAddress, message, auth.GasPrice)
	assert.NoError(t, err)
	receipt, err := bind.WaitMined(context.Background(), l2Cli, tx)
	if receipt.Status != types.ReceiptStatusSuccessful || err != nil {
		t.Fatalf("Call failed")
	}

	// extra block mined
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
	assert.Greater(t, height, int64(previousHeight))
	msgs, err := db.GetL2MessagesByStatus(orm.MsgPending)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(msgs))
}

func testFetchMultipleSentMessageInOneBlock(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(vp.Sub("db_config"))
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	previousHeight, err := l2Cli.BlockNumber(context.Background()) // shallow the global previousHeight
	assert.NoError(t, err)

	messageSenderPrivateKeys, err := config.UnmarshalPrivateKeys(vp.Sub("l2_config.relayer_config").GetStringSlice("message_sender_private_keys"))
	assert.NoError(t, err)
	auth := prepareAuth(t, l2Cli, messageSenderPrivateKeys[0])

	_, trx, instance, err := mock_bridge.DeployMockBridge(auth, l2Cli)
	assert.NoError(t, err)
	address, err := bind.WaitDeployed(context.Background(), l2Cli, trx)
	assert.NoError(t, err)

	vp.Set("l2_config.l2_messenger_address", address.String())
	rc := prepareRelayerClient(l2Cli, db, vp.Sub("l2_config"))
	rc.Start()
	defer rc.Stop()

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
	assert.Greater(t, height, int64(previousHeight)) // height must be greater than previousHeight because confirmations is 0
	msgs, err := db.GetL2MessagesByStatus(orm.MsgPending)
	assert.NoError(t, err)
	assert.Equal(t, 5, len(msgs))
}

func prepareRelayerClient(l2Cli *ethclient.Client, db database.OrmFactory, v *viper.Viper) *l2.WatcherClient {
	return l2.NewL2WatcherClient(context.Background(), l2Cli, db, v)
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
