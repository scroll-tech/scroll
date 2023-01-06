// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package l1scrollmessenger

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

// IL1ScrollMessengerL2MessageProof is an auto generated low-level Go binding around an user-defined struct.
type IL1ScrollMessengerL2MessageProof struct {
	BatchIndex  *big.Int
	BlockHeight *big.Int
	MerkleProof []byte
}

// L1ScrollMessengerMetaData contains all meta data concerning the L1ScrollMessenger contract.
var L1ScrollMessengerMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"msgHash\",\"type\":\"bytes32\"}],\"name\":\"FailedRelayedMessage\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"msgHash\",\"type\":\"bytes32\"}],\"name\":\"MessageDropped\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"Paused\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"msgHash\",\"type\":\"bytes32\"}],\"name\":\"RelayedMessage\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"message\",\"type\":\"bytes\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"messageNonce\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"}],\"name\":\"SentMessage\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"Unpaused\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_oldDuration\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_newDuration\",\"type\":\"uint256\"}],\"name\":\"UpdateDropDelayDuration\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_oldGasOracle\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_newGasOracle\",\"type\":\"address\"}],\"name\":\"UpdateGasOracle\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_oldWhitelist\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"_newWhitelist\",\"type\":\"address\"}],\"name\":\"UpdateWhitelist\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"dropDelayDuration\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_fee\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_deadline\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"}],\"name\":\"dropMessage\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"gasOracle\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_rollup\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"isMessageDropped\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"isMessageExecuted\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"isMessageRelayed\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"pause\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"paused\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_fee\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_deadline\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_nonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"batchIndex\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"blockHeight\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"merkleProof\",\"type\":\"bytes\"}],\"internalType\":\"structIL1ScrollMessenger.L2MessageProof\",\"name\":\"_proof\",\"type\":\"tuple\"}],\"name\":\"relayMessageWithProof\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_fee\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_deadline\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"_queueIndex\",\"type\":\"uint256\"},{\"internalType\":\"uint32\",\"name\":\"_oldGasLimit\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"_newGasLimit\",\"type\":\"uint32\"}],\"name\":\"replayMessage\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"rollup\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_fee\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_message\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"_gasLimit\",\"type\":\"uint256\"}],\"name\":\"sendMessage\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_newDuration\",\"type\":\"uint256\"}],\"name\":\"updateDropDelayDuration\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newGasOracle\",\"type\":\"address\"}],\"name\":\"updateGasOracle\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newWhitelist\",\"type\":\"address\"}],\"name\":\"updateWhitelist\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"whitelist\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"xDomainMessageSender\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"stateMutability\":\"payable\",\"type\":\"receive\"}]",
	Bin: "0x608060405234801561001057600080fd5b50611ae1806100206000396000f3fe60806040526004361061012e5760003560e01c80638456cb59116100ab578063c4d66de81161006f578063c4d66de81461033c578063cb23bcb51461035c578063e30484041461037c578063ed885bfe146103a0578063f2fde38b146103c0578063f7f7469a146103e057600080fd5b80638456cb59146102b65780638da5cb5b146102cb57806393e59dc1146102e95780639545a74714610309578063b2267a7b1461032957600080fd5b80635d62a8dd116100f25780635d62a8dd146102095780636e296e451461024157806370cee67f14610261578063715018a6146102815780637cecd1e51461029657600080fd5b8063396c16b71461013a5780633d0f963e1461017f5780633df3390d146101a157806352a3e089146101d15780635c975abb146101f157600080fd5b3661013557005b600080fd5b34801561014657600080fd5b5061016a6101553660046113dd565b609d6020526000908152604090205460ff1681565b60405190151581526020015b60405180910390f35b34801561018b57600080fd5b5061019f61019a366004611412565b610410565b005b3480156101ad57600080fd5b5061016a6101bc3660046113dd565b609c6020526000908152604090205460ff1681565b3480156101dd57600080fd5b5061019f6101ec366004611500565b6104a5565b3480156101fd57600080fd5b5060655460ff1661016a565b34801561021557600080fd5b50609854610229906001600160a01b031681565b6040516001600160a01b039091168152602001610176565b34801561024d57600080fd5b50609754610229906001600160a01b031681565b34801561026d57600080fd5b5061019f61027c366004611412565b6108fe565b34801561028d57600080fd5b5061019f610982565b3480156102a257600080fd5b5061019f6102b13660046113dd565b6109b8565b3480156102c257600080fd5b5061019f610a20565b3480156102d757600080fd5b506033546001600160a01b0316610229565b3480156102f557600080fd5b50609954610229906001600160a01b031681565b34801561031557600080fd5b5061019f610324366004611604565b610a52565b61019f6103373660046116ab565b610a80565b34801561034857600080fd5b5061019f610357366004611412565b610d83565b34801561036857600080fd5b50609e54610229906001600160a01b031681565b34801561038857600080fd5b50610392609a5481565b604051908152602001610176565b3480156103ac57600080fd5b5061019f6103bb36600461170a565b610e96565b3480156103cc57600080fd5b5061019f6103db366004611412565b6111ba565b3480156103ec57600080fd5b5061016a6103fb3660046113dd565b609b6020526000908152604090205460ff1681565b6033546001600160a01b031633146104435760405162461bcd60e51b815260040161043a90611799565b60405180910390fd5b609980546001600160a01b038381166001600160a01b031983168117909355604080519190921680825260208201939093527f22d1c35fe072d2e42c3c8f9bd4a0d34aa84a0101d020a62517b33fdb3174e5f791015b60405180910390a15050565b60655460ff16156104c85760405162461bcd60e51b815260040161043a906117ce565b60995433906001600160a01b0316801580610548575060405163efc7840160e01b81526001600160a01b03838116600483015282169063efc7840190602401602060405180830381865afa158015610524573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061054891906117f8565b61058d5760405162461bcd60e51b81526020600482015260166024820152751cd95b99195c881b9bdd081dda1a5d195b1a5cdd195960521b604482015260640161043a565b6097546001600160a01b03166001146105df5760405162461bcd60e51b815260206004820152601460248201527330b63932b0b23c9034b71032bc32b1baba34b7b760611b604482015260640161043a565b60008a8a8a8a8a8a8a6040516020016105fe979695949392919061183e565b60408051601f1981840301815291815281516020928301206000818152609d90935291205490915060ff16156106765760405162461bcd60e51b815260206004820152601d60248201527f4d657373616765207375636365737366756c6c79206578656375746564000000604482015260640161043a565b609e5484516020860151604051637142ab0160e11b8152600481019290925260248201526001600160a01b039091169063e285560290604401602060405180830381865afa1580156106cc573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906106f091906117f8565b6107325760405162461bcd60e51b815260206004820152601360248201527234b73b30b634b21039ba30ba3290383937b7b360691b604482015260640161043a565b6097546001600160a01b03908116908c16036107895760405162461bcd60e51b815260206004820152601660248201527534b73b30b634b21036b2b9b9b0b3b29039b2b73232b960511b604482015260640161043a565b609780546001600160a01b0319166001600160a01b038d8116919091179091556040516000918c16908b906107bf9089906118a0565b60006040518083038185875af1925050503d80600081146107fc576040519150601f19603f3d011682016040523d82523d6000602084013e610801565b606091505b5050609780546001600160a01b031916600117905590508015610863576000828152609d6020526040808220805460ff191660011790555183917f4641df4a962071e12719d8c8c8e5ac7fc4d97b927346a3d7a335b1f7517e133c91a261088f565b60405182907f99d0e048484baa1b1540b1367cb128acd7ab2946d1ed91ec10e3c85e4bf51b8f90600090a25b60408051602081018490526bffffffffffffffffffffffff193360601b169181019190915243605482015260009060740160408051601f1981840301815291815281516020928301206000908152609b9092529020805460ff1916600117905550505050505050505050505050565b6033546001600160a01b031633146109285760405162461bcd60e51b815260040161043a90611799565b609880546001600160a01b038381166001600160a01b031983168117909355604080519190921680825260208201939093527f9ed5ec28f252b3e7f62f1ace8e54c5ebabf4c61cc2a7c33a806365b2ff7ecc5e9101610499565b6033546001600160a01b031633146109ac5760405162461bcd60e51b815260040161043a90611799565b6109b66000611255565b565b6033546001600160a01b031633146109e25760405162461bcd60e51b815260040161043a90611799565b609a80549082905560408051828152602081018490527f8767db55656d87982bde23dfa77887931b21ecc3386f5764bf02ef0070d117429101610499565b6033546001600160a01b03163314610a4a5760405162461bcd60e51b815260040161043a90611799565b6109b66112a7565b60655460ff1615610a755760405162461bcd60e51b815260040161043a906117ce565b505050505050505050565b60655460ff1615610aa35760405162461bcd60e51b815260040161043a906117ce565b60995433906001600160a01b0316801580610b23575060405163efc7840160e01b81526001600160a01b03838116600483015282169063efc7840190602401602060405180830381865afa158015610aff573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610b2391906117f8565b610b685760405162461bcd60e51b81526020600482015260166024820152751cd95b99195c881b9bdd081dda1a5d195b1a5cdd195960521b604482015260640161043a565b84341015610ba95760405162461bcd60e51b815260206004820152600e60248201526d63616e6e6f74207061792066656560901b604482015260640161043a565b6000609a5442610bb991906118bc565b6098549091506000906001600160a01b031615610c4a57609854604051639856cf9f60e01b81526001600160a01b0390911690639856cf9f90610c049033908c908b9060040161190f565b602060405180830381865afa158015610c21573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610c459190611944565b610c4d565b60005b905080871015610c8f5760405162461bcd60e51b815260206004820152600d60248201526c199959481d1bdbc81cdb585b1b609a1b604482015260640161043a565b600087340390506000609e60009054906101000a90046001600160a01b03166001600160a01b0316632fc9931a338c858d898e8e6040518863ffffffff1660e01b8152600401610ce5979695949392919061195d565b6020604051808303816000875af1158015610d04573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610d289190611944565b9050896001600160a01b03167f806b28931bc6fbe6c146babfb83d5c2b47e971edb43b4566f010577a0ee7d9f433848c888d878e604051610d6f97969594939291906119ae565b60405180910390a250505050505050505050565b600054610100900460ff16610d9e5760005460ff1615610da2565b303b155b610e055760405162461bcd60e51b815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201526d191e481a5b9a5d1a585b1a5e995960921b606482015260840161043a565b600054610100900460ff16158015610e27576000805461ffff19166101011790555b610e2f61131c565b610e3761134b565b610e56609780546001600160a01b031916600117905562093a80609a55565b609e80546001600160a01b0384166001600160a01b0319918216179091556097805490911660011790558015610e92576000805461ff00191690555b5050565b60655460ff1615610eb95760405162461bcd60e51b815260040161043a906117ce565b834211610efe5760405162461bcd60e51b81526020600482015260136024820152721b595cdcd859d9481b9bdd08195e1c1a5c9959606a1b604482015260640161043a565b609e5460408051633d0b3d4560e11b815290516001600160a01b03909216916000918391637a167a8a916004808201926020929091908290030181865afa158015610f4d573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610f719190611944565b905084811115610fc35760405162461bcd60e51b815260206004820152601860248201527f6d65737361676520616c72656164792065786563757465640000000000000000604482015260640161043a565b60405163447273d760e01b8152600481018690526000906001600160a01b0384169063447273d790602401602060405180830381865afa15801561100b573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061102f9190611944565b905060008b8b8b8b8b8b8b8b6040516020016110529897969594939291906119f7565b6040516020818303038152906040528051906020012090508181146110b95760405162461bcd60e51b815260206004820152601760248201527f6d6573736167652068617368206d69736d617463686564000000000000000000604482015260640161043a565b6000818152609c602052604090205460ff16156111185760405162461bcd60e51b815260206004820152601760248201527f6d65737361676520616c72656164792064726f70706564000000000000000000604482015260640161043a565b6000818152609c60205260409020805460ff191660011790556001600160a01b038c163b611181576001600160a01b038c166108fc6111578b8d6118bc565b6040518115909202916000818181858888f1935050505015801561117f573d6000803e3d6000fd5b505b60405181907f6629230ca69c43f97674dd064896b819957583c8d20a870e4fb28b05c5d29f2990600090a2505050505050505050505050565b6033546001600160a01b031633146111e45760405162461bcd60e51b815260040161043a90611799565b6001600160a01b0381166112495760405162461bcd60e51b815260206004820152602660248201527f4f776e61626c653a206e6577206f776e657220697320746865207a65726f206160448201526564647265737360d01b606482015260840161043a565b61125281611255565b50565b603380546001600160a01b038381166001600160a01b0319831681179093556040519116919082907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e090600090a35050565b60655460ff16156112ca5760405162461bcd60e51b815260040161043a906117ce565b6065805460ff191660011790557f62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a2586112ff3390565b6040516001600160a01b03909116815260200160405180910390a1565b600054610100900460ff166113435760405162461bcd60e51b815260040161043a90611a60565b6109b661137a565b600054610100900460ff166113725760405162461bcd60e51b815260040161043a90611a60565b6109b66113aa565b600054610100900460ff166113a15760405162461bcd60e51b815260040161043a90611a60565b6109b633611255565b600054610100900460ff166113d15760405162461bcd60e51b815260040161043a90611a60565b6065805460ff19169055565b6000602082840312156113ef57600080fd5b5035919050565b80356001600160a01b038116811461140d57600080fd5b919050565b60006020828403121561142457600080fd5b61142d826113f6565b9392505050565b634e487b7160e01b600052604160045260246000fd5b6040516060810167ffffffffffffffff8111828210171561146d5761146d611434565b60405290565b600082601f83011261148457600080fd5b813567ffffffffffffffff8082111561149f5761149f611434565b604051601f8301601f19908116603f011681019082821181831017156114c7576114c7611434565b816040528381528660208588010111156114e057600080fd5b836020870160208301376000602085830101528094505050505092915050565b600080600080600080600080610100898b03121561151d57600080fd5b611526896113f6565b975061153460208a016113f6565b965060408901359550606089013594506080890135935060a0890135925060c089013567ffffffffffffffff8082111561156d57600080fd5b6115798c838d01611473565b935060e08b013591508082111561158f57600080fd5b908a01906060828d0312156115a357600080fd5b6115ab61144a565b82358152602083013560208201526040830135828111156115cb57600080fd5b6115d78e828601611473565b6040830152508093505050509295985092959890939650565b803563ffffffff8116811461140d57600080fd5b60008060008060008060008060006101208a8c03121561162357600080fd5b61162c8a6113f6565b985061163a60208b016113f6565b975060408a0135965060608a0135955060808a0135945060a08a013567ffffffffffffffff81111561166b57600080fd5b6116778c828d01611473565b94505060c08a0135925061168d60e08b016115f0565b915061169c6101008b016115f0565b90509295985092959850929598565b600080600080608085870312156116c157600080fd5b6116ca856113f6565b935060208501359250604085013567ffffffffffffffff8111156116ed57600080fd5b6116f987828801611473565b949793965093946060013593505050565b600080600080600080600080610100898b03121561172757600080fd5b611730896113f6565b975061173e60208a016113f6565b965060408901359550606089013594506080890135935060a0890135925060c089013567ffffffffffffffff81111561177657600080fd5b6117828b828c01611473565b92505060e089013590509295985092959890939650565b6020808252818101527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e6572604082015260600190565b60208082526010908201526f14185d5cd8589b194e881c185d5cd95960821b604082015260600190565b60006020828403121561180a57600080fd5b8151801515811461142d57600080fd5b60005b8381101561183557818101518382015260200161181d565b50506000910152565b60006bffffffffffffffffffffffff19808a60601b168352808960601b16601484015250866028830152856048830152846068830152836088830152825161188d8160a885016020870161181a565b9190910160a80198975050505050505050565b600082516118b281846020870161181a565b9190910192915050565b808201808211156118dd57634e487b7160e01b600052601160045260246000fd5b92915050565b600081518084526118fb81602086016020860161181a565b601f01601f19169290920160200192915050565b6001600160a01b0384811682528316602082015260606040820181905260009061193b908301846118e3565b95945050505050565b60006020828403121561195657600080fd5b5051919050565b600060018060a01b03808a16835280891660208401525086604083015285606083015284608083015260e060a083015261199a60e08301856118e3565b90508260c083015298975050505050505050565b60018060a01b038816815286602082015285604082015284606082015260e0608082015260006119e160e08301866118e3565b60a08301949094525060c0015295945050505050565b60006bffffffffffffffffffffffff19808b60601b168352808a60601b166014840152508760288301528660488301528560688301528460888301528351611a468160a885016020880161181a565b60a892019182019290925260c80198975050505050505050565b6020808252602b908201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960408201526a6e697469616c697a696e6760a81b60608201526080019056fea26469706673582212204e6a2db0d0b6778304f10589b17cf8f93019d0d218e44fafc55153cf97c93f2a64736f6c63430008110033",
}

// L1ScrollMessengerABI is the input ABI used to generate the binding from.
// Deprecated: Use L1ScrollMessengerMetaData.ABI instead.
var L1ScrollMessengerABI = L1ScrollMessengerMetaData.ABI

// L1ScrollMessengerBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use L1ScrollMessengerMetaData.Bin instead.
var L1ScrollMessengerBin = L1ScrollMessengerMetaData.Bin

// DeployL1ScrollMessenger deploys a new Ethereum contract, binding an instance of L1ScrollMessenger to it.
func DeployL1ScrollMessenger(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *L1ScrollMessenger, error) {
	parsed, err := L1ScrollMessengerMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(L1ScrollMessengerBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &L1ScrollMessenger{L1ScrollMessengerCaller: L1ScrollMessengerCaller{contract: contract}, L1ScrollMessengerTransactor: L1ScrollMessengerTransactor{contract: contract}, L1ScrollMessengerFilterer: L1ScrollMessengerFilterer{contract: contract}}, nil
}

// L1ScrollMessenger is an auto generated Go binding around an Ethereum contract.
type L1ScrollMessenger struct {
	L1ScrollMessengerCaller     // Read-only binding to the contract
	L1ScrollMessengerTransactor // Write-only binding to the contract
	L1ScrollMessengerFilterer   // Log filterer for contract events
}

// L1ScrollMessengerCaller is an auto generated read-only Go binding around an Ethereum contract.
type L1ScrollMessengerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1ScrollMessengerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type L1ScrollMessengerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1ScrollMessengerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type L1ScrollMessengerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L1ScrollMessengerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type L1ScrollMessengerSession struct {
	Contract     *L1ScrollMessenger // Generic contract binding to set the session for
	CallOpts     bind.CallOpts      // Call options to use throughout this session
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// L1ScrollMessengerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type L1ScrollMessengerCallerSession struct {
	Contract *L1ScrollMessengerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts            // Call options to use throughout this session
}

// L1ScrollMessengerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type L1ScrollMessengerTransactorSession struct {
	Contract     *L1ScrollMessengerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts            // Transaction auth options to use throughout this session
}

// L1ScrollMessengerRaw is an auto generated low-level Go binding around an Ethereum contract.
type L1ScrollMessengerRaw struct {
	Contract *L1ScrollMessenger // Generic contract binding to access the raw methods on
}

// L1ScrollMessengerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type L1ScrollMessengerCallerRaw struct {
	Contract *L1ScrollMessengerCaller // Generic read-only contract binding to access the raw methods on
}

// L1ScrollMessengerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type L1ScrollMessengerTransactorRaw struct {
	Contract *L1ScrollMessengerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewL1ScrollMessenger creates a new instance of L1ScrollMessenger, bound to a specific deployed contract.
func NewL1ScrollMessenger(address common.Address, backend bind.ContractBackend) (*L1ScrollMessenger, error) {
	contract, err := bindL1ScrollMessenger(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &L1ScrollMessenger{L1ScrollMessengerCaller: L1ScrollMessengerCaller{contract: contract}, L1ScrollMessengerTransactor: L1ScrollMessengerTransactor{contract: contract}, L1ScrollMessengerFilterer: L1ScrollMessengerFilterer{contract: contract}}, nil
}

// NewL1ScrollMessengerCaller creates a new read-only instance of L1ScrollMessenger, bound to a specific deployed contract.
func NewL1ScrollMessengerCaller(address common.Address, caller bind.ContractCaller) (*L1ScrollMessengerCaller, error) {
	contract, err := bindL1ScrollMessenger(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &L1ScrollMessengerCaller{contract: contract}, nil
}

// NewL1ScrollMessengerTransactor creates a new write-only instance of L1ScrollMessenger, bound to a specific deployed contract.
func NewL1ScrollMessengerTransactor(address common.Address, transactor bind.ContractTransactor) (*L1ScrollMessengerTransactor, error) {
	contract, err := bindL1ScrollMessenger(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &L1ScrollMessengerTransactor{contract: contract}, nil
}

// NewL1ScrollMessengerFilterer creates a new log filterer instance of L1ScrollMessenger, bound to a specific deployed contract.
func NewL1ScrollMessengerFilterer(address common.Address, filterer bind.ContractFilterer) (*L1ScrollMessengerFilterer, error) {
	contract, err := bindL1ScrollMessenger(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &L1ScrollMessengerFilterer{contract: contract}, nil
}

// bindL1ScrollMessenger binds a generic wrapper to an already deployed contract.
func bindL1ScrollMessenger(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(L1ScrollMessengerABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L1ScrollMessenger *L1ScrollMessengerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L1ScrollMessenger.Contract.L1ScrollMessengerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L1ScrollMessenger *L1ScrollMessengerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1ScrollMessenger.Contract.L1ScrollMessengerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L1ScrollMessenger *L1ScrollMessengerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L1ScrollMessenger.Contract.L1ScrollMessengerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L1ScrollMessenger *L1ScrollMessengerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L1ScrollMessenger.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L1ScrollMessenger *L1ScrollMessengerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1ScrollMessenger.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L1ScrollMessenger *L1ScrollMessengerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L1ScrollMessenger.Contract.contract.Transact(opts, method, params...)
}

// DropDelayDuration is a free data retrieval call binding the contract method 0xe3048404.
//
// Solidity: function dropDelayDuration() view returns(uint256)
func (_L1ScrollMessenger *L1ScrollMessengerCaller) DropDelayDuration(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L1ScrollMessenger.contract.Call(opts, &out, "dropDelayDuration")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// DropDelayDuration is a free data retrieval call binding the contract method 0xe3048404.
//
// Solidity: function dropDelayDuration() view returns(uint256)
func (_L1ScrollMessenger *L1ScrollMessengerSession) DropDelayDuration() (*big.Int, error) {
	return _L1ScrollMessenger.Contract.DropDelayDuration(&_L1ScrollMessenger.CallOpts)
}

// DropDelayDuration is a free data retrieval call binding the contract method 0xe3048404.
//
// Solidity: function dropDelayDuration() view returns(uint256)
func (_L1ScrollMessenger *L1ScrollMessengerCallerSession) DropDelayDuration() (*big.Int, error) {
	return _L1ScrollMessenger.Contract.DropDelayDuration(&_L1ScrollMessenger.CallOpts)
}

// GasOracle is a free data retrieval call binding the contract method 0x5d62a8dd.
//
// Solidity: function gasOracle() view returns(address)
func (_L1ScrollMessenger *L1ScrollMessengerCaller) GasOracle(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1ScrollMessenger.contract.Call(opts, &out, "gasOracle")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GasOracle is a free data retrieval call binding the contract method 0x5d62a8dd.
//
// Solidity: function gasOracle() view returns(address)
func (_L1ScrollMessenger *L1ScrollMessengerSession) GasOracle() (common.Address, error) {
	return _L1ScrollMessenger.Contract.GasOracle(&_L1ScrollMessenger.CallOpts)
}

// GasOracle is a free data retrieval call binding the contract method 0x5d62a8dd.
//
// Solidity: function gasOracle() view returns(address)
func (_L1ScrollMessenger *L1ScrollMessengerCallerSession) GasOracle() (common.Address, error) {
	return _L1ScrollMessenger.Contract.GasOracle(&_L1ScrollMessenger.CallOpts)
}

// IsMessageDropped is a free data retrieval call binding the contract method 0x3df3390d.
//
// Solidity: function isMessageDropped(bytes32 ) view returns(bool)
func (_L1ScrollMessenger *L1ScrollMessengerCaller) IsMessageDropped(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var out []interface{}
	err := _L1ScrollMessenger.contract.Call(opts, &out, "isMessageDropped", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsMessageDropped is a free data retrieval call binding the contract method 0x3df3390d.
//
// Solidity: function isMessageDropped(bytes32 ) view returns(bool)
func (_L1ScrollMessenger *L1ScrollMessengerSession) IsMessageDropped(arg0 [32]byte) (bool, error) {
	return _L1ScrollMessenger.Contract.IsMessageDropped(&_L1ScrollMessenger.CallOpts, arg0)
}

// IsMessageDropped is a free data retrieval call binding the contract method 0x3df3390d.
//
// Solidity: function isMessageDropped(bytes32 ) view returns(bool)
func (_L1ScrollMessenger *L1ScrollMessengerCallerSession) IsMessageDropped(arg0 [32]byte) (bool, error) {
	return _L1ScrollMessenger.Contract.IsMessageDropped(&_L1ScrollMessenger.CallOpts, arg0)
}

// IsMessageExecuted is a free data retrieval call binding the contract method 0x396c16b7.
//
// Solidity: function isMessageExecuted(bytes32 ) view returns(bool)
func (_L1ScrollMessenger *L1ScrollMessengerCaller) IsMessageExecuted(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var out []interface{}
	err := _L1ScrollMessenger.contract.Call(opts, &out, "isMessageExecuted", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsMessageExecuted is a free data retrieval call binding the contract method 0x396c16b7.
//
// Solidity: function isMessageExecuted(bytes32 ) view returns(bool)
func (_L1ScrollMessenger *L1ScrollMessengerSession) IsMessageExecuted(arg0 [32]byte) (bool, error) {
	return _L1ScrollMessenger.Contract.IsMessageExecuted(&_L1ScrollMessenger.CallOpts, arg0)
}

// IsMessageExecuted is a free data retrieval call binding the contract method 0x396c16b7.
//
// Solidity: function isMessageExecuted(bytes32 ) view returns(bool)
func (_L1ScrollMessenger *L1ScrollMessengerCallerSession) IsMessageExecuted(arg0 [32]byte) (bool, error) {
	return _L1ScrollMessenger.Contract.IsMessageExecuted(&_L1ScrollMessenger.CallOpts, arg0)
}

// IsMessageRelayed is a free data retrieval call binding the contract method 0xf7f7469a.
//
// Solidity: function isMessageRelayed(bytes32 ) view returns(bool)
func (_L1ScrollMessenger *L1ScrollMessengerCaller) IsMessageRelayed(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var out []interface{}
	err := _L1ScrollMessenger.contract.Call(opts, &out, "isMessageRelayed", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsMessageRelayed is a free data retrieval call binding the contract method 0xf7f7469a.
//
// Solidity: function isMessageRelayed(bytes32 ) view returns(bool)
func (_L1ScrollMessenger *L1ScrollMessengerSession) IsMessageRelayed(arg0 [32]byte) (bool, error) {
	return _L1ScrollMessenger.Contract.IsMessageRelayed(&_L1ScrollMessenger.CallOpts, arg0)
}

// IsMessageRelayed is a free data retrieval call binding the contract method 0xf7f7469a.
//
// Solidity: function isMessageRelayed(bytes32 ) view returns(bool)
func (_L1ScrollMessenger *L1ScrollMessengerCallerSession) IsMessageRelayed(arg0 [32]byte) (bool, error) {
	return _L1ScrollMessenger.Contract.IsMessageRelayed(&_L1ScrollMessenger.CallOpts, arg0)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_L1ScrollMessenger *L1ScrollMessengerCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1ScrollMessenger.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_L1ScrollMessenger *L1ScrollMessengerSession) Owner() (common.Address, error) {
	return _L1ScrollMessenger.Contract.Owner(&_L1ScrollMessenger.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_L1ScrollMessenger *L1ScrollMessengerCallerSession) Owner() (common.Address, error) {
	return _L1ScrollMessenger.Contract.Owner(&_L1ScrollMessenger.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_L1ScrollMessenger *L1ScrollMessengerCaller) Paused(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _L1ScrollMessenger.contract.Call(opts, &out, "paused")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_L1ScrollMessenger *L1ScrollMessengerSession) Paused() (bool, error) {
	return _L1ScrollMessenger.Contract.Paused(&_L1ScrollMessenger.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_L1ScrollMessenger *L1ScrollMessengerCallerSession) Paused() (bool, error) {
	return _L1ScrollMessenger.Contract.Paused(&_L1ScrollMessenger.CallOpts)
}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_L1ScrollMessenger *L1ScrollMessengerCaller) Rollup(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1ScrollMessenger.contract.Call(opts, &out, "rollup")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_L1ScrollMessenger *L1ScrollMessengerSession) Rollup() (common.Address, error) {
	return _L1ScrollMessenger.Contract.Rollup(&_L1ScrollMessenger.CallOpts)
}

// Rollup is a free data retrieval call binding the contract method 0xcb23bcb5.
//
// Solidity: function rollup() view returns(address)
func (_L1ScrollMessenger *L1ScrollMessengerCallerSession) Rollup() (common.Address, error) {
	return _L1ScrollMessenger.Contract.Rollup(&_L1ScrollMessenger.CallOpts)
}

// Whitelist is a free data retrieval call binding the contract method 0x93e59dc1.
//
// Solidity: function whitelist() view returns(address)
func (_L1ScrollMessenger *L1ScrollMessengerCaller) Whitelist(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1ScrollMessenger.contract.Call(opts, &out, "whitelist")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Whitelist is a free data retrieval call binding the contract method 0x93e59dc1.
//
// Solidity: function whitelist() view returns(address)
func (_L1ScrollMessenger *L1ScrollMessengerSession) Whitelist() (common.Address, error) {
	return _L1ScrollMessenger.Contract.Whitelist(&_L1ScrollMessenger.CallOpts)
}

// Whitelist is a free data retrieval call binding the contract method 0x93e59dc1.
//
// Solidity: function whitelist() view returns(address)
func (_L1ScrollMessenger *L1ScrollMessengerCallerSession) Whitelist() (common.Address, error) {
	return _L1ScrollMessenger.Contract.Whitelist(&_L1ScrollMessenger.CallOpts)
}

// XDomainMessageSender is a free data retrieval call binding the contract method 0x6e296e45.
//
// Solidity: function xDomainMessageSender() view returns(address)
func (_L1ScrollMessenger *L1ScrollMessengerCaller) XDomainMessageSender(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L1ScrollMessenger.contract.Call(opts, &out, "xDomainMessageSender")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// XDomainMessageSender is a free data retrieval call binding the contract method 0x6e296e45.
//
// Solidity: function xDomainMessageSender() view returns(address)
func (_L1ScrollMessenger *L1ScrollMessengerSession) XDomainMessageSender() (common.Address, error) {
	return _L1ScrollMessenger.Contract.XDomainMessageSender(&_L1ScrollMessenger.CallOpts)
}

// XDomainMessageSender is a free data retrieval call binding the contract method 0x6e296e45.
//
// Solidity: function xDomainMessageSender() view returns(address)
func (_L1ScrollMessenger *L1ScrollMessengerCallerSession) XDomainMessageSender() (common.Address, error) {
	return _L1ScrollMessenger.Contract.XDomainMessageSender(&_L1ScrollMessenger.CallOpts)
}

// DropMessage is a paid mutator transaction binding the contract method 0xed885bfe.
//
// Solidity: function dropMessage(address _from, address _to, uint256 _value, uint256 _fee, uint256 _deadline, uint256 _nonce, bytes _message, uint256 _gasLimit) returns()
func (_L1ScrollMessenger *L1ScrollMessengerTransactor) DropMessage(opts *bind.TransactOpts, _from common.Address, _to common.Address, _value *big.Int, _fee *big.Int, _deadline *big.Int, _nonce *big.Int, _message []byte, _gasLimit *big.Int) (*types.Transaction, error) {
	return _L1ScrollMessenger.contract.Transact(opts, "dropMessage", _from, _to, _value, _fee, _deadline, _nonce, _message, _gasLimit)
}

// DropMessage is a paid mutator transaction binding the contract method 0xed885bfe.
//
// Solidity: function dropMessage(address _from, address _to, uint256 _value, uint256 _fee, uint256 _deadline, uint256 _nonce, bytes _message, uint256 _gasLimit) returns()
func (_L1ScrollMessenger *L1ScrollMessengerSession) DropMessage(_from common.Address, _to common.Address, _value *big.Int, _fee *big.Int, _deadline *big.Int, _nonce *big.Int, _message []byte, _gasLimit *big.Int) (*types.Transaction, error) {
	return _L1ScrollMessenger.Contract.DropMessage(&_L1ScrollMessenger.TransactOpts, _from, _to, _value, _fee, _deadline, _nonce, _message, _gasLimit)
}

// DropMessage is a paid mutator transaction binding the contract method 0xed885bfe.
//
// Solidity: function dropMessage(address _from, address _to, uint256 _value, uint256 _fee, uint256 _deadline, uint256 _nonce, bytes _message, uint256 _gasLimit) returns()
func (_L1ScrollMessenger *L1ScrollMessengerTransactorSession) DropMessage(_from common.Address, _to common.Address, _value *big.Int, _fee *big.Int, _deadline *big.Int, _nonce *big.Int, _message []byte, _gasLimit *big.Int) (*types.Transaction, error) {
	return _L1ScrollMessenger.Contract.DropMessage(&_L1ScrollMessenger.TransactOpts, _from, _to, _value, _fee, _deadline, _nonce, _message, _gasLimit)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _rollup) returns()
func (_L1ScrollMessenger *L1ScrollMessengerTransactor) Initialize(opts *bind.TransactOpts, _rollup common.Address) (*types.Transaction, error) {
	return _L1ScrollMessenger.contract.Transact(opts, "initialize", _rollup)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _rollup) returns()
func (_L1ScrollMessenger *L1ScrollMessengerSession) Initialize(_rollup common.Address) (*types.Transaction, error) {
	return _L1ScrollMessenger.Contract.Initialize(&_L1ScrollMessenger.TransactOpts, _rollup)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _rollup) returns()
func (_L1ScrollMessenger *L1ScrollMessengerTransactorSession) Initialize(_rollup common.Address) (*types.Transaction, error) {
	return _L1ScrollMessenger.Contract.Initialize(&_L1ScrollMessenger.TransactOpts, _rollup)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_L1ScrollMessenger *L1ScrollMessengerTransactor) Pause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1ScrollMessenger.contract.Transact(opts, "pause")
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_L1ScrollMessenger *L1ScrollMessengerSession) Pause() (*types.Transaction, error) {
	return _L1ScrollMessenger.Contract.Pause(&_L1ScrollMessenger.TransactOpts)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_L1ScrollMessenger *L1ScrollMessengerTransactorSession) Pause() (*types.Transaction, error) {
	return _L1ScrollMessenger.Contract.Pause(&_L1ScrollMessenger.TransactOpts)
}

// RelayMessageWithProof is a paid mutator transaction binding the contract method 0x52a3e089.
//
// Solidity: function relayMessageWithProof(address _from, address _to, uint256 _value, uint256 _fee, uint256 _deadline, uint256 _nonce, bytes _message, (uint256,uint256,bytes) _proof) returns()
func (_L1ScrollMessenger *L1ScrollMessengerTransactor) RelayMessageWithProof(opts *bind.TransactOpts, _from common.Address, _to common.Address, _value *big.Int, _fee *big.Int, _deadline *big.Int, _nonce *big.Int, _message []byte, _proof IL1ScrollMessengerL2MessageProof) (*types.Transaction, error) {
	return _L1ScrollMessenger.contract.Transact(opts, "relayMessageWithProof", _from, _to, _value, _fee, _deadline, _nonce, _message, _proof)
}

// RelayMessageWithProof is a paid mutator transaction binding the contract method 0x52a3e089.
//
// Solidity: function relayMessageWithProof(address _from, address _to, uint256 _value, uint256 _fee, uint256 _deadline, uint256 _nonce, bytes _message, (uint256,uint256,bytes) _proof) returns()
func (_L1ScrollMessenger *L1ScrollMessengerSession) RelayMessageWithProof(_from common.Address, _to common.Address, _value *big.Int, _fee *big.Int, _deadline *big.Int, _nonce *big.Int, _message []byte, _proof IL1ScrollMessengerL2MessageProof) (*types.Transaction, error) {
	return _L1ScrollMessenger.Contract.RelayMessageWithProof(&_L1ScrollMessenger.TransactOpts, _from, _to, _value, _fee, _deadline, _nonce, _message, _proof)
}

// RelayMessageWithProof is a paid mutator transaction binding the contract method 0x52a3e089.
//
// Solidity: function relayMessageWithProof(address _from, address _to, uint256 _value, uint256 _fee, uint256 _deadline, uint256 _nonce, bytes _message, (uint256,uint256,bytes) _proof) returns()
func (_L1ScrollMessenger *L1ScrollMessengerTransactorSession) RelayMessageWithProof(_from common.Address, _to common.Address, _value *big.Int, _fee *big.Int, _deadline *big.Int, _nonce *big.Int, _message []byte, _proof IL1ScrollMessengerL2MessageProof) (*types.Transaction, error) {
	return _L1ScrollMessenger.Contract.RelayMessageWithProof(&_L1ScrollMessenger.TransactOpts, _from, _to, _value, _fee, _deadline, _nonce, _message, _proof)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_L1ScrollMessenger *L1ScrollMessengerTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1ScrollMessenger.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_L1ScrollMessenger *L1ScrollMessengerSession) RenounceOwnership() (*types.Transaction, error) {
	return _L1ScrollMessenger.Contract.RenounceOwnership(&_L1ScrollMessenger.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_L1ScrollMessenger *L1ScrollMessengerTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _L1ScrollMessenger.Contract.RenounceOwnership(&_L1ScrollMessenger.TransactOpts)
}

// ReplayMessage is a paid mutator transaction binding the contract method 0x9545a747.
//
// Solidity: function replayMessage(address _from, address _to, uint256 _value, uint256 _fee, uint256 _deadline, bytes _message, uint256 _queueIndex, uint32 _oldGasLimit, uint32 _newGasLimit) returns()
func (_L1ScrollMessenger *L1ScrollMessengerTransactor) ReplayMessage(opts *bind.TransactOpts, _from common.Address, _to common.Address, _value *big.Int, _fee *big.Int, _deadline *big.Int, _message []byte, _queueIndex *big.Int, _oldGasLimit uint32, _newGasLimit uint32) (*types.Transaction, error) {
	return _L1ScrollMessenger.contract.Transact(opts, "replayMessage", _from, _to, _value, _fee, _deadline, _message, _queueIndex, _oldGasLimit, _newGasLimit)
}

// ReplayMessage is a paid mutator transaction binding the contract method 0x9545a747.
//
// Solidity: function replayMessage(address _from, address _to, uint256 _value, uint256 _fee, uint256 _deadline, bytes _message, uint256 _queueIndex, uint32 _oldGasLimit, uint32 _newGasLimit) returns()
func (_L1ScrollMessenger *L1ScrollMessengerSession) ReplayMessage(_from common.Address, _to common.Address, _value *big.Int, _fee *big.Int, _deadline *big.Int, _message []byte, _queueIndex *big.Int, _oldGasLimit uint32, _newGasLimit uint32) (*types.Transaction, error) {
	return _L1ScrollMessenger.Contract.ReplayMessage(&_L1ScrollMessenger.TransactOpts, _from, _to, _value, _fee, _deadline, _message, _queueIndex, _oldGasLimit, _newGasLimit)
}

// ReplayMessage is a paid mutator transaction binding the contract method 0x9545a747.
//
// Solidity: function replayMessage(address _from, address _to, uint256 _value, uint256 _fee, uint256 _deadline, bytes _message, uint256 _queueIndex, uint32 _oldGasLimit, uint32 _newGasLimit) returns()
func (_L1ScrollMessenger *L1ScrollMessengerTransactorSession) ReplayMessage(_from common.Address, _to common.Address, _value *big.Int, _fee *big.Int, _deadline *big.Int, _message []byte, _queueIndex *big.Int, _oldGasLimit uint32, _newGasLimit uint32) (*types.Transaction, error) {
	return _L1ScrollMessenger.Contract.ReplayMessage(&_L1ScrollMessenger.TransactOpts, _from, _to, _value, _fee, _deadline, _message, _queueIndex, _oldGasLimit, _newGasLimit)
}

// SendMessage is a paid mutator transaction binding the contract method 0xb2267a7b.
//
// Solidity: function sendMessage(address _to, uint256 _fee, bytes _message, uint256 _gasLimit) payable returns()
func (_L1ScrollMessenger *L1ScrollMessengerTransactor) SendMessage(opts *bind.TransactOpts, _to common.Address, _fee *big.Int, _message []byte, _gasLimit *big.Int) (*types.Transaction, error) {
	return _L1ScrollMessenger.contract.Transact(opts, "sendMessage", _to, _fee, _message, _gasLimit)
}

// SendMessage is a paid mutator transaction binding the contract method 0xb2267a7b.
//
// Solidity: function sendMessage(address _to, uint256 _fee, bytes _message, uint256 _gasLimit) payable returns()
func (_L1ScrollMessenger *L1ScrollMessengerSession) SendMessage(_to common.Address, _fee *big.Int, _message []byte, _gasLimit *big.Int) (*types.Transaction, error) {
	return _L1ScrollMessenger.Contract.SendMessage(&_L1ScrollMessenger.TransactOpts, _to, _fee, _message, _gasLimit)
}

// SendMessage is a paid mutator transaction binding the contract method 0xb2267a7b.
//
// Solidity: function sendMessage(address _to, uint256 _fee, bytes _message, uint256 _gasLimit) payable returns()
func (_L1ScrollMessenger *L1ScrollMessengerTransactorSession) SendMessage(_to common.Address, _fee *big.Int, _message []byte, _gasLimit *big.Int) (*types.Transaction, error) {
	return _L1ScrollMessenger.Contract.SendMessage(&_L1ScrollMessenger.TransactOpts, _to, _fee, _message, _gasLimit)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_L1ScrollMessenger *L1ScrollMessengerTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _L1ScrollMessenger.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_L1ScrollMessenger *L1ScrollMessengerSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _L1ScrollMessenger.Contract.TransferOwnership(&_L1ScrollMessenger.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_L1ScrollMessenger *L1ScrollMessengerTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _L1ScrollMessenger.Contract.TransferOwnership(&_L1ScrollMessenger.TransactOpts, newOwner)
}

// UpdateDropDelayDuration is a paid mutator transaction binding the contract method 0x7cecd1e5.
//
// Solidity: function updateDropDelayDuration(uint256 _newDuration) returns()
func (_L1ScrollMessenger *L1ScrollMessengerTransactor) UpdateDropDelayDuration(opts *bind.TransactOpts, _newDuration *big.Int) (*types.Transaction, error) {
	return _L1ScrollMessenger.contract.Transact(opts, "updateDropDelayDuration", _newDuration)
}

// UpdateDropDelayDuration is a paid mutator transaction binding the contract method 0x7cecd1e5.
//
// Solidity: function updateDropDelayDuration(uint256 _newDuration) returns()
func (_L1ScrollMessenger *L1ScrollMessengerSession) UpdateDropDelayDuration(_newDuration *big.Int) (*types.Transaction, error) {
	return _L1ScrollMessenger.Contract.UpdateDropDelayDuration(&_L1ScrollMessenger.TransactOpts, _newDuration)
}

// UpdateDropDelayDuration is a paid mutator transaction binding the contract method 0x7cecd1e5.
//
// Solidity: function updateDropDelayDuration(uint256 _newDuration) returns()
func (_L1ScrollMessenger *L1ScrollMessengerTransactorSession) UpdateDropDelayDuration(_newDuration *big.Int) (*types.Transaction, error) {
	return _L1ScrollMessenger.Contract.UpdateDropDelayDuration(&_L1ScrollMessenger.TransactOpts, _newDuration)
}

// UpdateGasOracle is a paid mutator transaction binding the contract method 0x70cee67f.
//
// Solidity: function updateGasOracle(address _newGasOracle) returns()
func (_L1ScrollMessenger *L1ScrollMessengerTransactor) UpdateGasOracle(opts *bind.TransactOpts, _newGasOracle common.Address) (*types.Transaction, error) {
	return _L1ScrollMessenger.contract.Transact(opts, "updateGasOracle", _newGasOracle)
}

// UpdateGasOracle is a paid mutator transaction binding the contract method 0x70cee67f.
//
// Solidity: function updateGasOracle(address _newGasOracle) returns()
func (_L1ScrollMessenger *L1ScrollMessengerSession) UpdateGasOracle(_newGasOracle common.Address) (*types.Transaction, error) {
	return _L1ScrollMessenger.Contract.UpdateGasOracle(&_L1ScrollMessenger.TransactOpts, _newGasOracle)
}

// UpdateGasOracle is a paid mutator transaction binding the contract method 0x70cee67f.
//
// Solidity: function updateGasOracle(address _newGasOracle) returns()
func (_L1ScrollMessenger *L1ScrollMessengerTransactorSession) UpdateGasOracle(_newGasOracle common.Address) (*types.Transaction, error) {
	return _L1ScrollMessenger.Contract.UpdateGasOracle(&_L1ScrollMessenger.TransactOpts, _newGasOracle)
}

// UpdateWhitelist is a paid mutator transaction binding the contract method 0x3d0f963e.
//
// Solidity: function updateWhitelist(address _newWhitelist) returns()
func (_L1ScrollMessenger *L1ScrollMessengerTransactor) UpdateWhitelist(opts *bind.TransactOpts, _newWhitelist common.Address) (*types.Transaction, error) {
	return _L1ScrollMessenger.contract.Transact(opts, "updateWhitelist", _newWhitelist)
}

// UpdateWhitelist is a paid mutator transaction binding the contract method 0x3d0f963e.
//
// Solidity: function updateWhitelist(address _newWhitelist) returns()
func (_L1ScrollMessenger *L1ScrollMessengerSession) UpdateWhitelist(_newWhitelist common.Address) (*types.Transaction, error) {
	return _L1ScrollMessenger.Contract.UpdateWhitelist(&_L1ScrollMessenger.TransactOpts, _newWhitelist)
}

// UpdateWhitelist is a paid mutator transaction binding the contract method 0x3d0f963e.
//
// Solidity: function updateWhitelist(address _newWhitelist) returns()
func (_L1ScrollMessenger *L1ScrollMessengerTransactorSession) UpdateWhitelist(_newWhitelist common.Address) (*types.Transaction, error) {
	return _L1ScrollMessenger.Contract.UpdateWhitelist(&_L1ScrollMessenger.TransactOpts, _newWhitelist)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_L1ScrollMessenger *L1ScrollMessengerTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L1ScrollMessenger.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_L1ScrollMessenger *L1ScrollMessengerSession) Receive() (*types.Transaction, error) {
	return _L1ScrollMessenger.Contract.Receive(&_L1ScrollMessenger.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_L1ScrollMessenger *L1ScrollMessengerTransactorSession) Receive() (*types.Transaction, error) {
	return _L1ScrollMessenger.Contract.Receive(&_L1ScrollMessenger.TransactOpts)
}

// L1ScrollMessengerFailedRelayedMessageIterator is returned from FilterFailedRelayedMessage and is used to iterate over the raw logs and unpacked data for FailedRelayedMessage events raised by the L1ScrollMessenger contract.
type L1ScrollMessengerFailedRelayedMessageIterator struct {
	Event *L1ScrollMessengerFailedRelayedMessage // Event containing the contract specifics and raw log

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
func (it *L1ScrollMessengerFailedRelayedMessageIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1ScrollMessengerFailedRelayedMessage)
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
		it.Event = new(L1ScrollMessengerFailedRelayedMessage)
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
func (it *L1ScrollMessengerFailedRelayedMessageIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1ScrollMessengerFailedRelayedMessageIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1ScrollMessengerFailedRelayedMessage represents a FailedRelayedMessage event raised by the L1ScrollMessenger contract.
type L1ScrollMessengerFailedRelayedMessage struct {
	MsgHash [32]byte
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterFailedRelayedMessage is a free log retrieval operation binding the contract event 0x99d0e048484baa1b1540b1367cb128acd7ab2946d1ed91ec10e3c85e4bf51b8f.
//
// Solidity: event FailedRelayedMessage(bytes32 indexed msgHash)
func (_L1ScrollMessenger *L1ScrollMessengerFilterer) FilterFailedRelayedMessage(opts *bind.FilterOpts, msgHash [][32]byte) (*L1ScrollMessengerFailedRelayedMessageIterator, error) {

	var msgHashRule []interface{}
	for _, msgHashItem := range msgHash {
		msgHashRule = append(msgHashRule, msgHashItem)
	}

	logs, sub, err := _L1ScrollMessenger.contract.FilterLogs(opts, "FailedRelayedMessage", msgHashRule)
	if err != nil {
		return nil, err
	}
	return &L1ScrollMessengerFailedRelayedMessageIterator{contract: _L1ScrollMessenger.contract, event: "FailedRelayedMessage", logs: logs, sub: sub}, nil
}

// WatchFailedRelayedMessage is a free log subscription operation binding the contract event 0x99d0e048484baa1b1540b1367cb128acd7ab2946d1ed91ec10e3c85e4bf51b8f.
//
// Solidity: event FailedRelayedMessage(bytes32 indexed msgHash)
func (_L1ScrollMessenger *L1ScrollMessengerFilterer) WatchFailedRelayedMessage(opts *bind.WatchOpts, sink chan<- *L1ScrollMessengerFailedRelayedMessage, msgHash [][32]byte) (event.Subscription, error) {

	var msgHashRule []interface{}
	for _, msgHashItem := range msgHash {
		msgHashRule = append(msgHashRule, msgHashItem)
	}

	logs, sub, err := _L1ScrollMessenger.contract.WatchLogs(opts, "FailedRelayedMessage", msgHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1ScrollMessengerFailedRelayedMessage)
				if err := _L1ScrollMessenger.contract.UnpackLog(event, "FailedRelayedMessage", log); err != nil {
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

// ParseFailedRelayedMessage is a log parse operation binding the contract event 0x99d0e048484baa1b1540b1367cb128acd7ab2946d1ed91ec10e3c85e4bf51b8f.
//
// Solidity: event FailedRelayedMessage(bytes32 indexed msgHash)
func (_L1ScrollMessenger *L1ScrollMessengerFilterer) ParseFailedRelayedMessage(log types.Log) (*L1ScrollMessengerFailedRelayedMessage, error) {
	event := new(L1ScrollMessengerFailedRelayedMessage)
	if err := _L1ScrollMessenger.contract.UnpackLog(event, "FailedRelayedMessage", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1ScrollMessengerMessageDroppedIterator is returned from FilterMessageDropped and is used to iterate over the raw logs and unpacked data for MessageDropped events raised by the L1ScrollMessenger contract.
type L1ScrollMessengerMessageDroppedIterator struct {
	Event *L1ScrollMessengerMessageDropped // Event containing the contract specifics and raw log

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
func (it *L1ScrollMessengerMessageDroppedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1ScrollMessengerMessageDropped)
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
		it.Event = new(L1ScrollMessengerMessageDropped)
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
func (it *L1ScrollMessengerMessageDroppedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1ScrollMessengerMessageDroppedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1ScrollMessengerMessageDropped represents a MessageDropped event raised by the L1ScrollMessenger contract.
type L1ScrollMessengerMessageDropped struct {
	MsgHash [32]byte
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterMessageDropped is a free log retrieval operation binding the contract event 0x6629230ca69c43f97674dd064896b819957583c8d20a870e4fb28b05c5d29f29.
//
// Solidity: event MessageDropped(bytes32 indexed msgHash)
func (_L1ScrollMessenger *L1ScrollMessengerFilterer) FilterMessageDropped(opts *bind.FilterOpts, msgHash [][32]byte) (*L1ScrollMessengerMessageDroppedIterator, error) {

	var msgHashRule []interface{}
	for _, msgHashItem := range msgHash {
		msgHashRule = append(msgHashRule, msgHashItem)
	}

	logs, sub, err := _L1ScrollMessenger.contract.FilterLogs(opts, "MessageDropped", msgHashRule)
	if err != nil {
		return nil, err
	}
	return &L1ScrollMessengerMessageDroppedIterator{contract: _L1ScrollMessenger.contract, event: "MessageDropped", logs: logs, sub: sub}, nil
}

// WatchMessageDropped is a free log subscription operation binding the contract event 0x6629230ca69c43f97674dd064896b819957583c8d20a870e4fb28b05c5d29f29.
//
// Solidity: event MessageDropped(bytes32 indexed msgHash)
func (_L1ScrollMessenger *L1ScrollMessengerFilterer) WatchMessageDropped(opts *bind.WatchOpts, sink chan<- *L1ScrollMessengerMessageDropped, msgHash [][32]byte) (event.Subscription, error) {

	var msgHashRule []interface{}
	for _, msgHashItem := range msgHash {
		msgHashRule = append(msgHashRule, msgHashItem)
	}

	logs, sub, err := _L1ScrollMessenger.contract.WatchLogs(opts, "MessageDropped", msgHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1ScrollMessengerMessageDropped)
				if err := _L1ScrollMessenger.contract.UnpackLog(event, "MessageDropped", log); err != nil {
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

// ParseMessageDropped is a log parse operation binding the contract event 0x6629230ca69c43f97674dd064896b819957583c8d20a870e4fb28b05c5d29f29.
//
// Solidity: event MessageDropped(bytes32 indexed msgHash)
func (_L1ScrollMessenger *L1ScrollMessengerFilterer) ParseMessageDropped(log types.Log) (*L1ScrollMessengerMessageDropped, error) {
	event := new(L1ScrollMessengerMessageDropped)
	if err := _L1ScrollMessenger.contract.UnpackLog(event, "MessageDropped", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1ScrollMessengerOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the L1ScrollMessenger contract.
type L1ScrollMessengerOwnershipTransferredIterator struct {
	Event *L1ScrollMessengerOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *L1ScrollMessengerOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1ScrollMessengerOwnershipTransferred)
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
		it.Event = new(L1ScrollMessengerOwnershipTransferred)
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
func (it *L1ScrollMessengerOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1ScrollMessengerOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1ScrollMessengerOwnershipTransferred represents a OwnershipTransferred event raised by the L1ScrollMessenger contract.
type L1ScrollMessengerOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_L1ScrollMessenger *L1ScrollMessengerFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*L1ScrollMessengerOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _L1ScrollMessenger.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &L1ScrollMessengerOwnershipTransferredIterator{contract: _L1ScrollMessenger.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_L1ScrollMessenger *L1ScrollMessengerFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *L1ScrollMessengerOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _L1ScrollMessenger.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1ScrollMessengerOwnershipTransferred)
				if err := _L1ScrollMessenger.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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

// ParseOwnershipTransferred is a log parse operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_L1ScrollMessenger *L1ScrollMessengerFilterer) ParseOwnershipTransferred(log types.Log) (*L1ScrollMessengerOwnershipTransferred, error) {
	event := new(L1ScrollMessengerOwnershipTransferred)
	if err := _L1ScrollMessenger.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1ScrollMessengerPausedIterator is returned from FilterPaused and is used to iterate over the raw logs and unpacked data for Paused events raised by the L1ScrollMessenger contract.
type L1ScrollMessengerPausedIterator struct {
	Event *L1ScrollMessengerPaused // Event containing the contract specifics and raw log

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
func (it *L1ScrollMessengerPausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1ScrollMessengerPaused)
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
		it.Event = new(L1ScrollMessengerPaused)
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
func (it *L1ScrollMessengerPausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1ScrollMessengerPausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1ScrollMessengerPaused represents a Paused event raised by the L1ScrollMessenger contract.
type L1ScrollMessengerPaused struct {
	Account common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterPaused is a free log retrieval operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_L1ScrollMessenger *L1ScrollMessengerFilterer) FilterPaused(opts *bind.FilterOpts) (*L1ScrollMessengerPausedIterator, error) {

	logs, sub, err := _L1ScrollMessenger.contract.FilterLogs(opts, "Paused")
	if err != nil {
		return nil, err
	}
	return &L1ScrollMessengerPausedIterator{contract: _L1ScrollMessenger.contract, event: "Paused", logs: logs, sub: sub}, nil
}

// WatchPaused is a free log subscription operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_L1ScrollMessenger *L1ScrollMessengerFilterer) WatchPaused(opts *bind.WatchOpts, sink chan<- *L1ScrollMessengerPaused) (event.Subscription, error) {

	logs, sub, err := _L1ScrollMessenger.contract.WatchLogs(opts, "Paused")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1ScrollMessengerPaused)
				if err := _L1ScrollMessenger.contract.UnpackLog(event, "Paused", log); err != nil {
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

// ParsePaused is a log parse operation binding the contract event 0x62e78cea01bee320cd4e420270b5ea74000d11b0c9f74754ebdbfc544b05a258.
//
// Solidity: event Paused(address account)
func (_L1ScrollMessenger *L1ScrollMessengerFilterer) ParsePaused(log types.Log) (*L1ScrollMessengerPaused, error) {
	event := new(L1ScrollMessengerPaused)
	if err := _L1ScrollMessenger.contract.UnpackLog(event, "Paused", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1ScrollMessengerRelayedMessageIterator is returned from FilterRelayedMessage and is used to iterate over the raw logs and unpacked data for RelayedMessage events raised by the L1ScrollMessenger contract.
type L1ScrollMessengerRelayedMessageIterator struct {
	Event *L1ScrollMessengerRelayedMessage // Event containing the contract specifics and raw log

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
func (it *L1ScrollMessengerRelayedMessageIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1ScrollMessengerRelayedMessage)
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
		it.Event = new(L1ScrollMessengerRelayedMessage)
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
func (it *L1ScrollMessengerRelayedMessageIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1ScrollMessengerRelayedMessageIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1ScrollMessengerRelayedMessage represents a RelayedMessage event raised by the L1ScrollMessenger contract.
type L1ScrollMessengerRelayedMessage struct {
	MsgHash [32]byte
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRelayedMessage is a free log retrieval operation binding the contract event 0x4641df4a962071e12719d8c8c8e5ac7fc4d97b927346a3d7a335b1f7517e133c.
//
// Solidity: event RelayedMessage(bytes32 indexed msgHash)
func (_L1ScrollMessenger *L1ScrollMessengerFilterer) FilterRelayedMessage(opts *bind.FilterOpts, msgHash [][32]byte) (*L1ScrollMessengerRelayedMessageIterator, error) {

	var msgHashRule []interface{}
	for _, msgHashItem := range msgHash {
		msgHashRule = append(msgHashRule, msgHashItem)
	}

	logs, sub, err := _L1ScrollMessenger.contract.FilterLogs(opts, "RelayedMessage", msgHashRule)
	if err != nil {
		return nil, err
	}
	return &L1ScrollMessengerRelayedMessageIterator{contract: _L1ScrollMessenger.contract, event: "RelayedMessage", logs: logs, sub: sub}, nil
}

// WatchRelayedMessage is a free log subscription operation binding the contract event 0x4641df4a962071e12719d8c8c8e5ac7fc4d97b927346a3d7a335b1f7517e133c.
//
// Solidity: event RelayedMessage(bytes32 indexed msgHash)
func (_L1ScrollMessenger *L1ScrollMessengerFilterer) WatchRelayedMessage(opts *bind.WatchOpts, sink chan<- *L1ScrollMessengerRelayedMessage, msgHash [][32]byte) (event.Subscription, error) {

	var msgHashRule []interface{}
	for _, msgHashItem := range msgHash {
		msgHashRule = append(msgHashRule, msgHashItem)
	}

	logs, sub, err := _L1ScrollMessenger.contract.WatchLogs(opts, "RelayedMessage", msgHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1ScrollMessengerRelayedMessage)
				if err := _L1ScrollMessenger.contract.UnpackLog(event, "RelayedMessage", log); err != nil {
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

// ParseRelayedMessage is a log parse operation binding the contract event 0x4641df4a962071e12719d8c8c8e5ac7fc4d97b927346a3d7a335b1f7517e133c.
//
// Solidity: event RelayedMessage(bytes32 indexed msgHash)
func (_L1ScrollMessenger *L1ScrollMessengerFilterer) ParseRelayedMessage(log types.Log) (*L1ScrollMessengerRelayedMessage, error) {
	event := new(L1ScrollMessengerRelayedMessage)
	if err := _L1ScrollMessenger.contract.UnpackLog(event, "RelayedMessage", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1ScrollMessengerSentMessageIterator is returned from FilterSentMessage and is used to iterate over the raw logs and unpacked data for SentMessage events raised by the L1ScrollMessenger contract.
type L1ScrollMessengerSentMessageIterator struct {
	Event *L1ScrollMessengerSentMessage // Event containing the contract specifics and raw log

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
func (it *L1ScrollMessengerSentMessageIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1ScrollMessengerSentMessage)
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
		it.Event = new(L1ScrollMessengerSentMessage)
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
func (it *L1ScrollMessengerSentMessageIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1ScrollMessengerSentMessageIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1ScrollMessengerSentMessage represents a SentMessage event raised by the L1ScrollMessenger contract.
type L1ScrollMessengerSentMessage struct {
	Target       common.Address
	Sender       common.Address
	Value        *big.Int
	Fee          *big.Int
	Deadline     *big.Int
	Message      []byte
	MessageNonce *big.Int
	GasLimit     *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterSentMessage is a free log retrieval operation binding the contract event 0x806b28931bc6fbe6c146babfb83d5c2b47e971edb43b4566f010577a0ee7d9f4.
//
// Solidity: event SentMessage(address indexed target, address sender, uint256 value, uint256 fee, uint256 deadline, bytes message, uint256 messageNonce, uint256 gasLimit)
func (_L1ScrollMessenger *L1ScrollMessengerFilterer) FilterSentMessage(opts *bind.FilterOpts, target []common.Address) (*L1ScrollMessengerSentMessageIterator, error) {

	var targetRule []interface{}
	for _, targetItem := range target {
		targetRule = append(targetRule, targetItem)
	}

	logs, sub, err := _L1ScrollMessenger.contract.FilterLogs(opts, "SentMessage", targetRule)
	if err != nil {
		return nil, err
	}
	return &L1ScrollMessengerSentMessageIterator{contract: _L1ScrollMessenger.contract, event: "SentMessage", logs: logs, sub: sub}, nil
}

// WatchSentMessage is a free log subscription operation binding the contract event 0x806b28931bc6fbe6c146babfb83d5c2b47e971edb43b4566f010577a0ee7d9f4.
//
// Solidity: event SentMessage(address indexed target, address sender, uint256 value, uint256 fee, uint256 deadline, bytes message, uint256 messageNonce, uint256 gasLimit)
func (_L1ScrollMessenger *L1ScrollMessengerFilterer) WatchSentMessage(opts *bind.WatchOpts, sink chan<- *L1ScrollMessengerSentMessage, target []common.Address) (event.Subscription, error) {

	var targetRule []interface{}
	for _, targetItem := range target {
		targetRule = append(targetRule, targetItem)
	}

	logs, sub, err := _L1ScrollMessenger.contract.WatchLogs(opts, "SentMessage", targetRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1ScrollMessengerSentMessage)
				if err := _L1ScrollMessenger.contract.UnpackLog(event, "SentMessage", log); err != nil {
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

// ParseSentMessage is a log parse operation binding the contract event 0x806b28931bc6fbe6c146babfb83d5c2b47e971edb43b4566f010577a0ee7d9f4.
//
// Solidity: event SentMessage(address indexed target, address sender, uint256 value, uint256 fee, uint256 deadline, bytes message, uint256 messageNonce, uint256 gasLimit)
func (_L1ScrollMessenger *L1ScrollMessengerFilterer) ParseSentMessage(log types.Log) (*L1ScrollMessengerSentMessage, error) {
	event := new(L1ScrollMessengerSentMessage)
	if err := _L1ScrollMessenger.contract.UnpackLog(event, "SentMessage", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1ScrollMessengerUnpausedIterator is returned from FilterUnpaused and is used to iterate over the raw logs and unpacked data for Unpaused events raised by the L1ScrollMessenger contract.
type L1ScrollMessengerUnpausedIterator struct {
	Event *L1ScrollMessengerUnpaused // Event containing the contract specifics and raw log

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
func (it *L1ScrollMessengerUnpausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1ScrollMessengerUnpaused)
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
		it.Event = new(L1ScrollMessengerUnpaused)
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
func (it *L1ScrollMessengerUnpausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1ScrollMessengerUnpausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1ScrollMessengerUnpaused represents a Unpaused event raised by the L1ScrollMessenger contract.
type L1ScrollMessengerUnpaused struct {
	Account common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterUnpaused is a free log retrieval operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_L1ScrollMessenger *L1ScrollMessengerFilterer) FilterUnpaused(opts *bind.FilterOpts) (*L1ScrollMessengerUnpausedIterator, error) {

	logs, sub, err := _L1ScrollMessenger.contract.FilterLogs(opts, "Unpaused")
	if err != nil {
		return nil, err
	}
	return &L1ScrollMessengerUnpausedIterator{contract: _L1ScrollMessenger.contract, event: "Unpaused", logs: logs, sub: sub}, nil
}

// WatchUnpaused is a free log subscription operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_L1ScrollMessenger *L1ScrollMessengerFilterer) WatchUnpaused(opts *bind.WatchOpts, sink chan<- *L1ScrollMessengerUnpaused) (event.Subscription, error) {

	logs, sub, err := _L1ScrollMessenger.contract.WatchLogs(opts, "Unpaused")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1ScrollMessengerUnpaused)
				if err := _L1ScrollMessenger.contract.UnpackLog(event, "Unpaused", log); err != nil {
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

// ParseUnpaused is a log parse operation binding the contract event 0x5db9ee0a495bf2e6ff9c91a7834c1ba4fdd244a5e8aa4e537bd38aeae4b073aa.
//
// Solidity: event Unpaused(address account)
func (_L1ScrollMessenger *L1ScrollMessengerFilterer) ParseUnpaused(log types.Log) (*L1ScrollMessengerUnpaused, error) {
	event := new(L1ScrollMessengerUnpaused)
	if err := _L1ScrollMessenger.contract.UnpackLog(event, "Unpaused", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1ScrollMessengerUpdateDropDelayDurationIterator is returned from FilterUpdateDropDelayDuration and is used to iterate over the raw logs and unpacked data for UpdateDropDelayDuration events raised by the L1ScrollMessenger contract.
type L1ScrollMessengerUpdateDropDelayDurationIterator struct {
	Event *L1ScrollMessengerUpdateDropDelayDuration // Event containing the contract specifics and raw log

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
func (it *L1ScrollMessengerUpdateDropDelayDurationIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1ScrollMessengerUpdateDropDelayDuration)
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
		it.Event = new(L1ScrollMessengerUpdateDropDelayDuration)
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
func (it *L1ScrollMessengerUpdateDropDelayDurationIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1ScrollMessengerUpdateDropDelayDurationIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1ScrollMessengerUpdateDropDelayDuration represents a UpdateDropDelayDuration event raised by the L1ScrollMessenger contract.
type L1ScrollMessengerUpdateDropDelayDuration struct {
	OldDuration *big.Int
	NewDuration *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterUpdateDropDelayDuration is a free log retrieval operation binding the contract event 0x8767db55656d87982bde23dfa77887931b21ecc3386f5764bf02ef0070d11742.
//
// Solidity: event UpdateDropDelayDuration(uint256 _oldDuration, uint256 _newDuration)
func (_L1ScrollMessenger *L1ScrollMessengerFilterer) FilterUpdateDropDelayDuration(opts *bind.FilterOpts) (*L1ScrollMessengerUpdateDropDelayDurationIterator, error) {

	logs, sub, err := _L1ScrollMessenger.contract.FilterLogs(opts, "UpdateDropDelayDuration")
	if err != nil {
		return nil, err
	}
	return &L1ScrollMessengerUpdateDropDelayDurationIterator{contract: _L1ScrollMessenger.contract, event: "UpdateDropDelayDuration", logs: logs, sub: sub}, nil
}

// WatchUpdateDropDelayDuration is a free log subscription operation binding the contract event 0x8767db55656d87982bde23dfa77887931b21ecc3386f5764bf02ef0070d11742.
//
// Solidity: event UpdateDropDelayDuration(uint256 _oldDuration, uint256 _newDuration)
func (_L1ScrollMessenger *L1ScrollMessengerFilterer) WatchUpdateDropDelayDuration(opts *bind.WatchOpts, sink chan<- *L1ScrollMessengerUpdateDropDelayDuration) (event.Subscription, error) {

	logs, sub, err := _L1ScrollMessenger.contract.WatchLogs(opts, "UpdateDropDelayDuration")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1ScrollMessengerUpdateDropDelayDuration)
				if err := _L1ScrollMessenger.contract.UnpackLog(event, "UpdateDropDelayDuration", log); err != nil {
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

// ParseUpdateDropDelayDuration is a log parse operation binding the contract event 0x8767db55656d87982bde23dfa77887931b21ecc3386f5764bf02ef0070d11742.
//
// Solidity: event UpdateDropDelayDuration(uint256 _oldDuration, uint256 _newDuration)
func (_L1ScrollMessenger *L1ScrollMessengerFilterer) ParseUpdateDropDelayDuration(log types.Log) (*L1ScrollMessengerUpdateDropDelayDuration, error) {
	event := new(L1ScrollMessengerUpdateDropDelayDuration)
	if err := _L1ScrollMessenger.contract.UnpackLog(event, "UpdateDropDelayDuration", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1ScrollMessengerUpdateGasOracleIterator is returned from FilterUpdateGasOracle and is used to iterate over the raw logs and unpacked data for UpdateGasOracle events raised by the L1ScrollMessenger contract.
type L1ScrollMessengerUpdateGasOracleIterator struct {
	Event *L1ScrollMessengerUpdateGasOracle // Event containing the contract specifics and raw log

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
func (it *L1ScrollMessengerUpdateGasOracleIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1ScrollMessengerUpdateGasOracle)
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
		it.Event = new(L1ScrollMessengerUpdateGasOracle)
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
func (it *L1ScrollMessengerUpdateGasOracleIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1ScrollMessengerUpdateGasOracleIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1ScrollMessengerUpdateGasOracle represents a UpdateGasOracle event raised by the L1ScrollMessenger contract.
type L1ScrollMessengerUpdateGasOracle struct {
	OldGasOracle common.Address
	NewGasOracle common.Address
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterUpdateGasOracle is a free log retrieval operation binding the contract event 0x9ed5ec28f252b3e7f62f1ace8e54c5ebabf4c61cc2a7c33a806365b2ff7ecc5e.
//
// Solidity: event UpdateGasOracle(address _oldGasOracle, address _newGasOracle)
func (_L1ScrollMessenger *L1ScrollMessengerFilterer) FilterUpdateGasOracle(opts *bind.FilterOpts) (*L1ScrollMessengerUpdateGasOracleIterator, error) {

	logs, sub, err := _L1ScrollMessenger.contract.FilterLogs(opts, "UpdateGasOracle")
	if err != nil {
		return nil, err
	}
	return &L1ScrollMessengerUpdateGasOracleIterator{contract: _L1ScrollMessenger.contract, event: "UpdateGasOracle", logs: logs, sub: sub}, nil
}

// WatchUpdateGasOracle is a free log subscription operation binding the contract event 0x9ed5ec28f252b3e7f62f1ace8e54c5ebabf4c61cc2a7c33a806365b2ff7ecc5e.
//
// Solidity: event UpdateGasOracle(address _oldGasOracle, address _newGasOracle)
func (_L1ScrollMessenger *L1ScrollMessengerFilterer) WatchUpdateGasOracle(opts *bind.WatchOpts, sink chan<- *L1ScrollMessengerUpdateGasOracle) (event.Subscription, error) {

	logs, sub, err := _L1ScrollMessenger.contract.WatchLogs(opts, "UpdateGasOracle")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1ScrollMessengerUpdateGasOracle)
				if err := _L1ScrollMessenger.contract.UnpackLog(event, "UpdateGasOracle", log); err != nil {
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

// ParseUpdateGasOracle is a log parse operation binding the contract event 0x9ed5ec28f252b3e7f62f1ace8e54c5ebabf4c61cc2a7c33a806365b2ff7ecc5e.
//
// Solidity: event UpdateGasOracle(address _oldGasOracle, address _newGasOracle)
func (_L1ScrollMessenger *L1ScrollMessengerFilterer) ParseUpdateGasOracle(log types.Log) (*L1ScrollMessengerUpdateGasOracle, error) {
	event := new(L1ScrollMessengerUpdateGasOracle)
	if err := _L1ScrollMessenger.contract.UnpackLog(event, "UpdateGasOracle", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L1ScrollMessengerUpdateWhitelistIterator is returned from FilterUpdateWhitelist and is used to iterate over the raw logs and unpacked data for UpdateWhitelist events raised by the L1ScrollMessenger contract.
type L1ScrollMessengerUpdateWhitelistIterator struct {
	Event *L1ScrollMessengerUpdateWhitelist // Event containing the contract specifics and raw log

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
func (it *L1ScrollMessengerUpdateWhitelistIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L1ScrollMessengerUpdateWhitelist)
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
		it.Event = new(L1ScrollMessengerUpdateWhitelist)
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
func (it *L1ScrollMessengerUpdateWhitelistIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L1ScrollMessengerUpdateWhitelistIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L1ScrollMessengerUpdateWhitelist represents a UpdateWhitelist event raised by the L1ScrollMessenger contract.
type L1ScrollMessengerUpdateWhitelist struct {
	OldWhitelist common.Address
	NewWhitelist common.Address
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterUpdateWhitelist is a free log retrieval operation binding the contract event 0x22d1c35fe072d2e42c3c8f9bd4a0d34aa84a0101d020a62517b33fdb3174e5f7.
//
// Solidity: event UpdateWhitelist(address _oldWhitelist, address _newWhitelist)
func (_L1ScrollMessenger *L1ScrollMessengerFilterer) FilterUpdateWhitelist(opts *bind.FilterOpts) (*L1ScrollMessengerUpdateWhitelistIterator, error) {

	logs, sub, err := _L1ScrollMessenger.contract.FilterLogs(opts, "UpdateWhitelist")
	if err != nil {
		return nil, err
	}
	return &L1ScrollMessengerUpdateWhitelistIterator{contract: _L1ScrollMessenger.contract, event: "UpdateWhitelist", logs: logs, sub: sub}, nil
}

// WatchUpdateWhitelist is a free log subscription operation binding the contract event 0x22d1c35fe072d2e42c3c8f9bd4a0d34aa84a0101d020a62517b33fdb3174e5f7.
//
// Solidity: event UpdateWhitelist(address _oldWhitelist, address _newWhitelist)
func (_L1ScrollMessenger *L1ScrollMessengerFilterer) WatchUpdateWhitelist(opts *bind.WatchOpts, sink chan<- *L1ScrollMessengerUpdateWhitelist) (event.Subscription, error) {

	logs, sub, err := _L1ScrollMessenger.contract.WatchLogs(opts, "UpdateWhitelist")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L1ScrollMessengerUpdateWhitelist)
				if err := _L1ScrollMessenger.contract.UnpackLog(event, "UpdateWhitelist", log); err != nil {
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

// ParseUpdateWhitelist is a log parse operation binding the contract event 0x22d1c35fe072d2e42c3c8f9bd4a0d34aa84a0101d020a62517b33fdb3174e5f7.
//
// Solidity: event UpdateWhitelist(address _oldWhitelist, address _newWhitelist)
func (_L1ScrollMessenger *L1ScrollMessengerFilterer) ParseUpdateWhitelist(log types.Log) (*L1ScrollMessengerUpdateWhitelist, error) {
	event := new(L1ScrollMessengerUpdateWhitelist)
	if err := _L1ScrollMessenger.contract.UnpackLog(event, "UpdateWhitelist", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
