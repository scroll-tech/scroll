package orm

import (
	"database/sql/driver"
	"fmt"
	"math/big"
)

// BigInt keep `Big.Int`'s feature and also can include sqlx's `Scanner` and `Value` interface.
type BigInt struct {
	big.Int
}

// NewInt allocates and returns a new BigInt set to x.
func NewInt(n int64) *BigInt {
	return &BigInt{
		Int: *(big.NewInt(n)),
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
	}

	switch t := value.(type) {
	case int64:
		b.SetInt64(value.(int64))
	case []uint8:
		_, ok := b.SetString(string(value.([]uint8)), 10)
		if !ok {
			return fmt.Errorf("failed to load value to []uint8: %v", value)
		}
	default:
		return fmt.Errorf("Could not scan type %T into BigInt", t)
	}

	return nil
}
