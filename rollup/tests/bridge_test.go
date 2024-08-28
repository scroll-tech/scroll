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

	"scroll-tech/database/migrate"

	"scroll-tech/common/database"
	tc "scroll-tech/common/testcontainers"
	"scroll-tech/common/utils"

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
	testApps  *tc.TestcontainerApps
	rollupApp *bcmd.MockApp

	// clients
	l1Client *ethclient.Client
	l2Client *ethclient.Client

	l1Auth *bind.TransactOpts
	l2Auth *bind.TransactOpts
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
	defer func() {
		if testApps != nil {
			testApps.Free()
		}
		if rollupApp != nil {
			rollupApp.Free()
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
		l2GethChainID *big.Int
	)

	testApps = tc.NewTestcontainerApps()
	assert.NoError(t, testApps.StartPostgresContainer())
	assert.NoError(t, testApps.StartL2GethContainer())
	assert.NoError(t, testApps.StartPoSL1Container())
	rollupApp = bcmd.NewRollupApp(testApps, "../conf/config.json")

	l1Client, err = testApps.GetPoSL1Client()
	assert.NoError(t, err)
	l2Client, err = testApps.GetL2GethClient()
	assert.NoError(t, err)
	l1GethChainID, err = l1Client.ChainID(context.Background())
	assert.NoError(t, err)
	l2GethChainID, err = l2Client.ChainID(context.Background())
	assert.NoError(t, err)

	l1Cfg, l2Cfg := rollupApp.Config.L1Config, rollupApp.Config.L2Config
	l1Cfg.RelayerConfig.SenderConfig.Confirmations = 0
	l2Cfg.Confirmations = 0
	l2Cfg.RelayerConfig.SenderConfig.Confirmations = 0

	pKey, err := crypto.ToECDSA(common.FromHex(l2Cfg.RelayerConfig.CommitSenderPrivateKey))
	assert.NoError(t, err)
	l1Auth, err = bind.NewKeyedTransactorWithChainID(pKey, l1GethChainID)
	assert.NoError(t, err)

	pKey, err = crypto.ToECDSA(common.FromHex(l2Cfg.RelayerConfig.GasOracleSenderPrivateKey))
	assert.NoError(t, err)
	l2Auth, err = bind.NewKeyedTransactorWithChainID(pKey, l2GethChainID)
	assert.NoError(t, err)

	port, err := rand.Int(rand.Reader, big.NewInt(10000))
	assert.NoError(t, err)
	svrPort := strconv.FormatInt(port.Int64()+40000, 10)
	l2Cfg.RelayerConfig.ChainMonitor.BaseURL = "http://localhost:" + svrPort
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
	mockL1ContractAddress := crypto.CreateAddress(l1Auth.From, nonce)
	tx := types.NewContractCreation(nonce, big.NewInt(0), 10000000, big.NewInt(1000000000), common.FromHex(mock_bridge.MockBridgeMetaData.Bin))
	signedTx, err := l1Auth.Signer(l1Auth.From, tx)
	assert.NoError(t, err)
	err = l1Client.SendTransaction(context.Background(), signedTx)
	assert.NoError(t, err)

	assert.Eventually(t, func() bool {
		_, isPending, getErr := l1Client.TransactionByHash(context.Background(), signedTx.Hash())
		return getErr == nil && !isPending
	}, 30*time.Second, time.Second)

	assert.Eventually(t, func() bool {
		receipt, getErr := l1Client.TransactionReceipt(context.Background(), signedTx.Hash())
		return getErr == nil && receipt.Status == gethTypes.ReceiptStatusSuccessful
	}, 30*time.Second, time.Second)

	assert.Eventually(t, func() bool {
		code, getErr := l1Client.CodeAt(context.Background(), mockL1ContractAddress, nil)
		return getErr == nil && len(code) > 0
	}, 30*time.Second, time.Second)

	// L2 ScrolChain contract
	nonce, err = l2Client.PendingNonceAt(context.Background(), l2Auth.From)
	assert.NoError(t, err)
	mockL2ContractAddress := crypto.CreateAddress(l2Auth.From, nonce)
	tx = types.NewContractCreation(nonce, big.NewInt(0), 2000000, big.NewInt(1000000000), common.FromHex(mock_bridge.MockBridgeMetaData.Bin))
	signedTx, err = l2Auth.Signer(l2Auth.From, tx)
	assert.NoError(t, err)
	err = l2Client.SendTransaction(context.Background(), signedTx)
	assert.NoError(t, err)

	assert.Eventually(t, func() bool {
		_, isPending, err := l2Client.TransactionByHash(context.Background(), signedTx.Hash())
		return err == nil && !isPending
	}, 30*time.Second, time.Second)

	assert.Eventually(t, func() bool {
		receipt, err := l2Client.TransactionReceipt(context.Background(), signedTx.Hash())
		return err == nil && receipt.Status == gethTypes.ReceiptStatusSuccessful
	}, 30*time.Second, time.Second)

	assert.Eventually(t, func() bool {
		code, err := l2Client.CodeAt(context.Background(), mockL2ContractAddress, nil)
		return err == nil && len(code) > 0
	}, 30*time.Second, time.Second)

	l1Config, l2Config := rollupApp.Config.L1Config, rollupApp.Config.L2Config
	l2Config.RelayerConfig.RollupContractAddress = mockL1ContractAddress

	l2Config.RelayerConfig.GasPriceOracleContractAddress = mockL1ContractAddress
	l1Config.RelayerConfig.GasPriceOracleContractAddress = mockL2ContractAddress
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
	t.Run("testCommitBatchAndFinalizeBatchOrBundleWithAllCodecVersions", testCommitBatchAndFinalizeBatchOrBundleWithAllCodecVersions)
	t.Run("TestCommitBatchAndFinalizeBatchOrBundleCrossingAllTransitions", testCommitBatchAndFinalizeBatchOrBundleCrossingAllTransitions)

	// l1/l2 gas oracle
	t.Run("TestImportL1GasPrice", testImportL1GasPrice)
	t.Run("TestImportL1GasPriceAfterCurie", testImportL1GasPriceAfterCurie)
	t.Run("TestImportDefaultL1GasPriceDueToL1GasPriceSpike", testImportDefaultL1GasPriceDueToL1GasPriceSpike)
	t.Run("TestImportL2GasPrice", testImportL2GasPrice)
}
