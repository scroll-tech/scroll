package tests

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"scroll-tech/common/docker"
	"testing"

	"scroll-tech/bridge/config"
	"scroll-tech/bridge/mock_bridge"

	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"
)

var (
	// config
	cfg *config.Config

	// private key
	privateKey *ecdsa.PrivateKey

	// docker consider handler.
	l1gethImg docker.ImgInstance
	l2gethImg docker.ImgInstance
	dbImg     docker.ImgInstance

	// clients
	l1Client *ethclient.Client
	l2Client *ethclient.Client

	// auth
	l1Auth *bind.TransactOpts
	l2Auth *bind.TransactOpts

	// l1 messenger contract
	l1MessengerInstance *mock_bridge.MockBridgeL1
	l1MessengerAddress  common.Address

	// l1 rollup contract
	l1RollupInstance *mock_bridge.MockBridgeL1
	l1RollupAddress  common.Address

	// l2 messenger contract
	l2MessengerInstance *mock_bridge.MockBridgeL2
	l2MessengerAddress  common.Address
)

func setupEnv(t *testing.T) {
	var err error
	privateKey, err = crypto.ToECDSA(common.FromHex("1212121212121212121212121212121212121212121212121212121212121212"))
	assert.NoError(t, err)

	// Load config.
	cfg, err = config.NewConfig("../config.json")
	assert.NoError(t, err)
	cfg.L1Config.Confirmations = 0
	cfg.L1Config.RelayerConfig.MessageSenderPrivateKeys = []*ecdsa.PrivateKey{privateKey}
	cfg.L1Config.RelayerConfig.RollupSenderPrivateKeys = []*ecdsa.PrivateKey{privateKey}
	cfg.L2Config.Confirmations = 0
	cfg.L2Config.RelayerConfig.MessageSenderPrivateKeys = []*ecdsa.PrivateKey{privateKey}
	cfg.L2Config.RelayerConfig.RollupSenderPrivateKeys = []*ecdsa.PrivateKey{privateKey}

	// Create l1geth container.
	l1gethImg = docker.NewTestL1Docker(t)
	cfg.L2Config.RelayerConfig.SenderConfig.Endpoint = l1gethImg.Endpoint()
	cfg.L1Config.Endpoint = l1gethImg.Endpoint()

	// Create l2geth container.
	l2gethImg = docker.NewTestL2Docker(t)
	cfg.L1Config.RelayerConfig.SenderConfig.Endpoint = l2gethImg.Endpoint()
	cfg.L2Config.Endpoint = l2gethImg.Endpoint()

	// Create db container.
	dbImg = docker.NewTestDBDocker(t, cfg.DBConfig.DriverName)
	cfg.DBConfig.DSN = dbImg.Endpoint()

	// Create l1geth and l2geth client.
	l1Client, err = ethclient.Dial(cfg.L1Config.Endpoint)
	assert.NoError(t, err)
	l2Client, err = ethclient.Dial(cfg.L2Config.Endpoint)
	assert.NoError(t, err)

	// Create l1 and l2 auth
	l1Auth = prepareAuth(t, l1Client, privateKey)
	l2Auth = prepareAuth(t, l2Client, privateKey)
}

func free(t *testing.T) {
	if dbImg != nil {
		assert.NoError(t, dbImg.Stop())
	}
	if l1gethImg != nil {
		assert.NoError(t, l1gethImg.Stop())
	}
	if l2gethImg != nil {
		assert.NoError(t, l2gethImg.Stop())
	}
}

func prepareContracts(t *testing.T) {
	var err error
	var tx *types.Transaction

	// L1 messenger contract
	_, tx, l1MessengerInstance, err = mock_bridge.DeployMockBridgeL1(l1Auth, l1Client)
	assert.NoError(t, err)
	l1MessengerAddress, err = bind.WaitDeployed(context.Background(), l1Client, tx)
	assert.NoError(t, err)

	// L1 rollup contract
	_, tx, l1RollupInstance, err = mock_bridge.DeployMockBridgeL1(l1Auth, l1Client)
	assert.NoError(t, err)
	l1RollupAddress, err = bind.WaitDeployed(context.Background(), l1Client, tx)
	assert.NoError(t, err)

	// L2 messenger contract
	_, tx, l2MessengerInstance, err = mock_bridge.DeployMockBridgeL2(l2Auth, l2Client)
	assert.NoError(t, err)
	l2MessengerAddress, err = bind.WaitDeployed(context.Background(), l2Client, tx)
	assert.NoError(t, err)

	cfg.L1Config.L1MessengerAddress = l1MessengerAddress
	cfg.L1Config.RollupContractAddress = l1RollupAddress
	cfg.L1Config.RelayerConfig.MessengerContractAddress = l2MessengerAddress

	cfg.L2Config.L2MessengerAddress = l2MessengerAddress
	cfg.L2Config.RelayerConfig.MessengerContractAddress = l1MessengerAddress
	cfg.L2Config.RelayerConfig.RollupContractAddress = l1RollupAddress
}

func prepareAuth(t *testing.T, client *ethclient.Client, privateKey *ecdsa.PrivateKey) *bind.TransactOpts {
	chainID, err := client.ChainID(context.Background())
	assert.NoError(t, err)
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	assert.NoError(t, err)
	auth.Value = big.NewInt(0) // in wei
	assert.NoError(t, err)
	return auth
}

func TestFunction(t *testing.T) {
	setupEnv(t)

	// l1 rollup and watch rollup events
	t.Run("TestCommitBatchAndFinalizeBatch", testCommitBatchAndFinalizeBatch)

	t.Cleanup(func() {
		free(t)
	})
}
