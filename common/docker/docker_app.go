package docker

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/modern-go/reflect2"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/cmd"
	"scroll-tech/common/utils"
	"scroll-tech/database"
)

var (
	l1StartPort = 10000
	l2StartPort = 20000
	dbStartPort = 30000
)

type AppAPI interface {
	WaitResult(t *testing.T, timeout time.Duration, keyword string) bool
	RunApp(waitResult func() bool)
	WaitExit()
	ExpectWithTimeout(t *testing.T, parallel bool, timeout time.Duration, keyword string)
}

// DockerApp is collection struct of runtime docker images
type DockerApp struct {
	L1gethImg ImgInstance
	L2gethImg ImgInstance
	DBImg     ImgInstance

	DBConfig *database.DBConfig
	dbFile   string

	// common time stamp.
	Timestamp int
}

// NewDockerApp returns new instance of dokerApp struct
func NewDockerApp() *DockerApp {
	timestamp := time.Now().Nanosecond()
	app := &DockerApp{
		Timestamp: timestamp,
		L1gethImg: newTestL1Docker(),
		L2gethImg: newTestL2Docker(),
		DBImg:     newTestDBDocker("postgres"),
		dbFile:    fmt.Sprintf("/tmp/%d_db-config.json", timestamp),
	}
	if err := app.mockDBConfig(); err != nil {
		panic(err)
	}
	return app
}

// RunImages runs all images togather
func (b *DockerApp) RunImages(t *testing.T) {
	b.RunDBImage(t)
	b.RunL1Geth(t)
	b.RunL2Geth(t)
}

func (b *DockerApp) RunDBImage(t *testing.T) {
	if b.DBImg.IsRunning() {
		return
	}
	assert.NoError(t, b.DBImg.Start())
	var isRun bool
	// try 5 times until the db is ready.
	utils.TryTimes(5, func() bool {
		db, _ := sqlx.Open("postgres", b.DBImg.Endpoint())
		isRun = db != nil && db.Ping() == nil
		return isRun
	})
	assert.Equal(t, true, isRun)
}

// Free clear all running images
func (b *DockerApp) Free() {
	if b.L1gethImg.IsRunning() {
		_ = b.L1gethImg.Stop()
	}
	if b.L2gethImg.IsRunning() {
		_ = b.L2gethImg.Stop()
	}
	if b.DBImg.IsRunning() {
		_ = b.DBImg.Stop()
		_ = os.Remove(b.dbFile)
	}
}

func (b *DockerApp) RunL1Geth(t *testing.T) {
	if b.L1gethImg.IsRunning() {
		return
	}
	assert.NoError(t, b.L1gethImg.Start())

	var isRun bool
	// try 3 times to get chainID until is ok.
	utils.TryTimes(3, func() bool {
		client, _ := ethclient.Dial(b.L1gethImg.Endpoint())
		if client != nil {
			if _, err := client.ChainID(context.Background()); err == nil {
				isRun = true
			}
		}
		return isRun
	})
	assert.Equal(t, true, isRun)
}

// L1Client returns a ethclient by dialing running l1geth
func (b *DockerApp) L1Client() (*ethclient.Client, error) {
	if b.L1gethImg == nil || reflect2.IsNil(b.L1gethImg) {
		return nil, fmt.Errorf("l1 geth is not running")
	}
	client, err := ethclient.Dial(b.L1gethImg.Endpoint())
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (b *DockerApp) RunL2Geth(t *testing.T) {
	if b.L2gethImg.IsRunning() {
		return
	}
	assert.NoError(t, b.L2gethImg.Start())

	var isRun bool
	// try 3 times to get chainID until is ok.
	utils.TryTimes(3, func() bool {
		client, _ := ethclient.Dial(b.L2gethImg.Endpoint())
		if client != nil {
			if _, err := client.ChainID(context.Background()); err == nil {
				isRun = true
			}
		}
		return isRun
	})
	assert.Equal(t, true, isRun)
}

// L2Client returns a ethclient by dialing running l2geth
func (b *DockerApp) L2Client() (*ethclient.Client, error) {
	if b.L2gethImg == nil || reflect2.IsNil(b.L2gethImg) {
		return nil, fmt.Errorf("l2 geth is not running")
	}
	client, err := ethclient.Dial(b.L2gethImg.Endpoint())
	if err != nil {
		return nil, err
	}
	return client, nil
}

// RunDBApp runs DB app with command
func (b *DockerApp) RunDBApp(t *testing.T, option, keyword string) {
	args := []string{option, "--config", b.dbFile}
	app := cmd.NewCmd("db_cli-test", args...)
	defer app.WaitExit()

	okCh := make(chan struct{}, 1)
	app.RegistFunc(keyword, func(buf string) {
		if strings.Contains(buf, keyword) {
			select {
			case okCh <- struct{}{}:
			default:
				return
			}
		}
	})
	defer app.UnRegistFunc(keyword)

	// Start process.
	app.RunApp(nil)

	select {
	case <-okCh:
		return
	case err := <-app.ErrChan:
		assert.Fail(t, err.Error())
	case <-time.After(time.Second * 3):
		assert.Fail(t, fmt.Sprintf("didn't get the desired result before timeout, keyword: %s", keyword))
	}
}

func (b *DockerApp) InitDB(t *testing.T) {
	// Init database.
	b.RunDBApp(t, "reset", "successful to reset")
	b.RunDBApp(t, "migrate", "current version:")
}

func (b *DockerApp) mockDBConfig() error {
	if b.DBConfig == nil {
		b.DBConfig = &database.DBConfig{
			DSN:        "",
			DriverName: "postgres",
			MaxOpenNum: 200,
			MaxIdleNum: 20,
		}
	}

	if b.DBImg != nil {
		b.DBConfig.DSN = b.DBImg.Endpoint()
	}
	data, err := json.Marshal(b.DBConfig)
	if err != nil {
		return err
	}

	return os.WriteFile(b.dbFile, data, 0644) //nolint:gosec
}

func newTestL1Docker() ImgInstance {
	id, _ := rand.Int(rand.Reader, big.NewInt(2000))
	return NewImgGeth("scroll_l1geth", "", "", 0, l1StartPort+int(id.Int64()))
}

func newTestL2Docker() ImgInstance {
	id, _ := rand.Int(rand.Reader, big.NewInt(2000))
	return NewImgGeth("scroll_l2geth", "", "", 0, l2StartPort+int(id.Int64()))
}

func newTestDBDocker(driverName string) ImgInstance {
	id, _ := rand.Int(rand.Reader, big.NewInt(2000))
	return NewImgDB(driverName, "123456", "test_db", dbStartPort+int(id.Int64()))
}
