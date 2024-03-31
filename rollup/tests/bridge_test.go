package tests

import (
	"context"
	"crypto/rand"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"scroll-tech/common/database"
	dockercompose "scroll-tech/common/docker-compose/l1"
	tc "scroll-tech/common/testcontainers"
	"scroll-tech/common/utils"
	"scroll-tech/database/migrate"

	"github.com/gin-gonic/gin"
	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	bcmd "scroll-tech/rollup/cmd"
	"scroll-tech/rollup/mock_bridge"
)

var (
	testApps     *tc.TestcontainerApps
	rollupApp    *bcmd.MockApp
	posL1TestEnv *dockercompose.PoSL1TestEnv

	// clients
	l1Client *ethclient.Client
	l2Client *ethclient.Client

	// l1Auth
	l1Auth *bind.TransactOpts
)

func setupDB(t *testing.T) *gorm.DB {
	dsn, err := testApps.GetDBEndPoint()
	assert.NoError(t, err)

	cfg := &database.Config{
		DSN:        dsn,
		DriverName: "postgres",
		MaxOpenNum: 200,
		MaxIdleNum: 20,
	}
	db, err := database.InitDB(cfg)
	assert.NoError(t, err)
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))
	return db
}

func TestMain(m *testing.M) {
	posL1TestEnv, err := dockercompose.NewPoSL1TestEnv()
	if err != nil {
		log.Crit("failed to create PoS L1 test environment", "err", err)
	}
	if err := posL1TestEnv.Start(); err != nil {
		log.Crit("failed to start PoS L1 test environment", "err", err)
	}
	defer func() {
		ctx := context.Background()
		if testApps != nil {
			testApps.Free(ctx)
		}
		if rollupApp != nil {
			rollupApp.Free()
		}
		if posL1TestEnv != nil {
			posL1TestEnv.Stop()
		}
	}()
	m.Run()
}

func setupEnv(t *testing.T) {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.LogfmtFormat()))
	glogger.Verbosity(log.LvlInfo)
	log.Root().SetHandler(glogger)

	var (
		err           error
		l1GethChainID *big.Int
	)

	testApps = tc.NewTestcontainerApps()
	assert.NoError(t, testApps.StartPostgresContainer())
	assert.NoError(t, testApps.StartL1GethContainer())
	assert.NoError(t, testApps.StartL2GethContainer())

	l1Client, err = testApps.GetL1GethClient()
	assert.NoError(t, err)
	l2Client, err = testApps.GetL2GethClient()
	assert.NoError(t, err)

	l1GethChainID, err = l1Client.ChainID(context.Background())
	assert.NoError(t, err)

	rollupApp = bcmd.NewRollupApp2(testApps, "../conf/config.json")
	l1Cfg, l2Cfg := rollupApp.Config.L1Config, rollupApp.Config.L2Config
	l1Cfg.Confirmations = 0
	l1Cfg.RelayerConfig.SenderConfig.Confirmations = 0
	l2Cfg.Confirmations = 0
	l2Cfg.RelayerConfig.SenderConfig.Confirmations = 0

	l1Auth, err = bind.NewKeyedTransactorWithChainID(rollupApp.Config.L2Config.RelayerConfig.CommitSenderPrivateKey, l1GethChainID)
	assert.NoError(t, err)

	port, err := rand.Int(rand.Reader, big.NewInt(10000))
	assert.NoError(t, err)
	svrPort := strconv.FormatInt(port.Int64()+40000, 10)
	rollupApp.Config.L2Config.RelayerConfig.ChainMonitor.BaseURL = "http://localhost:" + svrPort
}

func mockChainMonitorServer(baseURL string) (*http.Server, error) {
	router := gin.New()
	r := router.Group("/v1")
	r.GET("/batch_status", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, struct {
			ErrCode int    `json:"errcode"`
			ErrMsg  string `json:"errmsg"`
			Data    bool   `json:"data"`
		}{
			ErrCode: 0,
			ErrMsg:  "",
			Data:    true,
		})
	})
	return utils.StartHTTPServer(strings.Split(baseURL, "//")[1], router)
}

func prepareContracts(t *testing.T) {
	// L1 ScrolChain contract
	nonce, err := l1Client.PendingNonceAt(context.Background(), l1Auth.From)
	assert.NoError(t, err)
	scrollChainAddress := crypto.CreateAddress(l1Auth.From, nonce)
	tx := types.NewContractCreation(nonce, big.NewInt(0), 10000000, big.NewInt(1000000000), common.FromHex(mock_bridge.MockBridgeMetaData.Bin))
	signedTx, err := l1Auth.Signer(l1Auth.From, tx)
	assert.NoError(t, err)
	err = l1Client.SendTransaction(context.Background(), signedTx)
	assert.NoError(t, err)

	assert.Eventually(t, func() bool {
		_, isPending, err := l1Client.TransactionByHash(context.Background(), signedTx.Hash())
		return err == nil && !isPending
	}, 30*time.Second, time.Second)

	assert.Eventually(t, func() bool {
		receipt, err := l1Client.TransactionReceipt(context.Background(), signedTx.Hash())
		return err == nil && receipt.Status == gethTypes.ReceiptStatusSuccessful
	}, 30*time.Second, time.Second)

	assert.Eventually(t, func() bool {
		code, err := l1Client.CodeAt(context.Background(), scrollChainAddress, nil)
		return err == nil && len(code) > 0
	}, 30*time.Second, time.Second)

	l1Config, l2Config := rollupApp.Config.L1Config, rollupApp.Config.L2Config
	l1Config.ScrollChainContractAddress = scrollChainAddress
	l2Config.RelayerConfig.RollupContractAddress = scrollChainAddress
}

func TestFunction(t *testing.T) {
	setupEnv(t)
	srv, err := mockChainMonitorServer(rollupApp.Config.L2Config.RelayerConfig.ChainMonitor.BaseURL)
	assert.NoError(t, err)
	defer srv.Close()

	// process start test
	t.Run("TestProcessStart", testProcessStart)
	t.Run("TestProcessStartEnableMetrics", testProcessStartEnableMetrics)

	// l1 rollup and watch rollup events
	t.Run("TestCommitAndFinalizeGenesisBatch", testCommitAndFinalizeGenesisBatch)
	t.Run("TestCommitBatchAndFinalizeBatch", testCommitBatchAndFinalizeBatch)
	t.Run("TestCommitBatchAndFinalizeBatch4844", testCommitBatchAndFinalizeBatch4844)
	t.Run("TestCommitBatchAndFinalizeBatchBeforeAndPost4844", testCommitBatchAndFinalizeBatchBeforeAndPost4844)

	// l1/l2 gas oracle
	t.Run("TestImportL1GasPrice", testImportL1GasPrice)
	t.Run("TestImportL2GasPrice", testImportL2GasPrice)
}
