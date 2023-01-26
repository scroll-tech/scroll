package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strconv"

	"github.com/scroll-tech/go-ethereum/core/types"
)

var pattern = regexp.MustCompile(`^number=(\d{1,3})$`)

// ConfirmationType defines the type of confirmation logic used by the watcher or the relayer.
type ConfirmationType int

const (
	// Use the "finalized" Ethereum tag for confirmation.
	Finalized ConfirmationType = iota

	// Safe use the "safe" Ethereum tag for confirmation.
	Safe

	// Wait for a certain number of blocks before considering a block confirmed.
	Number
)

// ConfirmationParams defines the confirmation configuration parameters used by the watcher or the relayer.
type ConfirmationParams struct {
	// Type shows whether we confirm by specific block tags or by block number.
	Type ConfirmationType

	// Number specifies the number of blocks after which a block is considered confirmed.
	// This field can only be used when Type is set to Number.
	Number uint64
}

// UnmarshalJSON implements custom JSON decoding from JSON string to ConfirmationParams.
func (c *ConfirmationParams) UnmarshalJSON(input []byte) error {
	var raw string

	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}

	if raw == "finalized" {
		c.Type = Finalized
		return nil
	}

	if raw == "safe" {
		c.Type = Safe
		return nil
	}

	matches := pattern.FindStringSubmatch(raw)
	if len(matches) != 2 {
		return errors.New("invalid configuration value for 'confirmations'")
	}

	number, err := strconv.Atoi(matches[1])
	if err != nil {
		return errors.New("invalid configuration value for 'confirmations'")
	}

	c.Type = Number
	c.Number = uint64(number)
	return nil
}

// UnmarshalJSON implements custom JSON encoding from ConfirmationParams to JSON string.
func (c *ConfirmationParams) MarshalJSON() ([]byte, error) {
	var raw string

	switch c.Type {
	case Finalized:
		raw = "finalized"

	case Safe:
		raw = "safe"

	case Number:
		raw = fmt.Sprintf("number=%d", c.Number)

	default:
		return nil, errors.New("invalid configuration value for 'confirmations'")
	}

	return json.Marshal(&raw)
}

type ethClient interface {
	BlockNumber(ctx context.Context) (uint64, error)
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
}

// GetLatestConfirmedBlockNumber queries the RPC provider and returns the latest
// confirmed block number according to the provided confirmation parameters.
func GetLatestConfirmedBlockNumber(ctx context.Context, client ethClient, confirmations ConfirmationParams) (uint64, error) {
	switch confirmations.Type {
	case Finalized:
	case Safe:
		// TODO: chain to "finalized" or "safe"
		tag := big.NewInt(0)

		header, err := client.HeaderByNumber(ctx, tag)
		if err != nil {
			return 0, err
		}

		if !header.Number.IsUint64() {
			return 0, errors.New("received invalid block number")
		}

		return header.Number.Uint64(), nil

	case Number:
		number, err := client.BlockNumber(ctx)
		if err != nil {
			return 0, err
		}

		if number >= confirmations.Number {
			return number - confirmations.Number, nil
		}

		return 0, nil

	default:
		return 0, errors.New("invalid confirmation configuration")
	}

	return 0, nil
}
