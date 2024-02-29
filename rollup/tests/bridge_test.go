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

	"github.com/gin-gonic/gin"
	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"scroll-tech/common/database"
	"scroll-tech/common/docker"
	"scroll-tech/common/utils"

	"scroll-tech/database/migrate"

	bcmd "scroll-tech/rollup/cmd"
	"scroll-tech/rollup/mock_bridge"
)

var (
	base      *docker.App
	rollupApp *bcmd.MockApp

	// clients
	l1Client *ethclient.Client
	l2Client *ethclient.Client

	// auth
	l1Auth *bind.TransactOpts
	l2Auth *bind.TransactOpts

	// l1 rollup contract
	scrollChainInstance *mock_bridge.MockBridgeL1
	scrollChainAddress  common.Address
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
	rollupApp = bcmd.NewRollupApp(base, "../conf/config.json")
	defer rollupApp.Free()
	defer base.Free()
	m.Run()
}

func setupEnv(t *testing.T) {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.LogfmtFormat()))
	glogger.Verbosity(log.LvlInfo)
	log.Root().SetHandler(glogger)

	base.RunImages(t)

	var err error
	l1Client, err = base.L1Client()
	assert.NoError(t, err)
	l2Client, err = base.L2Client()
	assert.NoError(t, err)

	l1Cfg, l2Cfg := rollupApp.Config.L1Config, rollupApp.Config.L2Config
	l1Cfg.Confirmations = 0
	l1Cfg.RelayerConfig.SenderConfig.Confirmations = 0
	l2Cfg.Confirmations = 0
	l2Cfg.RelayerConfig.SenderConfig.Confirmations = 0

	l1Auth, err = bind.NewKeyedTransactorWithChainID(rollupApp.Config.L2Config.RelayerConfig.CommitSenderPrivateKey, base.L1gethImg.ChainID())
	assert.NoError(t, err)

	l2Auth, err = bind.NewKeyedTransactorWithChainID(rollupApp.Config.L1Config.RelayerConfig.GasOracleSenderPrivateKey, base.L2gethImg.ChainID())
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
	var err error
	var tx *types.Transaction

	// L1 ScrollChain contract
	_, tx, scrollChainInstance, err = mock_bridge.DeployMockBridgeL1(l1Auth, l1Client)
	assert.NoError(t, err)
	scrollChainAddress, err = bind.WaitDeployed(context.Background(), l1Client, tx)
	assert.NoError(t, err)

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

	// l1/l2 gas oracle
	t.Run("TestImportL1GasPrice", testImportL1GasPrice)
	t.Run("TestImportL2GasPrice", testImportL2GasPrice)
}
