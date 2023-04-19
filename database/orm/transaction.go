package orm

import (
	"fmt"
	"modernc.org/mathutil"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/scroll-tech/go-ethereum/core/types"

	stypes "scroll-tech/common/types"
)

type scrollTxOrm struct {
	db *sqlx.DB
}

var _ ScrollTxOrm = (*scrollTxOrm)(nil)

// NewScrollTxOrm create an ScrollTxOrm instance.
func NewScrollTxOrm(db *sqlx.DB) ScrollTxOrm {
	return &scrollTxOrm{db: db}
}

// SaveScrollTx stores tx message into db.
func (t *scrollTxOrm) SaveScrollTx(id, sender string, txType stypes.ScrollTxType, tx *types.Transaction, extraData string) error {
	if tx == nil {
		return nil
	}
	var target string
	if tx.To() != nil {
		target = tx.To().String()
	}
	_, err := t.db.Exec(
		t.db.Rebind("INSERT INTO scroll_transaction (id, tx_hash, sender, nonce, target, value, data, note, type) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);"),
		id,
		tx.Hash().String(),
		sender,
		tx.Nonce(),
		target,
		tx.Value().String(),
		tx.Data(),
		extraData,
		txType,
	)
	return err
}

// SetScrollTxConfirmedByID updates confirmed and txHash field and clean data content.
func (t *scrollTxOrm) SetScrollTxConfirmedByID(id string, txHash string) error {
	db := t.db
	_, err := db.Exec(db.Rebind("UPDATE scroll_transaction SET confirmed = true, data = '', note = '', tx_hash = ? WHERE id = ?;"), txHash, id)
	return err
}

// GetScrollTxs get scroll txs by params.
func (t *scrollTxOrm) GetScrollTxs(fields map[string]interface{}, args ...string) ([]*stypes.ScrollTx, error) {
	query := "select id, tx_hash, sender, nonce, target, value, data, note from scroll_transaction where 1 = 1"
	for key := range fields {
		query = query + fmt.Sprintf(" AND %s = :%s", key, key)
	}
	query = strings.Join(append([]string{query}, args...), " ")

	db := t.db
	rows, err := db.NamedQuery(db.Rebind(query), fields)
	if err != nil {
		return nil, err
	}

	var txMsgs []*stypes.ScrollTx
	for rows.Next() {
		txMsg := &stypes.ScrollTx{}
		if err = rows.StructScan(txMsg); err != nil {
			return nil, err
		}
		txMsgs = append(txMsgs, txMsg)
	}
	return txMsgs, nil
}

// GetTxByID returns tx message by message id.
func (t *scrollTxOrm) GetTxByID(id string) (*stypes.ScrollTx, error) {
	db := t.db
	row := db.QueryRowx(db.Rebind("SELECT id, tx_hash, sender, nonce, target, value, data FROM scroll_transaction WHERE id = ?"), id)
	txMsg := &stypes.ScrollTx{}
	if err := row.StructScan(txMsg); err != nil {
		return nil, err
	}
	return txMsg, nil
}

// GetL1TxMessages gets tx messages by scroll_transaction right join l1_message.
// sql i.g:
// select l1.msg_hash as id, tx.tx_hash, tx.sender, tx.nonce, tx.target, tx.value, tx.data
// from scroll_transaction as tx
// right join (select msg_hash
//
//	from l1_message
//	where 1 = 1 AND status = :status AND queue_index > :index
//	ORDER BY queue_index ASC
//	LIMIT 10) as l1 on tx.id = l1.msg_hash;
func (t *scrollTxOrm) GetL1TxMessages(fields map[string]interface{}, args ...string) (uint64, []*stypes.ScrollTx, error) {
	query := "select msg_hash, queue_index from l1_message where 1 = 1"
	for key := range fields {
		query = query + fmt.Sprintf(" AND %s = :%s", key, key)
	}
	query = strings.Join(append([]string{query}, args...), " ")
	query = fmt.Sprintf("select l1.queue_index as index, l1.msg_hash as id, tx.tx_hash, tx.sender, tx.nonce, tx.target, tx.value, tx.data from scroll_transaction as tx right join (%s) as l1 on tx.id = l1.msg_hash;", query)

	db := t.db
	rows, err := db.NamedQuery(db.Rebind(query), fields)
	if err != nil {
		return 0, nil, err
	}

	var (
		index  uint64
		txMsgs []*stypes.ScrollTx
	)
	for rows.Next() {
		warp := struct {
			Index uint64 `db:"index"`
			*stypes.ScrollTx
		}{}
		if err = rows.StructScan(&warp); err != nil {
			return 0, nil, err
		}
		index = mathutil.MaxUint64(index, warp.Index)
		txMsgs = append(txMsgs, warp.ScrollTx)
	}
	return index, txMsgs, nil
}

// GetL2TxMessages gets tx messages by transaction right join l2_message.
func (t *scrollTxOrm) GetL2TxMessages(fields map[string]interface{}, args ...string) (uint64, []*stypes.ScrollTx, error) {
	query := "select msg_hash, nonce from l2_message where 1 = 1"
	for key := range fields {
		query = query + fmt.Sprintf(" AND %s = :%s", key, key)
	}
	query = strings.Join(append([]string{query}, args...), " ")
	query = fmt.Sprintf("select l2.nonce as l2_nonce, l2.msg_hash as id, tx.tx_hash, tx.sender, tx.nonce, tx.target, tx.value, tx.data from scroll_transaction as tx right join (%s) as l2 on tx.id = l2.msg_hash;", query)

	db := t.db
	rows, err := db.NamedQuery(db.Rebind(query), fields)
	if err != nil {
		return 0, nil, err
	}

	var (
		nonce  uint64
		txMsgs []*stypes.ScrollTx
	)
	for rows.Next() {
		warp := struct {
			Nonce uint64 `db:"l2_nonce"`
			*stypes.ScrollTx
		}{}
		if err = rows.StructScan(&warp); err != nil {
			return 0, nil, err
		}
		nonce = mathutil.MaxUint64(nonce, warp.Nonce)
		txMsgs = append(txMsgs, warp.ScrollTx)
	}
	return nonce, txMsgs, nil
}

// GetBlockBatchTxMessages gets tx messages by transaction right join block_batch.
func (t *scrollTxOrm) GetBlockBatchTxMessages(fields map[string]interface{}, args ...string) (uint64, []*stypes.ScrollTx, error) {
	query := "select hash, index from block_batch where 1 = 1"
	for key := range fields {
		query = query + fmt.Sprintf(" AND %s = :%s", key, key)
	}
	query = strings.Join(append([]string{query}, args...), " ")
	query = fmt.Sprintf("select bt.index as index, bt.hash as id, tx.tx_hash, tx.sender, tx.nonce, tx.target, tx.value, tx.data, tx.note from scroll_transaction as tx right join (%s) as bt on tx.id = bt.hash;", query)

	db := t.db
	rows, err := db.NamedQuery(db.Rebind(query), fields)
	if err != nil {
		return 0, nil, err
	}

	var (
		index  uint64
		txMsgs []*stypes.ScrollTx
	)
	for rows.Next() {
		warp := struct {
			Index uint64 `db:"index"`
			*stypes.ScrollTx
		}{}
		if err = rows.StructScan(&warp); err != nil {
			return 0, nil, err
		}
		index = mathutil.MaxUint64(index, warp.Index)
		txMsgs = append(txMsgs, warp.ScrollTx)
	}
	return index, txMsgs, nil
}
