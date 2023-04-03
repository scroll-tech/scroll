package orm

import (
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/scroll-tech/go-ethereum/core/types"

	stypes "scroll-tech/common/types"
)

type txOrm struct {
	db *sqlx.DB
}

var _ TxOrm = (*txOrm)(nil)

// NewTxOrm create an TxOrm instance.
func NewTxOrm(db *sqlx.DB) TxOrm {
	return &txOrm{db: db}
}

// SaveTx stores tx message into db.
func (t *txOrm) SaveTx(id, sender string, tx *types.Transaction) error {
	if tx == nil {
		return nil
	}
	var target string
	if tx.To() != nil {
		target = tx.To().String()
	}
	_, err := t.db.Exec(
		t.db.Rebind("INSERT INTO transaction (id, tx_hash, sender, nonce, target, value, data) VALUES (?, ?, ?, ?, ?, ?, ?);"),
		id,
		tx.Hash().String(),
		sender,
		tx.Nonce(),
		target,
		tx.Value().String(),
		tx.Data(),
	)
	return err
}

// DeleteTxDataById remove data content by hash.
func (t *txOrm) DeleteTxDataById(id string) error {
	db := t.db
	_, err := db.Exec(db.Rebind("UPDATE transaction SET data = '' WHERE hash = ?;"), id)
	return err
}

// GetTxById returns tx message by message hash.
func (t *txOrm) GetTxById(id string) (*stypes.TxMessage, error) {
	db := t.db
	row := db.QueryRowx(db.Rebind("SELECT id, tx_hash, sender, nonce, target, value, data FROM transaction WHERE id = ?"), id)
	txMsg := &stypes.TxMessage{}
	if err := row.StructScan(txMsg); err != nil {
		return nil, err
	}
	return txMsg, nil
}

// GetL1TxMessages gets tx messages by transaction right join l1_message.
// sql i.g:
// select l1.msg_hash as hash, tx.tx_hash, tx.sender, tx.nonce, tx.target, tx.value, tx.data
// from transaction as tx
// right join (select msg_hash
//
//	from l1_message
//	where 1 = 1 AND status = :status AND queue_index > 0
//	ORDER BY queue_index ASC
//	LIMIT 10) as l1 on tx.hash = l1.msg_hash;
func (t *txOrm) GetL1TxMessages(fields map[string]interface{}, args ...string) ([]*stypes.TxMessage, error) {
	query := "select msg_hash from l1_message where 1 = 1"
	for key := range fields {
		query = query + fmt.Sprintf(" AND %s = :%s", key, key)
	}
	query = strings.Join(append([]string{query}, args...), " ")
	query = fmt.Sprintf("select l1.msg_hash as id, tx.tx_hash, tx.sender, tx.nonce, tx.target, tx.value, tx.data from transaction as tx right join (%s) as l1 on tx.id = l1.msg_hash;", query)

	db := t.db
	rows, err := db.NamedQuery(db.Rebind(query), fields)
	if err != nil {
		return nil, err
	}

	var txMsgs []*stypes.TxMessage
	for rows.Next() {
		msg := &stypes.TxMessage{}
		if err = rows.StructScan(msg); err != nil {
			return nil, err
		}
		txMsgs = append(txMsgs, msg)
	}
	return txMsgs, nil
}

// GetL2TxMessages gets tx messages by transaction right join l2_message.
func (t *txOrm) GetL2TxMessages(fields map[string]interface{}, args ...string) ([]*stypes.TxMessage, error) {
	query := "select msg_hash from l2_message where 1 = 1"
	for key := range fields {
		query = query + fmt.Sprintf(" AND %s = :%s", key, key)
	}
	query = strings.Join(append([]string{query}, args...), " ")
	query = fmt.Sprintf("select l2.msg_hash as id, tx.tx_hash, tx.sender, tx.nonce, tx.target, tx.value, tx.data from transaction as tx right join (%s) as l2 on tx.id = l2.msg_hash;", query)

	db := t.db
	rows, err := db.NamedQuery(db.Rebind(query), fields)
	if err != nil {
		return nil, err
	}

	var txMsgs []*stypes.TxMessage
	for rows.Next() {
		msg := &stypes.TxMessage{}
		if err = rows.StructScan(msg); err != nil {
			return nil, err
		}
		txMsgs = append(txMsgs, msg)
	}
	return txMsgs, nil
}

// GetBlockBatchTxMessages gets tx messages by transaction right join block_batch.
func (t *txOrm) GetBlockBatchTxMessages(fields map[string]interface{}, args ...string) ([]*stypes.TxMessage, error) {
	query := "select hash from block_batch where 1 = 1"
	for key := range fields {
		query = query + fmt.Sprintf(" AND %s = :%s", key, key)
	}
	query = strings.Join(append([]string{query}, args...), " ")
	query = fmt.Sprintf("select bt.hash as id, tx.tx_hash, tx.sender, tx.nonce, tx.target, tx.value, tx.data from transaction as tx right join (%s) as bt on tx.id = bt.hash;", query)

	db := t.db
	rows, err := db.NamedQuery(db.Rebind(query), fields)
	if err != nil {
		return nil, err
	}

	var txMsgs []*stypes.TxMessage
	for rows.Next() {
		msg := &stypes.TxMessage{}
		if err = rows.StructScan(msg); err != nil {
			return nil, err
		}
		txMsgs = append(txMsgs, msg)
	}
	return txMsgs, nil
}
