package docker_test

import (
	"context"
	"scroll-tech/common/testcontainers"
	"testing"

	_ "github.com/lib/pq" //nolint:golint
	"github.com/stretchr/testify/assert"
)

var (
	testApps *testcontainers.TestcontainerApps
)

func TestMain(m *testing.M) {
	defer func() {
		if testApps != nil {
			testApps.Free()
		}
	}()
	m.Run()
}

func TestDB(t *testing.T) {
	testApps = testcontainers.NewTestcontainerApps()
	assert.NoError(t, testApps.StartPostgresContainer())

	db, err := testApps.GetDBClient()
	assert.NoError(t, err)
	assert.NoError(t, db.Ping())
}

func TestL1Geth(t *testing.T) {
	assert.NoError(t, testApps.StartL1GethContainer())

	client, err := testApps.GetL1GethClient()
	assert.NoError(t, err)

	chainID, err := client.ChainID(context.Background())
	assert.NoError(t, err)
	t.Logf("chainId: %s", chainID.String())
}

func TestL2Geth(t *testing.T) {
	assert.NoError(t, testApps.StartL2GethContainer())

	client, err := testApps.GetL2GethClient()
	assert.NoError(t, err)

	chainID, err := client.ChainID(context.Background())
	assert.NoError(t, err)
	t.Logf("chainId: %s", chainID.String())
}
