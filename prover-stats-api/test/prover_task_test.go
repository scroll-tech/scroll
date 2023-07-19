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

	"github.com/gin-gonic/gin"

	"testing"

	"scroll-tech/prover-stats-api/internal/config"
	"scroll-tech/prover-stats-api/internal/controller"
	"scroll-tech/prover-stats-api/internal/orm"
	"scroll-tech/prover-stats-api/internal/route"
	api_types "scroll-tech/prover-stats-api/internal/types"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

var (
	proverPubkey = "11111"
)

var (
	port      = ":12990"
	addr      = fmt.Sprintf("http://localhost%s", port)
	basicPath = fmt.Sprintf("%s/api/prover_task/v1", addr)
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
	router := gin.Default()
	controller.InitController(db)
	route.Route(router, cfg)
	go func() {
		router.Run(port)
	}()

	t.Run("testRequestToken", testRequestToken)
	t.Run("testGetProverTasksByProver", testGetProverTasksByProver)
	t.Run("testGetTotalRewards", testGetTotalRewards)
	t.Run("testGetProverTask", testGetProverTask)
}

func testRequestToken(t *testing.T) {
	data := getResp(t, fmt.Sprintf("%s/request_token?public_key=%s", basicPath, proverPubkey))
	token = fmt.Sprintf("Bearer %s", data.(map[string]interface{})["token"].(string))
	t.Log("token: ", token)
}

func testGetProverTasksByProver(t *testing.T) {
	data := getResp(t, fmt.Sprintf("%s/tasks?public_key=%s&page=%d&page_size=%d", basicPath, proverPubkey, 0, 10))
	tasks := data.([]api_types.ProverTaskSchema)
	assert.Equal(t, task2.TaskID, tasks[0].TaskID)
	assert.Equal(t, task1.TaskID, tasks[1].TaskID)
}

func testGetTotalRewards(t *testing.T) {
	data := getResp(t, fmt.Sprintf("%s/total_rewards?public_key=%s", basicPath, proverPubkey))
	schema := data.(api_types.ProverTotalRewardsSchema)
	assert.Equal(t, big.NewInt(22).String(), schema.Rewards)
}

func testGetProverTask(t *testing.T) {
	data := getResp(t, fmt.Sprintf("%s/task?task_id=1", basicPath))
	task := data.(api_types.ProverTaskSchema)
	assert.Equal(t, task1.TaskID, task.TaskID)
}

func getResp(t *testing.T, url string) interface{} {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", token)

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, resp.StatusCode, http.StatusOK)

	byt, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)

	res := new(api_types.Response)
	assert.NoError(t, json.Unmarshal(byt, res))
	t.Log("----byt is ", string(byt))
	assert.Equal(t, api_types.Success, res.ErrCode)
	return res.Data
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
