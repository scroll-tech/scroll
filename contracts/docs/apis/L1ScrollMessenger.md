# L1ScrollMessenger



> L1ScrollMessenger

The `L1ScrollMessenger` contract can: 1. send messages from layer 1 to layer 2; 2. relay messages from layer 2 layer 1; 3. replay failed message by replacing the gas limit; 4. drop expired message due to sequencer problems.

*All deposited Ether (including `WETH` deposited throng `L1WETHGateway`) will locked in this contract.*

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
function dropMessage(address _from, address _to, uint256 _value, uint256 _fee, uint256 _deadline, uint256 _nonce, bytes _message, uint256 _gasLimit) external nonpayable
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _from | address | undefined |
| _to | address | undefined |
| _value | uint256 | undefined |
| _fee | uint256 | undefined |
| _deadline | uint256 | undefined |
| _nonce | uint256 | undefined |
| _message | bytes | undefined |
| _gasLimit | uint256 | undefined |

### gasOracle

```solidity
function gasOracle() external view returns (address)
```

The gas oracle used to estimate transaction fee on layer 2.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

### initialize

```solidity
function initialize(address _rollup) external nonpayable
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _rollup | address | undefined |

### isMessageDropped

```solidity
function isMessageDropped(bytes32) external view returns (bool)
```

Mapping from message hash to drop status.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | bytes32 | undefined |

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bool | undefined |

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

### relayMessageWithProof

```solidity
function relayMessageWithProof(address _from, address _to, uint256 _value, uint256 _fee, uint256 _deadline, uint256 _nonce, bytes _message, IL1ScrollMessenger.L2MessageProof _proof) external nonpayable
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _from | address | undefined |
| _to | address | undefined |
| _value | uint256 | undefined |
| _fee | uint256 | undefined |
| _deadline | uint256 | undefined |
| _nonce | uint256 | undefined |
| _message | bytes | undefined |
| _proof | IL1ScrollMessenger.L2MessageProof | undefined |

### renounceOwnership

```solidity
function renounceOwnership() external nonpayable
```



*Leaves the contract without owner. It will not be possible to call `onlyOwner` functions anymore. Can only be called by the current owner. NOTE: Renouncing ownership will leave the contract without an owner, thereby removing any functionality that is only available to the owner.*


### replayMessage

```solidity
function replayMessage(address _from, address _to, uint256 _value, uint256 _fee, uint256 _deadline, bytes _message, uint256 _queueIndex, uint32 _oldGasLimit, uint32 _newGasLimit) external nonpayable
```

Replay an exsisting message.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _from | address | The address of the sender of the message. |
| _to | address | The address of the recipient of the message. |
| _value | uint256 | The msg.value passed to the message call. |
| _fee | uint256 | The amount of fee in ETH to charge. |
| _deadline | uint256 | The deadline of the message. |
| _message | bytes | The content of the message. |
| _queueIndex | uint256 | CTC Queue index for the message to replay. |
| _oldGasLimit | uint32 | Original gas limit used to send the message. |
| _newGasLimit | uint32 | New gas limit to be used for this message. |

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
function transferOwnership(address newOwner) external nonpayable
```



*Transfers ownership of the contract to a new account (`newOwner`). Can only be called by the current owner.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| newOwner | address | undefined |

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

### Unpaused

```solidity
event Unpaused(address account)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| account  | address | undefined |

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



