package test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"io"
	"net/http"
	"scroll-tech/common/database"
	"scroll-tech/common/docker"
	"scroll-tech/common/types"
	"scroll-tech/database/migrate"
	"scroll-tech/miner-api/cmd/app"
	"scroll-tech/miner-api/internal/config"
	"scroll-tech/miner-api/internal/orm"
	"testing"
)

var (
	proverPubkey = "11111"
)

var (
	port      = ":8990"
	addr      = fmt.Sprintf("http://localhost%s", port)
	basicPath = fmt.Sprintf("%s/api/v1/prover_task", addr)
)

func TestProverTaskAPIs(t *testing.T) {
	// start database image
	base := docker.NewDockerApp()
	defer base.Free()
	cfg, err := config.NewConfig("../config.json")
	assert.NoError(t, err)
	cfg.DBConfig.DSN = base.DBImg.Endpoint()
	base.RunDBImage(t)

	db, err := database.InitDB(cfg.DBConfig)
	assert.NoError(t, err)

	// insert some tasks
	insertSomeProverTasks(t, db)

	// run miner APIs
	app.RunMinerAPIs(db, port)

	t.Run("testGetProverTasksByProver", testGetProverTasksByProver)
	t.Run("testGetTotalRewards", testGetTotalRewards)
	t.Run("testGetProverTask", testGetProverTask)
}

func testGetProverTasksByProver(t *testing.T) {
	var tasks []*orm.ProverTask
	getResp(t, fmt.Sprintf("%s/tasks?pubkey=%s", basicPath, proverPubkey), &tasks)
	assert.Equal(t, task2, *tasks[0])
	assert.Equal(t, task1, *tasks[1])
}

func testGetTotalRewards(t *testing.T) {
	rewards := make(map[string]int)
	getResp(t, fmt.Sprintf("%s/total_rewards?pubkey=%s", basicPath, proverPubkey), &rewards)
	assert.Equal(t, 22, rewards["rewards"])
}

func testGetProverTask(t *testing.T) {
	var task orm.ProverTask
	getResp(t, fmt.Sprintf("%s/task?task_id=1", basicPath), &task)
	assert.Equal(t, task1, task)
}

func getResp(t *testing.T, url string, value interface{}) {
	resp, err := http.Get(url)
	assert.NoError(t, err)
	assert.Equal(t, resp.StatusCode, http.StatusOK)
	byt, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	err = json.Unmarshal(byt, value)
	assert.NoError(t, err)
}

var (
	task1 = orm.ProverTask{
		TaskID:          "1",
		ProverPublicKey: proverPubkey,
		ProverName:      "prover-0",
		ProvingStatus:   int16(types.RollerAssigned),
		Reward:          decimal.NewFromInt(10),
	}

	task2 = orm.ProverTask{
		TaskID:          "2",
		ProverPublicKey: proverPubkey,
		ProverName:      "prover-1",
		ProvingStatus:   int16(types.RollerAssigned),
		Reward:          decimal.NewFromInt(12),
	}
)

func insertSomeProverTasks(t *testing.T, db *gorm.DB) {
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	ptdb := orm.NewProverTask(db)
	err = ptdb.SetProverTask(context.Background(), &task1)
	assert.NoError(t, err)

	err = ptdb.SetProverTask(context.Background(), &task2)
	assert.NoError(t, err)
}
