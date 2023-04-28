package docker

import (
	"encoding/json"
	"github.com/scroll-tech/go-ethereum/common"
	"os"
)

// L1Contracts stores pre-deployed contracts address of scroll_l1geth
type L1Contracts struct {
	L2GasPriceOracle       common.Address `json:"L2GasPriceOracle"`
	L1ScrollChain          common.Address `json:"L1ScrollChain"`
	L1MessageQueue         common.Address `json:"L1MessageQueue"`
	L1ScrollMessenger      common.Address `json:"L1ScrollMessenger"`
	L1GatewayRouter        common.Address `json:"L1GatewayRouter"`
	L1StandardERC20        common.Address `json:"L1StandardERC20"`
	L1StandardERC20Gateway common.Address `json:"L1StandardERC20Gateway"`
}

// L2Contracts stores pre-deployed contracts address of scroll_l2geth
type L2Contracts struct {
	L1BlockContainer             common.Address `json:"L1BlockContainer"`
	L1GasPriceOracle             common.Address `json:"L1GasPriceOracle"`
	L2ProxyAdmin                 common.Address `json:"L2ProxyAdmin"`
	L2ScrollStandardERC20Factory common.Address `json:"L2ScrollStandardERC20Factory"`
	L2ScrollMessenger            common.Address `json:"L2ScrollMessenger"`
	L2MessageQueue               common.Address `json:"L2MessageQueue"`
	L2TxFeeVault                 common.Address `json:"L2TxFeeVault"`
	L2GatewayRouter              common.Address `json:"L2GatewayRouter"`
	L2StandardERC20              common.Address `json:"L2StandardERC20"`
	L2StandardERC20Gateway       common.Address `json:"L2StandardERC20Gateway"`
}

type ContractsList struct {
	L1Contracts *L1Contracts   `json:"l1_contracts,omitempty"`
	L2Contracts *L2Contracts   `json:"l2_contracts,omitempty"`
	ERC20       common.Address `json:"erc20,omitempty"`
	Greeter     common.Address `json:"greeter,omitempty"`
}

func GetContractsList(file string) (*ContractsList, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var contractsList = &ContractsList{}
	return contractsList, json.Unmarshal(data, &contractsList)
}
