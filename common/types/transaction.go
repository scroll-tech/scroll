package types

import "database/sql"

// TxType scroll tx type (l1_message_tx, l1_gasOracle_tx, l2_message_tx, l2_gasOracle_tx, l2_rollupCommit_tx, l2_rollupFinalize_tx)
type TxType int

const (
	// UndefinedTx undefined scroll tx type
	UndefinedTx TxType = iota
	// L1MessageTx l1 relayer message tx
	L1MessageTx
	// L1GasOracleTx l1 relayer gas oracle tx
	L1GasOracleTx
	// L2MessageTx l2 relayer message tx
	L2MessageTx
	// L2GasOracleTx l2 gas oracle tx
	L2GasOracleTx
	// L2RollUpCommitTx l2 rollup commit tx
	L2RollUpCommitTx
	// L2RollupFinalizeTx l2 rollup finalize tx
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
