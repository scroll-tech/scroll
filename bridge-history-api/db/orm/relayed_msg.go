package orm

import (
	"database/sql"
	"errors"

	"github.com/ethereum/go-ethereum/log"
	"github.com/jmoiron/sqlx"
)

// RelayedMsg is the struct for relayed_msg table
type RelayedMsg struct {
	MsgHash    string `json:"msg_hash" db:"msg_hash"`
	Height     uint64 `json:"height" db:"height"`
	Layer1Hash string `json:"layer1_hash" db:"layer1_hash"`
	Layer2Hash string `json:"layer2_hash" db:"layer2_hash"`
}

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
	}
	_, err = dbTx.NamedExec(`insert into relayed_msg(msg_hash, height, layer1_hash, layer2_hash) values(:msg_hash, :height, :layer1_hash, :layer2_hash);`, messageMaps)
	if err != nil {
		log.Error("BatchInsertRelayedMsgDBTx: failed to insert relayed msgs", "err", err)
		return err
	}
	return nil
}

func (l *relayedMsgOrm) GetRelayedMsgByHash(msgHash string) (*RelayedMsg, error) {
	result := &RelayedMsg{}
	row := l.db.QueryRowx(`SELECT msg_hash, height, layer1_hash, layer2_hash FROM relayed_msg WHERE msg_hash = $1 AND deleted_at IS NULL;`, msgHash)
	if err := row.StructScan(result); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return result, nil
}

func (l *relayedMsgOrm) GetLatestRelayedHeightOnL1() (int64, error) {
	row := l.db.QueryRow(`SELECT height FROM relayed_msg WHERE layer1_hash != '' AND deleted_at IS NULL ORDER BY height DESC LIMIT 1;`)
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

func (l *relayedMsgOrm) GetLatestRelayedHeightOnL2() (int64, error) {
	row := l.db.QueryRow(`SELECT height FROM relayed_msg WHERE layer2_hash != '' AND deleted_at IS NULL ORDER BY height DESC LIMIT 1;`)
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

func (l *relayedMsgOrm) DeleteL1RelayedHashAfterHeightDBTx(dbTx *sqlx.Tx, height int64) error {
	_, err := dbTx.Exec(`UPDATE relayed_msg SET deleted_at = current_timestamp WHERE height > $1 AND layer1_hash != '';`, height)
	return err
}

func (l *relayedMsgOrm) DeleteL2RelayedHashAfterHeightDBTx(dbTx *sqlx.Tx, height int64) error {
	_, err := dbTx.Exec(`UPDATE relayed_msg SET deleted_at = current_timestamp WHERE height > $1 AND layer2_hash != '';`, height)
	return err
}
