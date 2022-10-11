package docker

import (
	"context"
	"testing"

	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"

	"scroll-tech/store"
	"scroll-tech/store/config"
	"scroll-tech/store/migrate"
)

func TestL1Geth(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	img := NewImgGeth(t, "scroll_l1geth", "", "", 8535, 0)
	assert.NoError(t, img.Start())
	defer img.Stop()

	client, err := ethclient.Dial(img.Endpoint())
	assert.NoError(t, err)

	chainID, err := client.ChainID(ctx)
	assert.NoError(t, err)
	t.Logf("chainId: %s", chainID.String())
}

func TestL2Geth(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	img := NewImgGeth(t, "scroll_l2geth", "", "", 8535, 0)
	assert.NoError(t, img.Start())
	defer img.Stop()

	client, err := ethclient.Dial(img.Endpoint())
	assert.NoError(t, err)

	chainID, err := client.ChainID(ctx)
	assert.NoError(t, err)
	t.Logf("chainId: %s", chainID.String())
}

func TestDB(t *testing.T) {
	img := NewImgDB(t, "postgres", "123456", "test", 5433)
	assert.NoError(t, img.Start())
	defer img.Stop()

	db, err := store.NewConnection(&config.DBConfig{
		DriverName: "postgres",
		DSN:        img.Endpoint(),
	})
	assert.NoError(t, err)

	version := int64(0)
	assert.NoError(t, migrate.Rollback(db.DB, &version))

	assert.NoError(t, migrate.Migrate(db.DB))

	vs, err := migrate.Current(db.DB)
	assert.NoError(t, err)

	t.Logf("current version:%d", vs)
}
