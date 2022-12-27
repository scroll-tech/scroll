package integration

import (
	"context"
	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/crypto"
	"math/big"
	"testing"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/utils"
	"scroll-tech/database"
	"scroll-tech/database/orm"
)

func testContracts(t *testing.T) {
	// migrate db.
	runDBCliApp(t, "reset", "successful to reset")
	runDBCliApp(t, "migrate", "current version:")

	// Start bridge process.
	bridgeCmd := runBridgeApp(t)
	bridgeCmd.RunApp(true)
	bridgeCmd.ExpectWithTimeout(false, time.Second*10, "Start bridge successfully")
	defer bridgeCmd.WaitExit()

	// Start coordinator process.
	coordinatorCmd := runCoordinatorApp(t, "--ws", "--ws.port", "8391")
	coordinatorCmd.RunApp(true)
	coordinatorCmd.ExpectWithTimeout(false, time.Second*10, "Start coordinator successfully")
	defer coordinatorCmd.WaitExit()

	// Start roller process.
	rollerCmd := runRollerApp(t)
	rollerCmd.RunApp(true)
	rollerCmd.ExpectWithTimeout(false, time.Second*20, "roller start successfully")
	defer rollerCmd.WaitExit()

	// test native call.
	t.Run("testNative", testNative)
	t.Run("testERC20", testERC20)
}

func testNative(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(&database.DBConfig{
		DriverName: "postgres",
		DSN:        dbImg.Endpoint(),
	})
	assert.NoError(t, err)
	defer db.Close()

	// create and send native tx.
	to := common.HexToAddress("0x1c5a77d9fa7ef466951b2f01f724bca3a5820b63")
	err = native(context.Background(), to, big.NewInt(100))
	assert.NoError(t, err)

	number, err := l2Client.BlockNumber(context.Background())
	assert.NoError(t, err)

	// Wait all the ids were verified.
	utils.TryTimes(20, func() bool {
		var (
			id     string
			status orm.ProvingStatus
		)
		id, err = db.GetBatchIDByNumber(number)
		if err != nil {
			return false
		}
		status, err = db.GetProvingStatusByID(id)
		return err == nil && status == orm.ProvingTaskVerified
	})
	assert.NoError(t, err)
}

func testERC20(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(&database.DBConfig{
		DriverName: "postgres",
		DSN:        dbImg.Endpoint(),
	})
	assert.NoError(t, err)
	defer db.Close()

	pk, _ := crypto.GenerateKey()
	auth, _ := bind.NewKeyedTransactorWithChainID(pk, big.NewInt(53077))

	// erc20 operations.
	err = newERC20(context.Background(), l2Client, l2Root, auth)
	assert.NoError(t, err)

	// Wait all the ids were verified.
	number, err := l2Client.BlockNumber(context.Background())
	assert.NoError(t, err)
	utils.TryTimes(20, func() bool {
		var (
			id     string
			status orm.ProvingStatus
		)
		id, err = db.GetBatchIDByNumber(number)
		if err != nil {
			return false
		}
		status, err = db.GetProvingStatusByID(id)
		return err == nil && status == orm.ProvingTaskVerified
	})
	assert.NoError(t, err)
}

func testNFT(t *testing.T) {

}

func testGreeter(t *testing.T) {}

func testSuShi(t *testing.T) {}

func testDao(t *testing.T) {}

func testUniswapV2(t *testing.T) {}
