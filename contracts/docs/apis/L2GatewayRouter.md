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

### counterpart

```solidity
function counterpart() external view returns (address)
```

The address of corresponding L1/L2 Gateway contract.




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
function finalizeDepositETH(address _from, address _to, uint256 _amount, bytes _data) external payable
```

Complete ETH deposit from L1 to L2 and send fund to recipient&#39;s account in L2.

*This function should only be called by L2ScrollMessenger.      This function should also only be called by L1GatewayRouter in L1.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _from | address | The address of account who deposit ETH in L1. |
| _to | address | The address of recipient in L2 to receive ETH. |
| _amount | uint256 | The amount of ETH to deposit. |
| _data | bytes | Optional data to forward to recipient&#39;s account. |

### finalizeDropMessage

```solidity
function finalizeDropMessage() external payable
```






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
function initialize(address _defaultERC20Gateway, address _counterpart, address _messenger) external nonpayable
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _defaultERC20Gateway | address | undefined |
| _counterpart | address | undefined |
| _messenger | address | undefined |

### messenger

```solidity
function messenger() external view returns (address)
```

The address of L1ScrollMessenger/L2ScrollMessenger contract.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

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


### router

```solidity
function router() external view returns (address)
```

The address of L1GatewayRouter/L2GatewayRouter contract.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

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
| _token | address | The address of token in L2. |
| _amount | uint256 | The amount of token to transfer. |
| _gasLimit | uint256 | Unused, but included for potential forward compatibility considerations. |

### withdrawERC20

```solidity
function withdrawERC20(address _token, address _to, uint256 _amount, uint256 _gasLimit) external payable
```

Withdraw of some token to a recipient&#39;s account on L1.

*Make this function payable to send relayer fee in Ether.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _token | address | The address of token in L2. |
| _to | address | The address of recipient&#39;s account on L1. |
| _amount | uint256 | The amount of token to transfer. |
| _gasLimit | uint256 | Unused, but included for potential forward compatibility considerations. |

### withdrawERC20AndCall

```solidity
function withdrawERC20AndCall(address _token, address _to, uint256 _amount, bytes _data, uint256 _gasLimit) external payable
```

Withdraw of some token to a recipient&#39;s account on L1 and call.

*Make this function payable to send relayer fee in Ether.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _token | address | The address of token in L2. |
| _to | address | The address of recipient&#39;s account on L1. |
| _amount | uint256 | The amount of token to transfer. |
| _data | bytes | Optional data to forward to recipient&#39;s account. |
| _gasLimit | uint256 | Unused, but included for potential forward compatibility considerations. |

### withdrawETH

```solidity
function withdrawETH(address _to, uint256 _gasLimit) external payable
```

Withdraw ETH to caller&#39;s account in L1.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _to | address | The address of recipient&#39;s account on L1. |
| _gasLimit | uint256 | Gas limit required to complete the withdraw on L1. |

### withdrawETH

```solidity
function withdrawETH(uint256 _gasLimit) external payable
```

Withdraw ETH to caller&#39;s account in L1.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _gasLimit | uint256 | Gas limit required to complete the withdraw on L1. |



## Events

### FinalizeDepositERC20

```solidity
event FinalizeDepositERC20(address indexed _l1Token, address indexed _l2Token, address indexed _from, address _to, uint256 _amount, bytes _data)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token `indexed` | address | undefined |
| _l2Token `indexed` | address | undefined |
| _from `indexed` | address | undefined |
| _to  | address | undefined |
| _amount  | uint256 | undefined |
| _data  | bytes | undefined |

### FinalizeDepositETH

```solidity
event FinalizeDepositETH(address indexed _from, address indexed _to, uint256 _amount, bytes _data)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _from `indexed` | address | undefined |
| _to `indexed` | address | undefined |
| _amount  | uint256 | undefined |
| _data  | bytes | undefined |

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
event SetDefaultERC20Gateway(address indexed _defaultERC20Gateway)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _defaultERC20Gateway `indexed` | address | undefined |

### SetERC20Gateway

```solidity
event SetERC20Gateway(address indexed _token, address indexed _gateway)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _token `indexed` | address | undefined |
| _gateway `indexed` | address | undefined |

### WithdrawERC20

```solidity
event WithdrawERC20(address indexed _l1Token, address indexed _l2Token, address indexed _from, address _to, uint256 _amount, bytes _data)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token `indexed` | address | undefined |
| _l2Token `indexed` | address | undefined |
| _from `indexed` | address | undefined |
| _to  | address | undefined |
| _amount  | uint256 | undefined |
| _data  | bytes | undefined |

### WithdrawETH

```solidity
event WithdrawETH(address indexed _from, address indexed _to, uint256 _amount, bytes _data)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _from `indexed` | address | undefined |
| _to `indexed` | address | undefined |
| _amount  | uint256 | undefined |
| _data  | bytes | undefined |



