package types

import "database/sql"

// ScrollTxType scroll tx type (l1_message_tx, l1_gasOracle_tx, l2_message_tx, l2_gasOracle_tx, l2_rollupCommit_tx, l2_rollupFinalize_tx)
type ScrollTxType int

const (
	// UndefinedTx undefined scroll tx type
	UndefinedTx ScrollTxType = iota
	// L1toL2MessageTx is sent by l1 relayer but to L2
	L1toL2MessageTx
	// L1toL2GasOracleTx  is sent by l1 relayer but to L2
	L1toL2GasOracleTx
	// L2toL1MessageTx  is sent by l2 relayer but to L1
	L2toL1MessageTx
	// L2toL1GasOracleTx  is sent by l2 relayer but to L1
	L2toL1GasOracleTx
	// RollUpCommitTx  is sent to L1
	RollUpCommitTx
	// RollupFinalizeTx  is sent to L1
	RollupFinalizeTx
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
	Note   sql.NullString `json:"note" db:"note"`
}
