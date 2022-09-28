# L2ERC721Gateway



> L2ERC721Gateway

The `L2ERC721Gateway` is used to withdraw ERC721 compatible NFTs in layer 2 and finalize deposit the NFTs from layer 1.

*The withdrawn NFTs tokens will be burned directly. On finalizing deposit, the corresponding NFT will be minted and transfered to the recipient. This will be changed if we have more specific scenarios.*

## Methods

### batchWithdrawERC721

```solidity
function batchWithdrawERC721(address _token, uint256[] _tokenIds, uint256 _gasLimit) external nonpayable
```

Batch withdraw a list of ERC721 NFT to caller&#39;s account on layer 1.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _token | address | The address of ERC721 NFT in layer 2. |
| _tokenIds | uint256[] | The list of token ids to withdraw. |
| _gasLimit | uint256 | Unused, but included for potential forward compatibility considerations. |

### batchWithdrawERC721

```solidity
function batchWithdrawERC721(address _token, address _to, uint256[] _tokenIds, uint256 _gasLimit) external nonpayable
```

Batch withdraw a list of ERC721 NFT to caller&#39;s account on layer 1.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _token | address | The address of ERC721 NFT in layer 2. |
| _to | address | The address of recipient in layer 1. |
| _tokenIds | uint256[] | The list of token ids to withdraw. |
| _gasLimit | uint256 | Unused, but included for potential forward compatibility considerations. |

### counterpart

```solidity
function counterpart() external view returns (address)
```

The address of corresponding L1/L2 Gateway contract.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

### finalizeBatchDepositERC721

```solidity
function finalizeBatchDepositERC721(address _l1Token, address _l2Token, address _from, address _to, uint256[] _tokenIds) external nonpayable
```

Complete ERC721 deposit from layer 1 to layer 2 and send NFT to recipient&#39;s account in layer 2.

*Requirements:  - The function should only be called by L2ScrollMessenger.  - The function should also only be called by L1ERC721Gateway in layer 1.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token | address | The address of corresponding layer 1 token. |
| _l2Token | address | The address of corresponding layer 2 token. |
| _from | address | The address of account who withdraw the token in layer 1. |
| _to | address | The address of recipient in layer 2 to receive the token. |
| _tokenIds | uint256[] | The list of token ids to withdraw. |

### finalizeDepositERC721

```solidity
function finalizeDepositERC721(address _l1Token, address _l2Token, address _from, address _to, uint256 _tokenId) external nonpayable
```

Complete ERC721 deposit from layer 1 to layer 2 and send NFT to recipient&#39;s account in layer 2.

*Requirements:  - The function should only be called by L2ScrollMessenger.  - The function should also only be called by L1ERC721Gateway in layer 1.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token | address | The address of corresponding layer 1 token. |
| _l2Token | address | The address of corresponding layer 2 token. |
| _from | address | The address of account who withdraw the token in layer 1. |
| _to | address | The address of recipient in layer 2 to receive the token. |
| _tokenId | uint256 | The token id to withdraw. |

### finalizeDropMessage

```solidity
function finalizeDropMessage() external payable
```






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

The address of L1ScrollMessenger/L2ScrollMessenger contract.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

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

### tokenMapping

```solidity
function tokenMapping(address) external view returns (address)
```

Mapping from layer 2 token address to layer 1 token address for ERC721 NFT.



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
| _l1Token | address | The address of ERC721 token in layer 1. |

### withdrawERC721

```solidity
function withdrawERC721(address _token, uint256 _tokenId, uint256 _gasLimit) external nonpayable
```

Withdraw some ERC721 NFT to caller&#39;s account on layer 1.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _token | address | The address of ERC721 NFT in layer 2. |
| _tokenId | uint256 | The token id to withdraw. |
| _gasLimit | uint256 | Unused, but included for potential forward compatibility considerations. |

### withdrawERC721

```solidity
function withdrawERC721(address _token, address _to, uint256 _tokenId, uint256 _gasLimit) external nonpayable
```

Withdraw some ERC721 NFT to caller&#39;s account on layer 1.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _token | address | The address of ERC721 NFT in layer 2. |
| _to | address | The address of recipient in layer 1. |
| _tokenId | uint256 | The token id to withdraw. |
| _gasLimit | uint256 | Unused, but included for potential forward compatibility considerations. |



## Events

### BatchWithdrawERC721

```solidity
event BatchWithdrawERC721(address indexed _l1Token, address indexed _l2Token, address indexed _from, address _to, uint256[] _tokenIds)
```

Emitted when the ERC721 NFT is batch transfered to gateway in layer 2.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token `indexed` | address | undefined |
| _l2Token `indexed` | address | undefined |
| _from `indexed` | address | undefined |
| _to  | address | undefined |
| _tokenIds  | uint256[] | undefined |

### FinalizeBatchDepositERC721

```solidity
event FinalizeBatchDepositERC721(address indexed _l1Token, address indexed _l2Token, address indexed _from, address _to, uint256[] _tokenIds)
```

Emitted when the ERC721 NFT is batch transfered to recipient in layer 2.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token `indexed` | address | undefined |
| _l2Token `indexed` | address | undefined |
| _from `indexed` | address | undefined |
| _to  | address | undefined |
| _tokenIds  | uint256[] | undefined |

### FinalizeDepositERC721

```solidity
event FinalizeDepositERC721(address indexed _l1Token, address indexed _l2Token, address indexed _from, address _to, uint256 _tokenId)
```

Emitted when the ERC721 NFT is transfered to recipient in layer 2.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token `indexed` | address | undefined |
| _l2Token `indexed` | address | undefined |
| _from `indexed` | address | undefined |
| _to  | address | undefined |
| _tokenId  | uint256 | undefined |

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

Emitted when token mapping for ERC721 token is updated.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _l2Token  | address | undefined |
| _l1Token  | address | The address of ERC721 token in layer 1. |

### WithdrawERC721

```solidity
event WithdrawERC721(address indexed _l1Token, address indexed _l2Token, address indexed _from, address _to, uint256 _tokenId)
```

Emitted when the ERC721 NFT is transfered to gateway in layer 2.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token `indexed` | address | undefined |
| _l2Token `indexed` | address | undefined |
| _from `indexed` | address | undefined |
| _to  | address | undefined |
| _tokenId  | uint256 | undefined |



