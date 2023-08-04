package test

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"scroll-tech/database/migrate"

	"scroll-tech/common/database"
	"scroll-tech/common/docker"
	"scroll-tech/common/types"
	"scroll-tech/common/utils"

	"scroll-tech/prover-stats-api/internal/config"
	"scroll-tech/prover-stats-api/internal/controller"
	"scroll-tech/prover-stats-api/internal/orm"
	"scroll-tech/prover-stats-api/internal/route"
	apitypes "scroll-tech/prover-stats-api/internal/types"
)

var (
	proverPubkey = "11111"
)

var (
	addr      = randomURL()
	basicPath = fmt.Sprintf("http://%s/api/prover_task/v1", addr)
	token     string
)

func randomURL() string {
	id, _ := rand.Int(rand.Reader, big.NewInt(2000-1))
	return fmt.Sprintf("localhost:%d", 10000+2000+id.Int64())
}

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
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}
	go func() {
		if err = srv.ListenAndServe(); err != http.ErrServerClosed {
			assert.Failf(t, "addr: %s", srv.Addr)
		}
	}()
	defer srv.Shutdown(context.Background())

	t.Run("testRequestToken", testRequestToken)
	t.Run("testGetProverTasksByProver", testGetProverTasksByProver)
	t.Run("testGetTotalRewards", testGetTotalRewards)
	t.Run("testGetProverTask", testGetProverTask)
}

func testRequestToken(t *testing.T) {
	data := getResp(t, fmt.Sprintf("%s/request_token?public_key=%s", basicPath, proverPubkey))
	if utils.IsNil(data) {
		return
	}
	token = fmt.Sprintf("Bearer %s", data.(map[string]interface{})["token"].(string))
	t.Log("token: ", token)
}

func testGetProverTasksByProver(t *testing.T) {
	datas := getResp(t, fmt.Sprintf("%s/tasks?public_key=%s&page=%d&page_size=%d", basicPath, proverPubkey, 1, 10))
	if utils.IsNil(datas) {
		return
	}
	anys := datas.([]interface{})
	var tasks []map[string]interface{}
	for _, data := range anys {
		task := data.(map[string]interface{})
		tasks = append(tasks, task)
	}

	assert.Equal(t, task2.TaskID, tasks[0]["task_id"])
	assert.Equal(t, task1.TaskID, tasks[1]["task_id"])
}

func testGetTotalRewards(t *testing.T) {
	data := getResp(t, fmt.Sprintf("%s/total_rewards?public_key=%s", basicPath, proverPubkey))
	schema := data.(map[string]interface{})
	assert.Equal(t, big.NewInt(22).String(), schema["rewards"])
}

func testGetProverTask(t *testing.T) {
	data := getResp(t, fmt.Sprintf("%s/task?task_id=1", basicPath))
	task := data.(map[string]interface{})
	assert.Equal(t, task1.TaskID, task["task_id"])
}

func getResp(t *testing.T, url string) interface{} {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	assert.NoError(t, err)
	req.Header.Add("Authorization", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		assert.Fail(t, fmt.Sprintf("url: %s", url), err.Error())
		return nil
	}
	defer resp.Body.Close()
	assert.Equal(t, resp.StatusCode, http.StatusOK)

	byt, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)

	res := new(apitypes.Response)
	assert.NoError(t, json.Unmarshal(byt, res))
	t.Log("----byt is ", string(byt))
	assert.Equal(t, apitypes.Success, res.ErrCode)
	return res.Data
}

var (
	task1 = orm.ProverTask{
		TaskID:          "1",
		ProverPublicKey: proverPubkey,
		ProverName:      "prover-0",
		ProvingStatus:   int16(types.ProverAssigned),
		Reward:          decimal.NewFromInt(10),
	}

	task2 = orm.ProverTask{
		TaskID:          "2",
		ProverPublicKey: proverPubkey,
		ProverName:      "prover-1",
		ProvingStatus:   int16(types.ProverAssigned),
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
