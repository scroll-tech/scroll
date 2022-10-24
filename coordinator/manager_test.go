package coordinator_test

import (
	"context"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"

	"scroll-tech/common/docker"
	"scroll-tech/common/utils"
	"scroll-tech/database"
	"scroll-tech/database/orm"

	"scroll-tech/bridge/l2"
	"scroll-tech/bridge/mock"

	client2 "scroll-tech/coordinator/client"

	db_config "scroll-tech/database"

	bridge_config "scroll-tech/bridge/config"

	"scroll-tech/coordinator"
	coordinator_config "scroll-tech/coordinator/config"
)

const managerURL = "localhost:8132"
const managerPort = ":8132"

var (
	TEST_CONFIG = &mock.TestConfig{
		L2GethTestConfig: mock.L2GethTestConfig{
			HPort: 8536,
			WPort: 0,
		},
		DbTestconfig: mock.DbTestconfig{
			DbName: "testmanager_db",
			DbPort: 5436,
			DB_CONFIG: &db_config.DBConfig{
				DriverName: utils.GetEnvWithDefault("TEST_DB_DRIVER", "postgres"),
				DSN:        utils.GetEnvWithDefault("TEST_DB_DSN", "postgres://postgres:123456@localhost:5436/testmanager_db?sslmode=disable"),
			},
		},
	}
)

var (
	cfg            *bridge_config.Config
	l2backend      *l2.Backend
	imgGeth, imgDb docker.ImgInstance
	db             database.OrmFactory
	rollerManager  *coordinator.Manager
	handle         *http.Server
)

func setEnv(t *testing.T) {
	var err error
	cfg, err = bridge_config.NewConfig("./config.json")
	if err != nil {
		t.Error(err)
		return
	}

	// create docker instance
	l2backend, imgGeth, imgDb = mock.L2gethDocker(t, cfg, TEST_CONFIG)

	// reset db and return orm handler
	db = mock.ResetDB(t, TEST_CONFIG.DB_CONFIG)

	// start roller manager
	rollerManager = setupRollerManager(t, "", db)

	// start ws service
	handle, _, err = utils.StartWSEndpoint(managerURL, rollerManager.APIs())
	assert.NoError(t, err)
}

func TestApis(t *testing.T) {
	setEnv(t)

	t.Run("TestHandshake", testHandshake)
	t.Run("TestSeveralConnections", testSeveralConnections)
	t.Run("TestIdleRollerSelection", testIdleRollerSelection)

	t.Cleanup(func() {
		handle.Shutdown(context.Background())
		rollerManager.Stop()
		l2backend.Stop()
		imgGeth.Stop()
		imgDb.Stop()
	})
}

func testHandshake(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// reset db
	mock.ResetDB(t, TEST_CONFIG.DB_CONFIG)

	// create a new
	client, err := client2.DialContext(ctx, "ws://"+managerURL)
	assert.NoError(t, err)

	stopCh := make(chan struct{})
	mock.PerformHandshake(t, 1, "roller_test", client, stopCh)

	assert.Equal(t, 1, rollerManager.GetNumberOfIdleRollers())

	close(stopCh)
}

func testSeveralConnections(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		batch  = 100
		stopCh = make(chan struct{})
		eg     = errgroup.Group{}
	)
	for i := 0; i < batch; i++ {
		idx := i
		eg.Go(func() error {
			// create a new ws connection
			client, err := client2.DialContext(ctx, "ws://"+managerURL)
			assert.NoError(t, err)
			mock.PerformHandshake(t, 1, "roller_test"+strconv.Itoa(idx), client, stopCh)
			return nil
		})
	}
	assert.NoError(t, eg.Wait())

	// check roller's idle connections
	assert.Equal(t, batch, rollerManager.GetNumberOfIdleRollers())

	// close connection
	close(stopCh)

	var (
		tick     = time.Tick(time.Second)
		tickStop = time.Tick(time.Second * 10)
	)
	for {
		select {
		case <-tick:
			if rollerManager.GetNumberOfIdleRollers() == 0 {
				return
			}
		case <-tickStop:
			t.Error("roller connect is blocked")
			return
		}
	}
}

func testIdleRollerSelection(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// reset db
	traceDB := orm.BlockResultOrm(mock.ResetDB(t, TEST_CONFIG.DB_CONFIG))

	// create l2geth client
	ethClient, err := ethclient.DialContext(ctx, imgGeth.Endpoint())
	assert.NoError(t, err)

	// create ws connections.
	batch := 20
	stopCh := make(chan struct{})
	for i := 0; i < batch; i++ {
		var client *client2.Client
		client, err = client2.DialContext(ctx, "ws://"+managerURL)
		assert.NoError(t, err)
		mock.PerformHandshake(t, 1, "roller_test"+strconv.Itoa(i), client, stopCh)
	}
	assert.Equal(t, batch, rollerManager.GetNumberOfIdleRollers())

	// send two txs
	mock.SendTxToL2Client(t, ethClient, cfg.L2Config.RelayerConfig.PrivateKey)
	mock.SendTxToL2Client(t, ethClient, cfg.L2Config.RelayerConfig.PrivateKey)

	// verify proof status
	var (
		number   int64 = 1
		latest   int64
		tick     = time.Tick(time.Second)
		tickStop = time.Tick(time.Second * 20)
	)
	for {
		select {
		case <-tick:
			// get the latest number
			if latest, err = traceDB.GetBlockResultsLatestHeight(); err != nil || latest <= 0 {
				continue
			}
			if number > latest {
				close(stopCh)
				return
			}
			status, err := traceDB.GetBlockStatusByNumber(uint64(number))
			if err == nil && (status == orm.BlockVerified || status == orm.BlockSkipped) {
				number++
			}
		case <-tickStop:
			t.Error("failed to check proof status")
			close(stopCh)
			return
		}
	}
}

func setupRollerManager(t *testing.T, verifierEndpoint string, orm database.OrmFactory) *coordinator.Manager {
	rollerManager, err := coordinator.New(context.Background(), &coordinator_config.RollerManagerConfig{
		Endpoint:          managerPort,
		RollersPerSession: 1,
		VerifierEndpoint:  verifierEndpoint,
		CollectionTime:    1,
	}, orm)
	assert.NoError(t, err)

	assert.NoError(t, rollerManager.Start())

	return rollerManager
}
