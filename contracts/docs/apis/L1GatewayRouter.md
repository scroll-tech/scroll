# L1GatewayRouter



> L1GatewayRouter

The `L1GatewayRouter` is the main entry for depositing Ether and ERC20 tokens. All deposited tokens are routed to corresponding gateways.

*One can also use this contract to query L1/L2 token address mapping. In the future, ERC-721 and ERC-1155 tokens will be added to the router too.*

## Methods

### ERC20Gateway

```solidity
function ERC20Gateway(address) external view returns (address)
```

Mapping from ERC20 token address to corresponding L1ERC20Gateway.



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

The addess of default ERC20 gateway, normally the L1StandardERC20Gateway contract.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

### depositERC20

```solidity
function depositERC20(address _token, uint256 _amount, uint256 _gasLimit) external payable
```

Deposit some token to a caller&#39;s account on L2.

*Make this function payable to send relayer fee in Ether.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _token | address | The address of token in L1. |
| _amount | uint256 | The amount of token to transfer. |
| _gasLimit | uint256 | Gas limit required to complete the deposit on L2. |

### depositERC20

```solidity
function depositERC20(address _token, address _to, uint256 _amount, uint256 _gasLimit) external payable
```

Deposit some token to a recipient&#39;s account on L2.

*Make this function payable to send relayer fee in Ether.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _token | address | The address of token in L1. |
| _to | address | The address of recipient&#39;s account on L2. |
| _amount | uint256 | The amount of token to transfer. |
| _gasLimit | uint256 | Gas limit required to complete the deposit on L2. |

### depositERC20AndCall

```solidity
function depositERC20AndCall(address _token, address _to, uint256 _amount, bytes _data, uint256 _gasLimit) external payable
```

Deposit some token to a recipient&#39;s account on L2 and call.

*Make this function payable to send relayer fee in Ether.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _token | address | The address of token in L1. |
| _to | address | The address of recipient&#39;s account on L2. |
| _amount | uint256 | The amount of token to transfer. |
| _data | bytes | Optional data to forward to recipient&#39;s account. |
| _gasLimit | uint256 | Gas limit required to complete the deposit on L2. |

### depositETH

```solidity
function depositETH(address _to, uint256 _gasLimit) external payable
```

Deposit ETH to recipient&#39;s account in L2.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _to | address | The address of recipient&#39;s account on L2. |
| _gasLimit | uint256 | Gas limit required to complete the deposit on L2. |

### depositETH

```solidity
function depositETH(uint256 _gasLimit) external payable
```

Deposit ETH to call&#39;s account in L2.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _gasLimit | uint256 | Gas limit required to complete the deposit on L2. |

### finalizeDropMessage

```solidity
function finalizeDropMessage() external payable
```






### finalizeWithdrawERC20

```solidity
function finalizeWithdrawERC20(address, address, address, address, uint256, bytes) external payable
```

Complete ERC20 withdraw from L2 to L1 and send fund to recipient&#39;s account in L1.

*Make this function payable to handle WETH deposit/withdraw.      The function should only be called by L1ScrollMessenger.      The function should also only be called by L2ERC20Gateway in L2.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |
| _1 | address | undefined |
| _2 | address | undefined |
| _3 | address | undefined |
| _4 | uint256 | undefined |
| _5 | bytes | undefined |

### finalizeWithdrawETH

```solidity
function finalizeWithdrawETH(address _from, address _to, uint256 _amount, bytes _data) external payable
```

Complete ETH withdraw from L2 to L1 and send fund to recipient&#39;s account in L1.

*This function should only be called by L1ScrollMessenger.      This function should also only be called by L2GatewayRouter in L2.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _from | address | The address of account who withdraw ETH in L2. |
| _to | address | The address of recipient in L1 to receive ETH. |
| _amount | uint256 | The amount of ETH to withdraw. |
| _data | bytes | Optional data to forward to recipient&#39;s account. |

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

### getL2ERC20Address

```solidity
function getL2ERC20Address(address _l1Address) external view returns (address)
```

Return the corresponding l2 token address given l1 token address.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Address | address | undefined |

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



## Events

### DepositERC20

```solidity
event DepositERC20(address indexed _l1Token, address indexed _l2Token, address indexed _from, address _to, uint256 _amount, bytes _data)
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

### DepositETH

```solidity
event DepositETH(address indexed _from, address indexed _to, uint256 _amount, bytes _data)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _from `indexed` | address | undefined |
| _to `indexed` | address | undefined |
| _amount  | uint256 | undefined |
| _data  | bytes | undefined |

### FinalizeWithdrawERC20

```solidity
event FinalizeWithdrawERC20(address indexed _l1Token, address indexed _l2Token, address indexed _from, address _to, uint256 _amount, bytes _data)
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

### FinalizeWithdrawETH

```solidity
event FinalizeWithdrawETH(address indexed _from, address indexed _to, uint256 _amount, bytes _data)
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



