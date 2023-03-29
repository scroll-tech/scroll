package types

import (
	"github.com/scroll-tech/go-ethereum/common"
	"math/big"
)

type TxMessage struct {
	Hash     common.Hash     `json:"hash" db:"hash"`
	TxHash   common.Hash     `json:"tx_hash" db:"tx_hash"`
	Sender   common.Address  `json:"sender" db:"sender"`
	Nonce    uint64          `json:"nonce" db:"nonce"`
	Target   *common.Address `json:"target" db:"target"`
	Gas      *big.Int        `json:"gas" db:"gas"`
	GasLimit uint64          `json:"gas_limit" db:"gas_limit"`
	Value    *big.Int        `json:"value" db:"value"`
	Data     string          `json:"data" db:"data"`
}
