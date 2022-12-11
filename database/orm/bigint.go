package orm

import (
	"database/sql/driver"
	"fmt"
	"math/big"
)

// BigInt is wrapper of big.Int to enable database read/write
type BigInt big.Int

// Value implements the driver.Valuer interface
func (b *BigInt) Value() (driver.Value, error) {
	if b != nil {
		return (*big.Int)(b).String(), nil
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
		(*big.Int)(b).SetInt64(value.(int64))
	case []uint8:
		_, ok := (*big.Int)(b).SetString(string(value.([]uint8)), 10)
		if !ok {
			return fmt.Errorf("failed to load value to []uint8: %v", value)
		}
	default:
		return fmt.Errorf("Could not scan type %T into BigInt", t)
	}

	return nil
}
