package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/modern-go/reflect2"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"scroll-tech/common/cmd"
	"scroll-tech/common/docker"
	"scroll-tech/database"
)

type dockerApp struct {
	l1gethImg docker.ImgInstance
	l2gethImg docker.ImgInstance

	dbImg    docker.ImgInstance
	dbConfig *database.DBConfig
	dbFile   string

	// coordinator ws port
	wsPort int64
	// common time stamp.
	timestamp int
}

func newDockerApp() *dockerApp {
	timestamp := time.Now().Nanosecond()
	return &dockerApp{
		timestamp: timestamp,
		dbFile:    fmt.Sprintf("/tmp/%d_db-config.json", timestamp),
	}
}

func (b *dockerApp) runImages(t *testing.T) {
	b.runDBImage(t)
	b.runL1Geth(t)
	b.runL2Geth(t)
}

func (b *dockerApp) runDBImage(t *testing.T) {
	if b.dbImg != nil {
		return
	}
	b.dbImg = docker.NewTestDBDocker(t, "postgres")
	if err := b.mockDBConfig(); err != nil {
		_ = b.dbImg.Stop()
		b.dbImg = nil
		_ = os.Remove(b.dbFile)
		t.Fatal(err)
	}
}

func (b *dockerApp) runDBApp(t *testing.T, option, keyword string) {
	args := []string{option, "--config", b.dbFile}
	app := cmd.NewCmd("db_cli-test", args...)
	defer app.WaitExit()

	// Wait expect result.
	app.ExpectWithTimeout(t, true, time.Second*3, keyword)
	app.RunApp(nil)
}

func (b *dockerApp) free() {
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

func (b *dockerApp) dbEndpoint() string {
	return b.dbImg.Endpoint()
}

func (b *dockerApp) runL1Geth(t *testing.T) {
	if b.l1gethImg != nil {
		return
	}
	b.l1gethImg = docker.NewTestL1Docker(t)
}

func (b *dockerApp) l1Client() (*ethclient.Client, error) {
	if b.l1gethImg == nil || reflect2.IsNil(b.l1gethImg) {
		return nil, fmt.Errorf("l1 geth is not running")
	}
	client, err := ethclient.Dial(b.l1gethImg.Endpoint())
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (b *dockerApp) runL2Geth(t *testing.T) {
	if b.l2gethImg != nil {
		return
	}
	b.l2gethImg = docker.NewTestL2Docker(t)
}

func (b *dockerApp) l2Client() (*ethclient.Client, error) {
	if b.l2gethImg == nil || reflect2.IsNil(b.l2gethImg) {
		return nil, fmt.Errorf("l2 geth is not running")
	}
	client, err := ethclient.Dial(b.l2gethImg.Endpoint())
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (b *dockerApp) mockDBConfig() error {
	if b.dbConfig == nil {
		cfg, err := database.NewConfig("../../database/config.json")
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

	return os.WriteFile(b.dbFile, data, 0644)
}
