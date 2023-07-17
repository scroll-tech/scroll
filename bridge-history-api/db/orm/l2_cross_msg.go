package orm

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type l2CrossMsgOrm struct {
	db *sqlx.DB
}

// NewL2CrossMsgOrm create an NewL2CrossMsgOrm instance
func NewL2CrossMsgOrm(db *sqlx.DB) L2CrossMsgOrm {
	return &l2CrossMsgOrm{db: db}
}

func (l *l2CrossMsgOrm) GetL2CrossMsgByHash(l2Hash common.Hash) (*CrossMsg, error) {
	result := &CrossMsg{}
	row := l.db.QueryRowx(`SELECT * FROM cross_message WHERE layer2_hash = $1 AND deleted_at IS NULL;`, l2Hash.String())
	if err := row.StructScan(result); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return result, nil
}

// GetL2CrossMsgByAddress returns all layer2 cross messages under given address
// Warning: return empty slice if no data found
func (l *l2CrossMsgOrm) GetL2CrossMsgByAddress(sender common.Address) ([]*CrossMsg, error) {
	var results []*CrossMsg
	rows, err := l.db.Queryx(`SELECT * FROM cross_message WHERE sender = $1 AND msg_type = $2 AND deleted_at IS NULL;`, sender.String(), Layer2Msg)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err = rows.Close(); err != nil {
			log.Warn("failed to close rows", "err", err)
		}
	}()
	for rows.Next() {
		msg := &CrossMsg{}
		if err = rows.StructScan(msg); err != nil {
			break
		}
		results = append(results, msg)
	}
	if len(results) == 0 && errors.Is(err, sql.ErrNoRows) {
		// log.Warn("no unprocessed layer1 messages in db", "err", err)
	} else if err != nil {
		return nil, err
	}
	return results, nil

}

func (l *l2CrossMsgOrm) DeleteL2CrossMsgFromHeightDBTx(dbTx *sqlx.Tx, height int64) error {
	_, err := dbTx.Exec(`UPDATE cross_message SET deleted_at = current_timestamp where height > $1 AND msg_type = $2 ;`, height, Layer2Msg)
	if err != nil {
		log.Error("DeleteL1CrossMsgAfterHeightDBTx: failed to delete", "height", height, "err", err)
		return err
	}
	return nil
}

func (l *l2CrossMsgOrm) BatchInsertL2CrossMsgDBTx(dbTx *sqlx.Tx, messages []*CrossMsg) error {
	if len(messages) == 0 {
		return nil
	}

	var err error
	messageMaps := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		messageMaps[i] = map[string]interface{}{
			"height":       msg.Height,
			"sender":       msg.Sender,
			"target":       msg.Target,
			"asset":        msg.Asset,
			"msg_hash":     msg.MsgHash,
			"layer2_hash":  msg.Layer2Hash,
			"layer1_token": msg.Layer1Token,
			"layer2_token": msg.Layer2Token,
			"token_ids":    msg.TokenIDs,
			"amount":       msg.Amount,
			"msg_type":     Layer2Msg,
		}
	}
	_, err = dbTx.NamedExec(`insert into cross_message(height, sender, target, asset, msg_hash, layer2_hash, layer1_token, layer2_token, token_ids, amount, msg_type) values(:height, :sender, :target, :asset, :msg_hash, :layer2_hash, :layer1_token, :layer2_token, :token_ids, :amount, :msg_type);`, messageMaps)
	if err != nil {
		log.Error("BatchInsertL2CrossMsgDBTx: failed to insert l2 cross msgs", "err", err)
		return err
	}
	return nil
}

func (l *l2CrossMsgOrm) UpdateL2CrossMsgHashDBTx(ctx context.Context, dbTx *sqlx.Tx, l2Hash, msgHash common.Hash) error {
	if _, err := dbTx.ExecContext(ctx, l.db.Rebind("update cross_message set msg_hash = ? where layer2_hash = ? AND deleted_at IS NULL;"), msgHash.String(), l2Hash.String()); err != nil {
		return err
	}
	return nil
}

func (l *l2CrossMsgOrm) UpdateL2CrossMsgHash(ctx context.Context, l2Hash, msgHash common.Hash) error {
	if _, err := l.db.ExecContext(ctx, l.db.Rebind("update cross_message set msg_hash = ? where layer2_hash = ? AND deleted_at IS NULL;"), msgHash.String(), l2Hash.String()); err != nil {
		return err
	}
	return nil
}

func (l *l2CrossMsgOrm) GetLatestL2ProcessedHeight() (int64, error) {
	row := l.db.QueryRowx(`SELECT height FROM cross_message WHERE msg_type = $1 AND deleted_at IS NULL ORDER BY id DESC LIMIT 1;`, Layer2Msg)
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

func (l *l2CrossMsgOrm) UpdateL2BlockTimestamp(height uint64, timestamp time.Time) error {
	if _, err := l.db.Exec(`UPDATE cross_message SET block_timestamp = $1 where height = $2 AND msg_type = $3 AND deleted_at IS NULL`, timestamp, height, Layer2Msg); err != nil {
		return err
	}
	return nil
}

func (l *l2CrossMsgOrm) GetL2EarliestNoBlockTimestampHeight() (uint64, error) {
	row := l.db.QueryRowx(`SELECT height FROM cross_message WHERE block_timestamp IS NULL AND msg_type = $1 AND deleted_at IS NULL ORDER BY height ASC LIMIT 1;`, Layer2Msg)
	var result uint64
	if err := row.Scan(&result); err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	return result, nil
}

func (l *l2CrossMsgOrm) GetL2CrossMsgByMsgHashList(msgHashList []string) ([]*CrossMsg, error) {
	var results []*CrossMsg
	rows, err := l.db.Queryx(`SELECT * FROM cross_message WHERE msg_hash = ANY($1) AND msg_type = $2 AND deleted_at IS NULL;`, pq.Array(msgHashList), Layer2Msg)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err = rows.Close(); err != nil {
			log.Warn("failed to close rows", "err", err)
		}
	}()	
	for rows.Next() {
		msg := &CrossMsg{}
		if err = rows.StructScan(msg); err != nil {
			break
		}
		results = append(results, msg)
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if len(results) == 0 {
		log.Debug("no L2CrossMsg under given msg hashes", "msg hash list", msgHashList)
	}
	return results, nil
}
