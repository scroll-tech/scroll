# L2ScrollMessenger



> L2ScrollMessenger

The `L2ScrollMessenger` contract can: 1. send messages from layer 2 to layer 1; 2. relay messages from layer 1 layer 2; 3. drop expired message due to sequencer problems.

*It should be a predeployed contract in layer 2 and should hold infinite amount of Ether (Specifically, `uint256(-1)`), which can be initialized in Genesis Block.*

## Methods

### blockContainer

```solidity
function blockContainer() external view returns (address)
```

The contract contains the list of L1 blocks.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

### counterpart

```solidity
function counterpart() external view returns (address)
```

The address of L1ScrollMessenger contract in L1.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

### feeVault

```solidity
function feeVault() external view returns (address)
```

The address of fee vault, collecting cross domain messaging fee.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

### initialize

```solidity
function initialize(address _counterpart, address _feeVault) external nonpayable
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _counterpart | address | undefined |
| _feeVault | address | undefined |

### isMessageExecuted

```solidity
function isMessageExecuted(bytes32) external view returns (bool)
```

Mapping from message hash to execution status.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | bytes32 | undefined |

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bool | undefined |

### isMessageRelayed

```solidity
function isMessageRelayed(bytes32) external view returns (bool)
```

Mapping from relay id to relay status.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | bytes32 | undefined |

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bool | undefined |

### isMessageSent

```solidity
function isMessageSent(bytes32) external view returns (bool)
```

Mapping from message hash to sent status.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | bytes32 | undefined |

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bool | undefined |

### messageQueue

```solidity
function messageQueue() external view returns (address)
```

The address of L2MessageQueue.




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

### pause

```solidity
function pause() external nonpayable
```

Pause the contract

*This function can only called by contract owner.*


### paused

```solidity
function paused() external view returns (bool)
```



*Returns true if the contract is paused, and false otherwise.*


#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bool | undefined |

### relayMessage

```solidity
function relayMessage(address _from, address _to, uint256 _value, uint256 _nonce, bytes _message) external nonpayable
```

execute L1 =&gt; L2 message

*Make sure this is only called by privileged accounts.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _from | address | undefined |
| _to | address | undefined |
| _value | uint256 | undefined |
| _nonce | uint256 | undefined |
| _message | bytes | undefined |

### relayMessageWithProof

```solidity
function relayMessageWithProof(address _from, address _to, uint256 _value, uint256 _nonce, bytes _message, IL2ScrollMessenger.L1MessageProof _proof) external nonpayable
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _from | address | undefined |
| _to | address | undefined |
| _value | uint256 | undefined |
| _nonce | uint256 | undefined |
| _message | bytes | undefined |
| _proof | IL2ScrollMessenger.L1MessageProof | undefined |

### renounceOwnership

```solidity
function renounceOwnership() external nonpayable
```



*Leaves the contract without owner. It will not be possible to call `onlyOwner` functions anymore. Can only be called by the current owner. NOTE: Renouncing ownership will leave the contract without an owner, thereby removing any functionality that is only available to the owner.*


### sendMessage

```solidity
function sendMessage(address _to, uint256 _value, bytes _message, uint256) external payable
```

Send cross chain message from L1 to L2.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _to | address | undefined |
| _value | uint256 | undefined |
| _message | bytes | undefined |
| _3 | uint256 | undefined |

### transferOwnership

```solidity
function transferOwnership(address newOwner) external nonpayable
```



*Transfers ownership of the contract to a new account (`newOwner`). Can only be called by the current owner.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| newOwner | address | undefined |

### updateWhitelist

```solidity
function updateWhitelist(address _newWhitelist) external nonpayable
```

Update whitelist contract.

*This function can only called by contract owner.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _newWhitelist | address | The address of new whitelist contract. |

### verifyMessageExecutionStatus

```solidity
function verifyMessageExecutionStatus(bytes32 _blockHash, bytes32 _msgHash, bytes _proof) external view returns (bool)
```

Check whether the message is executed in the corresponding L1 block.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _blockHash | bytes32 | The block hash where the message should in. |
| _msgHash | bytes32 | The hash of the message to check. |
| _proof | bytes | The encoded storage proof from eth_getProof. |

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bool | bool Return true is the message is executed in L1, otherwise return false. |

### verifyMessageInclusionStatus

```solidity
function verifyMessageInclusionStatus(bytes32 _blockHash, bytes32 _msgHash, bytes _proof) external view returns (bool)
```

Check whether the l1 message is included in the corresponding L1 block.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _blockHash | bytes32 | The block hash where the message should in. |
| _msgHash | bytes32 | The hash of the message to check. |
| _proof | bytes | The encoded storage proof from eth_getProof. |

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bool | bool Return true is the message is included in L1, otherwise return false. |

### whitelist

```solidity
function whitelist() external view returns (address)
```

The whitelist contract to track the sender who can call `sendMessage` in ScrollMessenger.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

### xDomainMessageSender

```solidity
function xDomainMessageSender() external view returns (address)
```

See {IScrollMessenger-xDomainMessageSender}




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |



## Events

### FailedRelayedMessage

```solidity
event FailedRelayedMessage(bytes32 indexed msgHash)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| msgHash `indexed` | bytes32 | undefined |

### MessageDropped

```solidity
event MessageDropped(bytes32 indexed msgHash)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| msgHash `indexed` | bytes32 | undefined |

### OwnershipTransferred

```solidity
event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| previousOwner `indexed` | address | undefined |
| newOwner `indexed` | address | undefined |

### Paused

```solidity
event Paused(address account)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| account  | address | undefined |

### RelayedMessage

```solidity
event RelayedMessage(bytes32 indexed msgHash)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| msgHash `indexed` | bytes32 | undefined |

### SentMessage

```solidity
event SentMessage(address indexed sender, address indexed target, uint256 value, bytes message, uint256 messageNonce)
```

Emitted when a cross domain message is sent



#### Parameters

| Name | Type | Description |
|---|---|---|
| sender `indexed` | address | undefined |
| target `indexed` | address | undefined |
| value  | uint256 | undefined |
| message  | bytes | undefined |
| messageNonce  | uint256 | undefined |

### Unpaused

```solidity
event Unpaused(address account)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| account  | address | undefined |

### UpdateWhitelist

```solidity
event UpdateWhitelist(address _oldWhitelist, address _newWhitelist)
```

Emitted when owner updates whitelist contract.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _oldWhitelist  | address | undefined |
| _newWhitelist  | address | undefined |



