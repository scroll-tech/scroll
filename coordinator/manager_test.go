package coordinator_test

import (
	"context"
	"crypto/ecdsa"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/scroll-tech/go-ethereum"
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
const newManagerURL = "localhost:8133"

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
	t.Run("TestFailedHandshake", testFailedHandshake)
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

	roller := newMockRoller(t, "roller_test")
	defer roller.close()
	roller.connectToCoordinator(t, managerURL)

	assert.Equal(t, 1, rollerManager.GetNumberOfIdleRollers())
}

func testFailedHandshake(t *testing.T) {
	// Create db handler and reset db.
	l2db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(l2db.GetDB().DB))
	defer l2db.Close()

	stopCh := make(chan struct{})

	// prepare
	name := "roller_test"
	wsURL := "ws://" + managerURL
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Try to perform handshake without token
	// create a new ws connection
	client, err := client2.DialContext(ctx, wsURL)
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
	_, err = client.RegisterAndSubscribe(ctx, taskCh, authMsg)
	assert.Error(t, err)

	// Try to perform handshake with timeouted token
	// create a new ws connection
	client, err = client2.DialContext(ctx, wsURL)
	assert.NoError(t, err)
	// create private key
	privkey, err = crypto.GenerateKey()
	assert.NoError(t, err)

	authMsg = &message.AuthMsg{
		Identity: &message.Identity{
			Name:      name,
			Timestamp: time.Now().UnixNano(),
		},
	}
	assert.NoError(t, authMsg.Sign(privkey))
	token, err := client.RequestToken(ctx, authMsg)
	assert.NoError(t, err)

	authMsg.Identity.Token = token
	assert.NoError(t, authMsg.Sign(privkey))

	tick := time.Tick(6 * time.Second)

	<-tick
	taskCh = make(chan *message.TaskMsg, 4)
	_, err = client.RegisterAndSubscribe(ctx, taskCh, authMsg)
	assert.Error(t, err)

	assert.Equal(t, 0, rollerManager.GetNumberOfIdleRollers())

	close(stopCh)
}

func testSeveralConnections(t *testing.T) {
	// Create db handler and reset db.
	l2db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(l2db.GetDB().DB))
	defer l2db.Close()

	var (
		batch   = 100
		eg      = errgroup.Group{}
		rollers = make([]*mockRoller, batch)
	)
	for i := 0; i < batch; i++ {
		idx := i
		eg.Go(func() error {
			rollers[idx] = newMockRoller(t, "roller_test"+strconv.Itoa(idx))
			rollers[idx].connectToCoordinator(t, managerURL)
			return nil
		})
	}
	assert.NoError(t, eg.Wait())

	// check roller's idle connections
	assert.Equal(t, batch, rollerManager.GetNumberOfIdleRollers())

	// close connection
	for i := 0; i < batch; i++ {
		rollers[i].close()
	}

	var (
		tick     = time.Tick(time.Second)
		tickStop = time.Tick(time.Second * 15)
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

	// create mock rollers.
	batch := 20
	rollers := make([]*mockRoller, batch)
	for i := 0; i < batch; i++ {
		rollers[i] = newMockRoller(t, "roller_test"+strconv.Itoa(i))
		defer rollers[i].close()
		rollers[i].connectToCoordinator(t, managerURL)
		go rollers[i].waitTaskAndSendProof(t, 1, false)
	}
	assert.Equal(t, batch, rollerManager.GetNumberOfIdleRollers())

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

	var ids = make([]string, 1)
	dbTx, err := l2db.Beginx()
	assert.NoError(t, err)
	for i := range ids {
		ID, err := l2db.NewBatchInDBTx(dbTx, &orm.BlockInfo{Number: uint64(i)}, &orm.BlockInfo{Number: uint64(i)}, "0f", 1, 194676)
		assert.NoError(t, err)
		ids[i] = ID
	}
	assert.NoError(t, dbTx.Commit())

	// create mock roller
	roller := newMockRoller(t, "roller_test")
	defer roller.close()
	roller.connectToCoordinator(t, managerURL)
	go roller.waitTaskAndSendProof(t, 1, true)

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

	var ids = make([]string, 1)
	dbTx, err := l2db.Beginx()
	assert.NoError(t, err)
	for i := range ids {
		ids[i], err = l2db.NewBatchInDBTx(dbTx, &orm.BlockInfo{Number: uint64(i)}, &orm.BlockInfo{Number: uint64(i)}, "0f", 1, 194676)
		assert.NoError(t, err)
	}
	assert.NoError(t, dbTx.Commit())

	// create mock roller
	roller := newMockRoller(t, "roller_test")
	roller.connectToCoordinator(t, managerURL)

	// wait 5 seconds, coordinator restarts before roller submits proof
	go roller.waitTaskAndSendProof(t, 5, true)

	// wait for coordinator to dispatch task
	<-time.After(3 * time.Second)

	// the coordinator will delete the roller if the subscription is closed.
	roller.close()

	// start new roller manager && ws service
	newRollerManager := setupRollerManager(t, "", cfg.DBConfig)
	handle, _, err = utils.StartWSEndpoint(newManagerURL, newRollerManager.APIs())
	assert.NoError(t, err)
	defer func() {
		newRollerManager.Stop()
		handle.Shutdown(context.Background())
	}()

	for i := range ids {
		_, err = newRollerManager.GetSessionInfo(ids[i])
		assert.NoError(t, err)
	}

	// will overwrite the roller client for `SubmitProof`
	roller.connectToCoordinator(t, newManagerURL)
	defer roller.close()

	// at this point, roller haven't submitted
	status, err := l2db.GetProvingStatusByID(ids[0])
	assert.NoError(t, err)
	assert.Equal(t, orm.ProvingTaskAssigned, status)

	// verify proof status
	var (
		tick     = time.Tick(500 * time.Millisecond)
		tickStop = time.Tick(15 * time.Second)
	)
	for len(ids) > 0 {
		select {
		case <-tick:
			// this proves that the roller submits to the new coordinator,
			// because the roller client for `submitProof` has been overwritten
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
		TokenTimeToLive:   5,
	}, db)
	assert.NoError(t, err)
	assert.NoError(t, rollerManager.Start())

	return rollerManager
}

type mockRoller struct {
	rollerName string
	privKey    *ecdsa.PrivateKey
	taskCh     chan *message.TaskMsg
	sub        ethereum.Subscription
	client     *client2.Client
	stopCh     chan struct{}
}

func newMockRoller(t *testing.T, rollerName string) *mockRoller {
	privKey, err := crypto.GenerateKey()
	assert.NoError(t, err)
	return &mockRoller{
		rollerName: rollerName,
		privKey:    privKey,
		taskCh:     make(chan *message.TaskMsg, 4)}
}

// connectToCoordinator sets up a websocket client to connect to the roller manager.
func (r *mockRoller) connectToCoordinator(t *testing.T, wsURL string) {
	// create a new ws connection
	var err error
	r.client, err = client2.Dial("ws://" + wsURL)
	assert.NoError(t, err)

	authMsg := &message.AuthMsg{
		Identity: &message.Identity{
			Name:      r.rollerName,
			Timestamp: time.Now().UnixNano(),
		},
	}
	assert.NoError(t, authMsg.Sign(r.privKey))

	token, err := client.RequestToken(context.Background(), authMsg)
	assert.NoError(t, err)
	authMsg.Identity.Token = token

	assert.NoError(t, authMsg.Sign(privkey))
	r.sub, err = r.client.RegisterAndSubscribe(context.Background(), r.taskCh, authMsg)
	assert.NoError(t, err)

	r.stopCh = make(chan struct{})

	go func() {
		<-r.stopCh
		r.sub.Unsubscribe()
	}()
}

// Wait for the proof task, after receiving the proof task, roller submits proof after proofTime secs.
func (r *mockRoller) waitTaskAndSendProof(t *testing.T, proofTime time.Duration, reconnectBeforeSendProof bool) {
	for {
		task := <-r.taskCh
		// simulate proof time
		<-time.After(proofTime * time.Second)
		if reconnectBeforeSendProof {
			// simulating the case that the roller first disconnects and then reconnects to the coordinator
			// the Subscription and its `Err()` channel will be closed, and the coordinator will `freeRoller()`
			r.reconnetToCoordinator(t)
		}
		proof := &message.ProofMsg{
			ProofDetail: &message.ProofDetail{
				ID:     task.ID,
				Status: message.StatusOk,
				Proof:  &message.AggProof{},
			},
		}
		assert.NoError(t, proof.Sign(r.privKey))
		ok, err := r.client.SubmitProof(context.Background(), proof)
		assert.NoError(t, err)
		assert.Equal(t, true, ok)
	}
}

func (r *mockRoller) reconnetToCoordinator(t *testing.T) {
	authMsg := &message.AuthMsg{
		Identity: &message.Identity{
			Name:      r.rollerName,
			Timestamp: time.Now().UnixNano(),
		},
	}
	assert.NoError(t, authMsg.Sign(r.privKey))
	r.sub.Unsubscribe()
	var err error
	r.sub, err = r.client.RegisterAndSubscribe(context.Background(), r.taskCh, authMsg)
	assert.NoError(t, err)
}

func (r *mockRoller) close() {
	close(r.stopCh)
}
