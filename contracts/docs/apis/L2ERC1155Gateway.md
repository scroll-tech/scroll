# L2ERC1155Gateway



> L2ERC1155Gateway

The `L2ERC1155Gateway` is used to withdraw ERC1155 compatible NFTs in layer 2 and finalize deposit the NFTs from layer 1.

*The withdrawn NFTs tokens will be burned directly. On finalizing deposit, the corresponding NFT will be minted and transfered to the recipient. This will be changed if we have more specific scenarios.*

## Methods

### batchWithdrawERC1155

```solidity
function batchWithdrawERC1155(address _token, uint256[] _tokenIds, uint256[] _amounts, uint256 _gasLimit) external nonpayable
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _token | address | undefined |
| _tokenIds | uint256[] | undefined |
| _amounts | uint256[] | undefined |
| _gasLimit | uint256 | undefined |

### batchWithdrawERC1155

```solidity
function batchWithdrawERC1155(address _token, address _to, uint256[] _tokenIds, uint256[] _amounts, uint256 _gasLimit) external nonpayable
```





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





#### Parameters

| Name | Type | Description |
|---|---|---|
| _counterpart | address | undefined |
| _messenger | address | undefined |

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
| _l2Token | address | undefined |
| _l1Token | address | The address of ERC1155 token in layer 1. |

### withdrawERC1155

```solidity
function withdrawERC1155(address _token, uint256 _tokenId, uint256 _amount, uint256 _gasLimit) external nonpayable
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _token | address | undefined |
| _tokenId | uint256 | undefined |
| _amount | uint256 | undefined |
| _gasLimit | uint256 | undefined |

### withdrawERC1155

```solidity
function withdrawERC1155(address _token, address _to, uint256 _tokenId, uint256 _amount, uint256 _gasLimit) external nonpayable
```





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
event BatchWithdrawERC1155(address indexed _l1Token, address indexed _l2Token, address indexed _from, address _to, uint256[] _tokenIds, uint256[] _amounts)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token `indexed` | address | undefined |
| _l2Token `indexed` | address | undefined |
| _from `indexed` | address | undefined |
| _to  | address | undefined |
| _tokenIds  | uint256[] | undefined |
| _amounts  | uint256[] | undefined |

### FinalizeBatchDepositERC1155

```solidity
event FinalizeBatchDepositERC1155(address indexed _l1Token, address indexed _l2Token, address indexed _from, address _to, uint256[] _tokenIds, uint256[] _amounts)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token `indexed` | address | undefined |
| _l2Token `indexed` | address | undefined |
| _from `indexed` | address | undefined |
| _to  | address | undefined |
| _tokenIds  | uint256[] | undefined |
| _amounts  | uint256[] | undefined |

### FinalizeDepositERC1155

```solidity
event FinalizeDepositERC1155(address indexed _l1Token, address indexed _l2Token, address indexed _from, address _to, uint256 _tokenId, uint256 _amount)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token `indexed` | address | undefined |
| _l2Token `indexed` | address | undefined |
| _from `indexed` | address | undefined |
| _to  | address | undefined |
| _tokenId  | uint256 | undefined |
| _amount  | uint256 | undefined |

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
event UpdateTokenMapping(address _l2Token, address _l1Token)
```

Emitted when token mapping for ERC1155 token is updated.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _l2Token  | address | undefined |
| _l1Token  | address | The address of ERC1155 token in layer 1. |

### WithdrawERC1155

```solidity
event WithdrawERC1155(address indexed _l1Token, address indexed _l2Token, address indexed _from, address _to, uint256 _tokenId, uint256 _amount)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token `indexed` | address | undefined |
| _l2Token `indexed` | address | undefined |
| _from `indexed` | address | undefined |
| _to  | address | undefined |
| _tokenId  | uint256 | undefined |
| _amount  | uint256 | undefined |



