package tests

import (
	"context"
	"testing"

	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"scroll-tech/common/database"
	"scroll-tech/common/docker"

	"scroll-tech/database/migrate"

	bcmd "scroll-tech/bridge/cmd"
	"scroll-tech/bridge/mock_bridge"
)

var (
	base      *docker.App
	bridgeApp *bcmd.MockApp

	// clients
	l1Client *ethclient.Client
	l2Client *ethclient.Client

	// auth
	l1Auth *bind.TransactOpts
	l2Auth *bind.TransactOpts

	// l1 rollup contract
	scrollChainInstance *mock_bridge.MockBridgeL1
	scrollChainAddress  common.Address

	// l2 messenger contract
	l2MessengerInstance *mock_bridge.MockBridgeL2
	l2MessengerAddress  common.Address
)

func setupDB(t *testing.T) *gorm.DB {
	cfg := &database.Config{
		DSN:        base.DBConfig.DSN,
		DriverName: base.DBConfig.DriverName,
		MaxOpenNum: base.DBConfig.MaxOpenNum,
		MaxIdleNum: base.DBConfig.MaxIdleNum,
	}
	db, err := database.InitDB(cfg)
	assert.NoError(t, err)
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))
	return db
}

func TestMain(m *testing.M) {
	base = docker.NewDockerApp()
	bridgeApp = bcmd.NewBridgeApp(base, "../conf/config.json")
	defer bridgeApp.Free()
	defer base.Free()
	m.Run()
}

func setupEnv(t *testing.T) {
	base.RunImages(t)

	var err error
	l1Client, err = base.L1Client()
	assert.NoError(t, err)
	l2Client, err = base.L2Client()
	assert.NoError(t, err)

	l1Cfg, l2Cfg := bridgeApp.Config.L1Config, bridgeApp.Config.L2Config
	l1Cfg.Confirmations = 0
	l1Cfg.RelayerConfig.SenderConfig.Confirmations = 0
	l2Cfg.Confirmations = 0
	l2Cfg.RelayerConfig.SenderConfig.Confirmations = 0

	l1Auth, err = bind.NewKeyedTransactorWithChainID(bridgeApp.Config.L2Config.RelayerConfig.CommitSenderPrivateKey, base.L1gethImg.ChainID())
	assert.NoError(t, err)

	l2Auth, err = bind.NewKeyedTransactorWithChainID(bridgeApp.Config.L1Config.RelayerConfig.CommitSenderPrivateKey, base.L2gethImg.ChainID())
	assert.NoError(t, err)
}

func prepareContracts(t *testing.T) {
	var err error
	var tx *types.Transaction

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

	l1Config, l2Config := bridgeApp.Config.L1Config, bridgeApp.Config.L2Config
	l1Config.ScrollChainContractAddress = scrollChainAddress

	l2Config.L2MessengerAddress = l2MessengerAddress
	l2Config.L2MessageQueueAddress = l2MessengerAddress
	l2Config.RelayerConfig.RollupContractAddress = scrollChainAddress
	l2Config.RelayerConfig.GasPriceOracleContractAddress = l2MessengerAddress
}

func TestFunction(t *testing.T) {
	setupEnv(t)

	// process start test
	t.Run("TestProcessStart", testProcessStart)
	t.Run("TestProcessStartEnableMetrics", testProcessStartEnableMetrics)

	// l1 rollup and watch rollup events
	t.Run("TestCommitBatchAndFinalizeBatch", testCommitBatchAndFinalizeBatch)

	// l1 message
	// t.Run("TestRelayL1MessageSucceed", testRelayL1MessageSucceed)

	// l2 message
	// TODO: add a "user relay l2msg Succeed" test

	// l1/l2 gas oracle
	t.Run("TestImportL1GasPrice", testImportL1GasPrice)
	t.Run("TestImportL2GasPrice", testImportL2GasPrice)

}
