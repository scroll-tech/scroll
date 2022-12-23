package integration

import (
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"math/big"
	"scroll-tech/database"
	"scroll-tech/database/migrate"
	"testing"
	"time"
)

func testNative(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(&database.DBConfig{
		DriverName: "postgres",
		DSN:        dbImg.Endpoint(),
	})
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()
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

	// create and send native tx.
	sender := newSender(t, l2gethImg.Endpoint())
	to := common.HexToAddress("")
	_, err = sender.SendTransaction("native_01", &to, big.NewInt(100), nil)
	assert.NoError(t, err)
}

func testERC20(t *testing.T) {}

func testNFT(t *testing.T) {}

func testGreeter(t *testing.T) {}

func testSuShi(t *testing.T) {}

func testDao(t *testing.T) {}

func testUniswapV2(t *testing.T) {}
