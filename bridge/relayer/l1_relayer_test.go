package relayer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"scroll-tech/common/types"
	"scroll-tech/common/utils"
	"scroll-tech/database/migrate"

	"scroll-tech/bridge/sender"

	"scroll-tech/database"
)

var (
	templateL1Message = []*types.L1Message{
		{
			QueueIndex: 1,
			MsgHash:    "msg_hash1",
			Height:     1,
			Sender:     "0x596a746661dbed76a84556111c2872249b070e15",
			Value:      "0x19ece",
			GasLimit:   11529940,
			Target:     "0x2c73620b223808297ea734d946813f0dd78eb8f7",
			Calldata:   "testdata",
			Layer1Hash: "hash0",
		},
		{
			QueueIndex: 2,
			MsgHash:    "msg_hash2",
			Height:     2,
			Sender:     "0x596a746661dbed76a84556111c2872249b070e15",
			Value:      "0x19ece",
			GasLimit:   11529940,
			Target:     "0x2c73620b223808297ea734d946813f0dd78eb8f7",
			Calldata:   "testdata",
			Layer1Hash: "hash1",
		},
	}
)

// testCreateNewRelayer test create new relayer instance and stop
func testCreateNewL1Relayer(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	relayer, err := NewLayer1Relayer(context.Background(), db, cfg.L2Config.RelayerConfig)
	assert.NoError(t, err)
	assert.NotNil(t, relayer)
}

func testL1RelayerProcessSaveEvents(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()
	l1Cfg := cfg.L1Config
	relayer, err := NewLayer1Relayer(context.Background(), db, l1Cfg.RelayerConfig)
	assert.NoError(t, err)
	assert.NoError(t, db.SaveL1Messages(context.Background(), templateL1Message))
	relayer.ProcessSavedEvents()
	msg1, err := db.GetL1MessageByQueueIndex(1)
	assert.NoError(t, err)
	assert.Equal(t, msg1.Status, types.MsgSubmitted)
	msg2, err := db.GetL1MessageByQueueIndex(2)
	assert.NoError(t, err)
	assert.Equal(t, msg2.Status, types.MsgSubmitted)
}

func testL1RelayerMsgConfirm(t *testing.T) {
	// Set up the database and defer closing it.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	// Insert test data.
	assert.NoError(t, db.SaveL1Messages(context.Background(),
		[]*types.L1Message{
			{MsgHash: "msg-1", QueueIndex: 0},
			{MsgHash: "msg-2", QueueIndex: 1},
		}))

	// Create and set up the Layer1 Relayer.
	l1Cfg := cfg.L1Config
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l1Relayer, err := NewLayer1Relayer(ctx, db, l1Cfg.RelayerConfig)
	assert.NoError(t, err)

	// Simulate message confirmations.
	l1Relayer.messageSender.SendConfirmation(&sender.Confirmation{
		ID:           "msg-1",
		IsSuccessful: true,
	})
	l1Relayer.messageSender.SendConfirmation(&sender.Confirmation{
		ID:           "msg-2",
		IsSuccessful: false,
	})

	// Check the database for the updated status using TryTimes.
	utils.TryTimes(5, func() bool {
		msg1, err1 := db.GetL1MessageByMsgHash("msg-1")
		msg2, err2 := db.GetL1MessageByMsgHash("msg-2")
		return err1 == nil && msg1.Status == types.MsgConfirmed &&
			err2 == nil && msg2.Status == types.MsgRelayFailed
	})
}

func testL1RelayerGasOracleConfirm(t *testing.T) {
	// Set up the database and defer closing it.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	// Insert test data.
	assert.NoError(t, db.InsertL1Blocks(context.Background(),
		[]*types.L1BlockInfo{
			{Hash: "gas-oracle-1", Number: 0},
			{Hash: "gas-oracle-2", Number: 1},
		}))

	// Create and set up the Layer2 Relayer.
	l1Cfg := cfg.L1Config
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l1Relayer, err := NewLayer1Relayer(ctx, db, l1Cfg.RelayerConfig)
	assert.NoError(t, err)

	// Simulate message confirmations.
	l1Relayer.gasOracleSender.SendConfirmation(&sender.Confirmation{
		ID:           "gas-oracle-1",
		IsSuccessful: true,
	})
	l1Relayer.gasOracleSender.SendConfirmation(&sender.Confirmation{
		ID:           "gas-oracle-2",
		IsSuccessful: false,
	})

	// Check the database for the updated status using TryTimes.
	utils.TryTimes(5, func() bool {
		msg1, err1 := db.GetL1BlockInfos(map[string]interface{}{"hash": "gas-oracle-1"})
		msg2, err2 := db.GetL1BlockInfos(map[string]interface{}{"hash": "gas-oracle-2"})
		return err1 == nil && len(msg1) == 1 && msg1[0].GasOracleStatus == types.GasOracleImported &&
			err2 == nil && len(msg2) == 1 && msg2[0].GasOracleStatus == types.GasOracleFailed
	})
}
