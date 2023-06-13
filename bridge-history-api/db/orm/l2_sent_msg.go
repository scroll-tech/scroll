package orm

import (
	"context"
	"database/sql"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/jmoiron/sqlx"
)

type l2SentMsgOrm struct {
	db *sqlx.DB
}

// NewBridgeBatchOrm create an NewBridgeBatchOrm instance
func NewL2SentMsgOrm(db *sqlx.DB) L2SentMsgOrm {
	return &l2SentMsgOrm{db: db}
}

func (l *l2SentMsgOrm) GetL2SentMsgByHash(l2Hash string) (*L2SentMsg, error) {
	result := &L2SentMsg{}
	row := l.db.QueryRowx(`SELECT * FROM l2_sent_msg WHERE layer2_hash = $1 AND NOT is_deleted;`, l2Hash)
	if err := row.StructScan(result); err != nil {
		return nil, err
	}
	return result, nil
}

func (l *l2SentMsgOrm) BatchInsertL2SentMsgDBTx(dbTx *sqlx.Tx, messages []*L2SentMsg) error {
	if len(messages) == 0 {
		return nil
	}
	var err error
	messageMaps := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		messageMaps[i] = map[string]interface{}{
			"msg_hash":         msg.MsgHash,
			"height":           msg.Height,
			"nonce":            msg.Nonce,
			"finalized_height": msg.FinalizedHeight,
			"layer1_hash":      msg.Layer1Hash,
			"batch_index":      msg.BatchIndex,
			"msg_proof":        msg.MsgProof,
			"msg_data":         msg.MsgData,
		}

		_, err = dbTx.NamedExec(`insert into l2_sent_msg(msg_hash, height, nonce, finalized_height, layer1_hash, batch_index, msg_proof, msg_data) values(:msg_hash, :height, :nonce, :layer1_hash, :finalized_height, :batch_index, :msg_proof, :msg_data);`, messageMaps[i])
		if err != nil && !strings.Contains(err.Error(), "pq: duplicate key value violates unique constraint \"l2_sent_msg_hash_uindex") {
			log.Error("BatchInsertL2SentMsgDBTx: failed to insert l2 sent msgs", "msg_Hash", msg.MsgHash, "height", msg.Height, "err", err)
			break
		}
	}
	return err
}

func (l *l2SentMsgOrm) GetLatestSentMsgHeightOnL2() (int64, error) {
	row := l.db.QueryRow(`SELECT height FROM l2_sent_msg WHERE layer2_hash != '' AND NOT is_deleted ORDER BY nonce DESC LIMIT 1;`)
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

func (l *l2SentMsgOrm) UpdateL2SentMsgL1HashDBTx(ctx context.Context, dbTx *sqlx.Tx, l1Hash, msgHash common.Hash) error {
	if _, err := dbTx.ExecContext(ctx, l.db.Rebind("update public.l2_sent_msg set layer1_hash = ? where msg_hash = ? AMD NOT is_deleted;"), l1Hash.String(), msgHash.String()); err != nil {
		return err
	}
	return nil
}

func (l *l2SentMsgOrm) UpdateL2MessageProofInDbTx(ctx context.Context, dbTx *sqlx.Tx, msgHash string, proof string, batch_index uint64) error {
	if _, err := dbTx.ExecContext(ctx, l.db.Rebind("update public.l2_sent_msg set msg_proof = ?, batch_index = ? where msg_hash = ? AND NOT is_deleted;"), proof, batch_index, msgHash); err != nil {
		return err
	}
	return nil
}

func (l *l2SentMsgOrm) GetLatestL2SentMsgBactchIndex() (uint64, error) {
	row := l.db.QueryRow(`SELECT batch_index FROM l2_sent_msg WHERE msg_proof != null AND batch_index != null AND is_deleted ORDER BY batch_index DESC LIMIT 1;`)
	var result sql.NullInt64
	if err := row.Scan(&result); err != nil {
		if err == sql.ErrNoRows || !result.Valid {
			return 0, nil
		}
		return 0, err
	}
	if result.Valid {
		return uint64(result.Int64), nil
	}
	return 0, nil
}

func (l *l2SentMsgOrm) GetL2SentMsgMsgHashByHeightRange(startHeight, endHeight uint64) ([]*L2SentMsg, error) {
	var result []*L2SentMsg
	err := l.db.Select(&result, `SELECT * FROM l2_sent_msg WHERE height >= $1 AND height <= $2 AND NOT is_deleted ORDERED BY nonce ASC;`, startHeight, endHeight)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (l *l2SentMsgOrm) GetL2SentMessageByNonce(nonce uint64) (*L2SentMsg, error) {
	var result *L2SentMsg
	err := l.db.Select(&result, `SELECT * FROM l2_sent_msg WHERE nonce = $1 AND NOT is_deleted;`, nonce)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (l *l2SentMsgOrm) GetLastL2MessageNonceLEHeight(endBlockNumber uint64) (sql.NullInt64, error) {
	row := l.db.QueryRow(`SELECT MAX(nonce) FROM l2_message WHERE height <= $1;`, endBlockNumber)
	var nonce sql.NullInt64
	err := row.Scan(&nonce)
	return nonce, err
}

func (l *l2SentMsgOrm) DeleteL2SentMsgAfterHeightDBTx(dbTx *sqlx.Tx, height int64) error {
	_, err := dbTx.Exec(`UPDATE l2_sent_msg SET is_deleted = true WHERE height > $1 AND layer1_hash != '';`, height)
	return err
}

func (l *l2SentMsgOrm) ResetL2SentMsgL1HashAfterHeightDBTx(dbTx *sqlx.Tx, height int64) error {
	_, err := dbTx.Exec(`UPDATE l2_sent_msg SET layer1_hash = '' WHERE height > $1  AND NOT is_deleted;`, height)
	return err
}
