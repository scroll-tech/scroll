package bytecode

import (
	"context"
	"fmt"
	"math/big"

	geth "github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
)

type ContractAPI interface {
	GetAddress() common.Address
	GetParsers() map[common.Hash]func(log *types.Log) (interface{}, error)
	GetABI() *abi.ABI
}

type Event struct {
	Name string
	ID   common.Hash
}

type ContractsFilter struct {
	contractAPIs []ContractAPI
	parsers      map[common.Hash]func(log *types.Log) (interface{}, error)
	queries      map[common.Address][]common.Hash
	handlers     map[common.Hash]func(vLog *types.Log, value interface{}) error
}

func NewContractsFilter(cAPIs ...ContractAPI) *ContractsFilter {
	parsers := make(map[common.Hash]func(log *types.Log) (interface{}, error))
	for _, cABI := range cAPIs {
		for id, parse := range cABI.GetParsers() {
			parsers[id] = parse
		}
	}
	return &ContractsFilter{
		contractAPIs: cAPIs,
		parsers:      parsers,
		queries:      make(map[common.Address][]common.Hash),
		handlers:     make(map[common.Hash]func(vLog *types.Log, value interface{}) error),
	}
}

func (c *ContractsFilter) ParseLogs(ctx context.Context, client *ethclient.Client, start, end uint64) error {
	query := &geth.FilterQuery{
		FromBlock: big.NewInt(0).SetUint64(start),
		ToBlock:   big.NewInt(0).SetUint64(end),
		Addresses: make([]common.Address, 0, len(c.queries)),
		Topics:    make([][]common.Hash, 1),
	}
	for addr, ids := range c.queries {
		query.Addresses = append(query.Addresses, addr)
		query.Topics[0] = append(query.Topics[0], ids...)
	}

	logs, err := client.FilterLogs(ctx, *query)
	if err != nil {
		return err
	}
	for i := 0; i < len(logs); i++ {
		vLog := &logs[i]
		_id := vLog.Topics[0]
		if _, exist := c.handlers[_id]; exist {
			val, err := c.parsers[_id](vLog)
			if err != nil {
				return err
			}
			if err = c.handlers[_id](vLog, val); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *ContractsFilter) RegisterSig(sigHash common.Hash, handle func(vLog *types.Log, value interface{}) error) error {
	for _, api := range c.contractAPIs {
		addr := api.GetAddress()
		for _, val := range api.GetABI().Events {
			if _, exist := c.parsers[sigHash]; !exist {
				return fmt.Errorf("can't parse this event, event ID: %s", sigHash.String())
			}
			if val.ID == sigHash {
				if c.queries[addr] == nil {
					c.queries[addr] = []common.Hash{}
				}
				c.queries[addr] = append(c.queries[addr], val.ID)
				c.handlers[val.ID] = handle
				return nil
			}
		}
	}
	return fmt.Errorf("no event hash can match this one, event ID: %s", sigHash.String())
}
