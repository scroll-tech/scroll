# L2WETHGateway



> L2WETHGateway

The `L2WETHGateway` contract is used to withdraw `WETH` token in layer 2 and finalize deposit `WETH` from layer 1.

*The WETH tokens are not held in the gateway. It will first be unwrapped as Ether and then the Ether will be sent to the `L2ScrollMessenger` contract. On finalizing deposit, the Ether will be transfered from `L2ScrollMessenger`, then wrapped as WETH and finally transfer to recipient.*

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
| _l1Token | address | The address of corresponding L1 token. |
| _l2Token | address | The address of corresponding L2 token. |
| _from | address | The address of account who deposits the token in L1. |
| _to | address | The address of recipient in L2 to receive the token. |
| _amount | uint256 | The amount of the token to deposit. |
| _data | bytes | Optional data to forward to recipient&#39;s account. |

### finalizeDropMessage

```solidity
function finalizeDropMessage() external payable
```






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
function initialize(address _counterpart, address _router, address _messenger, address _WETH, address _l1WETH) external nonpayable
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _counterpart | address | undefined |
| _router | address | undefined |
| _messenger | address | undefined |
| _WETH | address | undefined |
| _l1WETH | address | undefined |

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

The address of L1ScrollMessenger/L2ScrollMessenger contract.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

### router

```solidity
function router() external view returns (address)
```

The address of L1GatewayRouter/L2GatewayRouter contract.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

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



