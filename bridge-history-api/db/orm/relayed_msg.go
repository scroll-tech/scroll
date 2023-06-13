package orm

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/ethereum/go-ethereum/log"
	"github.com/jmoiron/sqlx"
)

type relayedMsgOrm struct {
	db *sqlx.DB
}

// NewRelayedMsgOrm create an NewRelayedMsgOrm instance
func NewRelayedMsgOrm(db *sqlx.DB) RelayedMsgOrm {
	return &relayedMsgOrm{db: db}
}

func (l *relayedMsgOrm) BatchInsertRelayedMsgDBTx(dbTx *sqlx.Tx, messages []*RelayedMsg) error {
	if len(messages) == 0 {
		return nil
	}
	var err error
	messageMaps := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		messageMaps[i] = map[string]interface{}{
			"msg_hash":    msg.MsgHash,
			"height":      msg.Height,
			"layer1_hash": msg.Layer1Hash,
			"layer2_hash": msg.Layer2Hash,
		}

		_, err = dbTx.NamedExec(`insert into relayed_msg(msg_hash, height, layer1_hash, layer2_hash) values(:msg_hash, :height, :layer1_hash, :layer2_hash);`, messageMaps[i])
		if err != nil && !strings.Contains(err.Error(), "pq: duplicate key value violates unique constraint \"relayed_msg_hash_uindex") {
			log.Error("BatchInsertRelayedMsgDBTx: failed to insert l1 cross msgs", "msg_Hashe", msg.MsgHash, "height", msg.Height, "err", err)
			break
		}
	}
	return err
}

func (l *relayedMsgOrm) GetRelayedMsgByHash(msg_hash string) (*RelayedMsg, error) {
	result := &RelayedMsg{}
	row := l.db.QueryRowx(`SELECT msg_hash, height, layer1_hash, layer2_hash FROM relayed_msg WHERE msg_hash = $1 AND NOT is_deleted;`, msg_hash)
	if err := row.StructScan(result); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return result, nil
}

func (l *relayedMsgOrm) GetLatestRelayedHeightOnL1() (int64, error) {
	row := l.db.QueryRow(`SELECT height FROM relayed_msg WHERE layer1_hash != '' AND NOT is_deleted ORDER BY height DESC LIMIT 1;`)
	var result sql.NullInt64
	if err := row.Scan(&result); err != nil {
		if err == sql.ErrNoRows || !result.Valid {
			return -1, nil
		}
		return 0, err
	}
	if result.Valid {
		return result.Int64, nil
	}
	return 0, nil
}

func (l *relayedMsgOrm) DeleteL2RelayedHashAfterHeightDBTx(dbTx *sqlx.Tx, height int64) error {
	_, err := dbTx.Exec(`UPDATE relayed_msg SET is_deleted = true WHERE height > $1 AND layer2_hash != '';`, height)
	return err
}
