package types

import "database/sql"

// TxType scroll tx type (l1_message_tx, l1_gasOracle_tx, l2_message_tx, l2_gasOracle_tx, l2_rollupCommit_tx, l2_rollupFinalize_tx)
type TxType int

const (
	// UndefinedTx undefined scroll tx type
	UndefinedTx TxType = iota
	// L1MessageTx is sent by l1 relayer but to L2
	L1MessageTx
	// L1GasOracleTx  is sent by l1 relayer but to L2
	L1GasOracleTx
	// L2MessageTx  is sent by l2 relayer but to L1
	L2MessageTx
	// L2GasOracleTx  is sent by l2 relayer but to L1
	L2GasOracleTx
	// L2RollUpCommitTx  is sent to L2
	L2RollUpCommitTx
	// L2RollupFinalizeTx  is sent to L2
	L2RollupFinalizeTx
)

// ScrollTx Contains tx message, hash is the index.
type ScrollTx struct {
	ID     string         `json:"id" db:"id"`
	TxHash sql.NullString `json:"tx_hash" db:"tx_hash"`
	Sender sql.NullString `json:"sender" db:"sender"`
	Nonce  sql.NullInt64  `json:"nonce" db:"nonce"`
	Target sql.NullString `json:"target" db:"target"`
	Value  sql.NullString `json:"value" db:"value"`
	Data   []byte         `json:"data" db:"data"`
}
