# L2GatewayRouter



> L2GatewayRouter

The `L2GatewayRouter` is the main entry for withdrawing Ether and ERC20 tokens. All deposited tokens are routed to corresponding gateways.

*One can also use this contract to query L1/L2 token address mapping. In the future, ERC-721 and ERC-1155 tokens will be added to the router too.*

## Methods

### ERC20Gateway

```solidity
function ERC20Gateway(address) external view returns (address)
```

Mapping from L2 ERC20 token address to corresponding L2ERC20Gateway.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

### defaultERC20Gateway

```solidity
function defaultERC20Gateway() external view returns (address)
```

The addess of default L2 ERC20 gateway, normally the L2StandardERC20Gateway contract.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

### ethGateway

```solidity
function ethGateway() external view returns (address)
```

The address of L2ETHGateway.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

### finalizeDepositERC20

```solidity
function finalizeDepositERC20(address, address, address, address, uint256, bytes) external payable
```

Complete a deposit from L1 to L2 and send fund to recipient&#39;s account in L2.

*Make this function payable to handle WETH deposit/withdraw.      The function should only be called by L2ScrollMessenger.      The function should also only be called by L1ERC20Gateway in L1.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |
| _1 | address | undefined |
| _2 | address | undefined |
| _3 | address | undefined |
| _4 | uint256 | undefined |
| _5 | bytes | undefined |

### finalizeDepositETH

```solidity
function finalizeDepositETH(address, address, uint256, bytes) external payable
```

Complete ETH deposit from L1 to L2 and send fund to recipient&#39;s account in L2.

*This function should only be called by L2ScrollMessenger.      This function should also only be called by L1GatewayRouter in L1.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |
| _1 | address | undefined |
| _2 | uint256 | undefined |
| _3 | bytes | undefined |

### getERC20Gateway

```solidity
function getERC20Gateway(address _token) external view returns (address)
```

Return the corresponding gateway address for given token address.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _token | address | The address of token to query. |

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

### getL1ERC20Address

```solidity
function getL1ERC20Address(address _l2Address) external view returns (address)
```

Return the corresponding l1 token address given l2 token address.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _l2Address | address | undefined |

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

### getL2ERC20Address

```solidity
function getL2ERC20Address(address) external pure returns (address)
```

Return the corresponding l2 token address given l1 token address.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

### initialize

```solidity
function initialize(address _ethGateway, address _defaultERC20Gateway) external nonpayable
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _ethGateway | address | undefined |
| _defaultERC20Gateway | address | undefined |

### owner

```solidity
function owner() external view returns (address)
```



*Returns the address of the current owner.*


#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

### renounceOwnership

```solidity
function renounceOwnership() external nonpayable
```



*Leaves the contract without owner. It will not be possible to call `onlyOwner` functions anymore. Can only be called by the current owner. NOTE: Renouncing ownership will leave the contract without an owner, thereby removing any functionality that is only available to the owner.*


### setDefaultERC20Gateway

```solidity
function setDefaultERC20Gateway(address _defaultERC20Gateway) external nonpayable
```

Update the address of default ERC20 gateway contract.

*This function should only be called by contract owner.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _defaultERC20Gateway | address | The address to update. |

### setERC20Gateway

```solidity
function setERC20Gateway(address[] _tokens, address[] _gateways) external nonpayable
```

Update the mapping from token address to gateway address.

*This function should only be called by contract owner.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _tokens | address[] | The list of addresses of tokens to update. |
| _gateways | address[] | The list of addresses of gateways to update. |

### setETHGateway

```solidity
function setETHGateway(address _ethGateway) external nonpayable
```

Update the address of ETH gateway contract.

*This function should only be called by contract owner.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _ethGateway | address | The address to update. |

### transferOwnership

```solidity
function transferOwnership(address newOwner) external nonpayable
```



*Transfers ownership of the contract to a new account (`newOwner`). Can only be called by the current owner.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| newOwner | address | undefined |

### withdrawERC20

```solidity
function withdrawERC20(address _token, uint256 _amount, uint256 _gasLimit) external payable
```

Withdraw of some token to a caller&#39;s account on L1.

*Make this function payable to send relayer fee in Ether.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _token | address | undefined |
| _amount | uint256 | undefined |
| _gasLimit | uint256 | undefined |

### withdrawERC20

```solidity
function withdrawERC20(address _token, address _to, uint256 _amount, uint256 _gasLimit) external payable
```

Withdraw of some token to a recipient&#39;s account on L1.

*Make this function payable to send relayer fee in Ether.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _token | address | undefined |
| _to | address | undefined |
| _amount | uint256 | undefined |
| _gasLimit | uint256 | undefined |

### withdrawERC20AndCall

```solidity
function withdrawERC20AndCall(address _token, address _to, uint256 _amount, bytes _data, uint256 _gasLimit) external payable
```

Withdraw of some token to a recipient&#39;s account on L1 and call.

*Make this function payable to send relayer fee in Ether.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _token | address | undefined |
| _to | address | undefined |
| _amount | uint256 | undefined |
| _data | bytes | undefined |
| _gasLimit | uint256 | undefined |

### withdrawETH

```solidity
function withdrawETH(address _to, uint256 _amount, uint256 _gasLimit) external payable
```

Withdraw ETH to caller&#39;s account in L1.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _to | address | undefined |
| _amount | uint256 | undefined |
| _gasLimit | uint256 | undefined |

### withdrawETH

```solidity
function withdrawETH(uint256 _amount, uint256 _gasLimit) external payable
```

Withdraw ETH to caller&#39;s account in L1.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _amount | uint256 | undefined |
| _gasLimit | uint256 | undefined |

### withdrawETHAndCall

```solidity
function withdrawETHAndCall(address _to, uint256 _amount, bytes _data, uint256 _gasLimit) external payable
```

Withdraw ETH to caller&#39;s account in L1.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _to | address | undefined |
| _amount | uint256 | undefined |
| _data | bytes | undefined |
| _gasLimit | uint256 | undefined |



## Events

### FinalizeDepositERC20

```solidity
event FinalizeDepositERC20(address indexed l1Token, address indexed l2Token, address indexed from, address to, uint256 amount, bytes data)
```

Emitted when ERC20 token is deposited from L1 to L2 and transfer to recipient.



#### Parameters

| Name | Type | Description |
|---|---|---|
| l1Token `indexed` | address | The address of the token in L1. |
| l2Token `indexed` | address | The address of the token in L2. |
| from `indexed` | address | The address of sender in L1. |
| to  | address | The address of recipient in L2. |
| amount  | uint256 | The amount of token withdrawn from L1 to L2. |
| data  | bytes | The optional calldata passed to recipient in L2. |

### FinalizeDepositETH

```solidity
event FinalizeDepositETH(address indexed from, address indexed to, uint256 amount, bytes data)
```

Emitted when ETH is deposited from L1 to L2 and transfer to recipient.



#### Parameters

| Name | Type | Description |
|---|---|---|
| from `indexed` | address | The address of sender in L1. |
| to `indexed` | address | The address of recipient in L2. |
| amount  | uint256 | The amount of ETH deposited from L1 to L2. |
| data  | bytes | The optional calldata passed to recipient in L2. |

### OwnershipTransferred

```solidity
event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| previousOwner `indexed` | address | undefined |
| newOwner `indexed` | address | undefined |

### SetDefaultERC20Gateway

```solidity
event SetDefaultERC20Gateway(address indexed defaultERC20Gateway)
```

Emitted when the address of default ERC20 Gateway is updated.



#### Parameters

| Name | Type | Description |
|---|---|---|
| defaultERC20Gateway `indexed` | address | The address of new default ERC20 Gateway. |

### SetERC20Gateway

```solidity
event SetERC20Gateway(address indexed token, address indexed gateway)
```

Emitted when the `gateway` for `token` is updated.



#### Parameters

| Name | Type | Description |
|---|---|---|
| token `indexed` | address | The address of token updated. |
| gateway `indexed` | address | The corresponding address of gateway updated. |

### SetETHGateway

```solidity
event SetETHGateway(address indexed ethGateway)
```

Emitted when the address of ETH Gateway is updated.



#### Parameters

| Name | Type | Description |
|---|---|---|
| ethGateway `indexed` | address | The address of new ETH Gateway. |

### WithdrawERC20

```solidity
event WithdrawERC20(address indexed l1Token, address indexed l2Token, address indexed from, address to, uint256 amount, bytes data)
```

Emitted when someone withdraw ERC20 token from L2 to L1.



#### Parameters

| Name | Type | Description |
|---|---|---|
| l1Token `indexed` | address | The address of the token in L1. |
| l2Token `indexed` | address | The address of the token in L2. |
| from `indexed` | address | The address of sender in L2. |
| to  | address | The address of recipient in L1. |
| amount  | uint256 | The amount of token will be deposited from L2 to L1. |
| data  | bytes | The optional calldata passed to recipient in L1. |

### WithdrawETH

```solidity
event WithdrawETH(address indexed from, address indexed to, uint256 amount, bytes data)
```

Emitted when someone withdraw ETH from L2 to L1.



#### Parameters

| Name | Type | Description |
|---|---|---|
| from `indexed` | address | The address of sender in L2. |
| to `indexed` | address | The address of recipient in L1. |
| amount  | uint256 | The amount of ETH will be deposited from L2 to L1. |
| data  | bytes | The optional calldata passed to recipient in L1. |



