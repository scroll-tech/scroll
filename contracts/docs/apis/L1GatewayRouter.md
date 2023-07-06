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
function depositETH(uint256 _amount, uint256 _gasLimit) external payable
```

Deposit ETH to caller&#39;s account in L2.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _amount | uint256 | undefined |
| _gasLimit | uint256 | undefined |

### depositETH

```solidity
function depositETH(address _to, uint256 _amount, uint256 _gasLimit) external payable
```

Deposit ETH to some recipient&#39;s account in L2.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _to | address | undefined |
| _amount | uint256 | undefined |
| _gasLimit | uint256 | undefined |

### depositETHAndCall

```solidity
function depositETHAndCall(address _to, uint256 _amount, bytes _data, uint256 _gasLimit) external payable
```

Deposit ETH to some recipient&#39;s account in L2 and call the target contract.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _to | address | undefined |
| _amount | uint256 | undefined |
| _data | bytes | undefined |
| _gasLimit | uint256 | undefined |

### ethGateway

```solidity
function ethGateway() external view returns (address)
```

The address of L1ETHGateway.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

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
function finalizeWithdrawETH(address, address, uint256, bytes) external payable
```

Complete ETH withdraw from L2 to L1 and send fund to recipient&#39;s account in L1.

*This function should only be called by L1ScrollMessenger.      This function should also only be called by L1ETHGateway in L2.*

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
function initialize(address _ethGateway, address _defaultERC20Gateway) external nonpayable
```

Initialize the storage of L1GatewayRouter.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _ethGateway | address | The address of L1ETHGateway contract. |
| _defaultERC20Gateway | address | The address of default ERC20 Gateway contract. |

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



*Leaves the contract without owner. It will not be possible to call `onlyOwner` functions. Can only be called by the current owner. NOTE: Renouncing ownership will leave the contract without an owner, thereby disabling any functionality that is only available to the owner.*


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



## Events

### DepositERC20

```solidity
event DepositERC20(address indexed l1Token, address indexed l2Token, address indexed from, address to, uint256 amount, bytes data)
```

Emitted when someone deposit ERC20 token from L1 to L2.



#### Parameters

| Name | Type | Description |
|---|---|---|
| l1Token `indexed` | address | undefined |
| l2Token `indexed` | address | undefined |
| from `indexed` | address | undefined |
| to  | address | undefined |
| amount  | uint256 | undefined |
| data  | bytes | undefined |

### DepositETH

```solidity
event DepositETH(address indexed from, address indexed to, uint256 amount, bytes data)
```

Emitted when someone deposit ETH from L1 to L2.



#### Parameters

| Name | Type | Description |
|---|---|---|
| from `indexed` | address | undefined |
| to `indexed` | address | undefined |
| amount  | uint256 | undefined |
| data  | bytes | undefined |

### FinalizeWithdrawERC20

```solidity
event FinalizeWithdrawERC20(address indexed l1Token, address indexed l2Token, address indexed from, address to, uint256 amount, bytes data)
```

Emitted when ERC20 token is withdrawn from L2 to L1 and transfer to recipient.



#### Parameters

| Name | Type | Description |
|---|---|---|
| l1Token `indexed` | address | undefined |
| l2Token `indexed` | address | undefined |
| from `indexed` | address | undefined |
| to  | address | undefined |
| amount  | uint256 | undefined |
| data  | bytes | undefined |

### FinalizeWithdrawETH

```solidity
event FinalizeWithdrawETH(address indexed from, address indexed to, uint256 amount, bytes data)
```

Emitted when ETH is withdrawn from L2 to L1 and transfer to recipient.



#### Parameters

| Name | Type | Description |
|---|---|---|
| from `indexed` | address | undefined |
| to `indexed` | address | undefined |
| amount  | uint256 | undefined |
| data  | bytes | undefined |

### Initialized

```solidity
event Initialized(uint8 version)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| version  | uint8 | undefined |

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
| defaultERC20Gateway `indexed` | address | undefined |

### SetERC20Gateway

```solidity
event SetERC20Gateway(address indexed token, address indexed gateway)
```

Emitted when the `gateway` for `token` is updated.



#### Parameters

| Name | Type | Description |
|---|---|---|
| token `indexed` | address | undefined |
| gateway `indexed` | address | undefined |

### SetETHGateway

```solidity
event SetETHGateway(address indexed ethGateway)
```

Emitted when the address of ETH Gateway is updated.



#### Parameters

| Name | Type | Description |
|---|---|---|
| ethGateway `indexed` | address | undefined |



