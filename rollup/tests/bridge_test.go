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

	"scroll-tech/common/database"
	"scroll-tech/common/docker"
	dockercompose "scroll-tech/common/docker-compose/l1"
	"scroll-tech/common/utils"

	"scroll-tech/database/migrate"

	bcmd "scroll-tech/rollup/cmd"
	"scroll-tech/rollup/mock_bridge"
)

var (
	base         *docker.App
	rollupApp    *bcmd.MockApp
	posL1TestEnv *dockercompose.PoSL1TestEnv

	// clients
	l1Client *ethclient.Client
	l2Client *ethclient.Client

	// l1Auth
	l1Auth *bind.TransactOpts
)

func setupDB(t *testing.T) *gorm.DB {
	cfg := &database.Config{
		DSN:         base.DBConfig.DSN,
		DriverName:  base.DBConfig.DriverName,
		MaxOpenNum:  base.DBConfig.MaxOpenNum,
		MaxIdleNum:  base.DBConfig.MaxIdleNum,
		MaxLifetime: base.DBConfig.MaxLifetime,
		MaxIdleTime: base.DBConfig.MaxIdleTime,
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
	defer base.Free()

	rollupApp = bcmd.NewRollupApp(base, "../conf/config.json")
	defer rollupApp.Free()

	var err error
	posL1TestEnv, err = dockercompose.NewPoSL1TestEnv()
	if err != nil {
		log.Crit("failed to create PoS L1 test environment", "err", err)
	}
	if err := posL1TestEnv.Start(); err != nil {
		log.Crit("failed to start PoS L1 test environment", "err", err)
	}
	defer posL1TestEnv.Stop()

	m.Run()
}

func setupEnv(t *testing.T) {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.LogfmtFormat()))
	glogger.Verbosity(log.LvlInfo)
	log.Root().SetHandler(glogger)

	var err error
	l1Client, err = posL1TestEnv.L1Client()
	assert.NoError(t, err)
	chainID, err := l1Client.ChainID(context.Background())
	assert.NoError(t, err)
	l1Auth, err = bind.NewKeyedTransactorWithChainID(rollupApp.Config.L2Config.RelayerConfig.CommitSenderPrivateKey, chainID)
	assert.NoError(t, err)
	rollupApp.Config.L1Config.Endpoint = posL1TestEnv.Endpoint()
	rollupApp.Config.L2Config.RelayerConfig.SenderConfig.Endpoint = posL1TestEnv.Endpoint()

	base.RunImages(t)

	l2Client, err = base.L2Client()
	assert.NoError(t, err)

	l1Cfg, l2Cfg := rollupApp.Config.L1Config, rollupApp.Config.L2Config
	l1Cfg.Confirmations = 0
	l1Cfg.RelayerConfig.SenderConfig.Confirmations = 0
	l2Cfg.Confirmations = 0
	l2Cfg.RelayerConfig.SenderConfig.Confirmations = 0

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
