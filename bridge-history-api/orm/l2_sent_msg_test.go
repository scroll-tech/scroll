package orm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"bridge-history-api/orm/migrate"

	"scroll-tech/common/database"
	"scroll-tech/common/docker"
)

func TestGetClaimableL2SentMsgByAddress(t *testing.T) {
	base := docker.NewDockerApp()
	base.RunDBImage(t)

	db, err := database.InitDB(
		&database.Config{
			DSN:        base.DBConfig.DSN,
			DriverName: base.DBConfig.DriverName,
			MaxOpenNum: base.DBConfig.MaxOpenNum,
			MaxIdleNum: base.DBConfig.MaxIdleNum,
		},
	)
	assert.NoError(t, err)

	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	l2SentMsgOrm := NewL2SentMsg(db)
	relayedMsgOrm := NewRelayedMsg(db)

	msgs, err := l2SentMsgOrm.GetClaimableL2SentMsgByAddress(context.Background(), "sender1")
	assert.NoError(t, err)
	assert.Len(t, msgs, 0)

	l2SentMsgs := []*L2SentMsg{
		{
			Sender:   "sender1",
			MsgHash:  "hash1",
			MsgProof: "proof1",
			Nonce:    0,
		},
		{
			OriginalSender: "sender1",
			MsgHash:        "hash2",
			MsgProof:       "proof2",
			Nonce:          1,
		},
		{
			OriginalSender: "sender1",
			MsgHash:        "hash3",
			MsgProof:       "",
			Nonce:          2,
		},
	}
	relayedMsgs := []*RelayedMsg{
		{
			MsgHash: "hash2",
		},
		{
			MsgHash: "hash3",
		},
	}
	err = l2SentMsgOrm.InsertL2SentMsg(context.Background(), l2SentMsgs)
	assert.NoError(t, err)
	err = relayedMsgOrm.InsertRelayedMsg(context.Background(), relayedMsgs)
	assert.NoError(t, err)

	msgs, err = l2SentMsgOrm.GetClaimableL2SentMsgByAddress(context.Background(), "sender1")
	assert.NoError(t, err)
	assert.Len(t, msgs, 1)
	assert.Equal(t, "hash1", msgs[0].MsgHash)
}
