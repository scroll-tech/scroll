package docker_test

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" //nolint:golint
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/docker"
)

var (
	base *docker.App
)

func TestMain(m *testing.M) {
	base = docker.NewDockerApp()

	m.Run()

	base.Free()
}

func TestDB(t *testing.T) {
	base.RunDBImage(t)

	db, err := sqlx.Open("postgres", base.DBImg.Endpoint())
	assert.NoError(t, err)
	assert.NoError(t, db.Ping())
}

func TestL1Geth(t *testing.T) {
	base.RunL1Geth(t)

	client, err := base.L1Client()
	assert.NoError(t, err)

	chainID, err := client.ChainID(context.Background())
	assert.NoError(t, err)
	t.Logf("chainId: %s", chainID.String())
}

func TestL2Geth(t *testing.T) {
	base.RunL2Geth(t)

	client, err := base.L2Client()
	assert.NoError(t, err)

	chainID, err := client.ChainID(context.Background())
	assert.NoError(t, err)
	t.Logf("chainId: %s", chainID.String())
}
