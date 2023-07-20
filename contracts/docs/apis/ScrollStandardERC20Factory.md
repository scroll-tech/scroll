# ScrollStandardERC20Factory



> ScrollStandardERC20Factory

The `ScrollStandardERC20Factory` is used to deploy `ScrollStandardERC20` for `L2StandardERC20Gateway`. It uses the `Clones` contract to deploy contract with minimum gas usage.

*The implementation of deployed token is non-upgradable. This design may be changed in the future.*

## Methods

### computeL2TokenAddress

```solidity
function computeL2TokenAddress(address _gateway, address _l1Token) external view returns (address)
```

Compute the corresponding l2 token address given l1 token address.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _gateway | address | The address of gateway contract. |
| _l1Token | address | The address of l1 token. |

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

### deployL2Token

```solidity
function deployL2Token(address _gateway, address _l1Token) external nonpayable returns (address)
```

Deploy the corresponding l2 token address given l1 token address.

*This function should only be called by owner to avoid DDoS attack on StandardTokenBridge.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _gateway | address | The address of gateway contract. |
| _l1Token | address | The address of l1 token. |

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

### implementation

```solidity
function implementation() external view returns (address)
```

The address of `ScrollStandardERC20` implementation.




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

### DeployToken

```solidity
event DeployToken(address indexed _l1Token, address indexed _l2Token)
```

Emitted when a l2 token is deployed.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token `indexed` | address | undefined |
| _l2Token `indexed` | address | undefined |

### OwnershipTransferred

```solidity
event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| previousOwner `indexed` | address | undefined |
| newOwner `indexed` | address | undefined |



