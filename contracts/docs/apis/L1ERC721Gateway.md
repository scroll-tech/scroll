# L1ERC721Gateway



> L1ERC721Gateway

The `L1ERC721Gateway` is used to deposit ERC721 compatible NFT on layer 1 and finalize withdraw the NFTs from layer 2.

*The deposited NFTs are held in this gateway. On finalizing withdraw, the corresponding NFT will be transfer to the recipient directly. This will be changed if we have more specific scenarios.*

## Methods

### batchDepositERC721

```solidity
function batchDepositERC721(address _token, address _to, uint256[] _tokenIds, uint256 _gasLimit) external payable
```

Deposit a list of some ERC721 NFT to a recipient&#39;s account on layer 2.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _token | address | The address of ERC721 NFT on layer 1. |
| _to | address | The address of recipient on layer 2. |
| _tokenIds | uint256[] | The list of token ids to deposit. |
| _gasLimit | uint256 | Estimated gas limit required to complete the deposit on layer 2. |

### batchDepositERC721

```solidity
function batchDepositERC721(address _token, uint256[] _tokenIds, uint256 _gasLimit) external payable
```

Deposit a list of some ERC721 NFT to caller&#39;s account on layer 2.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _token | address | The address of ERC721 NFT on layer 1. |
| _tokenIds | uint256[] | The list of token ids to deposit. |
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

### depositERC721

```solidity
function depositERC721(address _token, address _to, uint256 _tokenId, uint256 _gasLimit) external payable
```

Deposit some ERC721 NFT to a recipient&#39;s account on layer 2.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _token | address | The address of ERC721 NFT on layer 1. |
| _to | address | The address of recipient on layer 2. |
| _tokenId | uint256 | The token id to deposit. |
| _gasLimit | uint256 | Estimated gas limit required to complete the deposit on layer 2. |

### depositERC721

```solidity
function depositERC721(address _token, uint256 _tokenId, uint256 _gasLimit) external payable
```

Deposit some ERC721 NFT to caller&#39;s account on layer 2.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _token | address | The address of ERC721 NFT on layer 1. |
| _tokenId | uint256 | The token id to deposit. |
| _gasLimit | uint256 | Estimated gas limit required to complete the deposit on layer 2. |

### finalizeBatchWithdrawERC721

```solidity
function finalizeBatchWithdrawERC721(address _l1Token, address _l2Token, address _from, address _to, uint256[] _tokenIds) external nonpayable
```

Complete ERC721 batch withdraw from layer 2 to layer 1 and send NFT to recipient&#39;s account on layer 1.

*Requirements:  - The function should only be called by L1ScrollMessenger.  - The function should also only be called by L2ERC721Gateway on layer 2.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token | address | The address of corresponding layer 1 token. |
| _l2Token | address | The address of corresponding layer 2 token. |
| _from | address | The address of account who withdraw the token on layer 2. |
| _to | address | The address of recipient on layer 1 to receive the token. |
| _tokenIds | uint256[] | The list of token ids to withdraw. |

### finalizeWithdrawERC721

```solidity
function finalizeWithdrawERC721(address _l1Token, address _l2Token, address _from, address _to, uint256 _tokenId) external nonpayable
```

Complete ERC721 withdraw from layer 2 to layer 1 and send NFT to recipient&#39;s account on layer 1.

*Requirements:  - The function should only be called by L1ScrollMessenger.  - The function should also only be called by L2ERC721Gateway on layer 2.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token | address | The address of corresponding layer 1 token. |
| _l2Token | address | The address of corresponding layer 2 token. |
| _from | address | The address of account who withdraw the token on layer 2. |
| _to | address | The address of recipient on layer 1 to receive the token. |
| _tokenId | uint256 | The token id to withdraw. |

### initialize

```solidity
function initialize(address _counterpart, address _messenger) external nonpayable
```

Initialize the storage of L1ERC721Gateway.

*The parameters `_counterpart` and `_messenger` are no longer used.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _counterpart | address | The address of L2ERC721Gateway in L2. |
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

### onERC721Received

```solidity
function onERC721Received(address, address, uint256, bytes) external nonpayable returns (bytes4)
```



*See {IERC721Receiver-onERC721Received}. Always returns `IERC721Receiver.onERC721Received.selector`.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |
| _1 | address | undefined |
| _2 | uint256 | undefined |
| _3 | bytes | undefined |

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

### tokenMapping

```solidity
function tokenMapping(address) external view returns (address)
```

Mapping from l1 token address to l2 token address for ERC721 NFT.



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
| _l1Token | address | The address of ERC721 token on layer 1. |
| _l2Token | address | The address of corresponding ERC721 token on layer 2. |



## Events

### BatchDepositERC721

```solidity
event BatchDepositERC721(address indexed _l1Token, address indexed _l2Token, address indexed _from, address _to, uint256[] _tokenIds)
```

Emitted when the ERC721 NFT is batch deposited to gateway on layer 1.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token `indexed` | address | undefined |
| _l2Token `indexed` | address | undefined |
| _from `indexed` | address | undefined |
| _to  | address | undefined |
| _tokenIds  | uint256[] | undefined |

### BatchRefundERC721

```solidity
event BatchRefundERC721(address indexed token, address indexed recipient, uint256[] tokenIds)
```

Emitted when a batch of ERC721 tokens are refunded.



#### Parameters

| Name | Type | Description |
|---|---|---|
| token `indexed` | address | undefined |
| recipient `indexed` | address | undefined |
| tokenIds  | uint256[] | undefined |

### DepositERC721

```solidity
event DepositERC721(address indexed _l1Token, address indexed _l2Token, address indexed _from, address _to, uint256 _tokenId)
```

Emitted when the ERC721 NFT is deposited to gateway on layer 1.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token `indexed` | address | undefined |
| _l2Token `indexed` | address | undefined |
| _from `indexed` | address | undefined |
| _to  | address | undefined |
| _tokenId  | uint256 | undefined |

### FinalizeBatchWithdrawERC721

```solidity
event FinalizeBatchWithdrawERC721(address indexed _l1Token, address indexed _l2Token, address indexed _from, address _to, uint256[] _tokenIds)
```

Emitted when the ERC721 NFT is batch transferred to recipient on layer 1.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token `indexed` | address | undefined |
| _l2Token `indexed` | address | undefined |
| _from `indexed` | address | undefined |
| _to  | address | undefined |
| _tokenIds  | uint256[] | undefined |

### FinalizeWithdrawERC721

```solidity
event FinalizeWithdrawERC721(address indexed _l1Token, address indexed _l2Token, address indexed _from, address _to, uint256 _tokenId)
```

Emitted when the ERC721 NFT is transferred to recipient on layer 1.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token `indexed` | address | undefined |
| _l2Token `indexed` | address | undefined |
| _from `indexed` | address | undefined |
| _to  | address | undefined |
| _tokenId  | uint256 | undefined |

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

### RefundERC721

```solidity
event RefundERC721(address indexed token, address indexed recipient, uint256 tokenId)
```

Emitted when some ERC721 token is refunded.



#### Parameters

| Name | Type | Description |
|---|---|---|
| token `indexed` | address | undefined |
| recipient `indexed` | address | undefined |
| tokenId  | uint256 | undefined |

### UpdateTokenMapping

```solidity
event UpdateTokenMapping(address indexed l1Token, address indexed oldL2Token, address indexed newL2Token)
```

Emitted when token mapping for ERC721 token is updated.



#### Parameters

| Name | Type | Description |
|---|---|---|
| l1Token `indexed` | address | The address of ERC721 token in layer 1. |
| oldL2Token `indexed` | address | The address of the old corresponding ERC721 token in layer 2. |
| newL2Token `indexed` | address | The address of the new corresponding ERC721 token in layer 2. |



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



