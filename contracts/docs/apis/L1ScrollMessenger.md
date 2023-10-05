# L1ScrollMessenger



> L1ScrollMessenger

The `L1ScrollMessenger` contract can: 1. send messages from layer 1 to layer 2; 2. relay messages from layer 2 layer 1; 3. replay failed message by replacing the gas limit; 4. drop expired message due to sequencer problems.

*All deposited Ether (including `WETH` deposited throng `L1WETHGateway`) will locked in this contract.*

## Methods

### counterpart

```solidity
function counterpart() external view returns (address)
```

The address of counterpart ScrollMessenger contract in L1/L2.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

### dropMessage

```solidity
function dropMessage(address _from, address _to, uint256 _value, uint256 _messageNonce, bytes _message) external nonpayable
```

Drop a skipped message.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _from | address | undefined |
| _to | address | undefined |
| _value | uint256 | undefined |
| _messageNonce | uint256 | undefined |
| _message | bytes | undefined |

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
function initialize(address _counterpart, address _feeVault, address _rollup, address _messageQueue) external nonpayable
```

Initialize the storage of L1ScrollMessenger.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _counterpart | address | The address of L2ScrollMessenger contract in L2. |
| _feeVault | address | The address of fee vault, which will be used to collect relayer fee. |
| _rollup | address | The address of ScrollChain contract. |
| _messageQueue | address | The address of L1MessageQueue contract. |

### isL1MessageDropped

```solidity
function isL1MessageDropped(bytes32) external view returns (bool)
```

Mapping from L1 message hash to drop status.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | bytes32 | undefined |

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bool | undefined |

### isL2MessageExecuted

```solidity
function isL2MessageExecuted(bytes32) external view returns (bool)
```

Mapping from L2 message hash to a boolean value indicating if the message has been successfully executed.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | bytes32 | undefined |

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bool | undefined |

### maxReplayTimes

```solidity
function maxReplayTimes() external view returns (uint256)
```

The maximum number of times each L1 message can be replayed.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined |

### messageQueue

```solidity
function messageQueue() external view returns (address)
```

The address of L1MessageQueue contract.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

### messageSendTimestamp

```solidity
function messageSendTimestamp(bytes32) external view returns (uint256)
```

Mapping from L1 message hash to the timestamp when the message is sent.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | bytes32 | undefined |

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined |

### owner

```solidity
function owner() external view returns (address)
```



*Returns the address of the current owner.*


#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

### paused

```solidity
function paused() external view returns (bool)
```



*Returns true if the contract is paused, and false otherwise.*


#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bool | undefined |

### prevReplayIndex

```solidity
function prevReplayIndex(uint256) external view returns (uint256)
```

Mapping from queue index to previous replay queue index.

*If a message `x` was replayed 3 times with index `q1`, `q2` and `q3`, the value of `prevReplayIndex` and `replayStates` will be `replayStates[hash(x)].lastIndex = q3`, `replayStates[hash(x)].times = 3`, `prevReplayIndex[q3] = q2`, `prevReplayIndex[q2] = q1`, `prevReplayIndex[q1] = x` and `prevReplayIndex[x]=nil`.The index `x` that `prevReplayIndex[x]=nil` is used as the termination of the list. Usually we use `0` to represent `nil`, but we cannot distinguish it with the first message with index zero. So a nonzero offset `1` is added to the value of `prevReplayIndex[x]` to avoid such situation.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined |

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined |

### relayMessageWithProof

```solidity
function relayMessageWithProof(address _from, address _to, uint256 _value, uint256 _nonce, bytes _message, IL1ScrollMessenger.L2MessageProof _proof) external nonpayable
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _from | address | undefined |
| _to | address | undefined |
| _value | uint256 | undefined |
| _nonce | uint256 | undefined |
| _message | bytes | undefined |
| _proof | IL1ScrollMessenger.L2MessageProof | undefined |

### renounceOwnership

```solidity
function renounceOwnership() external nonpayable
```



*Leaves the contract without owner. It will not be possible to call `onlyOwner` functions. Can only be called by the current owner. NOTE: Renouncing ownership will leave the contract without an owner, thereby disabling any functionality that is only available to the owner.*


### replayMessage

```solidity
function replayMessage(address _from, address _to, uint256 _value, uint256 _messageNonce, bytes _message, uint32 _newGasLimit, address _refundAddress) external payable
```

Replay an existing message.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _from | address | undefined |
| _to | address | undefined |
| _value | uint256 | undefined |
| _messageNonce | uint256 | undefined |
| _message | bytes | undefined |
| _newGasLimit | uint32 | undefined |
| _refundAddress | address | undefined |

### replayStates

```solidity
function replayStates(bytes32) external view returns (uint128 times, uint128 lastIndex)
```

Mapping from L1 message hash to replay state.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | bytes32 | undefined |

#### Returns

| Name | Type | Description |
|---|---|---|
| times | uint128 | undefined |
| lastIndex | uint128 | undefined |

### rollup

```solidity
function rollup() external view returns (address)
```

The address of Rollup contract.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

### sendMessage

```solidity
function sendMessage(address _to, uint256 _value, bytes _message, uint256 _gasLimit, address _refundAddress) external payable
```

Send cross chain message from L1 to L2 or L2 to L1.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _to | address | undefined |
| _value | uint256 | undefined |
| _message | bytes | undefined |
| _gasLimit | uint256 | undefined |
| _refundAddress | address | undefined |

### sendMessage

```solidity
function sendMessage(address _to, uint256 _value, bytes _message, uint256 _gasLimit) external payable
```

Send cross chain message from L1 to L2 or L2 to L1.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _to | address | undefined |
| _value | uint256 | undefined |
| _message | bytes | undefined |
| _gasLimit | uint256 | undefined |

### setPause

```solidity
function setPause(bool _status) external nonpayable
```

Pause the contract

*This function can only called by contract owner.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _status | bool | The pause status to update. |

### transferOwnership

```solidity
function transferOwnership(address newOwner) external nonpayable
```



*Transfers ownership of the contract to a new account (`newOwner`). Can only be called by the current owner.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| newOwner | address | undefined |

### updateFeeVault

```solidity
function updateFeeVault(address _newFeeVault) external nonpayable
```

Update fee vault contract.

*This function can only called by contract owner.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _newFeeVault | address | The address of new fee vault contract. |

### updateMaxReplayTimes

```solidity
function updateMaxReplayTimes(uint256 _newMaxReplayTimes) external nonpayable
```

Update max replay times.

*This function can only called by contract owner.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _newMaxReplayTimes | uint256 | The new max replay times. |

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
event FailedRelayedMessage(bytes32 indexed messageHash)
```

Emitted when a cross domain message is failed to relay.



#### Parameters

| Name | Type | Description |
|---|---|---|
| messageHash `indexed` | bytes32 | undefined |

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
event RelayedMessage(bytes32 indexed messageHash)
```

Emitted when a cross domain message is relayed successfully.



#### Parameters

| Name | Type | Description |
|---|---|---|
| messageHash `indexed` | bytes32 | undefined |

### SentMessage

```solidity
event SentMessage(address indexed sender, address indexed target, uint256 value, uint256 messageNonce, uint256 gasLimit, bytes message)
```

Emitted when a cross domain message is sent.



#### Parameters

| Name | Type | Description |
|---|---|---|
| sender `indexed` | address | undefined |
| target `indexed` | address | undefined |
| value  | uint256 | undefined |
| messageNonce  | uint256 | undefined |
| gasLimit  | uint256 | undefined |
| message  | bytes | undefined |

### Unpaused

```solidity
event Unpaused(address account)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| account  | address | undefined |

### UpdateFeeVault

```solidity
event UpdateFeeVault(address _oldFeeVault, address _newFeeVault)
```

Emitted when owner updates fee vault contract.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _oldFeeVault  | address | undefined |
| _newFeeVault  | address | undefined |

### UpdateMaxReplayTimes

```solidity
event UpdateMaxReplayTimes(uint256 oldMaxReplayTimes, uint256 newMaxReplayTimes)
```

Emitted when the maximum number of times each message can be replayed is updated.



#### Parameters

| Name | Type | Description |
|---|---|---|
| oldMaxReplayTimes  | uint256 | undefined |
| newMaxReplayTimes  | uint256 | undefined |



