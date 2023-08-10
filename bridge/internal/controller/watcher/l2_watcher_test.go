package watcher

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"testing"

	"gorm.io/gorm"

	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/database"
	cutils "scroll-tech/common/utils"

	"scroll-tech/bridge/internal/orm"
	"scroll-tech/bridge/mock_bridge"
)

func setupL2Watcher(t *testing.T) (*L2WatcherClient, *gorm.DB) {
	db := setupDB(t)
	l2cfg := cfg.L2Config
	watcher := NewL2WatcherClient(context.Background(), l2Cli, l2cfg.L2MessageQueueAddress, l2cfg.WithdrawTrieRootSlot, db)
	return watcher, db
}

func testFetchRunningMissingBlocks(t *testing.T) {
	_, db := setupL2Watcher(t)
	defer database.CloseDB(db)

	auth := prepareAuth(t, l2Cli, cfg.L2Config.RelayerConfig.RollupSenderPrivateKeys[0])

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
	return NewL2WatcherClient(context.Background(), l2Cli, contractAddr, common.Hash{}, db)
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
