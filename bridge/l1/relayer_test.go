package l1_test

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"

	"scroll-tech/database/migrate"

	"scroll-tech/bridge/config"
	"scroll-tech/bridge/l1"

	"scroll-tech/database"

	"scroll-tech/common/docker"
)
	"scroll-tech/common/docker"
	"scroll-tech/common/utils"
)

var (
	templateLayer1Message = []*orm.Layer1Message{
		{
			Content: orm.MsgContent{
				Sender:   common.HexToAddress("0x596a746661dbed76a84556111c2872249b070e15"),
				Value:    big.NewInt(106190),
				Fee:      big.NewInt(106190),
				GasLimit: big.NewInt(11529940),
				Deadline: big.NewInt(time.Now().Unix()),
				Target:   common.HexToAddress("0x2c73620b223808297ea734d946813f0dd78eb8f7"),
				Calldata: []byte("testdata"),
			},
			Nonce:      1,
			Height:     1,
			Layer1Hash: "hash0",
		},
		{
			Content: orm.MsgContent{
				Sender:   common.HexToAddress("0x596a746661dbed76a84556111c2872249b070e15"),
				Value:    big.NewInt(106190),
				Fee:      big.NewInt(106190),
				GasLimit: big.NewInt(11529940),
				Deadline: big.NewInt(time.Now().Unix()),
				Target:   common.HexToAddress("0x2c73620b223808297ea734d946813f0dd78eb8f7"),
				Calldata: []byte("testdata"),
			},
			Nonce:      2,
			Height:     2,
			Layer1Hash: "hash1",
		},
	}
	l1Docker docker.ImgInstance
	l2Docker docker.ImgInstance
	dbDocker docker.ImgInstance
)

func setupEnv(t *testing.T) {
	l1Docker = mock.NewTestL1Docker(t, TEST_CONFIG)
	l2Docker = mock.NewTestL2Docker(t, TEST_CONFIG)
	dbDocker = mock.GetDbDocker(t, TEST_CONFIG)
}

// testCreateNewL1Relayer test create new relayer instance and stop
func testCreateNewL1Relayer(t *testing.T) {
	cfg, err := config.NewConfig("../config.json")
	assert.NoError(t, err)
	l1docker := docker.NewTestL1Docker(t)
	defer l1docker.Stop()
	cfg.L2Config.RelayerConfig.SenderConfig.Endpoint = l1docker.Endpoint()
	cfg.L1Config.Endpoint = l1docker.Endpoint()

	client, err := ethclient.Dial(l1docker.Endpoint())
	assert.NoError(t, err)

	dbImg := docker.NewTestDBDocker(t, cfg.DBConfig.DriverName)
	defer dbImg.Stop()
	cfg.DBConfig.DSN = dbImg.Endpoint()

	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	relayer, err := l1.NewLayer1Relayer(context.Background(), client, 1, db, cfg.L2Config.RelayerConfig)
	assert.NoError(t, err)
	relayer.Start()
	defer relayer.Stop()
	defer db.Close()

}

func testProcessSavedEvents(t *testing.T) {
	cfg, err := config.NewConfig("../config.json")
	assert.NoError(t, err)

	// set up endpoint for test config
	cfg.L1Config.RelayerConfig.SenderConfig.Endpoint = l2Docker.Endpoint()
	cfg.L1Config.Endpoint = l1Docker.Endpoint()
	client, err := ethclient.Dial(l1Docker.Endpoint())
	assert.NoError(t, err)
	db, err := database.NewOrmFactory(TEST_CONFIG.DB_CONFIG)
	assert.NoError(t, err)
	migrate.Migrate(db.GetDB().DB)

	relayer, err := l1.NewLayer1Relayer(context.Background(), client, 1, db, cfg.L1Config.RelayerConfig)
	assert.NoError(t, err)

	err = db.SaveLayer1Messages(context.Background(), templateLayer1Message)
	assert.NoError(t, err)

	// process one msg per call
	relayer.ProcessSavedEvents()

	msgs, err := db.GetL1UnprocessedMessages()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(msgs))

	msg, err := db.GetLayer1MessageByLayer1Hash("hash0")
	assert.NoError(t, err)
	assert.Equal(t, orm.MsgSubmitted, msg.Status)
	assert.Equal(t, []byte("testdata"), msg.Content.Calldata)

	relayer.ProcessSavedEvents()

	msgs, err = db.GetL1UnprocessedMessages()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(msgs))

	msg, err = db.GetLayer1MessageByLayer1Hash("hash1")
	assert.NoError(t, err)
	assert.Equal(t, orm.MsgSubmitted, msg.Status)
	assert.Equal(t, []byte("testdata"), msg.Content.Calldata)

	defer db.Close()

}

func TestFunction(t *testing.T) {
	setupEnv(t)

	// Run l2 watcher test cases.
	t.Run("Test Create New L1Relayer", testCreateNewL1Relayer)
	t.Run("Test Relayer process saved event", testProcessSavedEvents)

	// Teardown
	t.Cleanup(func() {
		assert.NoError(t, l1Docker.Stop())
		assert.NoError(t, l2Docker.Stop())
		assert.NoError(t, dbDocker.Stop())
	})
}
