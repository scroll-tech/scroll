package types

import (
	"bytes"
	"database/sql/driver"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/rlp"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
)

// RowConsumptionDb is simply RowConsumption data type, but with Scanner/Valuer implementations
type RowConsumptionDb gethTypes.RowConsumption

// Scan scan value into RowConsumptionDb, implements sql.Scanner interface
func (rc *RowConsumptionDb) Scan(value interface{}) error {
	data, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal RowConsumptionDb value:", value))
	}
	var result RowConsumptionDb
	if err := rlp.Decode(bytes.NewReader(data), result); err != nil {
		return errors.New(fmt.Sprint("Failed to unmarshal RowConsumptionDb value:", value))
	}
	*rc = result
	return nil
}

// Value return RowConsumptionDb value, implement driver.Valuer interface
func (rc RowConsumptionDb) Value() (driver.Value, error) {
	return rlp.EncodeToBytes(rc)
}
