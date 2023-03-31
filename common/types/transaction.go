package types

import "database/sql"

type TxMessage struct {
	Hash   string         `json:"hash" db:"hash"`
	TxHash sql.NullString `json:"tx_hash" db:"tx_hash"`
	Sender sql.NullString `json:"sender" db:"sender"`
	Nonce  sql.NullInt64  `json:"nonce" db:"nonce"`
	Target sql.NullString `json:"target" db:"target"`
	Value  sql.NullString `json:"value" db:"value"`
	Data   []byte         `json:"data" db:"data"`
}
