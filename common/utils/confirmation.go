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

type ConfirmationType int

const (
	Finalized ConfirmationType = iota
	Safe
	Number
)

type ConfirmationParams struct {
	Type   ConfirmationType
	Number uint64 // only used with Number type
}

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

type EthClient interface {
	BlockNumber(ctx context.Context) (uint64, error)
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
}

func GetLatestConfirmedBlockNumber(ctx context.Context, client EthClient, confirmations ConfirmationParams) (uint64, error) {
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
		} else {
			return 0, nil
		}

	default:
		return 0, errors.New("invalid confirmation configuration")
	}

	return 0, nil
}
