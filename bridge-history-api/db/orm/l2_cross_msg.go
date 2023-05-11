package orm

import (
	"context"
	"database/sql"
	"errors"

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

func (l *l2CrossMsgOrm) GetL2CrossMsgByHash(l2Hash common.Hash) (*L2CrossMsg, error) {
	result := &L2CrossMsg{}
	row := l.db.QueryRowx(`SELECT * FROM l2_cross_message WHERE layer2_hash = $1;`, l2Hash.String())
	if err := row.StructScan(result); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return result, nil
}

func (l *l2CrossMsgOrm) GetL2CrossMsgsByAddressWithOffset(sender common.Address, offset int64, limit int64) ([]*L2CrossMsg, error) {
	para := sender.String()
	var results []*L2CrossMsg
	rows, err := l.db.Queryx(`SELECT * FROM l2_cross_message WHERE sender = $1 ORDER BY height DESC LIMIT $2 OFFSET $3;`, para, limit, offset)
	for rows.Next() {
		msg := &L2CrossMsg{}
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

// GetL2CrossMsgsByAddress returns all layer2 cross messages under given address
// Warning: return empty slice if no data found
func (l *l2CrossMsgOrm) GetL2CrossMsgByAddress(sender common.Address) ([]*L2CrossMsg, error) {
	para := sender.String()
	var results []*L2CrossMsg
	rows, err := l.db.Queryx(`SELECT * FROM l2_cross_message WHERE sender = $1;`, para)

	for rows.Next() {
		msg := &L2CrossMsg{}
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
	_, err := dbTx.Exec(`delete from l2_cross_message where height > $1;`, height)
	if err != nil {
		log.Error("DeleteL1CrossMsgAfterHeightDBTx: failed to delete", "height", height, "err", err)
		return err
	}
	return nil
}

func (l *l2CrossMsgOrm) BatchInsertL2CrossMsgDBTx(dbTx *sqlx.Tx, messages []*L2CrossMsg) error {
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
			"layer2_hash":  msg.Layer2Hash,
			"layer1_token": msg.Layer1Token,
			"layer2_token": msg.Layer2Token,
			"token_id":     msg.TokenID,
			"amount":       msg.Amount,
		}

		_, err := dbTx.NamedExec(`insert into l2_cross_message(height, sender, target, asset, layer2_hash, layer1_token, layer2_token, token_id, amount) values(:height, :sender, :target, :asset, :layer2_hash, :layer1_token, :layer2_token, :token_id, :amount);`, messageMaps[i])
		if err != nil {
			log.Error("BatchInsertL2CrossMsgDBTx: failed to insert l2 cross msgs", "layer2hash", msg.Layer2Hash, "heights", msg.Height, "err", err)
			break
		}
	}
	return err
}

func (l *l2CrossMsgOrm) UpdateL2CrossMsgHashDBTx(ctx context.Context, dbTx *sqlx.Tx, l2Hash, msgHash common.Hash) error {
	if _, err := dbTx.ExecContext(ctx, l.db.Rebind("update public.l2_cross_message set msg_hash = ? where layer2_hash = ?;"), msgHash.String(), l2Hash.String()); err != nil {
		return err
	}
	return nil
}

func (l *l2CrossMsgOrm) UpdateL2CrossMsgHash(ctx context.Context, l2Hash, msgHash common.Hash) error {
	if _, err := l.db.ExecContext(ctx, l.db.Rebind("update public.l2_cross_message set msg_hash = ? where layer2_hash = ?;"), msgHash.String(), l2Hash.String()); err != nil {
		return err
	}
	return nil
}

func (l *l2CrossMsgOrm) GetLatestL2ProcessedHeight() (int64, error) {
	row := l.db.QueryRowx(`SELECT MAX(height) FROM l2_cross_message WHERE layer2_hash IS NOT NULL;`)
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
