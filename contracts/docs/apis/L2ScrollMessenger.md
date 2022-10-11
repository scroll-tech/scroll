# L2ScrollMessenger



> L2ScrollMessenger

The `L2ScrollMessenger` contract can: 1. send messages from layer 2 to layer 1; 2. relay messages from layer 1 layer 2; 3. drop expired message due to sequencer problems.

*It should be a predeployed contract in layer 2 and should hold infinite amount of Ether (Specifically, `uint256(-1)`), which can be initialized in Genesis Block.*

## Methods

### dropDelayDuration

```solidity
function dropDelayDuration() external view returns (uint256)
```

The amount of seconds needed to wait if we want to drop message.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined |

### dropMessage

```solidity
function dropMessage(address, address, uint256, uint256, uint256, uint256, bytes, uint256) external nonpayable
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |
| _1 | address | undefined |
| _2 | uint256 | undefined |
| _3 | uint256 | undefined |
| _4 | uint256 | undefined |
| _5 | uint256 | undefined |
| _6 | bytes | undefined |
| _7 | uint256 | undefined |

### gasOracle

```solidity
function gasOracle() external view returns (address)
```

The gas oracle used to estimate transaction fee on layer 2.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

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

### messageNonce

```solidity
function messageNonce() external view returns (uint256)
```

Message nonce, used to avoid relay attack.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined |

### messagePasser

```solidity
function messagePasser() external view returns (contract L2ToL1MessagePasser)
```

Contract to store the sent message.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | contract L2ToL1MessagePasser | undefined |

### owner

```solidity
function owner() external view returns (address)
```

The address of the current owner.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

### relayMessage

```solidity
function relayMessage(address _from, address _to, uint256 _value, uint256 _fee, uint256 _deadline, uint256 _nonce, bytes _message) external nonpayable
```

execute L1 =&gt; L2 message

*Make sure this is only called by privileged accounts.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _from | address | The address of the sender of the message. |
| _to | address | The address of the recipient of the message. |
| _value | uint256 | The msg.value passed to the message call. |
| _fee | uint256 | The amount of fee in ETH to charge. |
| _deadline | uint256 | The deadline of the message. |
| _nonce | uint256 | The nonce of the message to avoid replay attack. |
| _message | bytes | The content of the message. |

### renounceOwnership

```solidity
function renounceOwnership() external nonpayable
```

Leaves the contract without owner. It will not be possible to call `onlyOwner` functions anymore. Can only be called by the current owner.

*Renouncing ownership will leave the contract without an owner, thereby removing any functionality that is only available to the owner.*


### sendMessage

```solidity
function sendMessage(address _to, uint256 _fee, bytes _message, uint256 _gasLimit) external payable
```

Send cross chain message (L1 =&gt; L2 or L2 =&gt; L1)

*Currently, only privileged accounts can call this function for safty. And adding an extra `_fee` variable make it more easy to upgrade to decentralized version.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _to | address | The address of account who recieve the message. |
| _fee | uint256 | The amount of fee in Ether the caller would like to pay to the relayer. |
| _message | bytes | The content of the message. |
| _gasLimit | uint256 | Unused, but included for potential forward compatibility considerations. |

### transferOwnership

```solidity
function transferOwnership(address _newOwner) external nonpayable
```

Transfers ownership of the contract to a new account (`newOwner`). Can only be called by the current owner.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _newOwner | address | undefined |

### updateDropDelayDuration

```solidity
function updateDropDelayDuration(uint256 _newDuration) external nonpayable
```

Update the drop delay duration.

*This function can only called by contract owner.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _newDuration | uint256 | The new delay duration to update. |

### updateGasOracle

```solidity
function updateGasOracle(address _newGasOracle) external nonpayable
```

Update the address of gas oracle.

*This function can only called by contract owner.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _newGasOracle | address | The address to update. |

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
event OwnershipTransferred(address indexed _oldOwner, address indexed _newOwner)
```

Emitted when owner is changed by current owner.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _oldOwner `indexed` | address | undefined |
| _newOwner `indexed` | address | undefined |

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
event SentMessage(address indexed target, address sender, uint256 value, uint256 fee, uint256 deadline, bytes message, uint256 messageNonce, uint256 gasLimit)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| target `indexed` | address | undefined |
| sender  | address | undefined |
| value  | uint256 | undefined |
| fee  | uint256 | undefined |
| deadline  | uint256 | undefined |
| message  | bytes | undefined |
| messageNonce  | uint256 | undefined |
| gasLimit  | uint256 | undefined |

### UpdateDropDelayDuration

```solidity
event UpdateDropDelayDuration(uint256 _oldDuration, uint256 _newDuration)
```

Emitted when owner updates drop delay duration



#### Parameters

| Name | Type | Description |
|---|---|---|
| _oldDuration  | uint256 | undefined |
| _newDuration  | uint256 | undefined |

### UpdateGasOracle

```solidity
event UpdateGasOracle(address _oldGasOracle, address _newGasOracle)
```

Emitted when owner updates gas oracle contract.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _oldGasOracle  | address | undefined |
| _newGasOracle  | address | undefined |

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



