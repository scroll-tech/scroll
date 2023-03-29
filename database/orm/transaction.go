package orm

import (
	"github.com/jmoiron/sqlx"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
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
func (t *txOrm) SaveTx(hash, sender string, tx *types.Transaction) error {
	if tx == nil {
		return nil
	}
	var target string
	if tx.To() != nil {
		target = tx.To().String()
	}
	_, err := t.db.Exec(
		t.db.Rebind(t.db.Rebind("INSERT INTO transaction (hash, tx_hash, sender, nonce, target, gas, gas_limit, value, data) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);")),
		hash,
		tx.Hash().String(),
		sender,
		tx.Nonce(),
		target,
		tx.GasPrice().Uint64(),
		tx.Gas(),
		hexutil.Encode(tx.Data()),
	)
	return err
}

// GetTxByHash returns tx message by message hash.
func (t *txOrm) GetTxByHash(hash string) (*stypes.TxMessage, error) {
	db := t.db
	row := db.QueryRowx(db.Rebind("SELECT * FROM transaction WHERE hash = ?"), hash)
	txMsg := &stypes.TxMessage{}
	return txMsg, row.Scan(&txMsg)
}
