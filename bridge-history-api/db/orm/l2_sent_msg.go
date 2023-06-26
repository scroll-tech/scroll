package orm

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/jmoiron/sqlx"
)

type L2SentMsg struct {
	ID         uint64     `json:"id" db:"id"`
	MsgHash    string     `json:"msg_hash" db:"msg_hash"`
	Sender     string     `json:"sender" db:"sender"`
	Target     string     `json:"target" db:"target"`
	Value      string     `json:"value" db:"value"`
	Height     uint64     `json:"height" db:"height"`
	Nonce      uint64     `json:"nonce" db:"nonce"`
	BatchIndex uint64     `json:"batch_index" db:"batch_index"`
	MsgProof   string     `json:"msg_proof" db:"msg_proof"`
	MsgData    string     `json:"msg_data" db:"msg_data"`
	IsDeleted  bool       `json:"is_deleted" db:"is_deleted"`
	CreatedAt  *time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  *time.Time `json:"updated_at" db:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at" db:"deleted_at"`
}

type l2SentMsgOrm struct {
	db *sqlx.DB
}

// NewL2SentMsgOrm create an NewRollupBatchOrm instance
func NewL2SentMsgOrm(db *sqlx.DB) L2SentMsgOrm {
	return &l2SentMsgOrm{db: db}
}

func (l *l2SentMsgOrm) GetL2SentMsgByHash(msgHash string) (*L2SentMsg, error) {
	result := &L2SentMsg{}
	row := l.db.QueryRowx(`SELECT * FROM l2_sent_msg WHERE msg_hash = $1 AND NOT is_deleted;`, msgHash)
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
			"sender":      msg.Sender,
			"target":      msg.Target,
			"value":       msg.Value,
			"msg_hash":    msg.MsgHash,
			"height":      msg.Height,
			"nonce":       msg.Nonce,
			"batch_index": msg.BatchIndex,
			"msg_proof":   msg.MsgProof,
			"msg_data":    msg.MsgData,
		}
		var exists bool
		err = dbTx.QueryRow(`SELECT EXISTS(SELECT 1 FROM l2_sent_msg WHERE (msg_hash = $1 OR nonce = $2) AND NOT is_deleted)`, msg.MsgHash, msg.Nonce).Scan(&exists)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("BatchInsertL2SentMsgDBTx: l2 sent msg_hash %v already exists at height %v", msg.MsgHash, msg.Height)
		}
	}
	_, err = dbTx.NamedExec(`insert into l2_sent_msg(sender, target, value, msg_hash, height, nonce, batch_index, msg_proof, msg_data) values(:sender, :target, :value, :msg_hash, :height, :nonce, :batch_index, :msg_proof, :msg_data);`, messageMaps)
	if err != nil {
		log.Error("BatchInsertL2SentMsgDBTx: failed to insert l2 sent msgs", "msg_Hash", "err", err)
		return err
	}
	return err
}

func (l *l2SentMsgOrm) GetLatestSentMsgHeightOnL2() (int64, error) {
	row := l.db.QueryRow(`SELECT height FROM l2_sent_msg WHERE NOT is_deleted ORDER BY nonce DESC LIMIT 1;`)
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

func (l *l2SentMsgOrm) UpdateL2MessageProofInDBTx(ctx context.Context, dbTx *sqlx.Tx, msgHash string, proof string, batch_index uint64) error {
	if _, err := dbTx.ExecContext(ctx, l.db.Rebind("update l2_sent_msg set msg_proof = ?, batch_index = ? where msg_hash = ? AND NOT is_deleted;"), proof, batch_index, msgHash); err != nil {
		return err
	}
	return nil
}

func (l *l2SentMsgOrm) GetLatestL2SentMsgBatchIndex() (int64, error) {
	row := l.db.QueryRow(`SELECT batch_index FROM l2_sent_msg WHERE msg_proof != '' AND NOT is_deleted ORDER BY batch_index DESC LIMIT 1;`)
	var result sql.NullInt64
	if err := row.Scan(&result); err != nil {
		if err == sql.ErrNoRows || !result.Valid {
			return -1, nil
		}
		return -1, err
	}
	if result.Valid {
		return result.Int64, nil
	}
	return -1, nil
}

func (l *l2SentMsgOrm) GetL2SentMsgMsgHashByHeightRange(startHeight, endHeight uint64) ([]*L2SentMsg, error) {
	var results []*L2SentMsg
	rows, err := l.db.Queryx(`SELECT * FROM l2_sent_msg WHERE height >= $1 AND height <= $2 AND NOT is_deleted ORDER BY nonce ASC;`, startHeight, endHeight)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		msg := &L2SentMsg{}
		if err = rows.StructScan(msg); err != nil {
			break
		}
		results = append(results, msg)
	}
	return results, err
}

func (l *l2SentMsgOrm) GetL2SentMessageByNonce(nonce uint64) (*L2SentMsg, error) {
	result := &L2SentMsg{}
	row := l.db.QueryRowx(`SELECT * FROM l2_sent_msg WHERE nonce = $1 AND NOT is_deleted;`, nonce)
	err := row.StructScan(result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (l *l2SentMsgOrm) GetLatestL2SentMsgLEHeight(endBlockNumber uint64) (*L2SentMsg, error) {
	result := &L2SentMsg{}
	row := l.db.QueryRowx(`select * from l2_sent_msg where height <= $1 AND NOT is_deleted order by nonce desc limit 1`, endBlockNumber)
	err := row.StructScan(result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (l *l2SentMsgOrm) DeleteL2SentMsgAfterHeightDBTx(dbTx *sqlx.Tx, height int64) error {
	_, err := dbTx.Exec(`UPDATE l2_sent_msg SET is_deleted = true WHERE height > $1;`, height)
	return err
}
