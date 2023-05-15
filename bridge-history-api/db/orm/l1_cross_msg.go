package orm

import (
	"context"
	"database/sql"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/jmoiron/sqlx"
)

type l1CrossMsgOrm struct {
	db *sqlx.DB
}

// NewL1CrossMsgOrm create an NewL1CrossMsgOrm instance
func NewL1CrossMsgOrm(db *sqlx.DB) L1CrossMsgOrm {
	return &l1CrossMsgOrm{db: db}
}

func (l *l1CrossMsgOrm) GetL1CrossMsgByHash(l1Hash common.Hash) (*CrossMsg, error) {
	result := &CrossMsg{}
	row := l.db.QueryRowx(`SELECT * FROM cross_message WHERE layer1_hash = $1 AND msg_type = $2 AND NOT is_deleted;`, l1Hash.String(), Layer1Msg)
	if err := row.StructScan(result); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return result, nil
}

// GetL1CrossMsgsByAddress returns all layer1 cross messages under given address
// Warning: return empty slice if no data found
func (l *l1CrossMsgOrm) GetL1CrossMsgsByAddress(sender common.Address) ([]*CrossMsg, error) {
	para := sender.String()
	var results []*CrossMsg
	rows, err := l.db.Queryx(`SELECT * FROM cross_message WHERE sender = $1 AND msg_type = 1 AND NOT is_deleted;`, para, Layer1Msg)

	for rows.Next() {
		msg := &CrossMsg{}
		if err = rows.StructScan(msg); err != nil {
			break
		}
		results = append(results, msg)
	}
	if len(results) == 0 && errors.Is(err, sql.ErrNoRows) {
	} else if err != nil {
		return nil, err
	}
	return results, nil
}

func (l *l1CrossMsgOrm) BatchInsertL1CrossMsgDBTx(dbTx *sqlx.Tx, messages []*CrossMsg) error {
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
			"amount":       msg.Amount,
			"asset":        msg.Asset,
			"layer1_hash":  msg.Layer1Hash,
			"layer1_token": msg.Layer1Token,
			"layer2_token": msg.Layer2Token,
			"token_id":     msg.TokenID,
			"msg_type":     Layer1Msg,
		}

		_, err := dbTx.NamedExec(`insert into cross_message(height, sender, target, asset, layer1_hash, layer1_token, layer2_token, token_id, amount, msg_type) select :height, :sender, :target, :asset, :layer1_hash, :layer1_token, :layer2_token, :token_id, :amount, :msg_type WHERE NOT EXISTS (SELECT 1 FROM cross_message WHERE layer1_hash = :layer1_hash);`, messageMaps[i])
		if err != nil {
			log.Error("BatchInsertL1CrossMsgDBTx: failed to insert l1 cross msgs", "l1hashes", msg.Layer1Hash, "heights", msg.Height, "err", err)
			break
		}
	}
	return err
}

// UpdateL1CrossMsgHashDBTx update l1 cross msg hash in db, no need to check msg_type since layer1_hash wont be empty if its layer1 msg
func (l *l1CrossMsgOrm) UpdateL1CrossMsgHashDBTx(ctx context.Context, dbTx *sqlx.Tx, l1Hash, msgHash common.Hash) error {
	if _, err := dbTx.ExecContext(ctx, l.db.Rebind("update public.cross_message set msg_hash = ? where layer1_hash = ? AND NOT is_deleted;"), msgHash.String(), l1Hash.String()); err != nil {
		return err
	}
	return nil

}

func (l *l1CrossMsgOrm) UpdateL1CrossMsgHash(ctx context.Context, l1Hash, msgHash common.Hash) error {
	if _, err := l.db.ExecContext(ctx, l.db.Rebind("update public.l1_cross_message set msg_hash = ? where layer1_hash = ? AND NOT is_deleted;"), msgHash.String(), l1Hash.String()); err != nil {
		return err
	}
	return nil

}

func (l *l1CrossMsgOrm) GetLatestL1ProcessedHeight() (int64, error) {
	row := l.db.QueryRowx(`SELECT MAX(height) FROM cross_message WHERE msg_type = $1 AND NOT is_deleted;`, Layer1Msg)
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

func (l *l1CrossMsgOrm) DeleteL1CrossMsgAfterHeightDBTx(dbTx *sqlx.Tx, height int64) error {
	if _, err := l.db.Exec(`UPDATE cross_message SET is_deleted = true WHERE height > $1 AND msg_type < $2;`, height, Layer1Msg); err != nil {
		return err
	}
	return nil
}
