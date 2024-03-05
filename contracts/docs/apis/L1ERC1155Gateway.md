# L1ERC1155Gateway



> L1ERC1155Gateway

The `L1ERC1155Gateway` is used to deposit ERC1155 compatible NFT on layer 1 and finalize withdraw the NFTs from layer 2.

*The deposited NFTs are held in this gateway. On finalizing withdraw, the corresponding NFT will be transfer to the recipient directly. This will be changed if we have more specific scenarios.*

## Methods

### batchDepositERC1155

```solidity
function batchDepositERC1155(address _token, uint256[] _tokenIds, uint256[] _amounts, uint256 _gasLimit) external payable
```

Deposit a list of some ERC1155 NFT to caller&#39;s account on layer 2.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _token | address | The address of ERC1155 NFT on layer 1. |
| _tokenIds | uint256[] | The list of token ids to deposit. |
| _amounts | uint256[] | The list of corresponding number of token to deposit. |
| _gasLimit | uint256 | Estimated gas limit required to complete the deposit on layer 2. |

### batchDepositERC1155

```solidity
function batchDepositERC1155(address _token, address _to, uint256[] _tokenIds, uint256[] _amounts, uint256 _gasLimit) external payable
```

Deposit a list of some ERC1155 NFT to a recipient&#39;s account on layer 2.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _token | address | The address of ERC1155 NFT on layer 1. |
| _to | address | The address of recipient on layer 2. |
| _tokenIds | uint256[] | The list of token ids to deposit. |
| _amounts | uint256[] | The list of corresponding number of token to deposit. |
| _gasLimit | uint256 | Estimated gas limit required to complete the deposit on layer 2. |

### counterpart

```solidity
function counterpart() external view returns (address)
```

The address of corresponding L1/L2 Gateway contract.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

### depositERC1155

```solidity
function depositERC1155(address _token, address _to, uint256 _tokenId, uint256 _amount, uint256 _gasLimit) external payable
```

Deposit some ERC1155 NFT to a recipient&#39;s account on layer 2.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _token | address | The address of ERC1155 NFT on layer 1. |
| _to | address | The address of recipient on layer 2. |
| _tokenId | uint256 | The token id to deposit. |
| _amount | uint256 | The amount of token to deposit. |
| _gasLimit | uint256 | Estimated gas limit required to complete the deposit on layer 2. |

### depositERC1155

```solidity
function depositERC1155(address _token, uint256 _tokenId, uint256 _amount, uint256 _gasLimit) external payable
```

Deposit some ERC1155 NFT to caller&#39;s account on layer 2.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _token | address | The address of ERC1155 NFT on layer 1. |
| _tokenId | uint256 | The token id to deposit. |
| _amount | uint256 | The amount of token to deposit. |
| _gasLimit | uint256 | Estimated gas limit required to complete the deposit on layer 2. |

### finalizeBatchWithdrawERC1155

```solidity
function finalizeBatchWithdrawERC1155(address _l1Token, address _l2Token, address _from, address _to, uint256[] _tokenIds, uint256[] _amounts) external nonpayable
```

Complete ERC1155 batch withdraw from layer 2 to layer 1 and send fund to recipient&#39;s account on layer 1.      The function should only be called by L1ScrollMessenger.      The function should also only be called by L2ERC1155Gateway on layer 2.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token | address | The address of corresponding layer 1 token. |
| _l2Token | address | The address of corresponding layer 2 token. |
| _from | address | The address of account who withdraw the token on layer 2. |
| _to | address | The address of recipient on layer 1 to receive the token. |
| _tokenIds | uint256[] | The list of token ids to withdraw. |
| _amounts | uint256[] | The list of corresponding number of token to withdraw. |

### finalizeWithdrawERC1155

```solidity
function finalizeWithdrawERC1155(address _l1Token, address _l2Token, address _from, address _to, uint256 _tokenId, uint256 _amount) external nonpayable
```

Complete ERC1155 withdraw from layer 2 to layer 1 and send fund to recipient&#39;s account on layer 1.      The function should only be called by L1ScrollMessenger.      The function should also only be called by L2ERC1155Gateway on layer 2.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token | address | The address of corresponding layer 1 token. |
| _l2Token | address | The address of corresponding layer 2 token. |
| _from | address | The address of account who withdraw the token on layer 2. |
| _to | address | The address of recipient on layer 1 to receive the token. |
| _tokenId | uint256 | The token id to withdraw. |
| _amount | uint256 | The amount of token to withdraw. |

### initialize

```solidity
function initialize(address _counterpart, address _messenger) external nonpayable
```

Initialize the storage of L1ERC1155Gateway.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _counterpart | address | The address of L2ERC1155Gateway in L2. |
| _messenger | address | The address of L1ScrollMessenger. |

### messenger

```solidity
function messenger() external view returns (address)
```

The address of corresponding L1ScrollMessenger/L2ScrollMessenger contract.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

### onDropMessage

```solidity
function onDropMessage(bytes _message) external payable
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _message | bytes | undefined |

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

Mapping from l1 token address to l2 token address for ERC1155 NFT.



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
function updateTokenMapping(address _l1Token, address _l2Token) external nonpayable
```

Update layer 2 to layer 2 token mapping.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token | address | The address of ERC1155 token on layer 1. |
| _l2Token | address | The address of corresponding ERC1155 token on layer 2. |



## Events

### BatchDepositERC1155

```solidity
event BatchDepositERC1155(address indexed _l1Token, address indexed _l2Token, address indexed _from, address _to, uint256[] _tokenIds, uint256[] _amounts)
```

Emitted when the ERC1155 NFT is batch deposited to gateway on layer 1.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token `indexed` | address | undefined |
| _l2Token `indexed` | address | undefined |
| _from `indexed` | address | undefined |
| _to  | address | undefined |
| _tokenIds  | uint256[] | undefined |
| _amounts  | uint256[] | undefined |

### BatchRefundERC1155

```solidity
event BatchRefundERC1155(address indexed token, address indexed recipient, uint256[] tokenIds, uint256[] amounts)
```

Emitted when some ERC1155 token is refunded.



#### Parameters

| Name | Type | Description |
|---|---|---|
| token `indexed` | address | undefined |
| recipient `indexed` | address | undefined |
| tokenIds  | uint256[] | undefined |
| amounts  | uint256[] | undefined |

### DepositERC1155

```solidity
event DepositERC1155(address indexed _l1Token, address indexed _l2Token, address indexed _from, address _to, uint256 _tokenId, uint256 _amount)
```

Emitted when the ERC1155 NFT is deposited to gateway on layer 1.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token `indexed` | address | undefined |
| _l2Token `indexed` | address | undefined |
| _from `indexed` | address | undefined |
| _to  | address | undefined |
| _tokenId  | uint256 | undefined |
| _amount  | uint256 | undefined |

### FinalizeBatchWithdrawERC1155

```solidity
event FinalizeBatchWithdrawERC1155(address indexed _l1Token, address indexed _l2Token, address indexed _from, address _to, uint256[] _tokenIds, uint256[] _amounts)
```

Emitted when the ERC1155 NFT is batch transferred to recipient on layer 1.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token `indexed` | address | undefined |
| _l2Token `indexed` | address | undefined |
| _from `indexed` | address | undefined |
| _to  | address | undefined |
| _tokenIds  | uint256[] | undefined |
| _amounts  | uint256[] | undefined |

### FinalizeWithdrawERC1155

```solidity
event FinalizeWithdrawERC1155(address indexed _l1Token, address indexed _l2Token, address indexed _from, address _to, uint256 _tokenId, uint256 _amount)
```

Emitted when the ERC1155 NFT is transferred to recipient on layer 1.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token `indexed` | address | undefined |
| _l2Token `indexed` | address | undefined |
| _from `indexed` | address | undefined |
| _to  | address | undefined |
| _tokenId  | uint256 | undefined |
| _amount  | uint256 | undefined |

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

### RefundERC1155

```solidity
event RefundERC1155(address indexed token, address indexed recipient, uint256 tokenId, uint256 amount)
```

Emitted when some ERC1155 token is refunded.



#### Parameters

| Name | Type | Description |
|---|---|---|
| token `indexed` | address | undefined |
| recipient `indexed` | address | undefined |
| tokenId  | uint256 | undefined |
| amount  | uint256 | undefined |

### UpdateTokenMapping

```solidity
event UpdateTokenMapping(address indexed l1Token, address indexed oldL2Token, address indexed newL2Token)
```

Emitted when token mapping for ERC1155 token is updated.



#### Parameters

| Name | Type | Description |
|---|---|---|
| l1Token `indexed` | address | The address of ERC1155 token in layer 1. |
| oldL2Token `indexed` | address | The address of the old corresponding ERC1155 token in layer 2. |
| newL2Token `indexed` | address | The address of the new corresponding ERC1155 token in layer 2. |



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



