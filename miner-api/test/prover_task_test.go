package test

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"io"
	"math/big"
	"net/http"
	"scroll-tech/common/database"
	"scroll-tech/common/docker"
	"scroll-tech/miner-api/cmd/app"
	"scroll-tech/miner-api/internal/config"
	"scroll-tech/miner-api/internal/orm"
	"testing"
	"time"
)

var (
	port, addr, basicPath string

	proverPubkey = "11111"
)

func init() {
	portInt, _ := rand.Int(rand.Reader, big.NewInt(2000))

	port = fmt.Sprintf(":%s", portInt.String())
	addr = fmt.Sprintf("http://localhost%s", port)
	basicPath = fmt.Sprintf("%s/api/v1/prover_task", addr)
}

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
	assert.Equal(t, "2", tasks[0].TaskID)
	assert.Equal(t, "1", tasks[1].TaskID)
}

func testGetTotalRewards(t *testing.T) {
	rewards := make(map[string]uint64)
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
		ID:              1,
		TaskID:          "1",
		ProverPublicKey: proverPubkey,
		ProverName:      "prover",
		TaskType:        0,
		ProvingStatus:   0,
		FailureType:     0,
		Reward:          10,
		Proof:           nil,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	task2 = orm.ProverTask{
		ID:              2,
		TaskID:          "2",
		ProverPublicKey: proverPubkey,
		ProverName:      "prover",
		TaskType:        0,
		ProvingStatus:   0,
		FailureType:     0,
		Reward:          12,
		Proof:           nil,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
)

func insertSomeProverTasks(t *testing.T, db *gorm.DB) {
	err := db.AutoMigrate(new(orm.ProverTask))
	assert.NoError(t, err)
	ptdb := orm.NewProverTask(db)
	err = ptdb.SetProverTask(context.Background(), &task1)
	assert.NoError(t, err)
	err = ptdb.SetProverTask(context.Background(), &task2)
	assert.NoError(t, err)
}
