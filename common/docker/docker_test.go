package docker_test

import (
	"testing"

	_ "github.com/lib/pq" //nolint:golint

	"scroll-tech/common/docker"
)

// func TestMain(m *testing.M) {
// 	base = NewDockerApp()

// 	m.Run()

// 	base.free()
// }

func TestStartProcess(t *testing.T) {
	base := docker.NewDockerApp()
	// Start l1geth l2geth postgres.
	base.RunImages(t)

	// migrate db.
	base.RunDBApp(t, "reset", "successful to reset")
	//base.runDBApp(t, "migrate", "current version:")
	base.Free()
}

// func TestDocker(t *testing.T) {
// 	t.Parallel()
// 	t.Run("testL1Geth", testL1Geth)
// 	t.Run("testL2Geth", testL2Geth)
// 	//t.Run("testDB", testDB)
// }

// func testL1Geth(t *testing.T) {
// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()

// 	img := base.l1gethImg

// 	client, err := ethclient.Dial(img.Endpoint())
// 	assert.NoError(t, err)

// 	chainID, err := client.ChainID(ctx)
// 	assert.NoError(t, err)
// 	t.Logf("chainId: %s", chainID.String())
// }

// func testL2Geth(t *testing.T) {
// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()

// 	img := base.l2gethImg

// 	client, err := ethclient.Dial(img.Endpoint())
// 	assert.NoError(t, err)

// 	chainID, err := client.ChainID(ctx)
// 	assert.NoError(t, err)
// 	t.Logf("chainId: %s", chainID.String())
// }

// func testDB(t *testing.T) {
// 	driverName := "postgres"
// 	dbImg := base.dbImg

// 	db, err := sqlx.Open(driverName, dbImg.Endpoint())
// 	assert.NoError(t, err)
// 	assert.NoError(t, db.Ping())
// }
