package tests

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"

	"scroll-tech/bridge/cmd/app"
	"scroll-tech/bridge/config"
	"scroll-tech/bridge/mock_bridge"

	"scroll-tech/common/docker"
)

var (
	// config
	cfg *config.Config

	// private key
	privateKey *ecdsa.PrivateKey

	bridgeApp *app.BridgeApp
	base      *docker.App

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
	scrollChainInstance *mock_bridge.MockBridgeL1
	scrollChainAddress  common.Address

	// l2 messenger contract
	l2MessengerInstance *mock_bridge.MockBridgeL2
	l2MessengerAddress  common.Address
)

func TestMain(m *testing.M) {
	base = docker.NewDockerApp()
	bridgeApp = app.NewBridgeApp(base, "../config.json")
	err := bridgeApp.MockConfig(false)
	if err != nil {
		panic(err)
	}
	// Load config.
	cfg = bridgeApp.Config

	m.Run()

	bridgeApp.Free()
	base.Free()
}

func setupEnv(t *testing.T) {
	// Start l1geth l2geth and postgres docker containers.
	base.RunImages(t)

	var err error
	privateKey, err = crypto.ToECDSA(common.FromHex("1212121212121212121212121212121212121212121212121212121212121212"))
	assert.NoError(t, err)
	// Create l1geth and l2geth client.
	l1Client, err = base.L1Client()
	assert.NoError(t, err)
	l2Client, err = base.L2Client()
	assert.NoError(t, err)

	// Create l1 and l2 auth
	l1Auth = prepareAuth(t, base.L1gethImg.ChainID(), privateKey)
	l2Auth = prepareAuth(t, base.L2gethImg.ChainID(), privateKey)
}

func prepareContracts(t *testing.T) {
	var err error
	var tx *types.Transaction

	// L1 messenger contract
	_, tx, l1MessengerInstance, err = mock_bridge.DeployMockBridgeL1(l1Auth, l1Client)
	assert.NoError(t, err)
	l1MessengerAddress, err = bind.WaitDeployed(context.Background(), l1Client, tx)
	assert.NoError(t, err)

	// L1 ScrolChain contract
	_, tx, scrollChainInstance, err = mock_bridge.DeployMockBridgeL1(l1Auth, l1Client)
	assert.NoError(t, err)
	scrollChainAddress, err = bind.WaitDeployed(context.Background(), l1Client, tx)
	assert.NoError(t, err)

	// L2 messenger contract
	_, tx, l2MessengerInstance, err = mock_bridge.DeployMockBridgeL2(l2Auth, l2Client)
	assert.NoError(t, err)
	l2MessengerAddress, err = bind.WaitDeployed(context.Background(), l2Client, tx)
	assert.NoError(t, err)

	cfg.L1Config.L1MessengerAddress = l1MessengerAddress
	cfg.L1Config.L1MessageQueueAddress = l1MessengerAddress
	cfg.L1Config.ScrollChainContractAddress = scrollChainAddress
	cfg.L1Config.RelayerConfig.MessengerContractAddress = l2MessengerAddress
	cfg.L1Config.RelayerConfig.GasPriceOracleContractAddress = l1MessengerAddress

	cfg.L2Config.L2MessengerAddress = l2MessengerAddress
	cfg.L2Config.L2MessageQueueAddress = l2MessengerAddress
	cfg.L2Config.RelayerConfig.MessengerContractAddress = l1MessengerAddress
	cfg.L2Config.RelayerConfig.RollupContractAddress = scrollChainAddress
	cfg.L2Config.RelayerConfig.GasPriceOracleContractAddress = l2MessengerAddress
}

func prepareAuth(t *testing.T, chainID *big.Int, privateKey *ecdsa.PrivateKey) *bind.TransactOpts {
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

	// l1 message
	t.Run("TestRelayL1MessageSucceed", testRelayL1MessageSucceed)

	// l2 message
	t.Run("TestRelayL2MessageSucceed", testRelayL2MessageSucceed)

	// l1/l2 gas oracle
	t.Run("TestImportL1GasPrice", testImportL1GasPrice)
	t.Run("TestImportL2GasPrice", testImportL2GasPrice)

}
