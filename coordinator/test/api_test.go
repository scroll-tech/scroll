package test

import (
	"compress/flate"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"

	"scroll-tech/common/docker"
	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
	"scroll-tech/common/utils"
	"scroll-tech/database/migrate"

	client2 "scroll-tech/coordinator/client"
	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/controller/api"
	"scroll-tech/coordinator/internal/controller/cron"
	"scroll-tech/coordinator/internal/logic/rollermanager"
	"scroll-tech/coordinator/internal/orm"
	coordinatorType "scroll-tech/coordinator/internal/types"
	coordinatorUtils "scroll-tech/coordinator/internal/utils"
)

var (
	base      *docker.App
	batchData *coordinatorType.BatchData
)

func TestMain(m *testing.M) {
	base = docker.NewDockerApp()
	m.Run()
	base.Free()
}

func randomURL() string {
	id, _ := rand.Int(rand.Reader, big.NewInt(2000-1))
	return fmt.Sprintf("localhost:%d", 10000+2000+id.Int64())
}

func setEnv(t *testing.T) (err error) {
	base.RunDBImage(t)
	templateBlockTrace, err := os.ReadFile("../testdata/blockTrace_02.json")
	if err != nil {
		return err
	}
	// unmarshal blockTrace
	wrappedBlock := &coordinatorType.WrappedBlock{}
	if err = json.Unmarshal(templateBlockTrace, wrappedBlock); err != nil {
		return err
	}

	parentBatch := &coordinatorType.BatchInfo{
		Index: 1,
		Hash:  "0x0000000000000000000000000000000000000000",
	}
	batchData = coordinatorType.NewBatchData(parentBatch, []*coordinatorType.WrappedBlock{wrappedBlock}, nil)

	return
}

func setupDB(t *testing.T) *gorm.DB {
	dbConf := config.DBConfig{
		DSN:        base.DBConfig.DSN,
		DriverName: base.DBConfig.DriverName,
		MaxOpenNum: base.DBConfig.MaxOpenNum,
		MaxIdleNum: base.DBConfig.MaxIdleNum,
	}
	db, err := coordinatorUtils.InitDB(&dbConf)
	assert.NoError(t, err)
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))
	return db
}

func setupCoordinator(t *testing.T, rollersPerSession uint8, wsURL string, db *gorm.DB) (*http.Server, *gorm.DB, *cron.Collector) {
	if db == nil {
		db = setupDB(t)
	}
	conf := config.Config{
		RollerManagerConfig: &config.RollerManagerConfig{
			RollersPerSession:  rollersPerSession,
			Verifier:           &config.VerifierConfig{MockMode: true},
			CollectionTime:     1,
			TokenTimeToLive:    5,
			MaxVerifierWorkers: 10,
			SessionAttempts:    2,
		},
	}
	tmpApi := api.APIs(&conf, db)
	handler, _, err := utils.StartWSEndpoint(strings.Split(wsURL, "//")[1], tmpApi, flate.NoCompression)
	assert.NoError(t, err)
	rollermanager.InitRollerManager()
	proofCollector := cron.NewCollector(context.Background(), db, &conf)
	return handler, db, proofCollector
}

func TestApis(t *testing.T) {
	// Set up the test environment.
	base = docker.NewDockerApp()
	assert.True(t, assert.NoError(t, setEnv(t)), "failed to setup the test environment.")

	t.Run("TestHandshake", testHandshake)
	t.Run("TestFailedHandshake", testFailedHandshake)
	t.Run("TestSeveralConnections", testSeveralConnections)
	t.Run("TestValidProof", testValidProof)
	t.Run("TestInvalidProof", testInvalidProof)
	t.Run("TestProofGeneratedFailed", testProofGeneratedFailed)
	t.Run("TestTimeoutProof", testTimeoutProof)
	t.Run("TestIdleRollerSelection", testIdleRollerSelection)
	t.Run("TestGracefulRestart", testGracefulRestart)
	//t.Run("TestListRollers", testListRollers)

	// Teardown
	t.Cleanup(func() {
		base.Free()
	})
}

func testHandshake(t *testing.T) {
	// Setup coordinator and ws server.
	wsURL := "ws://" + randomURL()
	handler, db, proofCollector := setupCoordinator(t, 1, wsURL, nil)
	defer func() {
		coordinatorUtils.CloseDB(db)
		handler.Shutdown(context.Background())
		proofCollector.Stop()
	}()

	roller := newMockRoller(t, "roller_test", wsURL)
	defer roller.close()

	assert.Equal(t, 1, rollermanager.Manager.GetNumberOfIdleRollers(message.BasicProve))
}

func testFailedHandshake(t *testing.T) {
	// Setup coordinator and ws server.
	wsURL := "ws://" + randomURL()
	handler, db, proofCollector := setupCoordinator(t, 1, wsURL, nil)
	defer func() {
		coordinatorUtils.CloseDB(db)
		handler.Shutdown(context.Background())
		proofCollector.Stop()
	}()

	// prepare
	name := "roller_test"
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
			Timestamp: uint32(time.Now().Unix()),
		},
	}
	assert.NoError(t, authMsg.SignWithKey(privkey))
	_, err = client.RegisterAndSubscribe(ctx, make(chan *message.TaskMsg, 4), authMsg)
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
			Timestamp: uint32(time.Now().Unix()),
		},
	}
	assert.NoError(t, authMsg.SignWithKey(privkey))
	token, err := client.RequestToken(ctx, authMsg)
	assert.NoError(t, err)

	authMsg.Identity.Token = token
	assert.NoError(t, authMsg.SignWithKey(privkey))

	<-time.After(6 * time.Second)
	_, err = client.RegisterAndSubscribe(ctx, make(chan *message.TaskMsg, 4), authMsg)
	assert.Error(t, err)

	assert.Equal(t, 0, rollermanager.Manager.GetNumberOfIdleRollers(message.BasicProve))
}

func testSeveralConnections(t *testing.T) {
	wsURL := "ws://" + randomURL()
	handler, db, proofCollector := setupCoordinator(t, 1, wsURL, nil)
	defer func() {
		coordinatorUtils.CloseDB(db)
		handler.Shutdown(context.Background())
		proofCollector.Stop()
	}()

	var (
		batch   = 100
		eg      = errgroup.Group{}
		rollers = make([]*mockRoller, batch)
	)
	for i := 0; i < batch; i++ {
		idx := i
		eg.Go(func() error {
			rollers[idx] = newMockRoller(t, "roller_test_"+strconv.Itoa(idx), wsURL)
			return nil
		})
	}
	assert.NoError(t, eg.Wait())

	// check roller's idle connections
	assert.Equal(t, batch, rollermanager.Manager.GetNumberOfIdleRollers(message.BasicProve))

	// close connection
	for _, roller := range rollers {
		roller.close()
	}

	var (
		tick     = time.Tick(time.Second)
		tickStop = time.Tick(time.Second * 15)
	)
	for {
		select {
		case <-tick:
			if rollermanager.Manager.GetNumberOfIdleRollers(message.BasicProve) == 0 {
				return
			}
		case <-tickStop:
			t.Error("roller connect is blocked")
			return
		}
	}
}

func testValidProof(t *testing.T) {
	wsURL := "ws://" + randomURL()
	handler, db, collector := setupCoordinator(t, 1, wsURL, nil)
	defer func() {
		coordinatorUtils.CloseDB(db)
		handler.Shutdown(context.Background())
		collector.Stop()
	}()

	// create mock rollers.
	rollers := make([]*mockRoller, 3)
	for i := 0; i < len(rollers); i++ {
		rollers[i] = newMockRoller(t, "roller_test"+strconv.Itoa(i), wsURL)
		// only roller 0 submits valid proof.
		proofStatus := verifiedSuccess
		if i > 0 {
			proofStatus = generatedFailed
		}
		rollers[i].waitTaskAndSendProof(t, time.Second, false, proofStatus)
	}
	defer func() {
		// close connection
		for _, roller := range rollers {
			roller.close()
		}
	}()
	assert.Equal(t, 3, rollermanager.Manager.GetNumberOfIdleRollers(message.BasicProve))

	var hashes = make([]string, 1)
	transErr := db.Transaction(func(tx *gorm.DB) error {
		for i := range hashes {
			assert.NoError(t, orm.AddBatchInfoToDB(tx, batchData))
			hashes[i] = batchData.Hash().Hex()
		}
		return nil
	})
	assert.NoError(t, transErr)

	// verify proof status
	var (
		tick     = time.Tick(500 * time.Millisecond)
		tickStop = time.Tick(time.Minute)
	)

	blockBatchOrm := orm.NewBlockBatch(db)
	for len(hashes) > 0 {
		select {
		case <-tick:
			tmpBlockBatches, err := blockBatchOrm.GetBlockBatches(map[string]interface{}{"hash": hashes[0]}, nil, 1)
			assert.NoError(t, err)
			assert.Equal(t, len(tmpBlockBatches), 1)
			if types.ProvingStatus(tmpBlockBatches[0].ProvingStatus) == types.ProvingTaskVerified {
				hashes = hashes[1:]
			}
		case <-tickStop:
			t.Error("failed to check proof status")
			return
		}
	}
}

func testInvalidProof(t *testing.T) {
	wsURL := "ws://" + randomURL()
	handler, db, collector := setupCoordinator(t, 3, wsURL, nil)
	defer func() {
		coordinatorUtils.CloseDB(db)
		handler.Shutdown(context.Background())
		collector.Stop()
	}()

	// create mock rollers.
	rollers := make([]*mockRoller, 3)
	for i := 0; i < len(rollers); i++ {
		rollers[i] = newMockRoller(t, "roller_test"+strconv.Itoa(i), wsURL)
		rollers[i].waitTaskAndSendProof(t, time.Second, false, verifiedFailed)
	}
	defer func() {
		for _, roller := range rollers {
			roller.close()
		}
	}()
	assert.Equal(t, 3, rollermanager.Manager.GetNumberOfIdleRollers(message.BasicProve))

	var hashes = make([]string, 1)
	transErr := db.Transaction(func(tx *gorm.DB) error {
		for i := range hashes {
			assert.NoError(t, orm.AddBatchInfoToDB(tx, batchData))
			hashes[i] = batchData.Hash().Hex()
		}
		return nil
	})
	assert.NoError(t, transErr)

	// verify proof status
	var (
		tick     = time.Tick(500 * time.Millisecond)
		tickStop = time.Tick(time.Minute)
	)

	blockBatchOrm := orm.NewBlockBatch(db)
	for len(hashes) > 0 {
		select {
		case <-tick:
			tmpBlockBatches, err := blockBatchOrm.GetBlockBatches(map[string]interface{}{"hash": hashes[0]}, nil, 1)
			assert.NoError(t, err)
			assert.Equal(t, len(tmpBlockBatches), 1)
			if types.ProvingStatus(tmpBlockBatches[0].ProvingStatus) == types.ProvingTaskFailed {
				hashes = hashes[1:]
			}
		case <-tickStop:
			t.Error("failed to check proof status")
			return
		}
	}
}

func testProofGeneratedFailed(t *testing.T) {
	wsURL := "ws://" + randomURL()
	handler, db, collector := setupCoordinator(t, 3, wsURL, nil)
	defer func() {
		coordinatorUtils.CloseDB(db)
		handler.Shutdown(context.Background())
		collector.Stop()
	}()

	// create mock rollers.
	rollers := make([]*mockRoller, 3)
	for i := 0; i < len(rollers); i++ {
		rollers[i] = newMockRoller(t, "roller_test"+strconv.Itoa(i), wsURL)
		rollers[i].waitTaskAndSendProof(t, time.Second, false, generatedFailed)
	}
	defer func() {
		// close connection
		for _, roller := range rollers {
			roller.close()
		}
	}()
	assert.Equal(t, 3, rollermanager.Manager.GetNumberOfIdleRollers(message.BasicProve))

	var hashes = make([]string, 1)
	transErr := db.Transaction(func(tx *gorm.DB) error {
		for i := range hashes {
			assert.NoError(t, orm.AddBatchInfoToDB(tx, batchData))
			hashes[i] = batchData.Hash().Hex()
		}
		return nil
	})
	assert.NoError(t, transErr)

	var (
		tick     = time.Tick(500 * time.Millisecond)
		tickStop = time.Tick(time.Minute)
	)
	blockBatchOrm := orm.NewBlockBatch(db)
	for len(hashes) > 0 {
		select {
		case <-tick:
			tmpBlockBatches, err := blockBatchOrm.GetBlockBatches(map[string]interface{}{"hash": hashes[0]}, nil, 1)
			assert.NoError(t, err)
			assert.Equal(t, len(tmpBlockBatches), 1)
			if types.ProvingStatus(tmpBlockBatches[0].ProvingStatus) == types.ProvingTaskFailed {
				hashes = hashes[1:]
			}
		case <-tickStop:
			t.Error("failed to check proof status")
			return
		}
	}
}

func testTimeoutProof(t *testing.T) {
	wsURL := "ws://" + randomURL()
	handler, db, collector := setupCoordinator(t, 1, wsURL, nil)
	defer func() {
		coordinatorUtils.CloseDB(db)
		handler.Shutdown(context.Background())
		collector.Stop()
	}()

	// create first mock roller, that will not send any proof.
	roller1 := newMockRoller(t, "roller_test"+strconv.Itoa(0), wsURL)
	defer func() {
		// close connection
		roller1.close()
	}()
	assert.Equal(t, 1, rollermanager.Manager.GetNumberOfIdleRollers(message.BasicProve))

	var (
		hashesAssigned = make([]string, 1)
		hashesVerified = make([]string, 1)
	)

	transErr := db.Transaction(func(tx *gorm.DB) error {
		for i := range hashesAssigned {
			assert.NoError(t, orm.AddBatchInfoToDB(tx, batchData))
			hashesAssigned[i] = batchData.Hash().Hex()
			hashesVerified[i] = batchData.Hash().Hex()
		}
		return nil
	})
	assert.NoError(t, transErr)

	blockBatchOrm := orm.NewBlockBatch(db)
	// verify proof status, it should be assigned, because roller didn't send any proof
	times := 0
	var tmpHashAggsigned string
	ok := utils.TryTimes(30, func() bool {
		times++
		t.Logf("check proving status assigned times:%d", times)
		tmpHashAggsigned = hashesAssigned[0]
		tmpBlockBatches, err := blockBatchOrm.GetBlockBatches(map[string]interface{}{"hash": hashesAssigned[0]}, nil, 1)
		if err != nil {
			return false
		}
		if len(tmpBlockBatches) != 1 {
			return false
		}
		if types.ProvingStatus(tmpBlockBatches[0].ProvingStatus) == types.ProvingTaskAssigned {
			hashesAssigned = hashesAssigned[1:]
		}
		return len(hashesAssigned) == 0
	})
	// because there is no roller run. the task is missing. so need change the status
	err := blockBatchOrm.UpdateProvingStatus(tmpHashAggsigned, types.ProvingTaskUnassigned)
	assert.NoError(t, err)
	assert.Falsef(t, !ok, "failed to check proof status")

	// create second mock roller, that will send valid proof.
	roller2 := newMockRoller(t, "roller_test"+strconv.Itoa(1), wsURL)
	roller2.waitTaskAndSendProof(t, time.Second, false, verifiedSuccess)
	defer func() {
		// close connection
		roller2.close()
	}()
	assert.Equal(t, 1, rollermanager.Manager.GetNumberOfIdleRollers(message.BasicProve))

	times = 0
	// verify proof status, it should be verified now, because second roller sent valid proof
	ok = utils.TryTimes(200, func() bool {
		times++
		t.Logf("check proving status verified times:%d", times)
		tmpBlockBatches, err := blockBatchOrm.GetBlockBatches(map[string]interface{}{"hash": hashesVerified[0]}, nil, 1)
		if err != nil {
			return false
		}
		if len(tmpBlockBatches) != 1 {
			return false
		}
		if types.ProvingStatus(tmpBlockBatches[0].ProvingStatus) == types.ProvingTaskVerified {
			hashesVerified = hashesVerified[1:]
		}
		return len(hashesVerified) == 0
	})
	assert.Falsef(t, !ok, "failed to check proof status")
}

func testIdleRollerSelection(t *testing.T) {
	wsURL := "ws://" + randomURL()
	handler, db, collector := setupCoordinator(t, 3, wsURL, nil)
	defer func() {
		coordinatorUtils.CloseDB(db)
		handler.Shutdown(context.Background())
		collector.Stop()
	}()

	// create mock rollers.
	rollers := make([]*mockRoller, 20)
	for i := 0; i < len(rollers); i++ {
		rollers[i] = newMockRoller(t, "roller_test"+strconv.Itoa(i), wsURL)
		rollers[i].waitTaskAndSendProof(t, time.Second, false, verifiedSuccess)
	}
	defer func() {
		for _, roller := range rollers {
			roller.close()
		}
	}()

	assert.Equal(t, len(rollers), rollermanager.Manager.GetNumberOfIdleRollers(message.BasicProve))

	var hashes = make([]string, 1)
	transErr := db.Transaction(func(tx *gorm.DB) error {
		for i := range hashes {
			assert.NoError(t, orm.AddBatchInfoToDB(tx, batchData))
			hashes[i] = batchData.Hash().Hex()
		}
		return nil
	})
	assert.NoError(t, transErr)

	// verify proof status
	var (
		tick     = time.Tick(500 * time.Millisecond)
		tickStop = time.Tick(time.Minute)
	)

	blockBatchOrm := orm.NewBlockBatch(db)
	for len(hashes) > 0 {
		select {
		case <-tick:
			tmpBlockBatches, err := blockBatchOrm.GetBlockBatches(map[string]interface{}{"hash": hashes[0]}, nil, 1)
			assert.NoError(t, err)
			assert.Equal(t, len(tmpBlockBatches), 1)
			if types.ProvingStatus(tmpBlockBatches[0].ProvingStatus) == types.ProvingTaskVerified {
				hashes = hashes[1:]
			}
		case <-tickStop:
			t.Error("failed to check proof status")
			return
		}
	}
}

func testGracefulRestart(t *testing.T) {
	wsURL := "ws://" + randomURL()
	handler, db, collector := setupCoordinator(t, 1, wsURL, nil)
	var hashes = make([]string, 1)
	transErr1 := db.Transaction(func(tx *gorm.DB) error {
		for i := range hashes {
			assert.NoError(t, orm.AddBatchInfoToDB(tx, batchData))
			hashes[i] = batchData.Hash().Hex()
		}
		return nil
	})
	assert.NoError(t, transErr1)

	// create mock roller
	roller := newMockRoller(t, "roller_test", wsURL)
	// wait 10 seconds, coordinator restarts before roller submits proof
	roller.waitTaskAndSendProof(t, 10*time.Second, false, verifiedSuccess)

	// wait for coordinator to dispatch task
	<-time.After(5 * time.Second)
	// the coordinator will delete the roller if the subscription is closed.
	roller.close()

	// Close rollerManager and ws handler.
	handler.Shutdown(context.Background())
	collector.Stop()

	// Setup new coordinator and ws server.
	newHandler, newDb, newCollector := setupCoordinator(t, 1, wsURL, db)
	defer func() {
		newHandler.Shutdown(context.Background())
		newCollector.Stop()
		coordinatorUtils.CloseDB(newDb)
	}()

	blockBatchOrm := orm.NewBlockBatch(newDb)
	for i := range hashes {
		// at this point, roller haven't submitted
		tmpBlockBatches, err := blockBatchOrm.GetBlockBatches(map[string]interface{}{"hash": hashes[i]}, nil, 1)
		assert.NoError(t, err)
		assert.Equal(t, len(tmpBlockBatches), 1)
		if types.ProvingStatus(tmpBlockBatches[0].ProvingStatus) == types.ProvingTaskAssigned {
			hashes = hashes[1:]
		}
	}

	// will overwrite the roller client for `SubmitProof`
	roller.waitTaskAndSendProof(t, time.Millisecond*500, true, verifiedSuccess)
	defer roller.close()

	// verify proof status
	var (
		tick     = time.Tick(500 * time.Millisecond)
		tickStop = time.Tick(time.Minute)
	)
	for len(hashes) > 0 {
		select {
		case <-tick:
			// this proves that the roller submits to the new coordinator,
			// because the roller client for `submitProof` has been overwritten
			tmpBlockBatches, err := blockBatchOrm.GetBlockBatches(map[string]interface{}{"hash": hashes[0]}, nil, 1)
			assert.NoError(t, err)
			assert.Equal(t, len(tmpBlockBatches), 1)
			if types.ProvingStatus(tmpBlockBatches[0].ProvingStatus) == types.ProvingTaskVerified {
				hashes = hashes[1:]
			}
		case <-tickStop:
			t.Error("failed to check proof status")
			return
		}
	}
}
