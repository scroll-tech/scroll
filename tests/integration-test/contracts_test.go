package integration

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/crypto"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/stretchr/testify/assert"

	"scroll-tech/database"
	"scroll-tech/database/orm"

	"scroll-tech/common/utils"
)

func testContracts(t *testing.T) {
	// Create configs.
	mockConfig(t)
	// migrate db.
	runDBCliApp(t, "reset", "successful to reset")
	runDBCliApp(t, "migrate", "current version:")

	// Start bridge process.
	bridgeCmd := runBridgeApp(t)
	bridgeCmd.RunApp(true)
	bridgeCmd.ExpectWithTimeout(false, time.Second*10, "Start bridge successfully")

	// Start coordinator process.
	coordinatorCmd := runCoordinatorApp(t, "--ws", "--ws.port", "8391")
	coordinatorCmd.RunApp(true)
	coordinatorCmd.ExpectWithTimeout(false, time.Second*10, "Start coordinator successfully")

	// Start roller process.
	rollerCmd := runRollerApp(t)
	rollerCmd.RunApp(true)
	rollerCmd.ExpectWithTimeout(false, time.Second*20, "roller start successfully")

	// test native call.
	t.Run("testNative", testNative)
	t.Run("testERC20", testERC20)
	t.Run("testNFT", testNFT)
	t.Run("testGreeter", testGreeter)
	t.Run("testSuShi", testSuShi)
	t.Run("testDao", testDao)
	t.Run("testUniswapV2", testUniswapV2)

	rollerCmd.WaitExit()
	bridgeCmd.WaitExit()
	coordinatorCmd.WaitExit()
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
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(&database.DBConfig{
		DriverName: "postgres",
		DSN:        dbImg.Endpoint(),
	})
	assert.NoError(t, err)
	defer db.Close()

	pk, _ := crypto.GenerateKey()
	auth, _ := bind.NewKeyedTransactorWithChainID(pk, big.NewInt(53077))

	// create new NFT operations.
	err = newNft(context.Background(), l2Client, l2Root, auth)
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

func testGreeter(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(&database.DBConfig{
		DriverName: "postgres",
		DSN:        dbImg.Endpoint(),
	})
	assert.NoError(t, err)
	defer db.Close()

	// create new Greeter operations.
	err = newGreeter(context.Background(), l2Client, l2Root)
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

func testSuShi(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(&database.DBConfig{
		DriverName: "postgres",
		DSN:        dbImg.Endpoint(),
	})
	assert.NoError(t, err)
	defer db.Close()

	// create new Sushi operations.
	err = newSushi(context.Background(), l2Client, l2Root)
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

func testDao(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(&database.DBConfig{
		DriverName: "postgres",
		DSN:        dbImg.Endpoint(),
	})
	assert.NoError(t, err)
	defer db.Close()

	// create new Dao operations.
	err = newDao(context.Background(), l2Client, l2Root)
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

func testUniswapV2(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(&database.DBConfig{
		DriverName: "postgres",
		DSN:        dbImg.Endpoint(),
	})
	assert.NoError(t, err)
	defer db.Close()

	pk, _ := crypto.GenerateKey()
	auth, _ := bind.NewKeyedTransactorWithChainID(pk, big.NewInt(53077))

	// add eth to auth.from
	err = native(context.Background(), auth.From, big.NewInt(1).Mul(big.NewInt(3e3), utils.Ether))
	assert.NoError(t, err)

	// create new uniswap operations.
	err = newUniswapv2(context.Background(), l2Client, l2Root, auth)
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
