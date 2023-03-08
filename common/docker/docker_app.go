package docker

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/modern-go/reflect2"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"

	"scroll-tech/database"

	"scroll-tech/common/cmd"
	"scroll-tech/common/utils"
)

var (
	l1StartPort = 10000
	l2StartPort = 20000
	dbStartPort = 30000
)

// App is collection struct of runtime docker images
type App struct {
	l1gethImg ImgInstance
	l2gethImg ImgInstance

	dbImg       ImgInstance
	dbConfig    *database.DBConfig
	dbOriginCfg string
	dbFile      string

	// common time stamp.
	timestamp int
}

// NewDockerApp returns new instance of dokerApp struct
func NewDockerApp(cfg string) *App {
	timestamp := time.Now().Nanosecond()
	return &App{
		timestamp:   timestamp,
		dbFile:      fmt.Sprintf("/tmp/%d_db-config.json", timestamp),
		dbOriginCfg: cfg,
	}
}

// RunImages runs all images togather
func (b *App) RunImages(t *testing.T) {
	b.runDBImage(t)
	b.runL1Geth(t)
	b.runL2Geth(t)
}

func (b *App) runDBImage(t *testing.T) {
	if b.dbImg != nil {
		return
	}
	b.dbImg = newTestDBDocker(t, "postgres")
	if err := b.mockDBConfig(); err != nil {
		_ = b.dbImg.Stop()
		b.dbImg = nil
		_ = os.Remove(b.dbFile)
		t.Fatal(err)
	}
}

// RunDBApp runs DB app with command
func (b *App) RunDBApp(t *testing.T, option, keyword string) {
	args := []string{option, "--config", b.dbFile}
	app := cmd.NewCmd(t, "db_cli-test", args...)
	defer app.WaitExit()

	// Wait expect result.
	app.ExpectWithTimeout(true, time.Second*3, keyword)
	app.RunApp(nil)
}

// Free clear all running images
func (b *App) Free() {
	if b.l1gethImg != nil {
		_ = b.l1gethImg.Stop()
		b.l1gethImg = nil
	}
	if b.l2gethImg != nil {
		_ = b.l2gethImg.Stop()
		b.l2gethImg = nil
	}
	if b.dbImg != nil {
		_ = b.dbImg.Stop()
		b.dbImg = nil
		_ = os.Remove(b.dbFile)
	}
}

// L1GethEndpoint returns l1gethimg endpoint
func (b *App) L1GethEndpoint() string {
	if b.l1gethImg != nil {
		return b.l1gethImg.Endpoint()
	}
	return ""
}

// L2GethEndpoint returns l2gethimg endpoint
func (b *App) L2GethEndpoint() string {
	if b.l2gethImg != nil {
		return b.l2gethImg.Endpoint()
	}
	return ""
}

// DbEndpoint returns the endpoint of the dbimg
func (b *App) DbEndpoint() string {
	return b.dbImg.Endpoint()
}

func (b *App) runL1Geth(t *testing.T) {
	if b.l1gethImg != nil {
		return
	}
	b.l1gethImg = newTestL1Docker(t)
}

// L1Client returns a ethclient by dialing running l1geth
func (b *App) L1Client() (*ethclient.Client, error) {
	if b.l1gethImg == nil || reflect2.IsNil(b.l1gethImg) {
		return nil, fmt.Errorf("l1 geth is not running")
	}
	client, err := ethclient.Dial(b.l1gethImg.Endpoint())
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (b *App) runL2Geth(t *testing.T) {
	if b.l2gethImg != nil {
		return
	}
	b.l2gethImg = newTestL2Docker(t)
}

// L2Client returns a ethclient by dialing running l2geth
func (b *App) L2Client() (*ethclient.Client, error) {
	if b.l2gethImg == nil || reflect2.IsNil(b.l2gethImg) {
		return nil, fmt.Errorf("l2 geth is not running")
	}
	client, err := ethclient.Dial(b.l2gethImg.Endpoint())
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (b *App) mockDBConfig() error {
	if b.dbConfig == nil {
		cfg, err := database.NewConfig(b.dbOriginCfg)
		if err != nil {
			return err
		}
		b.dbConfig = cfg
	}

	if b.dbImg != nil {
		b.dbConfig.DSN = b.dbImg.Endpoint()
	}
	data, err := json.Marshal(b.dbConfig)
	if err != nil {
		return err
	}

	return os.WriteFile(b.dbFile, data, 0644) //nolint:gosec
}

func newTestL1Docker(t *testing.T) ImgInstance {
	id, _ := rand.Int(rand.Reader, big.NewInt(2000))
	imgL1geth := NewImgGeth(t, "scroll_l1geth", "", "", 0, l1StartPort+int(id.Int64()))
	assert.NoError(t, imgL1geth.Start())

	// try 3 times to get chainID until is ok.
	utils.TryTimes(3, func() bool {
		client, _ := ethclient.Dial(imgL1geth.Endpoint())
		if client != nil {
			if _, err := client.ChainID(context.Background()); err == nil {
				return true
			}
		}
		return false
	})

	return imgL1geth
}

func newTestL2Docker(t *testing.T) ImgInstance {
	id, _ := rand.Int(rand.Reader, big.NewInt(2000))
	imgL2geth := NewImgGeth(t, "scroll_l2geth", "", "", 0, l2StartPort+int(id.Int64()))
	assert.NoError(t, imgL2geth.Start())

	// try 3 times to get chainID until is ok.
	utils.TryTimes(3, func() bool {
		client, _ := ethclient.Dial(imgL2geth.Endpoint())
		if client != nil {
			if _, err := client.ChainID(context.Background()); err == nil {
				return true
			}
		}
		return false
	})

	return imgL2geth
}

func newTestDBDocker(t *testing.T, driverName string) ImgInstance {
	id, _ := rand.Int(rand.Reader, big.NewInt(2000))
	imgDB := NewImgDB(t, driverName, "123456", "test_db", dbStartPort+int(id.Int64()))
	assert.NoError(t, imgDB.Start())

	// try 5 times until the db is ready.
	utils.TryTimes(5, func() bool {
		db, _ := sqlx.Open(driverName, imgDB.Endpoint())
		if db != nil {
			return db.Ping() == nil
		}
		return false
	})

	return imgDB
}
