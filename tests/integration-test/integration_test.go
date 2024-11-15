package integration_test

import (
	"context"
	"log"
	"math/big"
	"testing"

	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/common"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"scroll-tech/common/testcontainers"
	"scroll-tech/common/utils"
	"scroll-tech/common/version"
	"scroll-tech/database/migrate"
	"scroll-tech/integration-test/orm"

	capp "scroll-tech/coordinator/cmd/api/app"

	bcmd "scroll-tech/rollup/cmd"
)

var (
	testApps  *testcontainers.TestcontainerApps
	rollupApp *bcmd.MockApp
)

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
	testApps = testcontainers.NewTestcontainerApps()
	assert.NoError(t, testApps.StartPostgresContainer())
	assert.NoError(t, testApps.StartPoSL1Container())
	assert.NoError(t, testApps.StartL2GethContainer())
	rollupApp = bcmd.NewRollupApp(testApps, "../../rollup/conf/config.json")
}

func TestFunction(t *testing.T) {
	setupEnv(t)
	t.Run("TestCoordinatorProverInteraction", testCoordinatorProverInteraction)
	t.Run("TestProverReLogin", testProverReLogin)
	t.Run("TestERC20", testERC20)
	t.Run("TestGreeter", testGreeter)
}

func setupDB(t *testing.T) *gorm.DB {
	db, err := testApps.GetGormDBClient()
	assert.NoError(t, err)
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))
	return db
}

func testCoordinatorProverInteraction(t *testing.T) {
	db := setupDB(t)

	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	batchOrm := orm.NewBatch(db)
	chunkOrm := orm.NewChunk(db)
	l2BlockOrm := orm.NewL2Block(db)

	// Connect to l2geth client
	l2Client, err := testApps.GetL2GethClient()
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
	batch := &encoding.Batch{
		Index:                      0,
		TotalL1MessagePoppedBefore: 0,
		ParentBatchHash:            common.Hash{},
		Chunks:                     []*encoding.Chunk{chunk},
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

	coordinatorApp := capp.NewCoordinatorApp(testApps, "../../coordinator/conf/config.json", "./genesis.json")
	defer coordinatorApp.Free()

	// Run coordinator app.
	coordinatorApp.RunApp(t)
	coordinatorApp.WaitExit()
}

func testProverReLogin(t *testing.T) {
	client, err := testApps.GetGormDBClient()
	assert.NoError(t, err)
	db, err := client.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db))

	coordinatorApp := capp.NewCoordinatorApp(testApps, "../../coordinator/conf/config.json", "./genesis.json")
	defer coordinatorApp.Free()

	// Run coordinator app.
	coordinatorApp.RunApp(t) // login timeout: 1 sec

	coordinatorApp.WaitExit()
}
