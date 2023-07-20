package orm

import (
	"context"
	"database/sql"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/jmoiron/sqlx"
)

// L2SentMsg defines the struct for l2_sent_msg table record
type L2SentMsg struct {
	ID             uint64     `json:"id" db:"id"`
	OriginalSender string     `json:"original_sender" db:"original_sender"`
	TxHash         string     `json:"tx_hash" db:"tx_hash"`
	MsgHash        string     `json:"msg_hash" db:"msg_hash"`
	Sender         string     `json:"sender" db:"sender"`
	Target         string     `json:"target" db:"target"`
	Value          string     `json:"value" db:"value"`
	Height         uint64     `json:"height" db:"height"`
	Nonce          uint64     `json:"nonce" db:"nonce"`
	BatchIndex     uint64     `json:"batch_index" db:"batch_index"`
	MsgProof       string     `json:"msg_proof" db:"msg_proof"`
	MsgData        string     `json:"msg_data" db:"msg_data"`
	CreatedAt      *time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at" db:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at" db:"deleted_at"`
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
	row := l.db.QueryRowx(`SELECT * FROM l2_sent_msg WHERE msg_hash = $1 AND deleted_at IS NULL;`, msgHash)
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
			"original_sender": msg.OriginalSender,
			"tx_hash":         msg.TxHash,
			"sender":          msg.Sender,
			"target":          msg.Target,
			"value":           msg.Value,
			"msg_hash":        msg.MsgHash,
			"height":          msg.Height,
			"nonce":           msg.Nonce,
			"batch_index":     msg.BatchIndex,
			"msg_proof":       msg.MsgProof,
			"msg_data":        msg.MsgData,
		}
	}
	_, err = dbTx.NamedExec(`insert into l2_sent_msg(original_sender, tx_hash, sender, target, value, msg_hash, height, nonce, batch_index, msg_proof, msg_data) values(:original_sender, :tx_hash, :sender, :target, :value, :msg_hash, :height, :nonce, :batch_index, :msg_proof, :msg_data);`, messageMaps)
	if err != nil {
		log.Error("BatchInsertL2SentMsgDBTx: failed to insert l2 sent msgs", "err", err)
		return err
	}
	return err
}

func (l *l2SentMsgOrm) GetLatestSentMsgHeightOnL2() (int64, error) {
	row := l.db.QueryRow(`SELECT height FROM l2_sent_msg WHERE deleted_at IS NULL ORDER BY nonce DESC LIMIT 1;`)
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

func (l *l2SentMsgOrm) UpdateL2MessageProofInDBTx(ctx context.Context, dbTx *sqlx.Tx, msgHash string, proof string, batchIndex uint64) error {
	if _, err := dbTx.ExecContext(ctx, l.db.Rebind("update l2_sent_msg set msg_proof = ?, batch_index = ? where msg_hash = ? AND deleted_at IS NULL;"), proof, batchIndex, msgHash); err != nil {
		return err
	}
	return nil
}

func (l *l2SentMsgOrm) GetLatestL2SentMsgBatchIndex() (int64, error) {
	row := l.db.QueryRow(`SELECT batch_index FROM l2_sent_msg WHERE batch_index != 0 AND deleted_at IS NULL ORDER BY batch_index DESC LIMIT 1;`)
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
	rows, err := l.db.Queryx(`SELECT * FROM l2_sent_msg WHERE height >= $1 AND height <= $2 AND deleted_at IS NULL ORDER BY nonce ASC;`, startHeight, endHeight)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err = rows.Close(); err != nil {
			log.Error("failed to close rows", "err", err)
		}
	}()
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
	row := l.db.QueryRowx(`SELECT * FROM l2_sent_msg WHERE nonce = $1 AND deleted_at IS NULL;`, nonce)
	err := row.StructScan(result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (l *l2SentMsgOrm) GetLatestL2SentMsgLEHeight(endBlockNumber uint64) (*L2SentMsg, error) {
	result := &L2SentMsg{}
	row := l.db.QueryRowx(`select * from l2_sent_msg where height <= $1 AND deleted_at IS NULL order by nonce desc limit 1`, endBlockNumber)
	err := row.StructScan(result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (l *l2SentMsgOrm) DeleteL2SentMsgAfterHeightDBTx(dbTx *sqlx.Tx, height int64) error {
	_, err := dbTx.Exec(`UPDATE l2_sent_msg SET deleted_at = current_timestamp WHERE height > $1;`, height)
	return err
}

func (l *l2SentMsgOrm) GetClaimableL2SentMsgByAddressWithOffset(address string, offset int64, limit int64) ([]*L2SentMsg, error) {
	var results []*L2SentMsg
	rows, err := l.db.Queryx(`SELECT * FROM l2_sent_msg WHERE id NOT IN (SELECT l2_sent_msg.id FROM l2_sent_msg INNER JOIN relayed_msg ON l2_sent_msg.msg_hash = relayed_msg.msg_hash WHERE l2_sent_msg.deleted_at IS NULL AND relayed_msg.deleted_at IS NULL) AND (original_sender=$1 OR sender = $1) ORDER BY id DESC LIMIT $2 OFFSET $3;`, address, limit, offset)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err = rows.Close(); err != nil {
			log.Error("failed to close rows", "err", err)
		}
	}()
	for rows.Next() {
		msg := &L2SentMsg{}
		if err = rows.StructScan(msg); err != nil {
			break
		}
		results = append(results, msg)
	}
	return results, err
}

func (l *l2SentMsgOrm) GetClaimableL2SentMsgByAddressTotalNum(address string) (uint64, error) {
	var count uint64
	row := l.db.QueryRowx(`SELECT COUNT(*) FROM l2_sent_msg WHERE id NOT IN (SELECT l2_sent_msg.id FROM l2_sent_msg INNER JOIN relayed_msg ON l2_sent_msg.msg_hash = relayed_msg.msg_hash WHERE l2_sent_msg.deleted_at IS NULL AND relayed_msg.deleted_at IS NULL) AND (original_sender=$1 OR sender = $1);`, address)
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}
