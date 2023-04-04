package types

import (
	"database/sql"
	"github.com/scroll-tech/go-ethereum/common"
	"math/big"
)

// TxType scroll tx type (l1_message_tx, l1_gasOracle_tx, l2_message_tx, l2_gasOracle_tx, l2_rollupCommit_tx, l2_rollupFinalize_tx)
type TxType int

const (
	// UndefinedTx undefined scroll tx type
	UndefinedTx TxType = iota
	// L1toL2MessageTx is sent by l1 relayer but to L2
	L1toL2MessageTx
	// L1toL2GasOracleTx  is sent by l1 relayer but to L2
	L1toL2GasOracleTx
	// L2toL1MessageTx  is sent by l2 relayer but to L1
	L2toL1MessageTx
	// L2toL1GasOracleTx  is sent by l2 relayer but to L1
	L2toL1GasOracleTx
	// RollUpCommitTx  is sent to L2
	RollUpCommitTx
	// RollupFinalizeTx  is sent to L2
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
}

// GetTxHash returns `common.Hash` type of tx hash.
func (s *ScrollTx) GetTxHash() common.Hash {
	if s.TxHash.Valid {
		return common.HexToHash(s.TxHash.String)
	}
	return common.Hash{}
}

// GetSender returns `common.Address` type of sender address.
func (s *ScrollTx) GetSender() common.Address {
	if s.Sender.Valid {
		return common.HexToAddress(s.Sender.String)
	}
	return common.Address{}
}

// GetNonce returns `uint64` type of nonce value.
func (s *ScrollTx) GetNonce() uint64 {
	if s.Nonce.Valid {
		return uint64(s.Nonce.Int64)
	}
	return 0
}

// GetTarget returns `*common.Address`, if target is empty returns null.
func (s *ScrollTx) GetTarget() *common.Address {
	if s.Target.Valid {
		target := common.HexToAddress(s.Target.String)
		return &target
	}
	return nil
}

// GetValue returns `*big.Int` type of value.
func (s *ScrollTx) GetValue() *big.Int {
	if s.Value.Valid {
		if value, ok := big.NewInt(0).SetString(s.Value.String, 16); ok {
			return value
		}
	}
	return nil
}
