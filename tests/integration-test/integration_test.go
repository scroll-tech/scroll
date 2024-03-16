package integration_test

import (
	"context"
	"log"
	"math/big"
	"testing"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

	"scroll-tech/integration-test/orm"

	rapp "scroll-tech/prover/cmd/app"

	"scroll-tech/database/migrate"

	capp "scroll-tech/coordinator/cmd/api/app"

	"scroll-tech/common/database"
	"scroll-tech/common/docker"
	"scroll-tech/common/types/encoding"
	"scroll-tech/common/types/encoding/codecv0"
	"scroll-tech/common/utils"
	"scroll-tech/common/version"

	bcmd "scroll-tech/rollup/cmd"
)

var (
	base      *docker.App
	rollupApp *bcmd.MockApp
)

func TestMain(m *testing.M) {
	base = docker.NewDockerApp()
	rollupApp = bcmd.NewRollupApp(base, "../../rollup/conf/config.json")
	m.Run()
	rollupApp.Free()
	base.Free()
}

func TestCoordinatorProverInteraction(t *testing.T) {
	// Start postgres docker containers
	base.RunL2Geth(t)
	base.RunDBImage(t)

	// Init data
	dbCfg := &database.Config{
		DSN:         base.DBConfig.DSN,
		DriverName:  base.DBConfig.DriverName,
		MaxOpenNum:  base.DBConfig.MaxOpenNum,
		MaxIdleNum:  base.DBConfig.MaxIdleNum,
		MaxLifetime: base.DBConfig.MaxLifetime,
		MaxIdleTime: base.DBConfig.MaxIdleTime,
	}

	db, err := database.InitDB(dbCfg)
	assert.NoError(t, err)

	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	batchOrm := orm.NewBatch(db)
	chunkOrm := orm.NewChunk(db)
	l2BlockOrm := orm.NewL2Block(db)

	// Connect to l2geth client
	l2Client, err := base.L2Client()
	if err != nil {
		log.Fatalf("Failed to connect to the l2geth client: %v", err)
	}

	var header *gethTypes.Header
	success := utils.TryTimes(10, func() bool {
		header, err = l2Client.HeaderByNumber(context.Background(), big.NewInt(1))
		if err != nil {
			log.Printf("Failed to retrieve L2 genesis header: %v. Retrying...", err)
			return false
		}
		return true
	})

	if !success {
		log.Fatalf("Failed to retrieve L2 genesis header after multiple attempts: %v", err)
	}

	block := &encoding.Block{
		Header:         header,
		Transactions:   nil,
		WithdrawRoot:   common.Hash{},
		RowConsumption: &gethTypes.RowConsumption{},
	}
	chunk := &encoding.Chunk{Blocks: []*encoding.Block{block}}

	daChunk, err := codecv0.NewDAChunk(chunk, 0)
	assert.NoError(t, err)

	daChunkHash, err := daChunk.Hash()
	assert.NoError(t, err)

	batch := &encoding.Batch{
		Index:                      0,
		TotalL1MessagePoppedBefore: 0,
		ParentBatchHash:            common.Hash{},
		Chunks:                     []*encoding.Chunk{chunk},
		StartChunkIndex:            0,
		EndChunkIndex:              0,
		StartChunkHash:             daChunkHash,
		EndChunkHash:               daChunkHash,
	}

	err = l2BlockOrm.InsertL2Blocks(context.Background(), []*encoding.Block{block})
	assert.NoError(t, err)
	dbChunk, err := chunkOrm.InsertChunk(context.Background(), chunk)
	assert.NoError(t, err)
	err = l2BlockOrm.UpdateChunkHashInRange(context.Background(), 0, 100, dbChunk.Hash)
	assert.NoError(t, err)
	dbBatch, err := batchOrm.InsertBatch(context.Background(), batch)
	assert.NoError(t, err)
	err = chunkOrm.UpdateBatchHashInRange(context.Background(), 0, 0, dbBatch.Hash)
	assert.NoError(t, err)
	t.Log(version.Version)

	base.Timestamp = time.Now().Nanosecond()
	coordinatorApp := capp.NewCoordinatorApp(base, "../../coordinator/conf/config.json")
	chunkProverApp := rapp.NewProverApp(base, utils.ChunkProverApp, "../../prover/config.json", coordinatorApp.HTTPEndpoint())
	batchProverApp := rapp.NewProverApp(base, utils.BatchProverApp, "../../prover/config.json", coordinatorApp.HTTPEndpoint())
	defer coordinatorApp.Free()
	defer chunkProverApp.Free()
	defer batchProverApp.Free()

	// Run coordinator app.
	coordinatorApp.RunApp(t)

	// Run prover app.
	chunkProverApp.ExpectWithTimeout(t, true, time.Second*40, "proof submitted successfully") // chunk prover login -> get task -> submit proof.
	batchProverApp.ExpectWithTimeout(t, true, time.Second*40, "proof submitted successfully") // batch prover login -> get task -> submit proof.

	// All task has been proven, coordinator would not return any task.
	chunkProverApp.ExpectWithTimeout(t, true, 60*time.Second, "get empty prover task")
	batchProverApp.ExpectWithTimeout(t, true, 60*time.Second, "get empty prover task")

	chunkProverApp.RunApp(t)
	batchProverApp.RunApp(t)

	// Free apps.
	chunkProverApp.WaitExit()
	batchProverApp.WaitExit()
	coordinatorApp.WaitExit()
}

func TestProverReLogin(t *testing.T) {
	// Start postgres docker containers.
	base.RunL2Geth(t)
	base.RunDBImage(t)

	assert.NoError(t, migrate.ResetDB(base.DBClient(t)))

	base.Timestamp = time.Now().Nanosecond()
	coordinatorApp := capp.NewCoordinatorApp(base, "../../coordinator/conf/config.json")
	chunkProverApp := rapp.NewProverApp(base, utils.ChunkProverApp, "../../prover/config.json", coordinatorApp.HTTPEndpoint())
	batchProverApp := rapp.NewProverApp(base, utils.BatchProverApp, "../../prover/config.json", coordinatorApp.HTTPEndpoint())
	defer coordinatorApp.Free()
	defer chunkProverApp.Free()
	defer batchProverApp.Free()

	// Run coordinator app.
	coordinatorApp.RunApp(t) // login timeout: 1 sec

	// Run prover app.
	chunkProverApp.ExpectWithTimeout(t, true, time.Second*40, "re-login success") // chunk prover login.
	batchProverApp.ExpectWithTimeout(t, true, time.Second*40, "re-login success") // batch prover login.

	chunkProverApp.RunApp(t)
	batchProverApp.RunApp(t)

	// Free apps.
	chunkProverApp.WaitExit()
	batchProverApp.WaitExit()
	coordinatorApp.WaitExit()
}
