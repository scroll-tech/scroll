package relayer_test

import (
	"context"
	"fmt"
	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"scroll-tech/common/types"
	"testing"

	"scroll-tech/database/migrate"

	"scroll-tech/bridge/relayer"

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

	relayer, err := relayer.NewLayer1Relayer(context.Background(), db, cfg.L2Config.RelayerConfig)
	assert.NoError(t, err)
	assert.NotNil(t, relayer)
}

func testL1CheckSubmittedMessages(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	auth, err := bind.NewKeyedTransactorWithChainID(cfg.L1Config.RelayerConfig.MessageSenderPrivateKeys[0], l1ChainID)
	assert.NoError(t, err)

	signedTx, err := mockTx(auth)
	assert.NoError(t, err)
	err = db.SaveTx(templateL1Message[0].MsgHash, auth.From.String(), types.L1toL2MessageTx, signedTx, "")
	assert.Nil(t, err)
	err = db.SaveL1Messages(context.Background(), templateL1Message)
	assert.NoError(t, err)
	err = db.UpdateLayer1Status(context.Background(), templateL1Message[0].MsgHash, types.MsgSubmitted)
	assert.NoError(t, err)

	cfg.L1Config.RelayerConfig.SenderConfig.Confirmations = 0
	relayer, err := relayer.NewLayer1Relayer(context.Background(), db, cfg.L1Config.RelayerConfig)
	assert.NoError(t, err)
	assert.NotNil(t, relayer)
	err = relayer.CheckSubmittedMessages()
	assert.Nil(t, err)

	relayer.WaitL1MsgSender()

	// check tx is confirmed.
	maxIndex, txMsgs, err := db.GetL1TxMessages(
		map[string]interface{}{"status": types.MsgConfirmed},
		fmt.Sprintf("AND queue_index > %d", 0),
		fmt.Sprintf("ORDER BY queue_index ASC LIMIT %d", 10),
	)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(txMsgs))
	assert.Equal(t, templateL1Message[0].QueueIndex, maxIndex)

	// check tx is on chain.
	_, err = l2Cli.TransactionReceipt(context.Background(), common.HexToHash(txMsgs[0].TxHash.String))
	assert.NoError(t, err)
}
