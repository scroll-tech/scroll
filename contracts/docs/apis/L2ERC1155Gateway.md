# L2ERC1155Gateway



> L2ERC1155Gateway

The `L2ERC1155Gateway` is used to withdraw ERC1155 compatible NFTs on layer 2 and finalize deposit the NFTs from layer 1.

*The withdrawn NFTs tokens will be burned directly. On finalizing deposit, the corresponding NFT will be minted and transferred to the recipient. This will be changed if we have more specific scenarios.*

## Methods

### batchWithdrawERC1155

```solidity
function batchWithdrawERC1155(address _token, uint256[] _tokenIds, uint256[] _amounts, uint256 _gasLimit) external payable
```

Batch withdraw a list of ERC1155 NFT to caller&#39;s account on layer 1.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _token | address | undefined |
| _tokenIds | uint256[] | undefined |
| _amounts | uint256[] | undefined |
| _gasLimit | uint256 | undefined |

### batchWithdrawERC1155

```solidity
function batchWithdrawERC1155(address _token, address _to, uint256[] _tokenIds, uint256[] _amounts, uint256 _gasLimit) external payable
```

Batch withdraw a list of ERC1155 NFT to caller&#39;s account on layer 1.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _token | address | undefined |
| _to | address | undefined |
| _tokenIds | uint256[] | undefined |
| _amounts | uint256[] | undefined |
| _gasLimit | uint256 | undefined |

### counterpart

```solidity
function counterpart() external view returns (address)
```

The address of corresponding L1/L2 Gateway contract.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

### finalizeBatchDepositERC1155

```solidity
function finalizeBatchDepositERC1155(address _l1Token, address _l2Token, address _from, address _to, uint256[] _tokenIds, uint256[] _amounts) external nonpayable
```

Complete ERC1155 deposit from layer 1 to layer 2 and send NFT to recipient&#39;s account on layer 2.

*Requirements:  - The function should only be called by L2ScrollMessenger.  - The function should also only be called by L1ERC1155Gateway on layer 1.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token | address | undefined |
| _l2Token | address | undefined |
| _from | address | undefined |
| _to | address | undefined |
| _tokenIds | uint256[] | undefined |
| _amounts | uint256[] | undefined |

### finalizeDepositERC1155

```solidity
function finalizeDepositERC1155(address _l1Token, address _l2Token, address _from, address _to, uint256 _tokenId, uint256 _amount) external nonpayable
```

Complete ERC1155 deposit from layer 1 to layer 2 and send NFT to recipient&#39;s account on layer 2.

*Requirements:  - The function should only be called by L2ScrollMessenger.  - The function should also only be called by L1ERC1155Gateway on layer 1.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token | address | undefined |
| _l2Token | address | undefined |
| _from | address | undefined |
| _to | address | undefined |
| _tokenId | uint256 | undefined |
| _amount | uint256 | undefined |

### initialize

```solidity
function initialize(address _counterpart, address _messenger) external nonpayable
```

Initialize the storage of `L2ERC1155Gateway`.

*The parameters `_counterpart` and `_messenger` are no longer used.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _counterpart | address | The address of `L1ERC1155Gateway` contract in L1. |
| _messenger | address | The address of `L2ScrollMessenger` contract in L2. |

### messenger

```solidity
function messenger() external view returns (address)
```

The address of corresponding L1ScrollMessenger/L2ScrollMessenger contract.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

### onERC1155BatchReceived

```solidity
function onERC1155BatchReceived(address, address, uint256[], uint256[], bytes) external nonpayable returns (bytes4)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |
| _1 | address | undefined |
| _2 | uint256[] | undefined |
| _3 | uint256[] | undefined |
| _4 | bytes | undefined |

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bytes4 | undefined |

### onERC1155Received

```solidity
function onERC1155Received(address, address, uint256, uint256, bytes) external nonpayable returns (bytes4)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |
| _1 | address | undefined |
| _2 | uint256 | undefined |
| _3 | uint256 | undefined |
| _4 | bytes | undefined |

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bytes4 | undefined |

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

### supportsInterface

```solidity
function supportsInterface(bytes4 interfaceId) external view returns (bool)
```



*See {IERC165-supportsInterface}.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| interfaceId | bytes4 | undefined |

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bool | undefined |

### tokenMapping

```solidity
function tokenMapping(address) external view returns (address)
```

Mapping from layer 2 token address to layer 1 token address for ERC1155 NFT.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

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

### updateTokenMapping

```solidity
function updateTokenMapping(address _l2Token, address _l1Token) external nonpayable
```

Update layer 2 to layer 1 token mapping.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _l2Token | address | The address of corresponding ERC1155 token on layer 2. |
| _l1Token | address | The address of ERC1155 token on layer 1. |

### withdrawERC1155

```solidity
function withdrawERC1155(address _token, uint256 _tokenId, uint256 _amount, uint256 _gasLimit) external payable
```

Withdraw some ERC1155 NFT to caller&#39;s account on layer 1.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _token | address | undefined |
| _tokenId | uint256 | undefined |
| _amount | uint256 | undefined |
| _gasLimit | uint256 | undefined |

### withdrawERC1155

```solidity
function withdrawERC1155(address _token, address _to, uint256 _tokenId, uint256 _amount, uint256 _gasLimit) external payable
```

Withdraw some ERC1155 NFT to caller&#39;s account on layer 1.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _token | address | undefined |
| _to | address | undefined |
| _tokenId | uint256 | undefined |
| _amount | uint256 | undefined |
| _gasLimit | uint256 | undefined |



## Events

### BatchWithdrawERC1155

```solidity
event BatchWithdrawERC1155(address indexed l1Token, address indexed l2Token, address indexed from, address to, uint256[] tokenIds, uint256[] amounts)
```

Emitted when the ERC1155 NFT is batch transferred to gateway on layer 2.



#### Parameters

| Name | Type | Description |
|---|---|---|
| l1Token `indexed` | address | undefined |
| l2Token `indexed` | address | undefined |
| from `indexed` | address | undefined |
| to  | address | undefined |
| tokenIds  | uint256[] | undefined |
| amounts  | uint256[] | undefined |

### FinalizeBatchDepositERC1155

```solidity
event FinalizeBatchDepositERC1155(address indexed l1Token, address indexed l2Token, address indexed from, address to, uint256[] tokenIds, uint256[] amounts)
```

Emitted when the ERC1155 NFT is batch transferred to recipient on layer 2.



#### Parameters

| Name | Type | Description |
|---|---|---|
| l1Token `indexed` | address | undefined |
| l2Token `indexed` | address | undefined |
| from `indexed` | address | undefined |
| to  | address | undefined |
| tokenIds  | uint256[] | undefined |
| amounts  | uint256[] | undefined |

### FinalizeDepositERC1155

```solidity
event FinalizeDepositERC1155(address indexed l1Token, address indexed l2Token, address indexed from, address to, uint256 tokenId, uint256 amount)
```

Emitted when the ERC1155 NFT is transferred to recipient on layer 2.



#### Parameters

| Name | Type | Description |
|---|---|---|
| l1Token `indexed` | address | undefined |
| l2Token `indexed` | address | undefined |
| from `indexed` | address | undefined |
| to  | address | undefined |
| tokenId  | uint256 | undefined |
| amount  | uint256 | undefined |

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

### UpdateTokenMapping

```solidity
event UpdateTokenMapping(address indexed l2Token, address indexed oldL1Token, address indexed newL1Token)
```

Emitted when token mapping for ERC1155 token is updated.



#### Parameters

| Name | Type | Description |
|---|---|---|
| l2Token `indexed` | address | The address of corresponding ERC1155 token in layer 2. |
| oldL1Token `indexed` | address | The address of the old corresponding ERC1155 token in layer 1. |
| newL1Token `indexed` | address | The address of the new corresponding ERC1155 token in layer 1. |

### WithdrawERC1155

```solidity
event WithdrawERC1155(address indexed l1Token, address indexed l2Token, address indexed from, address to, uint256 tokenId, uint256 amount)
```

Emitted when the ERC1155 NFT is transferred to gateway on layer 2.



#### Parameters

| Name | Type | Description |
|---|---|---|
| l1Token `indexed` | address | undefined |
| l2Token `indexed` | address | undefined |
| from `indexed` | address | undefined |
| to  | address | undefined |
| tokenId  | uint256 | undefined |
| amount  | uint256 | undefined |



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



