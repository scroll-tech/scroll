package test

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"scroll-tech/common/testcontainers"
	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
	"scroll-tech/common/version"
	"scroll-tech/database/migrate"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/controller/api"
	"scroll-tech/coordinator/internal/controller/cron"
	"scroll-tech/coordinator/internal/orm"
	"scroll-tech/coordinator/internal/route"
)

var (
	conf *config.Config

	testApps *testcontainers.TestcontainerApps

	db                 *gorm.DB
	l2BlockOrm         *orm.L2Block
	chunkOrm           *orm.Chunk
	batchOrm           *orm.Batch
	proverTaskOrm      *orm.ProverTask
	proverBlockListOrm *orm.ProverBlockList

	block1       *encoding.Block
	block2       *encoding.Block
	chunk        *encoding.Chunk
	batch        *encoding.Batch
	tokenTimeout int
)

func TestMain(m *testing.M) {
	defer func() {
		if testApps != nil {
			testApps.Free()
		}
	}()
	m.Run()
}

func randomURL() string {
	id, _ := rand.Int(rand.Reader, big.NewInt(2000-1))
	return fmt.Sprintf("localhost:%d", 10000+2000+id.Int64())
}

func setupCoordinator(t *testing.T, proversPerSession uint8, coordinatorURL string, forks []string) (*cron.Collector, *http.Server) {
	var err error
	db, err = testApps.GetGormDBClient()

	assert.NoError(t, err)
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	tokenTimeout = 60
	conf = &config.Config{
		L2: &config.L2{
			ChainID: 111,
		},
		ProverManager: &config.ProverManager{
			ProversPerSession: proversPerSession,
			Verifier: &config.VerifierConfig{
				MockMode: true,
				LowVersionCircuit: &config.CircuitConfig{
					ParamsPath:       "",
					AssetsPath:       "",
					ForkName:         "homestead",
					MinProverVersion: "v4.2.0",
				},
				HighVersionCircuit: &config.CircuitConfig{
					ParamsPath:       "",
					AssetsPath:       "",
					ForkName:         "bernoulli",
					MinProverVersion: "v4.3.0",
				},
			},
			BatchCollectionTimeSec:  10,
			ChunkCollectionTimeSec:  10,
			BundleCollectionTimeSec: 10,
			SessionAttempts:         5,
		},
		Auth: &config.Auth{
			ChallengeExpireDurationSec: tokenTimeout,
			LoginExpireDurationSec:     tokenTimeout,
		},
	}

	var chainConf params.ChainConfig
	for _, forkName := range forks {
		switch forkName {
		case "bernoulli":
			chainConf.BernoulliBlock = big.NewInt(100)
		case "homestead":
			chainConf.HomesteadBlock = big.NewInt(0)
		}
	}

	proofCollector := cron.NewCollector(context.Background(), db, conf, nil)

	router := gin.New()
	api.InitController(conf, &chainConf, db, nil)
	route.Route(router, conf, nil)
	srv := &http.Server{
		Addr:    coordinatorURL,
		Handler: router,
	}
	go func() {
		runErr := srv.ListenAndServe()
		if runErr != nil && !errors.Is(runErr, http.ErrServerClosed) {
			assert.NoError(t, runErr)
		}
	}()
	time.Sleep(time.Second * 2)

	return proofCollector, srv
}

func setEnv(t *testing.T) {
	var err error

	version.Version = "v4.2.0"

	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.LogfmtFormat()))
	glogger.Verbosity(log.LvlInfo)
	log.Root().SetHandler(glogger)

	testApps = testcontainers.NewTestcontainerApps()
	assert.NoError(t, testApps.StartPostgresContainer())

	db, err = testApps.GetGormDBClient()
	assert.NoError(t, err)
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	batchOrm = orm.NewBatch(db)
	chunkOrm = orm.NewChunk(db)
	l2BlockOrm = orm.NewL2Block(db)
	proverTaskOrm = orm.NewProverTask(db)
	proverBlockListOrm = orm.NewProverBlockList(db)

	templateBlockTrace, err := os.ReadFile("../../common/testdata/blockTrace_02.json")
	assert.NoError(t, err)
	block1 = &encoding.Block{}
	err = json.Unmarshal(templateBlockTrace, block1)
	assert.NoError(t, err)

	templateBlockTrace, err = os.ReadFile("../../common/testdata/blockTrace_03.json")
	assert.NoError(t, err)
	block2 = &encoding.Block{}
	err = json.Unmarshal(templateBlockTrace, block2)
	assert.NoError(t, err)

	chunk = &encoding.Chunk{Blocks: []*encoding.Block{block1, block2}}
	assert.NoError(t, err)
	batch = &encoding.Batch{Chunks: []*encoding.Chunk{chunk}}

}

func TestApis(t *testing.T) {
	// Set up the test environment.
	setEnv(t)

	t.Run("TestHandshake", testHandshake)
	t.Run("TestFailedHandshake", testFailedHandshake)
	t.Run("TestGetTaskBlocked", testGetTaskBlocked)
	t.Run("TestOutdatedProverVersion", testOutdatedProverVersion)
	t.Run("TestValidProof", testValidProof)
	t.Run("TestInvalidProof", testInvalidProof)
	t.Run("TestProofGeneratedFailed", testProofGeneratedFailed)
	t.Run("TestTimeoutProof", testTimeoutProof)
}

func testHandshake(t *testing.T) {
	// Setup coordinator and http server.
	coordinatorURL := randomURL()
	proofCollector, httpHandler := setupCoordinator(t, 1, coordinatorURL, []string{"homestead"})
	defer func() {
		proofCollector.Stop()
		assert.NoError(t, httpHandler.Shutdown(context.Background()))
	}()

	chunkProver := newMockProver(t, "prover_chunk_test", coordinatorURL, message.ProofTypeChunk, version.Version)
	assert.True(t, chunkProver.healthCheckSuccess(t))
}

func testFailedHandshake(t *testing.T) {
	// Setup coordinator and http server.
	coordinatorURL := randomURL()
	proofCollector, httpHandler := setupCoordinator(t, 1, coordinatorURL, []string{"homestead"})
	defer func() {
		proofCollector.Stop()
	}()

	// Try to perform handshake without token
	chunkProver := newMockProver(t, "prover_chunk_test", coordinatorURL, message.ProofTypeChunk, version.Version)
	assert.True(t, chunkProver.healthCheckSuccess(t))

	// Try to perform handshake with server shutdown
	assert.NoError(t, httpHandler.Shutdown(context.Background()))
	time.Sleep(time.Second)
	batchProver := newMockProver(t, "prover_batch_test", coordinatorURL, message.ProofTypeBatch, version.Version)
	assert.True(t, batchProver.healthCheckFailure(t))
}

func testGetTaskBlocked(t *testing.T) {
	coordinatorURL := randomURL()
	collector, httpHandler := setupCoordinator(t, 3, coordinatorURL, []string{"homestead"})
	defer func() {
		collector.Stop()
		assert.NoError(t, httpHandler.Shutdown(context.Background()))
	}()

	chunkProver := newMockProver(t, "prover_chunk_test", coordinatorURL, message.ProofTypeChunk, version.Version)
	assert.True(t, chunkProver.healthCheckSuccess(t))

	batchProver := newMockProver(t, "prover_batch_test", coordinatorURL, message.ProofTypeBatch, version.Version)
	assert.True(t, batchProver.healthCheckSuccess(t))

	err := proverBlockListOrm.InsertProverPublicKey(context.Background(), chunkProver.proverName, chunkProver.publicKey())
	assert.NoError(t, err)

	expectedErr := fmt.Errorf("return prover task err:check prover task parameter failed, error:public key %s is blocked from fetching tasks. ProverName: %s, ProverVersion: %s", chunkProver.publicKey(), chunkProver.proverName, chunkProver.proverVersion)
	code, errMsg := chunkProver.tryGetProverTask(t, message.ProofTypeChunk)
	assert.Equal(t, types.ErrCoordinatorGetTaskFailure, code)
	assert.Equal(t, expectedErr, errors.New(errMsg))

	expectedErr = errors.New("get empty prover task")
	code, errMsg = batchProver.tryGetProverTask(t, message.ProofTypeBatch)
	assert.Equal(t, types.ErrCoordinatorEmptyProofData, code)
	assert.Equal(t, expectedErr, errors.New(errMsg))

	err = proverBlockListOrm.InsertProverPublicKey(context.Background(), batchProver.proverName, batchProver.publicKey())
	assert.NoError(t, err)

	err = proverBlockListOrm.DeleteProverPublicKey(context.Background(), chunkProver.publicKey())
	assert.NoError(t, err)

	expectedErr = errors.New("get empty prover task")
	code, errMsg = chunkProver.tryGetProverTask(t, message.ProofTypeChunk)
	assert.Equal(t, types.ErrCoordinatorEmptyProofData, code)
	assert.Equal(t, expectedErr, errors.New(errMsg))

	expectedErr = fmt.Errorf("return prover task err:check prover task parameter failed, error:public key %s is blocked from fetching tasks. ProverName: %s, ProverVersion: %s", batchProver.publicKey(), batchProver.proverName, batchProver.proverVersion)
	code, errMsg = batchProver.tryGetProverTask(t, message.ProofTypeBatch)
	assert.Equal(t, types.ErrCoordinatorGetTaskFailure, code)
	assert.Equal(t, expectedErr, errors.New(errMsg))
}

func testOutdatedProverVersion(t *testing.T) {
	coordinatorURL := randomURL()
	collector, httpHandler := setupCoordinator(t, 3, coordinatorURL, []string{"homestead"})
	defer func() {
		collector.Stop()
		assert.NoError(t, httpHandler.Shutdown(context.Background()))
	}()

	chunkProver := newMockProver(t, "prover_chunk_test", coordinatorURL, message.ProofTypeChunk, "v1.0.0")
	assert.True(t, chunkProver.healthCheckSuccess(t))

	batchProver := newMockProver(t, "prover_batch_test", coordinatorURL, message.ProofTypeBatch, "v1.999.999")
	assert.True(t, chunkProver.healthCheckSuccess(t))

	expectedErr := fmt.Errorf("check the login parameter failure: incompatible prover version. please upgrade your prover, minimum allowed version: %s, actual version: %s",
		conf.ProverManager.Verifier.LowVersionCircuit.MinProverVersion, chunkProver.proverVersion)
	code, errMsg := chunkProver.tryGetProverTask(t, message.ProofTypeChunk)
	assert.Equal(t, types.ErrJWTCommonErr, code)
	assert.Equal(t, expectedErr, errors.New(errMsg))

	expectedErr = fmt.Errorf("check the login parameter failure: incompatible prover version. please upgrade your prover, minimum allowed version: %s, actual version: %s",
		conf.ProverManager.Verifier.LowVersionCircuit.MinProverVersion, batchProver.proverVersion)
	code, errMsg = batchProver.tryGetProverTask(t, message.ProofTypeBatch)
	assert.Equal(t, types.ErrJWTCommonErr, code)
	assert.Equal(t, expectedErr, errors.New(errMsg))
}

func testValidProof(t *testing.T) {
	coordinatorURL := randomURL()
	collector, httpHandler := setupCoordinator(t, 3, coordinatorURL, []string{"homestead"})
	defer func() {
		collector.Stop()
		assert.NoError(t, httpHandler.Shutdown(context.Background()))
	}()

	err := l2BlockOrm.InsertL2Blocks(context.Background(), []*encoding.Block{block1, block2})
	assert.NoError(t, err)
	dbChunk, err := chunkOrm.InsertChunk(context.Background(), chunk)
	assert.NoError(t, err)
	err = l2BlockOrm.UpdateChunkHashInRange(context.Background(), 0, 100, dbChunk.Hash)
	assert.NoError(t, err)
	batch, err := batchOrm.InsertBatch(context.Background(), batch)
	assert.NoError(t, err)
	err = chunkOrm.UpdateBatchHashInRange(context.Background(), 0, 0, batch.Hash)
	assert.NoError(t, err)

	// create mock provers.
	provers := make([]*mockProver, 2)
	for i := 0; i < len(provers); i++ {
		var proofType message.ProofType
		if i%2 == 0 {
			proofType = message.ProofTypeChunk
		} else {
			proofType = message.ProofTypeBatch
		}

		provers[i] = newMockProver(t, "prover_test"+strconv.Itoa(i), coordinatorURL, proofType, version.Version)

		exceptProofStatus := verifiedSuccess
		proverTask, errCode, errMsg := provers[i].getProverTask(t, proofType)
		assert.Equal(t, types.Success, errCode)
		assert.Equal(t, "", errMsg)
		assert.NotNil(t, proverTask)
		provers[i].submitProof(t, proverTask, exceptProofStatus, types.Success)
	}

	// verify proof status
	var (
		tick     = time.Tick(1500 * time.Millisecond)
		tickStop = time.Tick(time.Minute)
	)

	var (
		chunkProofStatus    types.ProvingStatus
		batchProofStatus    types.ProvingStatus
		chunkActiveAttempts int16
		chunkMaxAttempts    int16
		batchActiveAttempts int16
		batchMaxAttempts    int16
	)

	for {
		select {
		case <-tick:
			chunkProofStatus, err = chunkOrm.GetProvingStatusByHash(context.Background(), dbChunk.Hash)
			assert.NoError(t, err)
			batchProofStatus, err = batchOrm.GetProvingStatusByHash(context.Background(), batch.Hash)
			assert.NoError(t, err)
			if chunkProofStatus == types.ProvingTaskVerified && batchProofStatus == types.ProvingTaskVerified {
				return
			}

			chunkActiveAttempts, chunkMaxAttempts, err = chunkOrm.GetAttemptsByHash(context.Background(), dbChunk.Hash)
			assert.NoError(t, err)
			assert.Equal(t, 1, int(chunkMaxAttempts))
			assert.Equal(t, 0, int(chunkActiveAttempts))

			batchActiveAttempts, batchMaxAttempts, err = batchOrm.GetAttemptsByHash(context.Background(), batch.Hash)
			assert.NoError(t, err)
			assert.Equal(t, 1, int(batchMaxAttempts))
			assert.Equal(t, 0, int(batchActiveAttempts))

		case <-tickStop:
			t.Error("failed to check proof status", "chunkProofStatus", chunkProofStatus.String(), "batchProofStatus", batchProofStatus.String())
			return
		}
	}
}

func testInvalidProof(t *testing.T) {
	// Setup coordinator and ws server.
	coordinatorURL := randomURL()
	collector, httpHandler := setupCoordinator(t, 3, coordinatorURL, []string{"darwinV2"})
	defer func() {
		collector.Stop()
		assert.NoError(t, httpHandler.Shutdown(context.Background()))
	}()

	err := l2BlockOrm.InsertL2Blocks(context.Background(), []*encoding.Block{block1, block2})
	assert.NoError(t, err)
	dbChunk, err := chunkOrm.InsertChunk(context.Background(), chunk)
	assert.NoError(t, err)
	err = l2BlockOrm.UpdateChunkHashInRange(context.Background(), 0, 100, dbChunk.Hash)
	assert.NoError(t, err)
	dbBatch, err := batchOrm.InsertBatch(context.Background(), batch)
	assert.NoError(t, err)
	err = chunkOrm.UpdateBatchHashInRange(context.Background(), 0, 100, dbBatch.Hash)
	assert.NoError(t, err)
	err = batchOrm.UpdateChunkProofsStatusByBatchHash(context.Background(), dbBatch.Hash, types.ChunkProofsStatusReady)
	assert.NoError(t, err)

	// create mock provers.
	provers := make([]*mockProver, 2)
	for i := 0; i < len(provers); i++ {
		var (
			proofType     message.ProofType
			provingStatus proofStatus
			exceptCode    int
		)

		if i%2 == 0 {
			proofType = message.ProofTypeChunk
			provingStatus = verifiedSuccess
			exceptCode = types.Success
		} else {
			proofType = message.ProofTypeBatch
			provingStatus = verifiedFailed
			exceptCode = types.ErrCoordinatorHandleZkProofFailure
		}

		provers[i] = newMockProver(t, "prover_test"+strconv.Itoa(i), coordinatorURL, proofType, version.Version)
		proverTask, errCode, errMsg := provers[i].getProverTask(t, proofType)
		assert.Equal(t, types.Success, errCode)
		assert.Equal(t, "", errMsg)
		assert.NotNil(t, proverTask)
		provers[i].submitProof(t, proverTask, provingStatus, exceptCode)
	}

	// verify proof status
	var (
		tick                = time.Tick(1500 * time.Millisecond)
		tickStop            = time.Tick(time.Minute)
		chunkProofStatus    types.ProvingStatus
		batchProofStatus    types.ProvingStatus
		batchActiveAttempts int16
		batchMaxAttempts    int16
		chunkActiveAttempts int16
		chunkMaxAttempts    int16
	)

	for {
		select {
		case <-tick:
			chunkProofStatus, err = chunkOrm.GetProvingStatusByHash(context.Background(), dbChunk.Hash)
			assert.NoError(t, err)
			batchProofStatus, err = batchOrm.GetProvingStatusByHash(context.Background(), dbBatch.Hash)
			assert.NoError(t, err)
			if chunkProofStatus == types.ProvingTaskVerified && batchProofStatus == types.ProvingTaskAssigned {
				return
			}

			chunkActiveAttempts, chunkMaxAttempts, err = chunkOrm.GetAttemptsByHash(context.Background(), dbChunk.Hash)
			assert.NoError(t, err)
			assert.Equal(t, 1, int(chunkMaxAttempts))
			assert.Equal(t, 0, int(chunkActiveAttempts))

			batchActiveAttempts, batchMaxAttempts, err = batchOrm.GetAttemptsByHash(context.Background(), dbBatch.Hash)
			assert.NoError(t, err)
			assert.Equal(t, 1, int(batchMaxAttempts))
			assert.Equal(t, 0, int(batchActiveAttempts))
		case <-tickStop:
			t.Error("failed to check proof status", "batchProofStatus", batchProofStatus.String())
			return
		}
	}
}

func testProofGeneratedFailed(t *testing.T) {
	// Setup coordinator and ws server.
	coordinatorURL := randomURL()
	collector, httpHandler := setupCoordinator(t, 3, coordinatorURL, []string{"darwinV2"})
	defer func() {
		collector.Stop()
		assert.NoError(t, httpHandler.Shutdown(context.Background()))
	}()

	err := l2BlockOrm.InsertL2Blocks(context.Background(), []*encoding.Block{block1, block2})
	assert.NoError(t, err)
	dbChunk, err := chunkOrm.InsertChunk(context.Background(), chunk)
	assert.NoError(t, err)
	err = l2BlockOrm.UpdateChunkHashInRange(context.Background(), 0, 100, dbChunk.Hash)
	assert.NoError(t, err)
	dbBatch, err := batchOrm.InsertBatch(context.Background(), batch)
	assert.NoError(t, err)
	err = chunkOrm.UpdateBatchHashInRange(context.Background(), 0, 100, dbBatch.Hash)
	assert.NoError(t, err)
	err = batchOrm.UpdateChunkProofsStatusByBatchHash(context.Background(), dbBatch.Hash, types.ChunkProofsStatusReady)
	assert.NoError(t, err)

	// create mock provers.
	provers := make([]*mockProver, 2)
	for i := 0; i < len(provers); i++ {
		var (
			proofType    message.ProofType
			exceptCode   int
			exceptErrMsg string
		)
		if i%2 == 0 {
			proofType = message.ProofTypeChunk
			exceptCode = types.Success
			exceptErrMsg = ""
		} else {
			proofType = message.ProofTypeBatch
			exceptCode = types.ErrCoordinatorGetTaskFailure
			exceptErrMsg = "return prover task err:coordinator internal error"
		}
		provers[i] = newMockProver(t, "prover_test"+strconv.Itoa(i), coordinatorURL, proofType, version.Version)
		proverTask, errCode, errMsg := provers[i].getProverTask(t, proofType)
		assert.NotNil(t, proverTask)
		assert.Equal(t, errCode, exceptCode)
		assert.Equal(t, errMsg, exceptErrMsg)
		if errCode == types.Success {
			provers[i].submitProof(t, proverTask, generatedFailed, types.ErrCoordinatorHandleZkProofFailure)
		}
	}

	// verify proof status
	var (
		tick     = time.Tick(1500 * time.Millisecond)
		tickStop = time.Tick(time.Minute)
	)

	var (
		chunkProofStatus             types.ProvingStatus
		batchProofStatus             types.ProvingStatus
		chunkProverTaskProvingStatus types.ProverProveStatus
		batchProverTaskProvingStatus types.ProverProveStatus
		chunkActiveAttempts          int16
		chunkMaxAttempts             int16
		batchActiveAttempts          int16
		batchMaxAttempts             int16
	)

	for {
		select {
		case <-tick:
			chunkProofStatus, err = chunkOrm.GetProvingStatusByHash(context.Background(), dbChunk.Hash)
			assert.NoError(t, err)
			batchProofStatus, err = batchOrm.GetProvingStatusByHash(context.Background(), dbBatch.Hash)
			assert.NoError(t, err)
			if chunkProofStatus == types.ProvingTaskAssigned && batchProofStatus == types.ProvingTaskAssigned {
				return
			}

			chunkActiveAttempts, chunkMaxAttempts, err = chunkOrm.GetAttemptsByHash(context.Background(), dbChunk.Hash)
			assert.NoError(t, err)
			assert.Equal(t, 1, int(chunkMaxAttempts))
			assert.Equal(t, 0, int(chunkActiveAttempts))

			batchActiveAttempts, batchMaxAttempts, err = batchOrm.GetAttemptsByHash(context.Background(), dbBatch.Hash)
			assert.NoError(t, err)
			assert.Equal(t, 1, int(batchMaxAttempts))
			assert.Equal(t, 0, int(batchActiveAttempts))

			chunkProverTaskProvingStatus, err = proverTaskOrm.GetProvingStatusByTaskID(context.Background(), message.ProofTypeChunk, dbChunk.Hash)
			assert.NoError(t, err)
			batchProverTaskProvingStatus, err = proverTaskOrm.GetProvingStatusByTaskID(context.Background(), message.ProofTypeBatch, dbBatch.Hash)
			assert.NoError(t, err)
			if chunkProverTaskProvingStatus == types.ProverProofInvalid && batchProverTaskProvingStatus == types.ProverProofInvalid {
				return
			}
		case <-tickStop:
			t.Error("failed to check proof status", "chunkProofStatus", chunkProofStatus.String(), "batchProofStatus", batchProofStatus.String())
			return
		}
	}
}

func testTimeoutProof(t *testing.T) {
	// Setup coordinator and ws server.
	coordinatorURL := randomURL()
	collector, httpHandler := setupCoordinator(t, 1, coordinatorURL, []string{"darwinV2"})
	defer func() {
		collector.Stop()
		assert.NoError(t, httpHandler.Shutdown(context.Background()))
	}()

	var (
		chunkActiveAttempts int16
		chunkMaxAttempts    int16
		batchActiveAttempts int16
		batchMaxAttempts    int16
	)

	err := l2BlockOrm.InsertL2Blocks(context.Background(), []*encoding.Block{block1, block2})
	assert.NoError(t, err)
	dbChunk, err := chunkOrm.InsertChunk(context.Background(), chunk)
	assert.NoError(t, err)
	err = l2BlockOrm.UpdateChunkHashInRange(context.Background(), 0, 100, dbChunk.Hash)
	assert.NoError(t, err)
	batch, err := batchOrm.InsertBatch(context.Background(), batch)
	assert.NoError(t, err)
	err = chunkOrm.UpdateBatchHashInRange(context.Background(), 0, 100, batch.Hash)
	assert.NoError(t, err)
	encodeData, err := json.Marshal(message.ChunkProof{})
	assert.NoError(t, err)
	assert.NotEmpty(t, encodeData)
	err = chunkOrm.UpdateProofAndProvingStatusByHash(context.Background(), dbChunk.Hash, encodeData, types.ProvingTaskUnassigned, 1)
	assert.NoError(t, err)
	err = batchOrm.UpdateChunkProofsStatusByBatchHash(context.Background(), batch.Hash, types.ChunkProofsStatusReady)
	assert.NoError(t, err)

	// create first chunk & batch mock prover, that will not send any proof.
	chunkProver1 := newMockProver(t, "prover_test"+strconv.Itoa(0), coordinatorURL, message.ProofTypeChunk, version.Version)
	proverChunkTask, errChunkCode, errChunkMsg := chunkProver1.getProverTask(t, message.ProofTypeChunk)
	assert.NotNil(t, proverChunkTask)
	assert.Equal(t, errChunkCode, types.Success)
	assert.Equal(t, errChunkMsg, "")

	batchProver1 := newMockProver(t, "prover_test"+strconv.Itoa(1), coordinatorURL, message.ProofTypeBatch, version.Version)
	proverBatchTask, errBatchCode, errBatchMsg := batchProver1.getProverTask(t, message.ProofTypeBatch)
	assert.NotNil(t, proverBatchTask)
	assert.Equal(t, errBatchCode, types.Success)
	assert.Equal(t, errBatchMsg, "")

	// verify proof status, it should be assigned, because prover didn't send any proof
	chunkProofStatus, err := chunkOrm.GetProvingStatusByHash(context.Background(), dbChunk.Hash)
	assert.NoError(t, err)
	assert.Equal(t, chunkProofStatus, types.ProvingTaskAssigned)

	batchProofStatus, err := batchOrm.GetProvingStatusByHash(context.Background(), batch.Hash)
	assert.NoError(t, err)
	assert.Equal(t, batchProofStatus, types.ProvingTaskAssigned)

	chunkActiveAttempts, chunkMaxAttempts, err = chunkOrm.GetAttemptsByHash(context.Background(), dbChunk.Hash)
	assert.NoError(t, err)
	assert.Equal(t, 1, int(chunkMaxAttempts))
	assert.Equal(t, 1, int(chunkActiveAttempts))

	batchActiveAttempts, batchMaxAttempts, err = batchOrm.GetAttemptsByHash(context.Background(), batch.Hash)
	assert.NoError(t, err)
	assert.Equal(t, 1, int(batchMaxAttempts))
	assert.Equal(t, 1, int(batchActiveAttempts))

	// wait coordinator to reset the prover task proving status
	time.Sleep(time.Duration(conf.ProverManager.BatchCollectionTimeSec*2) * time.Second)

	// create second mock prover, that will send valid proof.
	chunkProver2 := newMockProver(t, "prover_test"+strconv.Itoa(2), coordinatorURL, message.ProofTypeChunk, version.Version)
	proverChunkTask2, chunkTask2ErrCode, chunkTask2ErrMsg := chunkProver2.getProverTask(t, message.ProofTypeChunk)
	assert.NotNil(t, proverChunkTask2)
	assert.Equal(t, chunkTask2ErrCode, types.Success)
	assert.Equal(t, chunkTask2ErrMsg, "")
	chunkProver2.submitProof(t, proverChunkTask2, verifiedSuccess, types.Success)

	batchProver2 := newMockProver(t, "prover_test"+strconv.Itoa(3), coordinatorURL, message.ProofTypeBatch, version.Version)
	proverBatchTask2, batchTask2ErrCode, batchTask2ErrMsg := batchProver2.getProverTask(t, message.ProofTypeBatch)
	assert.NotNil(t, proverBatchTask2)
	assert.Equal(t, batchTask2ErrCode, types.Success)
	assert.Equal(t, batchTask2ErrMsg, "")
	batchProver2.submitProof(t, proverBatchTask2, verifiedSuccess, types.Success)

	// verify proof status, it should be verified now, because second prover sent valid proof
	chunkProofStatus2, err := chunkOrm.GetProvingStatusByHash(context.Background(), dbChunk.Hash)
	assert.NoError(t, err)
	assert.Equal(t, chunkProofStatus2, types.ProvingTaskVerified)

	batchProofStatus2, err := batchOrm.GetProvingStatusByHash(context.Background(), batch.Hash)
	assert.NoError(t, err)
	assert.Equal(t, batchProofStatus2, types.ProvingTaskVerified)

	chunkActiveAttempts, chunkMaxAttempts, err = chunkOrm.GetAttemptsByHash(context.Background(), dbChunk.Hash)
	assert.NoError(t, err)
	assert.Equal(t, 2, int(chunkMaxAttempts))
	assert.Equal(t, 0, int(chunkActiveAttempts))

	batchActiveAttempts, batchMaxAttempts, err = batchOrm.GetAttemptsByHash(context.Background(), batch.Hash)
	assert.NoError(t, err)
	assert.Equal(t, 2, int(batchMaxAttempts))
	assert.Equal(t, 0, int(batchActiveAttempts))
}
