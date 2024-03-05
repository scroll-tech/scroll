# L2WETHGateway



> L2WETHGateway

The `L2WETHGateway` contract is used to withdraw `WETH` token on layer 2 and finalize deposit `WETH` from layer 1.

*The WETH tokens are not held in the gateway. It will first be unwrapped as Ether and then the Ether will be sent to the `L2ScrollMessenger` contract. On finalizing deposit, the Ether will be transferred from `L2ScrollMessenger`, then wrapped as WETH and finally transfer to recipient.*

## Methods

### WETH

```solidity
function WETH() external view returns (address)
```

The address of L2 WETH address.




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

### finalizeDepositERC20

```solidity
function finalizeDepositERC20(address _l1Token, address _l2Token, address _from, address _to, uint256 _amount, bytes _data) external payable
```

Complete a deposit from L1 to L2 and send fund to recipient&#39;s account in L2.

*Make this function payable to handle WETH deposit/withdraw.      The function should only be called by L2ScrollMessenger.      The function should also only be called by L1ERC20Gateway in L1.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token | address | undefined |
| _l2Token | address | undefined |
| _from | address | undefined |
| _to | address | undefined |
| _amount | uint256 | undefined |
| _data | bytes | undefined |

### getL1ERC20Address

```solidity
function getL1ERC20Address(address) external view returns (address)
```

Return the corresponding l1 token address given l2 token address.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

### getL2ERC20Address

```solidity
function getL2ERC20Address(address) external view returns (address)
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
function initialize(address _counterpart, address _router, address _messenger) external nonpayable
```

Initialize the storage of `L2WETHGateway`.

*The parameters `_counterpart`, `_router` and `_messenger` are no longer used.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _counterpart | address | The address of `L1WETHGateway` contract in L1. |
| _router | address | The address of `L2GatewayRouter` contract in L2. |
| _messenger | address | The address of `L2ScrollMessenger` contract in L2. |

### l1WETH

```solidity
function l1WETH() external view returns (address)
```

The address of L1 WETH address.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

### messenger

```solidity
function messenger() external view returns (address)
```

The address of corresponding L1ScrollMessenger/L2ScrollMessenger contract.




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



*Leaves the contract without owner. It will not be possible to call `onlyOwner` functions. Can only be called by the current owner. NOTE: Renouncing ownership will leave the contract without an owner, thereby disabling any functionality that is only available to the owner.*


### router

```solidity
function router() external view returns (address)
```

The address of L1GatewayRouter/L2GatewayRouter contract.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

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



## Events

### FinalizeDepositERC20

```solidity
event FinalizeDepositERC20(address indexed l1Token, address indexed l2Token, address indexed from, address to, uint256 amount, bytes data)
```

Emitted when ERC20 token is deposited from L1 to L2 and transfer to recipient.



#### Parameters

| Name | Type | Description |
|---|---|---|
| l1Token `indexed` | address | undefined |
| l2Token `indexed` | address | undefined |
| from `indexed` | address | undefined |
| to  | address | undefined |
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

### WithdrawERC20

```solidity
event WithdrawERC20(address indexed l1Token, address indexed l2Token, address indexed from, address to, uint256 amount, bytes data)
```

Emitted when someone withdraw ERC20 token from L2 to L1.



#### Parameters

| Name | Type | Description |
|---|---|---|
| l1Token `indexed` | address | undefined |
| l2Token `indexed` | address | undefined |
| from `indexed` | address | undefined |
| to  | address | undefined |
| amount  | uint256 | undefined |
| data  | bytes | undefined |



## Errors

### ErrorCallerIsNotCounterpartGateway

```solidity
error ErrorCallerIsNotCounterpartGateway()
```



*Thrown when the cross chain sender is not the counterpart gateway contract.*


### ErrorCallerIsNotMessenger

```solidity
error ErrorCallerIsNotMessenger()
```



*Thrown when the caller is not corresponding `L1ScrollMessenger` or `L2ScrollMessenger`.*


### ErrorNotInDropMessageContext

```solidity
error ErrorNotInDropMessageContext()
```



*Thrown when ScrollMessenger is not dropping message.*


### ErrorZeroAddress

```solidity
error ErrorZeroAddress()
```



*Thrown when the given address is `address(0)`.*



