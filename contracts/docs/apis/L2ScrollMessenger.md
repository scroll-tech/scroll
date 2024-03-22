# L2ScrollMessenger



> L2ScrollMessenger

The `L2ScrollMessenger` contract can: 1. send messages from layer 2 to layer 1; 2. relay messages from layer 1 layer 2; 3. drop expired message due to sequencer problems.

*It should be a predeployed contract on layer 2 and should hold infinite amount of Ether (Specifically, `uint256(-1)`), which can be initialized in Genesis Block.*

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
function initialize(address) external nonpayable
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

### isL1MessageExecuted

```solidity
function isL1MessageExecuted(bytes32) external view returns (bool)
```

Mapping from L1 message hash to a boolean value indicating if the message has been successfully executed.



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

### messageSendTimestamp

```solidity
function messageSendTimestamp(bytes32) external view returns (uint256)
```

Mapping from L2 message hash to the timestamp when the message is sent.



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

### renounceOwnership

```solidity
function renounceOwnership() external nonpayable
```



*Leaves the contract without owner. It will not be possible to call `onlyOwner` functions. Can only be called by the current owner. NOTE: Renouncing ownership will leave the contract without an owner, thereby disabling any functionality that is only available to the owner.*


### sendMessage

```solidity
function sendMessage(address _to, uint256 _value, bytes _message, uint256 _gasLimit, address) external payable
```

Send cross chain message from L1 to L2 or L2 to L1.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _to | address | undefined |
| _value | uint256 | undefined |
| _message | bytes | undefined |
| _gasLimit | uint256 | undefined |
| _4 | address | undefined |

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
| messageHash `indexed` | bytes32 | The hash of the message. |

### Initialized

```solidity
event Initialized(uint8 version)
```



*Triggered when the contract has been initialized or reinitialized.*

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



*Emitted when the pause is triggered by `account`.*

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
| messageHash `indexed` | bytes32 | The hash of the message. |

### SentMessage

```solidity
event SentMessage(address indexed sender, address indexed target, uint256 value, uint256 messageNonce, uint256 gasLimit, bytes message)
```

Emitted when a cross domain message is sent.



#### Parameters

| Name | Type | Description |
|---|---|---|
| sender `indexed` | address | The address of the sender who initiates the message. |
| target `indexed` | address | The address of target contract to call. |
| value  | uint256 | The amount of value passed to the target contract. |
| messageNonce  | uint256 | The nonce of the message. |
| gasLimit  | uint256 | The optional gas limit passed to L1 or L2. |
| message  | bytes | The calldata passed to the target contract. |

### Unpaused

```solidity
event Unpaused(address account)
```



*Emitted when the pause is lifted by `account`.*

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
| _oldFeeVault  | address | The address of old fee vault contract. |
| _newFeeVault  | address | The address of new fee vault contract. |

### UpdateMaxFailedExecutionTimes

```solidity
event UpdateMaxFailedExecutionTimes(uint256 oldMaxFailedExecutionTimes, uint256 newMaxFailedExecutionTimes)
```

Emitted when the maximum number of times each message can fail in L2 is updated.



#### Parameters

| Name | Type | Description |
|---|---|---|
| oldMaxFailedExecutionTimes  | uint256 | The old maximum number of times each message can fail in L2. |
| newMaxFailedExecutionTimes  | uint256 | The new maximum number of times each message can fail in L2. |



## Errors

### ErrorZeroAddress

```solidity
error ErrorZeroAddress()
```



*Thrown when the given address is `address(0)`.*



