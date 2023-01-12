package docker

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" //nolint:golint
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"
)

func TestDocker(t *testing.T) {
	t.Parallel()

	t.Run("testL1Geth", testL1Geth)
	t.Run("testL2Geth", testL2Geth)
	t.Run("testDB", testDB)
}

func testL1Geth(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	img := NewTestL1Docker(t)
	defer img.Stop() //nolint:errcheck

	client, err := ethclient.Dial(img.Endpoint())
	assert.NoError(t, err)

	chainID, err := client.ChainID(ctx)
	assert.NoError(t, err)
	t.Logf("chainId: %s", chainID.String())
}

func testL2Geth(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	img := NewTestL2Docker(t)
	defer img.Stop() //nolint:errcheck

	client, err := ethclient.Dial(img.Endpoint())
	assert.NoError(t, err)

	chainID, err := client.ChainID(ctx)
	assert.NoError(t, err)
	t.Logf("chainId: %s", chainID.String())
}

func testDB(t *testing.T) {
	driverName := "postgres"
	dbImg := NewTestDBDocker(t, driverName)
	defer dbImg.Stop() //nolint:errcheck

	db, err := sqlx.Open(driverName, dbImg.Endpoint())
	assert.NoError(t, err)
	assert.NoError(t, db.Ping())
}
