package coordinator_test

import (
	"context"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"

	"scroll-tech/common/docker"
	"scroll-tech/common/message"
	"scroll-tech/common/utils"

	"scroll-tech/database"
	"scroll-tech/database/migrate"
	"scroll-tech/database/orm"

	"scroll-tech/coordinator"
	client2 "scroll-tech/coordinator/client"

	bridge_config "scroll-tech/bridge/config"

	coordinator_config "scroll-tech/coordinator/config"
)

const managerURL = "localhost:8132"

var (
	cfg           *bridge_config.Config
	dbImg         docker.ImgInstance
	rollerManager *coordinator.Manager
	handle        *http.Server
)

func setEnv(t *testing.T) error {
	var err error
	// Load config.
	cfg, err = bridge_config.NewConfig("../bridge/config.json")
	assert.NoError(t, err)

	// Create db container.
	dbImg = docker.NewTestDBDocker(t, cfg.DBConfig.DriverName)
	cfg.DBConfig.DSN = dbImg.Endpoint()

	// start roller manager
	rollerManager = setupRollerManager(t, "", cfg.DBConfig)

	// start ws service
	handle, _, err = utils.StartWSEndpoint(managerURL, rollerManager.APIs())
	assert.NoError(t, err)
	return err
}

func TestApis(t *testing.T) {
	// Set up the test environment.
	assert.True(t, assert.NoError(t, setEnv(t)), "failed to setup the test environment.")

	t.Run("TestHandshake", testHandshake)
	t.Run("TestSeveralConnections", testSeveralConnections)
	t.Run("TestIdleRollerSelection", testIdleRollerSelection)
	t.Run("TestRollerReconnect", testRollerReconnect)
	t.Run("TestGracefulRestart", testGracefulRestart)

	// Teardown
	t.Cleanup(func() {
		handle.Shutdown(context.Background())
		rollerManager.Stop()
		dbImg.Stop()
	})
}

func testHandshake(t *testing.T) {
	// Create db handler and reset db.
	l2db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(l2db.GetDB().DB))
	defer l2db.Close()

	stopCh := make(chan struct{})
	performHandshake(t, 1, false, "roller_test", stopCh)

	assert.Equal(t, 1, rollerManager.GetNumberOfIdleRollers())

	close(stopCh)
}

func testSeveralConnections(t *testing.T) {
	// Create db handler and reset db.
	l2db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(l2db.GetDB().DB))
	defer l2db.Close()

	var (
		batch  = 100
		stopCh = make(chan struct{})
		eg     = errgroup.Group{}
	)
	for i := 0; i < batch; i++ {
		idx := i
		eg.Go(func() error {
			performHandshake(t, 1, false, "roller_test"+strconv.Itoa(idx), stopCh)
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
	// Create db handler and reset db.
	l2db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(l2db.GetDB().DB))
	defer l2db.Close()

	// create ws connections.
	batch := 20
	stopCh := make(chan struct{})
	for i := 0; i < batch; i++ {
		performHandshake(t, 1, false, "roller_test"+strconv.Itoa(i), stopCh)
	}
	assert.Equal(t, batch, rollerManager.GetNumberOfIdleRollers())
	defer close(stopCh)

	var ids = make([]string, 2)
	dbTx, err := l2db.Beginx()
	assert.NoError(t, err)
	for i := range ids {
		ID, err := l2db.NewBatchInDBTx(dbTx, &orm.BlockInfo{Number: uint64(i)}, &orm.BlockInfo{Number: uint64(i)}, "0f", 1, 194676)
		assert.NoError(t, err)
		ids[i] = ID
	}
	assert.NoError(t, dbTx.Commit())

	// verify proof status
	var (
		tick     = time.Tick(500 * time.Millisecond)
		tickStop = time.Tick(10 * time.Second)
	)
	for len(ids) > 0 {
		select {
		case <-tick:
			status, err := l2db.GetProvingStatusByID(ids[0])
			assert.NoError(t, err)
			if status == orm.ProvingTaskVerified {
				ids = ids[1:]
			}
		case <-tickStop:
			t.Error("failed to check proof status")
			return
		}
	}
}

func testRollerReconnect(t *testing.T) {
	// Create db handler and reset db.
	l2db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(l2db.GetDB().DB))
	defer l2db.Close()

	var ids = make([]string, 2)
	dbTx, err := l2db.Beginx()
	assert.NoError(t, err)
	for i := range ids {
		ID, err := l2db.NewBatchInDBTx(dbTx, &orm.BlockInfo{Number: uint64(i)}, &orm.BlockInfo{Number: uint64(i)}, "0f", 1, 194676)
		assert.NoError(t, err)
		ids[i] = ID
	}
	assert.NoError(t, dbTx.Commit())

	// create ws connections.
	batch := 2
	stopCh := make(chan struct{})
	for i := 0; i < batch; i++ {
		performHandshake(t, 5, true, "roller_test"+strconv.Itoa(i), stopCh)
	}
	defer close(stopCh)

	// verify proof status
	var (
		tick     = time.Tick(500 * time.Millisecond)
		tickStop = time.Tick(15 * time.Second)
	)
	for len(ids) > 0 {
		select {
		case <-tick:
			status, err := l2db.GetProvingStatusByID(ids[0])
			assert.NoError(t, err)
			if status == orm.ProvingTaskVerified {
				ids = ids[1:]
			}
		case <-tickStop:
			t.Error("failed to check proof status")
			return
		}
	}
}

func testGracefulRestart(t *testing.T) {
	// Create db handler and reset db.
	l2db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(l2db.GetDB().DB))
	defer l2db.Close()

	var ids = make([]string, 2)
	dbTx, err := l2db.Beginx()
	assert.NoError(t, err)
	for i := range ids {
		ID, err := l2db.NewBatchInDBTx(dbTx, &orm.BlockInfo{Number: uint64(i)}, &orm.BlockInfo{Number: uint64(i)}, "0f", 1, 194676)
		assert.NoError(t, err)
		ids[i] = ID
	}
	assert.NoError(t, dbTx.Commit())

	// create ws connections.
	batch := 2
	stopCh := make(chan struct{})
	for i := 0; i < batch; i++ {
		performHandshake(t, 5, true, "roller_test"+strconv.Itoa(i), stopCh)
	}
	defer close(stopCh)

	// wait for task dispatch
	<-time.After(3 * time.Second)

	// restart coordinator
	rollerManager.Stop()
	rollerManager = setupRollerManager(t, "", cfg.DBConfig)

	// verify proof status
	var (
		tick     = time.Tick(500 * time.Millisecond)
		tickStop = time.Tick(15 * time.Second)
	)
	for len(ids) > 0 {
		select {
		case <-tick:
			status, err := l2db.GetProvingStatusByID(ids[0])
			assert.NoError(t, err)
			if status == orm.ProvingTaskVerified {
				ids = ids[1:]
			}
		case <-tickStop:
			t.Error("failed to check proof status")
			return
		}
	}
}

func setupRollerManager(t *testing.T, verifierEndpoint string, dbCfg *database.DBConfig) *coordinator.Manager {
	// Get db handler.
	db, err := database.NewOrmFactory(dbCfg)
	assert.True(t, assert.NoError(t, err), "failed to get db handler.")

	rollerManager, err := coordinator.New(context.Background(), &coordinator_config.RollerManagerConfig{
		RollersPerSession: 1,
		VerifierEndpoint:  verifierEndpoint,
		CollectionTime:    1,
	}, db)
	assert.NoError(t, err)
	assert.NoError(t, rollerManager.Start())

	return rollerManager
}

// performHandshake sets up a websocket client to connect to the roller manager.
func performHandshake(t *testing.T, proofTime time.Duration, isReconnect bool, name string, stopCh chan struct{}) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create a new ws connection
	client, err := client2.DialContext(ctx, "ws://"+managerURL)
	assert.NoError(t, err)

	// create private key
	privkey, err := crypto.GenerateKey()
	assert.NoError(t, err)

	authMsg := &message.AuthMsg{
		Identity: &message.Identity{
			Name:      name,
			Timestamp: time.Now().UnixNano(),
		},
	}
	assert.NoError(t, authMsg.Sign(privkey))

	taskCh := make(chan *message.TaskMsg, 4)
	sub, err := client.RegisterAndSubscribe(ctx, taskCh, authMsg)
	if err != nil {
		t.Error(err)
		return
	}

	go func() {
		for {
			select {
			case task := <-taskCh:
				id := task.ID
				// sleep several seconds to mock the proof process.
				<-time.After(proofTime * time.Second)
				if isReconnect {
					sub.Unsubscribe()
					sub, err = client.RegisterAndSubscribe(context.Background(), taskCh, authMsg)
					if err != nil {
						t.Error(err)
						return
					}
				}
				proof := &message.ProofMsg{
					ProofDetail: &message.ProofDetail{
						ID:     id,
						Status: message.StatusOk,
						Proof:  &message.AggProof{},
					},
				}
				assert.NoError(t, proof.Sign(privkey))
				ok, err := client.SubmitProof(context.Background(), proof)
				assert.NoError(t, err)
				assert.Equal(t, true, ok)
			case <-stopCh:
				sub.Unsubscribe()
				return
			}
		}
	}()
}
