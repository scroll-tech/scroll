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
	"scroll-tech/database/orm"

	"scroll-tech/bridge/config"
	"scroll-tech/bridge/l1"

	"scroll-tech/database"

	"scroll-tech/common/docker"
)

var (
	// config
	cfg                   *config.Config
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
	l1gethImg docker.ImgInstance
	l2gethImg docker.ImgInstance
	dbImg     docker.ImgInstance
)

func setupEnv(t *testing.T) {
	// Load config.
	var err error
	cfg, err = config.NewConfig("../config.json")
	assert.NoError(t, err)

	// Create l1geth container.
	l1gethImg = docker.NewTestL1Docker(t)
	cfg.L2Config.RelayerConfig.SenderConfig.Endpoint = l1gethImg.Endpoint()
	cfg.L1Config.Endpoint = l1gethImg.Endpoint()

	// Create l2geth container.
	l2gethImg = docker.NewTestL2Docker(t)
	cfg.L1Config.RelayerConfig.SenderConfig.Endpoint = l2gethImg.Endpoint()
	cfg.L2Config.Endpoint = l2gethImg.Endpoint()

	// Create db container.
	dbImg = docker.NewTestDBDocker(t, cfg.DBConfig.DriverName)
	cfg.DBConfig.DSN = dbImg.Endpoint()

}

// testCreateNewL1Relayer test create new relayer instance and stop
func testCreateNewL1Relayer(t *testing.T) {
	client, err := ethclient.Dial(l1gethImg.Endpoint())
	assert.NoError(t, err)

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
	// set up l1client
	client, err := ethclient.Dial(l1gethImg.Endpoint())
	assert.NoError(t, err)
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()
	relayer, err := l1.NewLayer1Relayer(context.Background(), client, 1, db, cfg.L1Config.RelayerConfig)
	assert.NoError(t, err)

	err = db.SaveLayer1Messages(context.Background(), templateLayer1Message)
	assert.NoError(t, err)

	// process msg
	relayer.ProcessSavedEvents()

	msgs, err := db.GetL1UnprocessedMessages()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(msgs))

	msg, err := db.GetLayer1MessageByLayer1Hash("hash1")
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
		assert.NoError(t, l1gethImg.Stop())
		assert.NoError(t, l2gethImg.Stop())
		assert.NoError(t, dbImg.Stop())
	})
}
