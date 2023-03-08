package docker_test

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" //nolint:golint
	"github.com/stretchr/testify/assert"

	_ "scroll-tech/database/cmd/app"

	"scroll-tech/common/docker"
)

var (
	base *docker.App
)

func TestMain(m *testing.M) {
	base = docker.NewDockerApp("../../database/config.json")

	m.Run()

	base.Free()
}

func TestStartProcess(t *testing.T) {
	base.RunImages(t)

	// migrate db.
	base.RunDBApp(t, "reset", "successful to reset")
	base.RunDBApp(t, "migrate", "current version:")
}

func TestDocker(t *testing.T) {
	base.RunImages(t)
	t.Parallel()
	t.Run("testL1Geth", testL1Geth)
	t.Run("testL2Geth", testL2Geth)
	t.Run("testDB", testDB)
}

func testL1Geth(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := base.L1Client()
	assert.NoError(t, err)

	chainID, err := client.ChainID(ctx)
	assert.NoError(t, err)
	t.Logf("chainId: %s", chainID.String())
}

func testL2Geth(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := base.L2Client()
	assert.NoError(t, err)

	chainID, err := client.ChainID(ctx)
	assert.NoError(t, err)
	t.Logf("chainId: %s", chainID.String())
}

func testDB(t *testing.T) {
	driverName := "postgres"

	db, err := sqlx.Open(driverName, base.DbEndpoint())
	assert.NoError(t, err)
	assert.NoError(t, db.Ping())
}
