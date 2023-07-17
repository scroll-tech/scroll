package test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"scroll-tech/common/database"
	"scroll-tech/common/docker"
	"scroll-tech/common/types"
	"scroll-tech/database/migrate"
	"testing"

	"scroll-tech/prover-stats-api/internal/controller"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"scroll-tech/prover-stats-api/internal/config"
	"scroll-tech/prover-stats-api/internal/orm"
)

var (
	proverPubkey = "11111"
)

var (
	port      = ":12990"
	addr      = fmt.Sprintf("http://localhost%s", port)
	basicPath = fmt.Sprintf("%s/api/v1/prover_task", addr)
	token     string
)

func TestProverTaskAPIs(t *testing.T) {
	// start database image
	base := docker.NewDockerApp()
	defer base.Free()
	cfg, err := config.NewConfig("../conf/config.json")
	assert.NoError(t, err)
	cfg.DBConfig.DSN = base.DBImg.Endpoint()
	base.RunDBImage(t)

	db, err := database.InitDB(cfg.DBConfig)
	assert.NoError(t, err)

	// insert some tasks
	insertSomeProverTasks(t, db)

	// run Prover Stats APIs
	controller.Route(db, port, cfg)

	t.Run("testRequestToken", testRequestToken)
	t.Run("testGetProverTasksByProver", testGetProverTasksByProver)
	t.Run("testGetTotalRewards", testGetTotalRewards)
	t.Run("testGetProverTask", testGetProverTask)
}

func testRequestToken(t *testing.T) {
	tokenByt := getResp(t, fmt.Sprintf("%s/request_token", basicPath))
	token = string(tokenByt)
	t.Log("token: ", token)
}

func testGetProverTasksByProver(t *testing.T) {
	var tasks []*orm.ProverTask
	byt := getResp(t, fmt.Sprintf("%s/tasks?pubkey=%s", basicPath, proverPubkey))
	assert.NoError(t, json.Unmarshal(byt, &tasks))
	assert.Equal(t, task2.TaskID, tasks[0].TaskID)
	assert.Equal(t, task1.TaskID, tasks[1].TaskID)
}

func testGetTotalRewards(t *testing.T) {
	rewards := getResp(t, fmt.Sprintf("%s/total_rewards?pubkey=%s", basicPath, proverPubkey))
	assert.Equal(t, big.NewInt(22), new(big.Int).SetBytes(rewards))
}

func testGetProverTask(t *testing.T) {
	var task orm.ProverTask
	byt := getResp(t, fmt.Sprintf("%s/task?task_id=1", basicPath))
	assert.NoError(t, json.Unmarshal(byt, &task))
	assert.Equal(t, task1.TaskID, task.TaskID)
}

func getResp(t *testing.T, url string) []byte {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", token)

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, resp.StatusCode, http.StatusOK)

	byt, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)

	res := new(controller.Resp)
	assert.NoError(t, json.Unmarshal(byt, res))
	assert.Equal(t, controller.OK, res.Code)
	return res.Object
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
