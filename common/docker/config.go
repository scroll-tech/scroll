package docker

import (
	"github.com/scroll-tech/go-ethereum/common"
)

// L1Contracts stores pre-deployed contracts address of scroll_l1geth
type L1Contracts struct {
	L2GasPriceOracle  common.Address `json:"L2GasPriceOracle"`
	L1Whitelist       common.Address `json:"L1Whitelist"`
	L1ScrollChain     common.Address `json:"L1ScrollChain"`
	L1MessageQueue    common.Address `json:"L1MessageQueue"`
	L1ScrollMessenger common.Address `json:"L1ScrollMessenger"`
	L1GatewayRouter   common.Address `json:"L1GatewayRouter"`
	L1ETHGateway      common.Address `json:"L1ETHGateway"`
}

// L2Contracts stores pre-deployed contracts address of scroll_l2geth
type L2Contracts struct {
	L1GasPriceOracle  common.Address `json:"L1GasPriceOracle"`
	L1BlockContainer  common.Address `json:"L1BlockContainer"`
	L2Whitelist       common.Address `json:"L2Whitelist"`
	L2ProxyAdmin      common.Address `json:"L2ProxyAdmin"`
	L2ScrollMessenger common.Address `json:"L2ScrollMessenger"`
	L2MessageQueue    common.Address `json:"L2MessageQueue"`
	L2TxFeeVault      common.Address `json:"L2TxFeeVault"`
	L2GatewayRouter   common.Address `json:"L2GatewayRouter"`
	L2ETHGateway      common.Address `json:"L2ETHGateway"`
}

// ContractsList all contracts addresses which are needed to be tested.
type ContractsList struct {
	L1Contracts *L1Contracts   `json:"l1_contracts,omitempty"`
	L2Contracts *L2Contracts   `json:"l2_contracts,omitempty"`
	ERC20       common.Address `json:"erc20,omitempty"`
	Greeter     common.Address `json:"greeter,omitempty"`
}
