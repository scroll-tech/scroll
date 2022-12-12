package bigint

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"math/big"
)

// BigInt is a wrapper around `Big.Int` that implements sqlx's `Scanner` and `Value` interfaces.
type BigInt struct {
	big.Int
}

// NewInt allocates and returns a new BigInt set to n.
func NewInt(n int64) *BigInt {
	return &BigInt{
		Int: *(big.NewInt(n)),
	}
}

// NewUInt allocates and returns a new BigInt set to n of type uint64.
func NewUInt(n uint64) *BigInt {
	return &BigInt{
		Int: *(big.NewInt(0).SetUint64(n)),
	}
}

// NewBigInt allocates and returns a new BigInt from big.Int
func NewBigInt(n *big.Int) *BigInt {
	return &BigInt{
		Int: *(big.NewInt(0).Set(n)),
	}
}

// SetBigInt set value by big.int field.
func (b *BigInt) SetBigInt(n *big.Int) {
	b.Set(n)
}

// BigInt return origin big.int type.
func (b *BigInt) BigInt() *big.Int {
	return &b.Int
}

// Value implements the driver.Valuer interface
func (b *BigInt) Value() (driver.Value, error) {
	if b != nil {
		return b.String(), nil
	}
	return nil, nil
}

// Scan implements the sql.Scanner interface
func (b *BigInt) Scan(value interface{}) error {
	if value == nil {
		b = nil
		return errors.New("could not scan nil value into BigInt")
	}

	switch t := value.(type) {
	case int64:
		b.SetInt64(value.(int64))
	case []uint8:
		_, ok := b.SetString(string(value.([]uint8)), 10)
		if !ok {
			return fmt.Errorf("failed to load BigInt value from []uint8: %v", value)
		}
	default:
		return fmt.Errorf("could not scan type %T into BigInt", t)
	}

	return nil
}
