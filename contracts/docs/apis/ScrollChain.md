# ScrollChain



> ScrollChain

This contract maintains data for the Scroll rollup.



## Methods

### addProver

```solidity
function addProver(address _account) external nonpayable
```

Add an account to the prover list.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _account | address | The address of account to add. |

### addSequencer

```solidity
function addSequencer(address _account) external nonpayable
```

Add an account to the sequencer list.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _account | address | The address of account to add. |

### blockCommitBatches

```solidity
function blockCommitBatches(uint256) external view returns (bool)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined |

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bool | undefined |

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

### committedBatchInfo

```solidity
function committedBatchInfo(uint256) external view returns (uint256 blockNumber, bool proofSubmitted)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined |

#### Returns

| Name | Type | Description |
|---|---|---|
| blockNumber | uint256 | undefined |
| proofSubmitted | bool | undefined |

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

### getBatchToProve

```solidity
function getBatchToProve() external view returns (uint256)
```






#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined |

### ideDeposit

```solidity
function ideDeposit() external view returns (contract IDeposit)
```






#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | contract IDeposit | undefined |

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

### incorrectProofHashPunishAmount

```solidity
function incorrectProofHashPunishAmount() external view returns (uint256)
```






#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined |

### initialize

```solidity
function initialize(address _messageQueue, address _verifier, uint256 _maxNumTxInChunk) external nonpayable
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _messageQueue | address | undefined |
| _verifier | address | undefined |
| _maxNumTxInChunk | uint256 | undefined |

### isAllLiquidated

```solidity
function isAllLiquidated() external view returns (bool)
```






#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bool | undefined |

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

### isCommitProofAllowed

```solidity
function isCommitProofAllowed(uint256 batchIndex) external view returns (bool)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| batchIndex | uint256 | undefined |

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bool | undefined |

### isCommitProofHashAllowed

```solidity
function isCommitProofHashAllowed(uint256 batchIndex) external view returns (bool)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| batchIndex | uint256 | undefined |

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

### maxNumTxInChunk

```solidity
function maxNumTxInChunk() external view returns (uint256)
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

### minDeposit

```solidity
function minDeposit() external view returns (uint256)
```






#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined |

### noProofPunishAmount

```solidity
function noProofPunishAmount() external view returns (uint256)
```






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

### proofCommitEpoch

```solidity
function proofCommitEpoch() external view returns (uint8)
```






#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint8 | undefined |

### proofHashCommitEpoch

```solidity
function proofHashCommitEpoch() external view returns (uint8)
```






#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint8 | undefined |

### proofNum

```solidity
function proofNum(address) external view returns (uint256)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined |

### proverCommitProofHash

```solidity
function proverCommitProofHash(uint256, address) external view returns (bytes32 proofHash, uint256 blockNumber, bool proof)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined |
| _1 | address | undefined |

#### Returns

| Name | Type | Description |
|---|---|---|
| proofHash | bytes32 | undefined |
| blockNumber | uint256 | undefined |
| proof | bool | undefined |

### proverLastLiquidated

```solidity
function proverLastLiquidated(address) external view returns (uint256)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined |

### proverLiquidation

```solidity
function proverLiquidation(address, uint256) external view returns (address prover, bool isSubmittedProofHash, uint256 submitHashBlockNumber, bool isSubmittedProof, uint256 submitProofBlockNumber, bool isLiquidated, uint64 finalNewBatch)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |
| _1 | uint256 | undefined |

#### Returns

| Name | Type | Description |
|---|---|---|
| prover | address | undefined |
| isSubmittedProofHash | bool | undefined |
| submitHashBlockNumber | uint256 | undefined |
| isSubmittedProof | bool | undefined |
| submitProofBlockNumber | uint256 | undefined |
| isLiquidated | bool | undefined |
| finalNewBatch | uint64 | undefined |

### proverPosition

```solidity
function proverPosition(address, bytes32) external view returns (uint256)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined |
| _1 | bytes32 | undefined |

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined |

### removeProver

```solidity
function removeProver(address _account) external nonpayable
```

Add an account from the prover list.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _account | address | The address of account to remove. |

### removeSequencer

```solidity
function removeSequencer(address _account) external nonpayable
```

Remove an account from the sequencer list.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _account | address | The address of account to remove. |

### renounceOwnership

```solidity
function renounceOwnership() external nonpayable
```



*Leaves the contract without owner. It will not be possible to call `onlyOwner` functions. Can only be called by the current owner. NOTE: Renouncing ownership will leave the contract without an owner, thereby disabling any functionality that is only available to the owner.*


### revertBatch

```solidity
function revertBatch(bytes _batchHeader, uint256 _count) external nonpayable
```

Revert a pending batch.

*If the owner want to revert a sequence of batches by sending multiple transactions,      make sure to revert recent batches first.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _batchHeader | bytes | undefined |
| _count | uint256 | undefined |

### setDeposit

```solidity
function setDeposit(contract IDeposit _ideDeposit) external nonpayable
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _ideDeposit | contract IDeposit | undefined |

### setIncorrectProofPunishAmount

```solidity
function setIncorrectProofPunishAmount(uint256 _amount) external nonpayable
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _amount | uint256 | undefined |

### setMinDeposit

```solidity
function setMinDeposit(uint256 _amount) external nonpayable
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _amount | uint256 | undefined |

### setNoProofPunishAmount

```solidity
function setNoProofPunishAmount(uint256 _amount) external nonpayable
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _amount | uint256 | undefined |

### setPause

```solidity
function setPause(bool _status) external nonpayable
```

Pause the contract



#### Parameters

| Name | Type | Description |
|---|---|---|
| _status | bool | The pause status to update. |

### setProofCommitEpoch

```solidity
function setProofCommitEpoch(uint8 _newCommitEpoch) external nonpayable
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _newCommitEpoch | uint8 | undefined |

### setProofHashCommitEpoch

```solidity
function setProofHashCommitEpoch(uint8 _newCommitEpoch) external nonpayable
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _newCommitEpoch | uint8 | undefined |

### setSlotAdapter

```solidity
function setSlotAdapter(contract ISlotAdapter _slotAdapter) external nonpayable
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _slotAdapter | contract ISlotAdapter | undefined |

### settle

```solidity
function settle(address _account) external nonpayable
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _account | address | undefined |

### slotAdapter

```solidity
function slotAdapter() external view returns (contract ISlotAdapter)
```






#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | contract ISlotAdapter | undefined |

### submitProofHash

```solidity
function submitProofHash(uint256 batchIndex, bytes32 _proofHash) external nonpayable
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| batchIndex | uint256 | undefined |
| _proofHash | bytes32 | undefined |

### transferOwnership

```solidity
function transferOwnership(address newOwner) external nonpayable
```



*Transfers ownership of the contract to a new account (`newOwner`). Can only be called by the current owner.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| newOwner | address | undefined |

### updateMaxNumTxInChunk

```solidity
function updateMaxNumTxInChunk(uint256 _maxNumTxInChunk) external nonpayable
```

Update the value of `maxNumTxInChunk`.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _maxNumTxInChunk | uint256 | The new value of `maxNumTxInChunk`. |

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
event CommitBatch(uint256 indexed batchIndex, bytes32 indexed batchHash)
```

Emitted when a new batch is committed.



#### Parameters

| Name | Type | Description |
|---|---|---|
| batchIndex `indexed` | uint256 | undefined |
| batchHash `indexed` | bytes32 | undefined |

### FinalizeBatch

```solidity
event FinalizeBatch(uint256 indexed batchIndex, bytes32 indexed batchHash, bytes32 stateRoot, bytes32 withdrawRoot)
```

Emitted when a batch is finalized.



#### Parameters

| Name | Type | Description |
|---|---|---|
| batchIndex `indexed` | uint256 | undefined |
| batchHash `indexed` | bytes32 | undefined |
| stateRoot  | bytes32 | undefined |
| withdrawRoot  | bytes32 | undefined |

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

### RevertBatch

```solidity
event RevertBatch(uint256 indexed batchIndex, bytes32 indexed batchHash)
```

revert a pending batch.



#### Parameters

| Name | Type | Description |
|---|---|---|
| batchIndex `indexed` | uint256 | undefined |
| batchHash `indexed` | bytes32 | undefined |

### SetProofCommitEpoch

```solidity
event SetProofCommitEpoch(uint8 newProofCommitEpoch)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| newProofCommitEpoch  | uint8 | undefined |

### SetProofHashCommitEpoch

```solidity
event SetProofHashCommitEpoch(uint8 newProofHashCommitEpoch)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| newProofHashCommitEpoch  | uint8 | undefined |

### SubmitProofHash

```solidity
event SubmitProofHash(address _prover, uint256 batchIndex, bytes32 _proofHash)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _prover  | address | undefined |
| batchIndex  | uint256 | undefined |
| _proofHash  | bytes32 | undefined |

### Unpaused

```solidity
event Unpaused(address account)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| account  | address | undefined |

### UpdateMaxNumTxInChunk

```solidity
event UpdateMaxNumTxInChunk(uint256 oldMaxNumTxInChunk, uint256 newMaxNumTxInChunk)
```

Emitted when the value of `maxNumTxInChunk` is updated.



#### Parameters

| Name | Type | Description |
|---|---|---|
| oldMaxNumTxInChunk  | uint256 | The old value of `maxNumTxInChunk`. |
| newMaxNumTxInChunk  | uint256 | The new value of `maxNumTxInChunk`. |

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
event UpdateVerifier(address indexed oldVerifier, address indexed newVerifier)
```

Emitted when the address of rollup verifier is updated.



#### Parameters

| Name | Type | Description |
|---|---|---|
| oldVerifier `indexed` | address | The address of old rollup verifier. |
| newVerifier `indexed` | address | The address of new rollup verifier. |



## Errors

### CommittedProof

```solidity
error CommittedProof()
```



*Thrown when prover already committed proof*


### CommittedProofHash

```solidity
error CommittedProofHash()
```



*Thrown when prover already committed proof hash*


### CommittedTimeout

```solidity
error CommittedTimeout()
```



*Thrown when commit time out*


### ErrCommitProof

```solidity
error ErrCommitProof()
```



*Thrown when prover submitted invalid proof*


### ErrorBatchHash

```solidity
error ErrorBatchHash(bytes32)
```



*Thrown when commit wrong batch hash*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | bytes32 | undefined |

### InsufficientPledge

```solidity
error InsufficientPledge()
```



*Thrown when Caller has not deposited*


### OnlyDeposit

```solidity
error OnlyDeposit()
```



*Thrown when Caller is not IDEposit contract*


### SlotAdapterEmpty

```solidity
error SlotAdapterEmpty()
```



*Thrown when the SlotAdapter address is ZeroAddress*


### SubmitProofEarly

```solidity
error SubmitProofEarly()
```



*Thrown when prover submit proof early*


### SubmitProofTooLate

```solidity
error SubmitProofTooLate()
```



*Thrown when prover submitted proof too late*


### UnCommittedProofHash

```solidity
error UnCommittedProofHash()
```



*Thrown when prover hasn&#39;t committed proof hash, but want to commit proof*


### ZeroAddress

```solidity
error ZeroAddress()
```



*Thrown when the address is ZeroAddress*


### submitProofHashNotAllowed

```solidity
error submitProofHashNotAllowed()
```



*Thrown when submit proof hash not allowed*


### submitProofNotAllowed

```solidity
error submitProofNotAllowed()
```



*Thrown when submit proof not allowed*



