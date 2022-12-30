package l1_test

import (
	"context"
	"testing"

	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"

	"scroll-tech/database"
	"scroll-tech/database/migrate"

	"scroll-tech/bridge/l1"
)

func testStartWatcher(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(vp.Sub("db_config"))
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	client, err := ethclient.Dial(l1gethImg.Endpoint())
	assert.NoError(t, err)

	watcher := l1.NewWatcher(context.Background(), client, db, vp.Sub("l2_config"))
	watcher.Start()
	defer watcher.Stop()
}
