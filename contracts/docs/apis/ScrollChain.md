# ScrollChain



> ScrollChain

This contract maintains data for the Scroll rollup.



## Methods

### commitBatch

```solidity
function commitBatch(uint8 _version, bytes _parentBatchHeader, bytes[] _chunks, bytes _skippedL1MessageBitmap) external nonpayable
```

Commit a batch of transactions on layer 1.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _version | uint8 | undefined |
| _parentBatchHeader | bytes | undefined |
| _chunks | bytes[] | undefined |
| _skippedL1MessageBitmap | bytes | undefined |

### committedBatches

```solidity
function committedBatches(uint256) external view returns (bytes32)
```

Return the batch hash of a committed batch.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined |

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bytes32 | undefined |

### finalizeBatchWithProof

```solidity
function finalizeBatchWithProof(bytes _batchHeader, bytes32 _prevStateRoot, bytes32 _postStateRoot, bytes32 _withdrawRoot, bytes _aggrProof) external nonpayable
```

Finalize a committed batch on layer 1.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _batchHeader | bytes | undefined |
| _prevStateRoot | bytes32 | undefined |
| _postStateRoot | bytes32 | undefined |
| _withdrawRoot | bytes32 | undefined |
| _aggrProof | bytes | undefined |

### finalizedStateRoots

```solidity
function finalizedStateRoots(uint256) external view returns (bytes32)
```

Return the state root of a committed batch.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined |

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bytes32 | undefined |

### importGenesisBatch

```solidity
function importGenesisBatch(bytes _batchHeader, bytes32 _stateRoot) external nonpayable
```

Import layer 2 genesis block



#### Parameters

| Name | Type | Description |
|---|---|---|
| _batchHeader | bytes | undefined |
| _stateRoot | bytes32 | undefined |

### initialize

```solidity
function initialize(address _messageQueue, address _verifier, uint256 _maxNumL2TxInChunk) external nonpayable
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _messageQueue | address | undefined |
| _verifier | address | undefined |
| _maxNumL2TxInChunk | uint256 | undefined |

### isBatchFinalized

```solidity
function isBatchFinalized(uint256 _batchIndex) external view returns (bool)
```

Return whether the batch is finalized by batch index.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _batchIndex | uint256 | undefined |

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bool | undefined |

### isProver

```solidity
function isProver(address) external view returns (bool)
```

Whether an account is a prover.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bool | undefined |

### isSequencer

```solidity
function isSequencer(address) external view returns (bool)
```

Whether an account is a sequencer.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bool | undefined |

### lastFinalizedBatchIndex

```solidity
function lastFinalizedBatchIndex() external view returns (uint256)
```

The latest finalized batch index.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined |

### layer2ChainId

```solidity
function layer2ChainId() external view returns (uint64)
```

The chain id of the corresponding layer 2 chain.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint64 | undefined |

### maxNumL2TxInChunk

```solidity
function maxNumL2TxInChunk() external view returns (uint256)
```

The maximum number of transactions allowed in each chunk.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined |

### messageQueue

```solidity
function messageQueue() external view returns (address)
```

The address of L1MessageQueue.




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

### renounceOwnership

```solidity
function renounceOwnership() external nonpayable
```



*Leaves the contract without owner. It will not be possible to call `onlyOwner` functions anymore. Can only be called by the current owner. NOTE: Renouncing ownership will leave the contract without an owner, thereby removing any functionality that is only available to the owner.*


### revertBatch

```solidity
function revertBatch(bytes _batchHeader, uint256 _count) external nonpayable
```

Revert a pending batch.

*one can only revert unfinalized batches.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _batchHeader | bytes | undefined |
| _count | uint256 | undefined |

### transferOwnership

```solidity
function transferOwnership(address newOwner) external nonpayable
```



*Transfers ownership of the contract to a new account (`newOwner`). Can only be called by the current owner.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| newOwner | address | undefined |

### updateMaxNumL2TxInChunk

```solidity
function updateMaxNumL2TxInChunk(uint256 _maxNumL2TxInChunk) external nonpayable
```

Update the value of `maxNumL2TxInChunk`.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _maxNumL2TxInChunk | uint256 | The new value of `maxNumL2TxInChunk`. |

### updateProver

```solidity
function updateProver(address _account, bool _status) external nonpayable
```

Update the status of prover.

*This function can only called by contract owner.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _account | address | The address of account to update. |
| _status | bool | The status of the account to update. |

### updateSequencer

```solidity
function updateSequencer(address _account, bool _status) external nonpayable
```

Update the status of sequencer.

*This function can only called by contract owner.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _account | address | The address of account to update. |
| _status | bool | The status of the account to update. |

### updateVerifier

```solidity
function updateVerifier(address _newVerifier) external nonpayable
```

Update the address verifier contract.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _newVerifier | address | The address of new verifier contract. |

### verifier

```solidity
function verifier() external view returns (address)
```

The address of RollupVerifier.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

### withdrawRoots

```solidity
function withdrawRoots(uint256) external view returns (bytes32)
```

Return the message root of a committed batch.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined |

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bytes32 | undefined |



## Events

### CommitBatch

```solidity
event CommitBatch(bytes32 indexed batchHash)
```

Emitted when a new batch is committed.



#### Parameters

| Name | Type | Description |
|---|---|---|
| batchHash `indexed` | bytes32 | undefined |

### FinalizeBatch

```solidity
event FinalizeBatch(bytes32 indexed batchHash, bytes32 stateRoot, bytes32 withdrawRoot)
```

Emitted when a batch is finalized.



#### Parameters

| Name | Type | Description |
|---|---|---|
| batchHash `indexed` | bytes32 | undefined |
| stateRoot  | bytes32 | undefined |
| withdrawRoot  | bytes32 | undefined |

### OwnershipTransferred

```solidity
event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| previousOwner `indexed` | address | undefined |
| newOwner `indexed` | address | undefined |

### RevertBatch

```solidity
event RevertBatch(bytes32 indexed batchHash)
```

revert a pending batch.



#### Parameters

| Name | Type | Description |
|---|---|---|
| batchHash `indexed` | bytes32 | undefined |

### UpdateMaxNumL2TxInChunk

```solidity
event UpdateMaxNumL2TxInChunk(uint256 oldMaxNumL2TxInChunk, uint256 newMaxNumL2TxInChunk)
```

Emitted when the value of `maxNumL2TxInChunk` is updated.



#### Parameters

| Name | Type | Description |
|---|---|---|
| oldMaxNumL2TxInChunk  | uint256 | The old value of `maxNumL2TxInChunk`. |
| newMaxNumL2TxInChunk  | uint256 | The new value of `maxNumL2TxInChunk`. |

### UpdateProver

```solidity
event UpdateProver(address indexed account, bool status)
```

Emitted when owner updates the status of prover.



#### Parameters

| Name | Type | Description |
|---|---|---|
| account `indexed` | address | The address of account updated. |
| status  | bool | The status of the account updated. |

### UpdateSequencer

```solidity
event UpdateSequencer(address indexed account, bool status)
```

Emitted when owner updates the status of sequencer.



#### Parameters

| Name | Type | Description |
|---|---|---|
| account `indexed` | address | The address of account updated. |
| status  | bool | The status of the account updated. |

### UpdateVerifier

```solidity
event UpdateVerifier(address oldVerifier, address newVerifier)
```

Emitted when the address of rollup verifier is updated.



#### Parameters

| Name | Type | Description |
|---|---|---|
| oldVerifier  | address | The address of old rollup verifier. |
| newVerifier  | address | The address of new rollup verifier. |



