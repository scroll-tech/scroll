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

// testCreateNewRelayer test create new relayer instance and stop
func testCreateNewL1Relayer(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(vp.Sub("db_config"))
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	client, err := ethclient.Dial(l1gethImg.Endpoint())
	assert.NoError(t, err)

	relayer, err := l1.NewLayer1Relayer(context.Background(), client, db, vp.Sub("l2_config.relayer_config"))
	assert.NoError(t, err)
	defer relayer.Stop()

	relayer.Start()
}
