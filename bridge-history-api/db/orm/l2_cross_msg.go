package orm

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/jmoiron/sqlx"
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
	row := l.db.QueryRowx(`SELECT * FROM l2_cross_message WHERE layer2_hash = $1 AND NOT is_deleted;`, l2Hash.String())
	if err := row.StructScan(result); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return result, nil
}

// GetL2CrossMsgsByAddress returns all layer2 cross messages under given address
// Warning: return empty slice if no data found
func (l *l2CrossMsgOrm) GetL2CrossMsgByAddress(sender common.Address) ([]*CrossMsg, error) {
	var results []*CrossMsg
	rows, err := l.db.Queryx(`SELECT * FROM cross_message WHERE sender = $1 AND msg_type = $2 AND NOT is_deleted;`, sender.String(), Layer2Msg)

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
	_, err := dbTx.Exec(`UPDATE cross_message SET is_deleted = true where height > $1 AND msg_type = $2 ;`, height, Layer2Msg)
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
			"height":        msg.Height,
			"sender":        msg.Sender,
			"target":        msg.Target,
			"asset":         msg.Asset,
			"layer2_hash":   msg.Layer2Hash,
			"layer1_token":  msg.Layer1Token,
			"layer2_token":  msg.Layer2Token,
			"token_ids":     msg.TokenIDs,
			"token_amounts": msg.TokenAmounts,
			"amount":        msg.Amount,
			"msg_type":      Layer2Msg,
		}

		_, err = dbTx.NamedExec(`insert into cross_message(height, sender, target, asset, layer2_hash, layer1_token, layer2_token, token_ids, amount, msg_type) select :height, :sender, :target, :asset, :layer2_hash, :layer1_token, :layer2_token, :token_ids, :token_amounts, :amount, :msg_type WHERE NOT EXISTS (SELECT 1 FROM cross_message WHERE layer2_hash = :layer2_hash AND NOT is_deleted);`, messageMaps[i])
		if err != nil {
			log.Error("BatchInsertL2CrossMsgDBTx: failed to insert l2 cross msgs", "layer2hash", msg.Layer2Hash, "heights", msg.Height, "err", err)
			break
		}
	}
	return err
}

func (l *l2CrossMsgOrm) UpdateL2CrossMsgHashDBTx(ctx context.Context, dbTx *sqlx.Tx, l2Hash, msgHash common.Hash) error {
	if _, err := dbTx.ExecContext(ctx, l.db.Rebind("update public.cross_message set msg_hash = ? where layer2_hash = ? AND NOT is_deleted;"), msgHash.String(), l2Hash.String()); err != nil {
		return err
	}
	return nil
}

func (l *l2CrossMsgOrm) UpdateL2CrossMsgHash(ctx context.Context, l2Hash, msgHash common.Hash) error {
	if _, err := l.db.ExecContext(ctx, l.db.Rebind("update public.cross_message set msg_hash = ? where layer2_hash = ? AND NOT is_deleted;"), msgHash.String(), l2Hash.String()); err != nil {
		return err
	}
	return nil
}

func (l *l2CrossMsgOrm) GetLatestL2ProcessedHeight() (int64, error) {
	row := l.db.QueryRowx(`SELECT height FROM cross_message WHERE msg_type = $1 AND NOT is_deleted ORDER BY id DESC LIMIT 1;`, Layer2Msg)
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

func (l *l2CrossMsgOrm) UpdateL2Blocktimestamp(height uint64, timestamp time.Time) error {
	if _, err := l.db.Exec(`UPDATE cross_message SET blocktimestamp = $1 where height = $2 AND msg_type = $3 AND NOT is_deleted`, timestamp, height, Layer2Msg); err != nil {
		return err
	}
	return nil
}

func (l *l2CrossMsgOrm) GetL2EarliestNoBlocktimestampHeight() (uint64, error) {
	row := l.db.QueryRowx(`SELECT height FROM cross_message WHERE blocktimestamp IS NULL AND msg_type = $1 AND NOT is_deleted ORDER BY height ASC LIMIT 1;`, Layer2Msg)
	var result uint64
	if err := row.Scan(&result); err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	return result, nil
}
