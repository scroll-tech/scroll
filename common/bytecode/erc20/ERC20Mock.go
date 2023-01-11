// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package erc20

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// ERC20MockMetaData contains all meta data concerning the ERC20Mock contract.
var ERC20MockMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"address\",\"name\":\"initialAccount\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"initialBalance\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Approval\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"}],\"name\":\"allowance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"approveInternal\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"burn\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"decimals\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"subtractedValue\",\"type\":\"uint256\"}],\"name\":\"decreaseAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"addedValue\",\"type\":\"uint256\"}],\"name\":\"increaseAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"mint\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"symbol\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"transferInternal\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6080604052604051620020f4380380620020f4833981810160405281019062000029919062000417565b838381600390816200003c919062000708565b5080600490816200004e919062000708565b5050506200006382826200006d60201b60201c565b505050506200090a565b600073ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff1603620000df576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401620000d69062000850565b60405180910390fd5b620000f360008383620001da60201b60201c565b8060026000828254620001079190620008a1565b92505081905550806000808473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020600082825401925050819055508173ffffffffffffffffffffffffffffffffffffffff16600073ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef83604051620001ba9190620008ed565b60405180910390a3620001d660008383620001df60201b60201c565b5050565b505050565b505050565b6000604051905090565b600080fd5b600080fd5b600080fd5b600080fd5b6000601f19601f8301169050919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b6200024d8262000202565b810181811067ffffffffffffffff821117156200026f576200026e62000213565b5b80604052505050565b600062000284620001e4565b905062000292828262000242565b919050565b600067ffffffffffffffff821115620002b557620002b462000213565b5b620002c08262000202565b9050602081019050919050565b60005b83811015620002ed578082015181840152602081019050620002d0565b60008484015250505050565b6000620003106200030a8462000297565b62000278565b9050828152602081018484840111156200032f576200032e620001fd565b5b6200033c848285620002cd565b509392505050565b600082601f8301126200035c576200035b620001f8565b5b81516200036e848260208601620002f9565b91505092915050565b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b6000620003a48262000377565b9050919050565b620003b68162000397565b8114620003c257600080fd5b50565b600081519050620003d681620003ab565b92915050565b6000819050919050565b620003f181620003dc565b8114620003fd57600080fd5b50565b6000815190506200041181620003e6565b92915050565b60008060008060808587031215620004345762000433620001ee565b5b600085015167ffffffffffffffff811115620004555762000454620001f3565b5b620004638782880162000344565b945050602085015167ffffffffffffffff811115620004875762000486620001f3565b5b620004958782880162000344565b9350506040620004a887828801620003c5565b9250506060620004bb8782880162000400565b91505092959194509250565b600081519050919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b600060028204905060018216806200051a57607f821691505b60208210810362000530576200052f620004d2565b5b50919050565b60008190508160005260206000209050919050565b60006020601f8301049050919050565b600082821b905092915050565b6000600883026200059a7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff826200055b565b620005a686836200055b565b95508019841693508086168417925050509392505050565b6000819050919050565b6000620005e9620005e3620005dd84620003dc565b620005be565b620003dc565b9050919050565b6000819050919050565b6200060583620005c8565b6200061d6200061482620005f0565b84845462000568565b825550505050565b600090565b6200063462000625565b62000641818484620005fa565b505050565b5b8181101562000669576200065d6000826200062a565b60018101905062000647565b5050565b601f821115620006b857620006828162000536565b6200068d846200054b565b810160208510156200069d578190505b620006b5620006ac856200054b565b83018262000646565b50505b505050565b600082821c905092915050565b6000620006dd60001984600802620006bd565b1980831691505092915050565b6000620006f88383620006ca565b9150826002028217905092915050565b6200071382620004c7565b67ffffffffffffffff8111156200072f576200072e62000213565b5b6200073b825462000501565b620007488282856200066d565b600060209050601f8311600181146200078057600084156200076b578287015190505b620007778582620006ea565b865550620007e7565b601f198416620007908662000536565b60005b82811015620007ba5784890151825560018201915060208501945060208101905062000793565b86831015620007da5784890151620007d6601f891682620006ca565b8355505b6001600288020188555050505b505050505050565b600082825260208201905092915050565b7f45524332303a206d696e7420746f20746865207a65726f206164647265737300600082015250565b600062000838601f83620007ef565b9150620008458262000800565b602082019050919050565b600060208201905081810360008301526200086b8162000829565b9050919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b6000620008ae82620003dc565b9150620008bb83620003dc565b9250828201905080821115620008d657620008d562000872565b5b92915050565b620008e781620003dc565b82525050565b6000602082019050620009046000830184620008dc565b92915050565b6117da806200091a6000396000f3fe608060405234801561001057600080fd5b50600436106100f55760003560e01c806340c10f19116100975780639dc29fac116100665780639dc29fac14610286578063a457c2d7146102a2578063a9059cbb146102d2578063dd62ed3e14610302576100f5565b806340c10f191461020057806356189cb41461021c57806370a082311461023857806395d89b4114610268576100f5565b8063222f5be0116100d3578063222f5be01461016657806323b872dd14610182578063313ce567146101b257806339509351146101d0576100f5565b806306fdde03146100fa578063095ea7b31461011857806318160ddd14610148575b600080fd5b610102610332565b60405161010f9190610f27565b60405180910390f35b610132600480360381019061012d9190610fe2565b6103c4565b60405161013f919061103d565b60405180910390f35b6101506103e7565b60405161015d9190611067565b60405180910390f35b610180600480360381019061017b9190611082565b6103f1565b005b61019c60048036038101906101979190611082565b610401565b6040516101a9919061103d565b60405180910390f35b6101ba610430565b6040516101c791906110f1565b60405180910390f35b6101ea60048036038101906101e59190610fe2565b610439565b6040516101f7919061103d565b60405180910390f35b61021a60048036038101906102159190610fe2565b610470565b005b61023660048036038101906102319190611082565b61047e565b005b610252600480360381019061024d919061110c565b61048e565b60405161025f9190611067565b60405180910390f35b6102706104d6565b60405161027d9190610f27565b60405180910390f35b6102a0600480360381019061029b9190610fe2565b610568565b005b6102bc60048036038101906102b79190610fe2565b610576565b6040516102c9919061103d565b60405180910390f35b6102ec60048036038101906102e79190610fe2565b6105ed565b6040516102f9919061103d565b60405180910390f35b61031c60048036038101906103179190611139565b610610565b6040516103299190611067565b60405180910390f35b606060038054610341906111a8565b80601f016020809104026020016040519081016040528092919081815260200182805461036d906111a8565b80156103ba5780601f1061038f576101008083540402835291602001916103ba565b820191906000526020600020905b81548152906001019060200180831161039d57829003601f168201915b5050505050905090565b6000806103cf610697565b90506103dc81858561069f565b600191505092915050565b6000600254905090565b6103fc838383610868565b505050565b60008061040c610697565b9050610419858285610ade565b610424858585610868565b60019150509392505050565b60006012905090565b600080610444610697565b90506104658185856104568589610610565b6104609190611208565b61069f565b600191505092915050565b61047a8282610b6a565b5050565b61048983838361069f565b505050565b60008060008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020549050919050565b6060600480546104e5906111a8565b80601f0160208091040260200160405190810160405280929190818152602001828054610511906111a8565b801561055e5780601f106105335761010080835404028352916020019161055e565b820191906000526020600020905b81548152906001019060200180831161054157829003601f168201915b5050505050905090565b6105728282610cc0565b5050565b600080610581610697565b9050600061058f8286610610565b9050838110156105d4576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016105cb906112ae565b60405180910390fd5b6105e1828686840361069f565b60019250505092915050565b6000806105f8610697565b9050610605818585610868565b600191505092915050565b6000600160008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054905092915050565b600033905090565b600073ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff160361070e576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161070590611340565b60405180910390fd5b600073ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff160361077d576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610774906113d2565b60405180910390fd5b80600160008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055508173ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff167f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b9258360405161085b9190611067565b60405180910390a3505050565b600073ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff16036108d7576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016108ce90611464565b60405180910390fd5b600073ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff1603610946576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161093d906114f6565b60405180910390fd5b610951838383610e8d565b60008060008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020549050818110156109d7576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016109ce90611588565b60405180910390fd5b8181036000808673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002081905550816000808573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020600082825401925050819055508273ffffffffffffffffffffffffffffffffffffffff168473ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef84604051610ac59190611067565b60405180910390a3610ad8848484610e92565b50505050565b6000610aea8484610610565b90507fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8114610b645781811015610b56576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610b4d906115f4565b60405180910390fd5b610b63848484840361069f565b5b50505050565b600073ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff1603610bd9576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610bd090611660565b60405180910390fd5b610be560008383610e8d565b8060026000828254610bf79190611208565b92505081905550806000808473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020600082825401925050819055508173ffffffffffffffffffffffffffffffffffffffff16600073ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef83604051610ca89190611067565b60405180910390a3610cbc60008383610e92565b5050565b600073ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff1603610d2f576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610d26906116f2565b60405180910390fd5b610d3b82600083610e8d565b60008060008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054905081811015610dc1576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610db890611784565b60405180910390fd5b8181036000808573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000208190555081600260008282540392505081905550600073ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef84604051610e749190611067565b60405180910390a3610e8883600084610e92565b505050565b505050565b505050565b600081519050919050565b600082825260208201905092915050565b60005b83811015610ed1578082015181840152602081019050610eb6565b60008484015250505050565b6000601f19601f8301169050919050565b6000610ef982610e97565b610f038185610ea2565b9350610f13818560208601610eb3565b610f1c81610edd565b840191505092915050565b60006020820190508181036000830152610f418184610eee565b905092915050565b600080fd5b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b6000610f7982610f4e565b9050919050565b610f8981610f6e565b8114610f9457600080fd5b50565b600081359050610fa681610f80565b92915050565b6000819050919050565b610fbf81610fac565b8114610fca57600080fd5b50565b600081359050610fdc81610fb6565b92915050565b60008060408385031215610ff957610ff8610f49565b5b600061100785828601610f97565b925050602061101885828601610fcd565b9150509250929050565b60008115159050919050565b61103781611022565b82525050565b6000602082019050611052600083018461102e565b92915050565b61106181610fac565b82525050565b600060208201905061107c6000830184611058565b92915050565b60008060006060848603121561109b5761109a610f49565b5b60006110a986828701610f97565b93505060206110ba86828701610f97565b92505060406110cb86828701610fcd565b9150509250925092565b600060ff82169050919050565b6110eb816110d5565b82525050565b600060208201905061110660008301846110e2565b92915050565b60006020828403121561112257611121610f49565b5b600061113084828501610f97565b91505092915050565b600080604083850312156111505761114f610f49565b5b600061115e85828601610f97565b925050602061116f85828601610f97565b9150509250929050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b600060028204905060018216806111c057607f821691505b6020821081036111d3576111d2611179565b5b50919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b600061121382610fac565b915061121e83610fac565b9250828201905080821115611236576112356111d9565b5b92915050565b7f45524332303a2064656372656173656420616c6c6f77616e63652062656c6f7760008201527f207a65726f000000000000000000000000000000000000000000000000000000602082015250565b6000611298602583610ea2565b91506112a38261123c565b604082019050919050565b600060208201905081810360008301526112c78161128b565b9050919050565b7f45524332303a20617070726f76652066726f6d20746865207a65726f2061646460008201527f7265737300000000000000000000000000000000000000000000000000000000602082015250565b600061132a602483610ea2565b9150611335826112ce565b604082019050919050565b600060208201905081810360008301526113598161131d565b9050919050565b7f45524332303a20617070726f766520746f20746865207a65726f20616464726560008201527f7373000000000000000000000000000000000000000000000000000000000000602082015250565b60006113bc602283610ea2565b91506113c782611360565b604082019050919050565b600060208201905081810360008301526113eb816113af565b9050919050565b7f45524332303a207472616e736665722066726f6d20746865207a65726f20616460008201527f6472657373000000000000000000000000000000000000000000000000000000602082015250565b600061144e602583610ea2565b9150611459826113f2565b604082019050919050565b6000602082019050818103600083015261147d81611441565b9050919050565b7f45524332303a207472616e7366657220746f20746865207a65726f206164647260008201527f6573730000000000000000000000000000000000000000000000000000000000602082015250565b60006114e0602383610ea2565b91506114eb82611484565b604082019050919050565b6000602082019050818103600083015261150f816114d3565b9050919050565b7f45524332303a207472616e7366657220616d6f756e742065786365656473206260008201527f616c616e63650000000000000000000000000000000000000000000000000000602082015250565b6000611572602683610ea2565b915061157d82611516565b604082019050919050565b600060208201905081810360008301526115a181611565565b9050919050565b7f45524332303a20696e73756666696369656e7420616c6c6f77616e6365000000600082015250565b60006115de601d83610ea2565b91506115e9826115a8565b602082019050919050565b6000602082019050818103600083015261160d816115d1565b9050919050565b7f45524332303a206d696e7420746f20746865207a65726f206164647265737300600082015250565b600061164a601f83610ea2565b915061165582611614565b602082019050919050565b600060208201905081810360008301526116798161163d565b9050919050565b7f45524332303a206275726e2066726f6d20746865207a65726f2061646472657360008201527f7300000000000000000000000000000000000000000000000000000000000000602082015250565b60006116dc602183610ea2565b91506116e782611680565b604082019050919050565b6000602082019050818103600083015261170b816116cf565b9050919050565b7f45524332303a206275726e20616d6f756e7420657863656564732062616c616e60008201527f6365000000000000000000000000000000000000000000000000000000000000602082015250565b600061176e602283610ea2565b915061177982611712565b604082019050919050565b6000602082019050818103600083015261179d81611761565b905091905056fea26469706673582212206a0466cc8279c36801cc7618e809890da047e4518cddd96836b2e2250be5535764736f6c63430008100033",
}

// ERC20MockABI is the input ABI used to generate the binding from.
// Deprecated: Use ERC20MockMetaData.ABI instead.
var ERC20MockABI = ERC20MockMetaData.ABI

// ERC20MockBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use ERC20MockMetaData.Bin instead.
var ERC20MockBin = ERC20MockMetaData.Bin

// DeployERC20Mock deploys a new Ethereum contract, binding an instance of ERC20Mock to it.
func DeployERC20Mock(auth *bind.TransactOpts, backend bind.ContractBackend, name string, symbol string, initialAccount common.Address, initialBalance *big.Int) (common.Address, *types.Transaction, *ERC20Mock, error) {
	parsed, err := ERC20MockMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(ERC20MockBin), backend, name, symbol, initialAccount, initialBalance)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &ERC20Mock{ERC20MockCaller: ERC20MockCaller{contract: contract}, ERC20MockTransactor: ERC20MockTransactor{contract: contract}, ERC20MockFilterer: ERC20MockFilterer{contract: contract}}, nil
}

// ERC20Mock is an auto generated Go binding around an Ethereum contract.
type ERC20Mock struct {
	ERC20MockCaller     // Read-only binding to the contract
	ERC20MockTransactor // Write-only binding to the contract
	ERC20MockFilterer   // Log filterer for contract events
}

// ERC20MockCaller is an auto generated read-only Go binding around an Ethereum contract.
type ERC20MockCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ERC20MockTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ERC20MockTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ERC20MockFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ERC20MockFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ERC20MockSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ERC20MockSession struct {
	Contract     *ERC20Mock        // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ERC20MockCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ERC20MockCallerSession struct {
	Contract *ERC20MockCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts    // Call options to use throughout this session
}

// ERC20MockTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ERC20MockTransactorSession struct {
	Contract     *ERC20MockTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// ERC20MockRaw is an auto generated low-level Go binding around an Ethereum contract.
type ERC20MockRaw struct {
	Contract *ERC20Mock // Generic contract binding to access the raw methods on
}

// ERC20MockCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ERC20MockCallerRaw struct {
	Contract *ERC20MockCaller // Generic read-only contract binding to access the raw methods on
}

// ERC20MockTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ERC20MockTransactorRaw struct {
	Contract *ERC20MockTransactor // Generic write-only contract binding to access the raw methods on
}

// NewERC20Mock creates a new instance of ERC20Mock, bound to a specific deployed contract.
func NewERC20Mock(address common.Address, backend bind.ContractBackend) (*ERC20Mock, error) {
	contract, err := bindERC20Mock(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ERC20Mock{ERC20MockCaller: ERC20MockCaller{contract: contract}, ERC20MockTransactor: ERC20MockTransactor{contract: contract}, ERC20MockFilterer: ERC20MockFilterer{contract: contract}}, nil
}

// NewERC20MockCaller creates a new read-only instance of ERC20Mock, bound to a specific deployed contract.
func NewERC20MockCaller(address common.Address, caller bind.ContractCaller) (*ERC20MockCaller, error) {
	contract, err := bindERC20Mock(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ERC20MockCaller{contract: contract}, nil
}

// NewERC20MockTransactor creates a new write-only instance of ERC20Mock, bound to a specific deployed contract.
func NewERC20MockTransactor(address common.Address, transactor bind.ContractTransactor) (*ERC20MockTransactor, error) {
	contract, err := bindERC20Mock(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ERC20MockTransactor{contract: contract}, nil
}

// NewERC20MockFilterer creates a new log filterer instance of ERC20Mock, bound to a specific deployed contract.
func NewERC20MockFilterer(address common.Address, filterer bind.ContractFilterer) (*ERC20MockFilterer, error) {
	contract, err := bindERC20Mock(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ERC20MockFilterer{contract: contract}, nil
}

// bindERC20Mock binds a generic wrapper to an already deployed contract.
func bindERC20Mock(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(ERC20MockABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ERC20Mock *ERC20MockRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ERC20Mock.Contract.ERC20MockCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ERC20Mock *ERC20MockRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ERC20Mock.Contract.ERC20MockTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ERC20Mock *ERC20MockRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ERC20Mock.Contract.ERC20MockTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ERC20Mock *ERC20MockCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ERC20Mock.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ERC20Mock *ERC20MockTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ERC20Mock.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ERC20Mock *ERC20MockTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ERC20Mock.Contract.contract.Transact(opts, method, params...)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_ERC20Mock *ERC20MockCaller) Allowance(opts *bind.CallOpts, owner common.Address, spender common.Address) (*big.Int, error) {
	var out []interface{}
	err := _ERC20Mock.contract.Call(opts, &out, "allowance", owner, spender)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_ERC20Mock *ERC20MockSession) Allowance(owner common.Address, spender common.Address) (*big.Int, error) {
	return _ERC20Mock.Contract.Allowance(&_ERC20Mock.CallOpts, owner, spender)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_ERC20Mock *ERC20MockCallerSession) Allowance(owner common.Address, spender common.Address) (*big.Int, error) {
	return _ERC20Mock.Contract.Allowance(&_ERC20Mock.CallOpts, owner, spender)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_ERC20Mock *ERC20MockCaller) BalanceOf(opts *bind.CallOpts, account common.Address) (*big.Int, error) {
	var out []interface{}
	err := _ERC20Mock.contract.Call(opts, &out, "balanceOf", account)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_ERC20Mock *ERC20MockSession) BalanceOf(account common.Address) (*big.Int, error) {
	return _ERC20Mock.Contract.BalanceOf(&_ERC20Mock.CallOpts, account)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_ERC20Mock *ERC20MockCallerSession) BalanceOf(account common.Address) (*big.Int, error) {
	return _ERC20Mock.Contract.BalanceOf(&_ERC20Mock.CallOpts, account)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_ERC20Mock *ERC20MockCaller) Decimals(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _ERC20Mock.contract.Call(opts, &out, "decimals")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_ERC20Mock *ERC20MockSession) Decimals() (uint8, error) {
	return _ERC20Mock.Contract.Decimals(&_ERC20Mock.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_ERC20Mock *ERC20MockCallerSession) Decimals() (uint8, error) {
	return _ERC20Mock.Contract.Decimals(&_ERC20Mock.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_ERC20Mock *ERC20MockCaller) Name(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _ERC20Mock.contract.Call(opts, &out, "name")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_ERC20Mock *ERC20MockSession) Name() (string, error) {
	return _ERC20Mock.Contract.Name(&_ERC20Mock.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_ERC20Mock *ERC20MockCallerSession) Name() (string, error) {
	return _ERC20Mock.Contract.Name(&_ERC20Mock.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_ERC20Mock *ERC20MockCaller) Symbol(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _ERC20Mock.contract.Call(opts, &out, "symbol")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_ERC20Mock *ERC20MockSession) Symbol() (string, error) {
	return _ERC20Mock.Contract.Symbol(&_ERC20Mock.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_ERC20Mock *ERC20MockCallerSession) Symbol() (string, error) {
	return _ERC20Mock.Contract.Symbol(&_ERC20Mock.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_ERC20Mock *ERC20MockCaller) TotalSupply(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ERC20Mock.contract.Call(opts, &out, "totalSupply")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_ERC20Mock *ERC20MockSession) TotalSupply() (*big.Int, error) {
	return _ERC20Mock.Contract.TotalSupply(&_ERC20Mock.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_ERC20Mock *ERC20MockCallerSession) TotalSupply() (*big.Int, error) {
	return _ERC20Mock.Contract.TotalSupply(&_ERC20Mock.CallOpts)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 amount) returns(bool)
func (_ERC20Mock *ERC20MockTransactor) Approve(opts *bind.TransactOpts, spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return _ERC20Mock.contract.Transact(opts, "approve", spender, amount)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 amount) returns(bool)
func (_ERC20Mock *ERC20MockSession) Approve(spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return _ERC20Mock.Contract.Approve(&_ERC20Mock.TransactOpts, spender, amount)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 amount) returns(bool)
func (_ERC20Mock *ERC20MockTransactorSession) Approve(spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return _ERC20Mock.Contract.Approve(&_ERC20Mock.TransactOpts, spender, amount)
}

// ApproveInternal is a paid mutator transaction binding the contract method 0x56189cb4.
//
// Solidity: function approveInternal(address owner, address spender, uint256 value) returns()
func (_ERC20Mock *ERC20MockTransactor) ApproveInternal(opts *bind.TransactOpts, owner common.Address, spender common.Address, value *big.Int) (*types.Transaction, error) {
	return _ERC20Mock.contract.Transact(opts, "approveInternal", owner, spender, value)
}

// ApproveInternal is a paid mutator transaction binding the contract method 0x56189cb4.
//
// Solidity: function approveInternal(address owner, address spender, uint256 value) returns()
func (_ERC20Mock *ERC20MockSession) ApproveInternal(owner common.Address, spender common.Address, value *big.Int) (*types.Transaction, error) {
	return _ERC20Mock.Contract.ApproveInternal(&_ERC20Mock.TransactOpts, owner, spender, value)
}

// ApproveInternal is a paid mutator transaction binding the contract method 0x56189cb4.
//
// Solidity: function approveInternal(address owner, address spender, uint256 value) returns()
func (_ERC20Mock *ERC20MockTransactorSession) ApproveInternal(owner common.Address, spender common.Address, value *big.Int) (*types.Transaction, error) {
	return _ERC20Mock.Contract.ApproveInternal(&_ERC20Mock.TransactOpts, owner, spender, value)
}

// Burn is a paid mutator transaction binding the contract method 0x9dc29fac.
//
// Solidity: function burn(address account, uint256 amount) returns()
func (_ERC20Mock *ERC20MockTransactor) Burn(opts *bind.TransactOpts, account common.Address, amount *big.Int) (*types.Transaction, error) {
	return _ERC20Mock.contract.Transact(opts, "burn", account, amount)
}

// Burn is a paid mutator transaction binding the contract method 0x9dc29fac.
//
// Solidity: function burn(address account, uint256 amount) returns()
func (_ERC20Mock *ERC20MockSession) Burn(account common.Address, amount *big.Int) (*types.Transaction, error) {
	return _ERC20Mock.Contract.Burn(&_ERC20Mock.TransactOpts, account, amount)
}

// Burn is a paid mutator transaction binding the contract method 0x9dc29fac.
//
// Solidity: function burn(address account, uint256 amount) returns()
func (_ERC20Mock *ERC20MockTransactorSession) Burn(account common.Address, amount *big.Int) (*types.Transaction, error) {
	return _ERC20Mock.Contract.Burn(&_ERC20Mock.TransactOpts, account, amount)
}

// DecreaseAllowance is a paid mutator transaction binding the contract method 0xa457c2d7.
//
// Solidity: function decreaseAllowance(address spender, uint256 subtractedValue) returns(bool)
func (_ERC20Mock *ERC20MockTransactor) DecreaseAllowance(opts *bind.TransactOpts, spender common.Address, subtractedValue *big.Int) (*types.Transaction, error) {
	return _ERC20Mock.contract.Transact(opts, "decreaseAllowance", spender, subtractedValue)
}

// DecreaseAllowance is a paid mutator transaction binding the contract method 0xa457c2d7.
//
// Solidity: function decreaseAllowance(address spender, uint256 subtractedValue) returns(bool)
func (_ERC20Mock *ERC20MockSession) DecreaseAllowance(spender common.Address, subtractedValue *big.Int) (*types.Transaction, error) {
	return _ERC20Mock.Contract.DecreaseAllowance(&_ERC20Mock.TransactOpts, spender, subtractedValue)
}

// DecreaseAllowance is a paid mutator transaction binding the contract method 0xa457c2d7.
//
// Solidity: function decreaseAllowance(address spender, uint256 subtractedValue) returns(bool)
func (_ERC20Mock *ERC20MockTransactorSession) DecreaseAllowance(spender common.Address, subtractedValue *big.Int) (*types.Transaction, error) {
	return _ERC20Mock.Contract.DecreaseAllowance(&_ERC20Mock.TransactOpts, spender, subtractedValue)
}

// IncreaseAllowance is a paid mutator transaction binding the contract method 0x39509351.
//
// Solidity: function increaseAllowance(address spender, uint256 addedValue) returns(bool)
func (_ERC20Mock *ERC20MockTransactor) IncreaseAllowance(opts *bind.TransactOpts, spender common.Address, addedValue *big.Int) (*types.Transaction, error) {
	return _ERC20Mock.contract.Transact(opts, "increaseAllowance", spender, addedValue)
}

// IncreaseAllowance is a paid mutator transaction binding the contract method 0x39509351.
//
// Solidity: function increaseAllowance(address spender, uint256 addedValue) returns(bool)
func (_ERC20Mock *ERC20MockSession) IncreaseAllowance(spender common.Address, addedValue *big.Int) (*types.Transaction, error) {
	return _ERC20Mock.Contract.IncreaseAllowance(&_ERC20Mock.TransactOpts, spender, addedValue)
}

// IncreaseAllowance is a paid mutator transaction binding the contract method 0x39509351.
//
// Solidity: function increaseAllowance(address spender, uint256 addedValue) returns(bool)
func (_ERC20Mock *ERC20MockTransactorSession) IncreaseAllowance(spender common.Address, addedValue *big.Int) (*types.Transaction, error) {
	return _ERC20Mock.Contract.IncreaseAllowance(&_ERC20Mock.TransactOpts, spender, addedValue)
}

// Mint is a paid mutator transaction binding the contract method 0x40c10f19.
//
// Solidity: function mint(address account, uint256 amount) returns()
func (_ERC20Mock *ERC20MockTransactor) Mint(opts *bind.TransactOpts, account common.Address, amount *big.Int) (*types.Transaction, error) {
	return _ERC20Mock.contract.Transact(opts, "mint", account, amount)
}

// Mint is a paid mutator transaction binding the contract method 0x40c10f19.
//
// Solidity: function mint(address account, uint256 amount) returns()
func (_ERC20Mock *ERC20MockSession) Mint(account common.Address, amount *big.Int) (*types.Transaction, error) {
	return _ERC20Mock.Contract.Mint(&_ERC20Mock.TransactOpts, account, amount)
}

// Mint is a paid mutator transaction binding the contract method 0x40c10f19.
//
// Solidity: function mint(address account, uint256 amount) returns()
func (_ERC20Mock *ERC20MockTransactorSession) Mint(account common.Address, amount *big.Int) (*types.Transaction, error) {
	return _ERC20Mock.Contract.Mint(&_ERC20Mock.TransactOpts, account, amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 amount) returns(bool)
func (_ERC20Mock *ERC20MockTransactor) Transfer(opts *bind.TransactOpts, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _ERC20Mock.contract.Transact(opts, "transfer", to, amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 amount) returns(bool)
func (_ERC20Mock *ERC20MockSession) Transfer(to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _ERC20Mock.Contract.Transfer(&_ERC20Mock.TransactOpts, to, amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 amount) returns(bool)
func (_ERC20Mock *ERC20MockTransactorSession) Transfer(to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _ERC20Mock.Contract.Transfer(&_ERC20Mock.TransactOpts, to, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 amount) returns(bool)
func (_ERC20Mock *ERC20MockTransactor) TransferFrom(opts *bind.TransactOpts, from common.Address, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _ERC20Mock.contract.Transact(opts, "transferFrom", from, to, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 amount) returns(bool)
func (_ERC20Mock *ERC20MockSession) TransferFrom(from common.Address, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _ERC20Mock.Contract.TransferFrom(&_ERC20Mock.TransactOpts, from, to, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 amount) returns(bool)
func (_ERC20Mock *ERC20MockTransactorSession) TransferFrom(from common.Address, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _ERC20Mock.Contract.TransferFrom(&_ERC20Mock.TransactOpts, from, to, amount)
}

// TransferInternal is a paid mutator transaction binding the contract method 0x222f5be0.
//
// Solidity: function transferInternal(address from, address to, uint256 value) returns()
func (_ERC20Mock *ERC20MockTransactor) TransferInternal(opts *bind.TransactOpts, from common.Address, to common.Address, value *big.Int) (*types.Transaction, error) {
	return _ERC20Mock.contract.Transact(opts, "transferInternal", from, to, value)
}

// TransferInternal is a paid mutator transaction binding the contract method 0x222f5be0.
//
// Solidity: function transferInternal(address from, address to, uint256 value) returns()
func (_ERC20Mock *ERC20MockSession) TransferInternal(from common.Address, to common.Address, value *big.Int) (*types.Transaction, error) {
	return _ERC20Mock.Contract.TransferInternal(&_ERC20Mock.TransactOpts, from, to, value)
}

// TransferInternal is a paid mutator transaction binding the contract method 0x222f5be0.
//
// Solidity: function transferInternal(address from, address to, uint256 value) returns()
func (_ERC20Mock *ERC20MockTransactorSession) TransferInternal(from common.Address, to common.Address, value *big.Int) (*types.Transaction, error) {
	return _ERC20Mock.Contract.TransferInternal(&_ERC20Mock.TransactOpts, from, to, value)
}

// ERC20MockApprovalIterator is returned from FilterApproval and is used to iterate over the raw logs and unpacked data for Approval events raised by the ERC20Mock contract.
type ERC20MockApprovalIterator struct {
	Event *ERC20MockApproval // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ERC20MockApprovalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ERC20MockApproval)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ERC20MockApproval)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ERC20MockApprovalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ERC20MockApprovalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ERC20MockApproval represents a Approval event raised by the ERC20Mock contract.
type ERC20MockApproval struct {
	Owner   common.Address
	Spender common.Address
	Value   *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterApproval is a free log retrieval operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_ERC20Mock *ERC20MockFilterer) FilterApproval(opts *bind.FilterOpts, owner []common.Address, spender []common.Address) (*ERC20MockApprovalIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _ERC20Mock.contract.FilterLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return &ERC20MockApprovalIterator{contract: _ERC20Mock.contract, event: "Approval", logs: logs, sub: sub}, nil
}

// WatchApproval is a free log subscription operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_ERC20Mock *ERC20MockFilterer) WatchApproval(opts *bind.WatchOpts, sink chan<- *ERC20MockApproval, owner []common.Address, spender []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _ERC20Mock.contract.WatchLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ERC20MockApproval)
				if err := _ERC20Mock.contract.UnpackLog(event, "Approval", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseApproval is a log parse operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_ERC20Mock *ERC20MockFilterer) ParseApproval(log types.Log) (*ERC20MockApproval, error) {
	event := new(ERC20MockApproval)
	if err := _ERC20Mock.contract.UnpackLog(event, "Approval", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ERC20MockTransferIterator is returned from FilterTransfer and is used to iterate over the raw logs and unpacked data for Transfer events raised by the ERC20Mock contract.
type ERC20MockTransferIterator struct {
	Event *ERC20MockTransfer // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ERC20MockTransferIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ERC20MockTransfer)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ERC20MockTransfer)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ERC20MockTransferIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ERC20MockTransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ERC20MockTransfer represents a Transfer event raised by the ERC20Mock contract.
type ERC20MockTransfer struct {
	From  common.Address
	To    common.Address
	Value *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterTransfer is a free log retrieval operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_ERC20Mock *ERC20MockFilterer) FilterTransfer(opts *bind.FilterOpts, from []common.Address, to []common.Address) (*ERC20MockTransferIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _ERC20Mock.contract.FilterLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &ERC20MockTransferIterator{contract: _ERC20Mock.contract, event: "Transfer", logs: logs, sub: sub}, nil
}

// WatchTransfer is a free log subscription operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_ERC20Mock *ERC20MockFilterer) WatchTransfer(opts *bind.WatchOpts, sink chan<- *ERC20MockTransfer, from []common.Address, to []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _ERC20Mock.contract.WatchLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ERC20MockTransfer)
				if err := _ERC20Mock.contract.UnpackLog(event, "Transfer", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseTransfer is a log parse operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_ERC20Mock *ERC20MockFilterer) ParseTransfer(log types.Log) (*ERC20MockTransfer, error) {
	event := new(ERC20MockTransfer)
	if err := _ERC20Mock.contract.UnpackLog(event, "Transfer", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
