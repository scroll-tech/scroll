package docker

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"

	"scroll-tech/database"

	"scroll-tech/common/utils"
)

var (
	l1StartPort = 10000
	l2StartPort = 20000
	dbStartPort = 30000
)

// AppAPI app interface.
type AppAPI interface {
	IsRunning() bool
	WaitResult(t *testing.T, timeout time.Duration, keyword string) bool
	RunApp(waitResult func() bool)
	WaitExit()
	ExpectWithTimeout(t *testing.T, parallel bool, timeout time.Duration, keyword string)
}

// App is collection struct of runtime docker images
type App struct {
	L1gethImg GethImgInstance
	L2gethImg GethImgInstance
	DBImg     ImgInstance

	dbClient     *sql.DB
	DBConfig     *database.DBConfig
	DBConfigFile string

	// common time stamp.
	Timestamp int
}

// NewDockerApp returns new instance of dockerApp struct
func NewDockerApp() *App {
	timestamp := time.Now().Nanosecond()
	app := &App{
		Timestamp:    timestamp,
		L1gethImg:    newTestL1Docker(),
		L2gethImg:    newTestL2Docker(),
		DBImg:        newTestDBDocker("postgres"),
		DBConfigFile: fmt.Sprintf("/tmp/%d_db-config.json", timestamp),
	}
	if err := app.mockDBConfig(); err != nil {
		panic(err)
	}
	return app
}

// RunImages runs all images together
func (b *App) RunImages(t *testing.T) {
	b.RunDBImage(t)
	b.RunL1Geth(t)
	b.RunL2Geth(t)
}

// RunDBImage starts postgres docker container.
func (b *App) RunDBImage(t *testing.T) {
	if b.DBImg.IsRunning() {
		return
	}
	assert.NoError(t, b.DBImg.Start())

	// try 5 times until the db is ready.
	ok := utils.TryTimes(10, func() bool {
		db, err := sqlx.Open("postgres", b.DBImg.Endpoint())
		return err == nil && db != nil && db.Ping() == nil
	})
	assert.True(t, ok)
}

// Free clear all running images, double check and recycle docker container.
func (b *App) Free() {
	if b.L1gethImg.IsRunning() {
		_ = b.L1gethImg.Stop()
	}
	if b.L2gethImg.IsRunning() {
		_ = b.L2gethImg.Stop()
	}
	if b.DBImg.IsRunning() {
		_ = b.DBImg.Stop()
		_ = os.Remove(b.DBConfigFile)
		if !utils.IsNil(b.dbClient) {
			_ = b.dbClient.Close()
			b.dbClient = nil
		}
	}
}

// RunL1Geth starts l1geth docker container.
func (b *App) RunL1Geth(t *testing.T) {
	if b.L1gethImg.IsRunning() {
		return
	}
	assert.NoError(t, b.L1gethImg.Start())
}

// L1Client returns a ethclient by dialing running l1geth
func (b *App) L1Client() (*ethclient.Client, error) {
	if utils.IsNil(b.L1gethImg) {
		return nil, fmt.Errorf("l1 geth is not running")
	}
	client, err := ethclient.Dial(b.L1gethImg.Endpoint())
	if err != nil {
		return nil, err
	}
	return client, nil
}

// RunL2Geth starts l2geth docker container.
func (b *App) RunL2Geth(t *testing.T) {
	if b.L2gethImg.IsRunning() {
		return
	}
	assert.NoError(t, b.L2gethImg.Start())
}

// L2Client returns a ethclient by dialing running l2geth
func (b *App) L2Client() (*ethclient.Client, error) {
	if utils.IsNil(b.L2gethImg) {
		return nil, fmt.Errorf("l2 geth is not running")
	}
	client, err := ethclient.Dial(b.L2gethImg.Endpoint())
	if err != nil {
		return nil, err
	}
	return client, nil
}

// DBClient create and return *sql.DB instance.
func (b *App) DBClient(t *testing.T) *sql.DB {
	if !utils.IsNil(b.dbClient) {
		return b.dbClient
	}
	var (
		cfg = b.DBConfig
		err error
	)
	b.dbClient, err = sql.Open(cfg.DriverName, cfg.DSN)
	assert.NoError(t, err)
	b.dbClient.SetMaxOpenConns(cfg.MaxOpenNum)
	b.dbClient.SetMaxIdleConns(cfg.MaxIdleNum)
	assert.NoError(t, b.dbClient.Ping())
	return b.dbClient
}

func (b *App) mockDBConfig() error {
	b.DBConfig = &database.DBConfig{
		DSN:        "",
		DriverName: "postgres",
		MaxOpenNum: 200,
		MaxIdleNum: 20,
	}

	if b.DBImg != nil {
		b.DBConfig.DSN = b.DBImg.Endpoint()
	}
	data, err := json.Marshal(b.DBConfig)
	if err != nil {
		return err
	}

	return os.WriteFile(b.DBConfigFile, data, 0644) //nolint:gosec
}

func newTestL1Docker() GethImgInstance {
	id, _ := rand.Int(rand.Reader, big.NewInt(2000))
	return NewImgGeth("scroll_l1geth", "", "", 0, l1StartPort+int(id.Int64()))
}

func newTestL2Docker() GethImgInstance {
	id, _ := rand.Int(rand.Reader, big.NewInt(2000))
	return NewImgGeth("scroll_l2geth", "", "", 0, l2StartPort+int(id.Int64()))
}

func newTestDBDocker(driverName string) ImgInstance {
	id, _ := rand.Int(rand.Reader, big.NewInt(2000))
	return NewImgDB(driverName, "123456", "test_db", dbStartPort+int(id.Int64()))
}
