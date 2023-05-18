// File: @openzeppelin/contracts-upgradeable/utils/AddressUpgradeable.sol

// SPDX-License-Identifier: MIT
// OpenZeppelin Contracts (last updated v4.5.0) (utils/Address.sol)

pragma solidity ^0.8.1;

/**
 * @dev Collection of functions related to the address type
 */
library AddressUpgradeable {
    /**
     * @dev Returns true if `account` is a contract.
     *
     * [IMPORTANT]
     * ====
     * It is unsafe to assume that an address for which this function returns
     * false is an externally-owned account (EOA) and not a contract.
     *
     * Among others, `isContract` will return false for the following
     * types of addresses:
     *
     *  - an externally-owned account
     *  - a contract in construction
     *  - an address where a contract will be created
     *  - an address where a contract lived, but was destroyed
     * ====
     *
     * [IMPORTANT]
     * ====
     * You shouldn't rely on `isContract` to protect against flash loan attacks!
     *
     * Preventing calls from contracts is highly discouraged. It breaks composability, breaks support for smart wallets
     * like Gnosis Safe, and does not provide security since it can be circumvented by calling from a contract
     * constructor.
     * ====
     */
    function isContract(address account) internal view returns (bool) {
        // This method relies on extcodesize/address.code.length, which returns 0
        // for contracts in construction, since the code is only stored at the end
        // of the constructor execution.

        return account.code.length > 0;
    }

    /**
     * @dev Replacement for Solidity's `transfer`: sends `amount` wei to
     * `recipient`, forwarding all available gas and reverting on errors.
     *
     * https://eips.ethereum.org/EIPS/eip-1884[EIP1884] increases the gas cost
     * of certain opcodes, possibly making contracts go over the 2300 gas limit
     * imposed by `transfer`, making them unable to receive funds via
     * `transfer`. {sendValue} removes this limitation.
     *
     * https://diligence.consensys.net/posts/2019/09/stop-using-soliditys-transfer-now/[Learn more].
     *
     * IMPORTANT: because control is transferred to `recipient`, care must be
     * taken to not create reentrancy vulnerabilities. Consider using
     * {ReentrancyGuard} or the
     * https://solidity.readthedocs.io/en/v0.5.11/security-considerations.html#use-the-checks-effects-interactions-pattern[checks-effects-interactions pattern].
     */
    function sendValue(address payable recipient, uint256 amount) internal {
        require(address(this).balance >= amount, "Address: insufficient balance");

        (bool success, ) = recipient.call{value: amount}("");
        require(success, "Address: unable to send value, recipient may have reverted");
    }

    /**
     * @dev Performs a Solidity function call using a low level `call`. A
     * plain `call` is an unsafe replacement for a function call: use this
     * function instead.
     *
     * If `target` reverts with a revert reason, it is bubbled up by this
     * function (like regular Solidity function calls).
     *
     * Returns the raw returned data. To convert to the expected return value,
     * use https://solidity.readthedocs.io/en/latest/units-and-global-variables.html?highlight=abi.decode#abi-encoding-and-decoding-functions[`abi.decode`].
     *
     * Requirements:
     *
     * - `target` must be a contract.
     * - calling `target` with `data` must not revert.
     *
     * _Available since v3.1._
     */
    function functionCall(address target, bytes memory data) internal returns (bytes memory) {
        return functionCall(target, data, "Address: low-level call failed");
    }

    /**
     * @dev Same as {xref-Address-functionCall-address-bytes-}[`functionCall`], but with
     * `errorMessage` as a fallback revert reason when `target` reverts.
     *
     * _Available since v3.1._
     */
    function functionCall(
        address target,
        bytes memory data,
        string memory errorMessage
    ) internal returns (bytes memory) {
        return functionCallWithValue(target, data, 0, errorMessage);
    }

    /**
     * @dev Same as {xref-Address-functionCall-address-bytes-}[`functionCall`],
     * but also transferring `value` wei to `target`.
     *
     * Requirements:
     *
     * - the calling contract must have an ETH balance of at least `value`.
     * - the called Solidity function must be `payable`.
     *
     * _Available since v3.1._
     */
    function functionCallWithValue(
        address target,
        bytes memory data,
        uint256 value
    ) internal returns (bytes memory) {
        return functionCallWithValue(target, data, value, "Address: low-level call with value failed");
    }

    /**
     * @dev Same as {xref-Address-functionCallWithValue-address-bytes-uint256-}[`functionCallWithValue`], but
     * with `errorMessage` as a fallback revert reason when `target` reverts.
     *
     * _Available since v3.1._
     */
    function functionCallWithValue(
        address target,
        bytes memory data,
        uint256 value,
        string memory errorMessage
    ) internal returns (bytes memory) {
        require(address(this).balance >= value, "Address: insufficient balance for call");
        require(isContract(target), "Address: call to non-contract");

        (bool success, bytes memory returndata) = target.call{value: value}(data);
        return verifyCallResult(success, returndata, errorMessage);
    }

    /**
     * @dev Same as {xref-Address-functionCall-address-bytes-}[`functionCall`],
     * but performing a static call.
     *
     * _Available since v3.3._
     */
    function functionStaticCall(address target, bytes memory data) internal view returns (bytes memory) {
        return functionStaticCall(target, data, "Address: low-level static call failed");
    }

    /**
     * @dev Same as {xref-Address-functionCall-address-bytes-string-}[`functionCall`],
     * but performing a static call.
     *
     * _Available since v3.3._
     */
    function functionStaticCall(
        address target,
        bytes memory data,
        string memory errorMessage
    ) internal view returns (bytes memory) {
        require(isContract(target), "Address: static call to non-contract");

        (bool success, bytes memory returndata) = target.staticcall(data);
        return verifyCallResult(success, returndata, errorMessage);
    }

    /**
     * @dev Tool to verifies that a low level call was successful, and revert if it wasn't, either by bubbling the
     * revert reason using the provided one.
     *
     * _Available since v4.3._
     */
    function verifyCallResult(
        bool success,
        bytes memory returndata,
        string memory errorMessage
    ) internal pure returns (bytes memory) {
        if (success) {
            return returndata;
        } else {
            // Look for revert reason and bubble it up if present
            if (returndata.length > 0) {
                // The easiest way to bubble the revert reason is using memory via assembly

                assembly {
                    let returndata_size := mload(returndata)
                    revert(add(32, returndata), returndata_size)
                }
            } else {
                revert(errorMessage);
            }
        }
    }
}

// File: @openzeppelin/contracts-upgradeable/proxy/utils/Initializable.sol


// OpenZeppelin Contracts (last updated v4.5.0) (proxy/utils/Initializable.sol)

pragma solidity ^0.8.0;

/**
 * @dev This is a base contract to aid in writing upgradeable contracts, or any kind of contract that will be deployed
 * behind a proxy. Since proxied contracts do not make use of a constructor, it's common to move constructor logic to an
 * external initializer function, usually called `initialize`. It then becomes necessary to protect this initializer
 * function so it can only be called once. The {initializer} modifier provided by this contract will have this effect.
 *
 * TIP: To avoid leaving the proxy in an uninitialized state, the initializer function should be called as early as
 * possible by providing the encoded function call as the `_data` argument to {ERC1967Proxy-constructor}.
 *
 * CAUTION: When used with inheritance, manual care must be taken to not invoke a parent initializer twice, or to ensure
 * that all initializers are idempotent. This is not verified automatically as constructors are by Solidity.
 *
 * [CAUTION]
 * ====
 * Avoid leaving a contract uninitialized.
 *
 * An uninitialized contract can be taken over by an attacker. This applies to both a proxy and its implementation
 * contract, which may impact the proxy. To initialize the implementation contract, you can either invoke the
 * initializer manually, or you can include a constructor to automatically mark it as initialized when it is deployed:
 *
 * [.hljs-theme-light.nopadding]
 * ```
 * /// @custom:oz-upgrades-unsafe-allow constructor
 * constructor() initializer {}
 * ```
 * ====
 */
abstract contract Initializable {
    /**
     * @dev Indicates that the contract has been initialized.
     */
    bool private _initialized;

    /**
     * @dev Indicates that the contract is in the process of being initialized.
     */
    bool private _initializing;

    /**
     * @dev Modifier to protect an initializer function from being invoked twice.
     */
    modifier initializer() {
        // If the contract is initializing we ignore whether _initialized is set in order to support multiple
        // inheritance patterns, but we only do this in the context of a constructor, because in other contexts the
        // contract may have been reentered.
        require(_initializing ? _isConstructor() : !_initialized, "Initializable: contract is already initialized");

        bool isTopLevelCall = !_initializing;
        if (isTopLevelCall) {
            _initializing = true;
            _initialized = true;
        }

        _;

        if (isTopLevelCall) {
            _initializing = false;
        }
    }

    /**
     * @dev Modifier to protect an initialization function so that it can only be invoked by functions with the
     * {initializer} modifier, directly or indirectly.
     */
    modifier onlyInitializing() {
        require(_initializing, "Initializable: contract is not initializing");
        _;
    }

    function _isConstructor() private view returns (bool) {
        return !AddressUpgradeable.isContract(address(this));
    }
}

// File: @openzeppelin/contracts-upgradeable/utils/ContextUpgradeable.sol


// OpenZeppelin Contracts v4.4.1 (utils/Context.sol)

pragma solidity ^0.8.0;

/**
 * @dev Provides information about the current execution context, including the
 * sender of the transaction and its data. While these are generally available
 * via msg.sender and msg.data, they should not be accessed in such a direct
 * manner, since when dealing with meta-transactions the account sending and
 * paying for execution may not be the actual sender (as far as an application
 * is concerned).
 *
 * This contract is only required for intermediate, library-like contracts.
 */
abstract contract ContextUpgradeable is Initializable {
    function __Context_init() internal onlyInitializing {
    }

    function __Context_init_unchained() internal onlyInitializing {
    }
    function _msgSender() internal view virtual returns (address) {
        return msg.sender;
    }

    function _msgData() internal view virtual returns (bytes calldata) {
        return msg.data;
    }

    /**
     * @dev This empty reserved space is put in place to allow future versions to add new
     * variables without shifting down storage in the inheritance chain.
     * See https://docs.openzeppelin.com/contracts/4.x/upgradeable#storage_gaps
     */
    uint256[50] private __gap;
}

// File: @openzeppelin/contracts-upgradeable/security/PausableUpgradeable.sol


// OpenZeppelin Contracts v4.4.1 (security/Pausable.sol)

pragma solidity ^0.8.0;


/**
 * @dev Contract module which allows children to implement an emergency stop
 * mechanism that can be triggered by an authorized account.
 *
 * This module is used through inheritance. It will make available the
 * modifiers `whenNotPaused` and `whenPaused`, which can be applied to
 * the functions of your contract. Note that they will not be pausable by
 * simply including this module, only once the modifiers are put in place.
 */
abstract contract PausableUpgradeable is Initializable, ContextUpgradeable {
    /**
     * @dev Emitted when the pause is triggered by `account`.
     */
    event Paused(address account);

    /**
     * @dev Emitted when the pause is lifted by `account`.
     */
    event Unpaused(address account);

    bool private _paused;

    /**
     * @dev Initializes the contract in unpaused state.
     */
    function __Pausable_init() internal onlyInitializing {
        __Pausable_init_unchained();
    }

    function __Pausable_init_unchained() internal onlyInitializing {
        _paused = false;
    }

    /**
     * @dev Returns true if the contract is paused, and false otherwise.
     */
    function paused() public view virtual returns (bool) {
        return _paused;
    }

    /**
     * @dev Modifier to make a function callable only when the contract is not paused.
     *
     * Requirements:
     *
     * - The contract must not be paused.
     */
    modifier whenNotPaused() {
        require(!paused(), "Pausable: paused");
        _;
    }

    /**
     * @dev Modifier to make a function callable only when the contract is paused.
     *
     * Requirements:
     *
     * - The contract must be paused.
     */
    modifier whenPaused() {
        require(paused(), "Pausable: not paused");
        _;
    }

    /**
     * @dev Triggers stopped state.
     *
     * Requirements:
     *
     * - The contract must not be paused.
     */
    function _pause() internal virtual whenNotPaused {
        _paused = true;
        emit Paused(_msgSender());
    }

    /**
     * @dev Returns to normal state.
     *
     * Requirements:
     *
     * - The contract must be paused.
     */
    function _unpause() internal virtual whenPaused {
        _paused = false;
        emit Unpaused(_msgSender());
    }

    /**
     * @dev This empty reserved space is put in place to allow future versions to add new
     * variables without shifting down storage in the inheritance chain.
     * See https://docs.openzeppelin.com/contracts/4.x/upgradeable#storage_gaps
     */
    uint256[49] private __gap;
}

// File: src/L1/rollup/IScrollChain.sol



pragma solidity ^0.8.0;

interface IScrollChain {
  /**********
   * Events *
   **********/

  /// @notice Emitted when a new batch is commited.
  /// @param batchHash The hash of the batch
  event CommitBatch(bytes32 indexed batchHash);

  /// @notice Emitted when a batch is reverted.
  /// @param batchHash The identification of the batch.
  event RevertBatch(bytes32 indexed batchHash);

  /// @notice Emitted when a batch is finalized.
  /// @param batchHash The hash of the batch
  event FinalizeBatch(bytes32 indexed batchHash);

  /***********
   * Structs *
   ***********/

  struct BlockContext {
    // The hash of this block.
    bytes32 blockHash;
    // The parent hash of this block.
    bytes32 parentHash;
    // The height of this block.
    uint64 blockNumber;
    // The timestamp of this block.
    uint64 timestamp;
    // The base fee of this block.
    // Currently, it is not used, because we disable EIP-1559.
    // We keep it for future proof.
    uint256 baseFee;
    // The gas limit of this block.
    uint64 gasLimit;
    // The number of transactions in this block, both L1 & L2 txs.
    uint16 numTransactions;
    // The number of l1 messages in this block.
    uint16 numL1Messages;
  }

  struct Batch {
    // The list of blocks in this batch
    BlockContext[] blocks; // MAX_NUM_BLOCKS = 100, about 5 min
    // The state root of previous batch.
    // The first batch will use 0x0 for prevStateRoot
    bytes32 prevStateRoot;
    // The state root of the last block in this batch.
    bytes32 newStateRoot;
    // The withdraw trie root of the last block in this batch.
    bytes32 withdrawTrieRoot;
    // The index of the batch.
    uint64 batchIndex;
    // The parent batch hash.
    bytes32 parentBatchHash;
    // Concatenated raw data of RLP encoded L2 txs
    bytes l2Transactions;
  }

  /*************************
   * Public View Functions *
   *************************/

  /// @notice Return whether the batch is finalized by batch hash.
  /// @param batchHash The hash of the batch to query.
  function isBatchFinalized(bytes32 batchHash) external view returns (bool);

  /// @notice Return the merkle root of L2 message tree.
  /// @param batchHash The hash of the batch to query.
  function getL2MessageRoot(bytes32 batchHash) external view returns (bytes32);

  /****************************
   * Public Mutated Functions *
   ****************************/

  /// @notice commit a batch in layer 1
  /// @param batch The layer2 batch to commit.
  function commitBatch(Batch memory batch) external;

  /// @notice commit a list of batches in layer 1
  /// @param batches The list of layer2 batches to commit.
  function commitBatches(Batch[] memory batches) external;

  /// @notice revert a pending batch.
  /// @dev one can only revert unfinalized batches.
  /// @param batchId The identification of the batch.
  function revertBatch(bytes32 batchId) external;

  /// @notice finalize commited batch in layer 1
  /// @dev will add more parameters if needed.
  /// @param batchId The identification of the commited batch.
  /// @param proof The corresponding proof of the commited batch.
  /// @param instances Instance used to verify, generated from batch.
  function finalizeBatchWithProof(
    bytes32 batchId,
    uint256[] memory proof,
    uint256[] memory instances
  ) external;
}

// File: src/L1/rollup/IL1MessageQueue.sol



pragma solidity ^0.8.0;

interface IL1MessageQueue {
  /**********
   * Events *
   **********/

  /// @notice Emitted when a new L1 => L2 transaction is appended to the queue.
  /// @param sender The address of account who initiates the transaction.
  /// @param target The address of account who will recieve the transaction.
  /// @param value The value passed with the transaction.
  /// @param queueIndex The index of this transaction in the queue.
  /// @param gasLimit Gas limit required to complete the message relay on L2.
  /// @param data The calldata of the transaction.
  event QueueTransaction(
    address indexed sender,
    address indexed target,
    uint256 value,
    uint256 queueIndex,
    uint256 gasLimit,
    bytes data
  );

  /*************************
   * Public View Functions *
   *************************/

  /// @notice Return the index of next appended message.
  /// @dev Also the total number of appended messages.
  function nextCrossDomainMessageIndex() external view returns (uint256);

  /// @notice Return the message of in `queueIndex`.
  /// @param queueIndex The index to query.
  function getCrossDomainMessage(uint256 queueIndex) external view returns (bytes32);

  /// @notice Return the amount of ETH should pay for cross domain message.
  /// @param sender The address of account who initiates the message in L1.
  /// @param target The address of account who will recieve the message in L2.
  /// @param message The content of the message.
  /// @param gasLimit Gas limit required to complete the message relay on L2.
  function estimateCrossDomainMessageFee(
    address sender,
    address target,
    bytes memory message,
    uint256 gasLimit
  ) external view returns (uint256);

  /****************************
   * Public Mutated Functions *
   ****************************/

  /// @notice Append a L1 to L2 message into this contract.
  /// @param target The address of target contract to call in L2.
  /// @param gasLimit The maximum gas should be used for relay this message in L2.
  /// @param data The calldata passed to target contract.
  function appendCrossDomainMessage(
    address target,
    uint256 gasLimit,
    bytes calldata data
  ) external;

  /// @notice Append an enforced transaction to this contract.
  /// @dev The address of sender should be an EOA.
  /// @param sender The address of sender who will initiate this transaction in L2.
  /// @param target The address of target contract to call in L2.
  /// @param value The value passed
  /// @param gasLimit The maximum gas should be used for this transaction in L2.
  /// @param data The calldata passed to target contract.
  function appendEnforcedTransaction(
    address sender,
    address target,
    uint256 value,
    uint256 gasLimit,
    bytes calldata data
  ) external;
}

// File: src/libraries/IScrollMessenger.sol



pragma solidity ^0.8.0;

interface IScrollMessenger {
  /**********
   * Events *
   **********/

  /// @notice Emitted when a cross domain message is sent.
  /// @param sender The address of the sender who initiates the message.
  /// @param target The address of target contract to call.
  /// @param value The amount of value passed to the target contract.
  /// @param messageNonce The nonce of the message.
  /// @param gasLimit The optional gas limit passed to L1 or L2.
  /// @param message The calldata passed to the target contract.
  event SentMessage(
    address indexed sender,
    address indexed target,
    uint256 value,
    uint256 messageNonce,
    uint256 gasLimit,
    bytes message
  );

  /// @notice Emitted when a cross domain message is relayed successfully.
  /// @param messageHash The hash of the message.
  event RelayedMessage(bytes32 indexed messageHash);

  /// @notice Emitted when a cross domain message is failed to relay.
  /// @param messageHash The hash of the message.
  event FailedRelayedMessage(bytes32 indexed messageHash);

  /*************************
   * Public View Functions *
   *************************/

  /// @notice Return the sender of a cross domain message.
  function xDomainMessageSender() external view returns (address);

  /****************************
   * Public Mutated Functions *
   ****************************/

  /// @notice Send cross chain message from L1 to L2 or L2 to L1.
  /// @param target The address of account who recieve the message.
  /// @param value The amount of ether passed when call target contract.
  /// @param message The content of the message.
  /// @param gasLimit Gas limit required to complete the message relay on corresponding chain.
  function sendMessage(
    address target,
    uint256 value,
    bytes calldata message,
    uint256 gasLimit
  ) external payable;
}

// File: src/L1/IL1ScrollMessenger.sol



pragma solidity ^0.8.0;

interface IL1ScrollMessenger is IScrollMessenger {
  /***********
   * Structs *
   ***********/

  struct L2MessageProof {
    // The hash of the batch where the message belongs to.
    bytes32 batchHash;
    // Concatenation of merkle proof for withdraw merkle trie.
    bytes merkleProof;
  }

  /****************************
   * Public Mutated Functions *
   ****************************/

  /// @notice Relay a L2 => L1 message with message proof.
  /// @param from The address of the sender of the message.
  /// @param to The address of the recipient of the message.
  /// @param value The msg.value passed to the message call.
  /// @param nonce The nonce of the message to avoid replay attack.
  /// @param message The content of the message.
  /// @param proof The proof used to verify the correctness of the transaction.
  function relayMessageWithProof(
    address from,
    address to,
    uint256 value,
    uint256 nonce,
    bytes memory message,
    L2MessageProof memory proof
  ) external;

  /// @notice Replay an exsisting message.
  /// @param from The address of the sender of the message.
  /// @param to The address of the recipient of the message.
  /// @param value The msg.value passed to the message call.
  /// @param queueIndex The queue index for the message to replay.
  /// @param message The content of the message.
  /// @param oldGasLimit Original gas limit used to send the message.
  /// @param newGasLimit New gas limit to be used for this message.
  function replayMessage(
    address from,
    address to,
    uint256 value,
    uint256 queueIndex,
    bytes memory message,
    uint32 oldGasLimit,
    uint32 newGasLimit
  ) external;
}

// File: src/libraries/constants/ScrollConstants.sol



pragma solidity ^0.8.0;

library ScrollConstants {
  /// @notice The address of default cross chain message sender.
  address internal constant DEFAULT_XDOMAIN_MESSAGE_SENDER = address(1);
}

// File: @openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol


// OpenZeppelin Contracts v4.4.1 (access/Ownable.sol)

pragma solidity ^0.8.0;


/**
 * @dev Contract module which provides a basic access control mechanism, where
 * there is an account (an owner) that can be granted exclusive access to
 * specific functions.
 *
 * By default, the owner account will be the one that deploys the contract. This
 * can later be changed with {transferOwnership}.
 *
 * This module is used through inheritance. It will make available the modifier
 * `onlyOwner`, which can be applied to your functions to restrict their use to
 * the owner.
 */
abstract contract OwnableUpgradeable is Initializable, ContextUpgradeable {
    address private _owner;

    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);

    /**
     * @dev Initializes the contract setting the deployer as the initial owner.
     */
    function __Ownable_init() internal onlyInitializing {
        __Ownable_init_unchained();
    }

    function __Ownable_init_unchained() internal onlyInitializing {
        _transferOwnership(_msgSender());
    }

    /**
     * @dev Returns the address of the current owner.
     */
    function owner() public view virtual returns (address) {
        return _owner;
    }

    /**
     * @dev Throws if called by any account other than the owner.
     */
    modifier onlyOwner() {
        require(owner() == _msgSender(), "Ownable: caller is not the owner");
        _;
    }

    /**
     * @dev Leaves the contract without owner. It will not be possible to call
     * `onlyOwner` functions anymore. Can only be called by the current owner.
     *
     * NOTE: Renouncing ownership will leave the contract without an owner,
     * thereby removing any functionality that is only available to the owner.
     */
    function renounceOwnership() public virtual onlyOwner {
        _transferOwnership(address(0));
    }

    /**
     * @dev Transfers ownership of the contract to a new account (`newOwner`).
     * Can only be called by the current owner.
     */
    function transferOwnership(address newOwner) public virtual onlyOwner {
        require(newOwner != address(0), "Ownable: new owner is the zero address");
        _transferOwnership(newOwner);
    }

    /**
     * @dev Transfers ownership of the contract to a new account (`newOwner`).
     * Internal function without access restriction.
     */
    function _transferOwnership(address newOwner) internal virtual {
        address oldOwner = _owner;
        _owner = newOwner;
        emit OwnershipTransferred(oldOwner, newOwner);
    }

    /**
     * @dev This empty reserved space is put in place to allow future versions to add new
     * variables without shifting down storage in the inheritance chain.
     * See https://docs.openzeppelin.com/contracts/4.x/upgradeable#storage_gaps
     */
    uint256[49] private __gap;
}

// File: src/libraries/common/IWhitelist.sol



pragma solidity ^0.8.0;

interface IWhitelist {
  /// @notice Check whether the sender is allowed to do something.
  /// @param _sender The address of sender.
  function isSenderAllowed(address _sender) external view returns (bool);
}

// File: src/libraries/ScrollMessengerBase.sol



pragma solidity ^0.8.0;



abstract contract ScrollMessengerBase is OwnableUpgradeable, IScrollMessenger {
  /**********
   * Events *
   **********/

  /// @notice Emitted when owner updates whitelist contract.
  /// @param _oldWhitelist The address of old whitelist contract.
  /// @param _newWhitelist The address of new whitelist contract.
  event UpdateWhitelist(address _oldWhitelist, address _newWhitelist);

  /// @notice Emitted when owner updates fee vault contract.
  /// @param _oldFeeVault The address of old fee vault contract.
  /// @param _newFeeVault The address of new fee vault contract.
  event UpdateFeeVault(address _oldFeeVault, address _newFeeVault);

  /*************
   * Variables *
   *************/

  /// @notice See {IScrollMessenger-xDomainMessageSender}
  address public override xDomainMessageSender;

  /// @notice The whitelist contract to track the sender who can call `sendMessage` in ScrollMessenger.
  address public whitelist;

  /// @notice The address of counterpart ScrollMessenger contract in L1/L2.
  address public counterpart;

  /// @notice The address of fee vault, collecting cross domain messaging fee.
  address public feeVault;

  /**********************
   * Function Modifiers *
   **********************/

  modifier onlyWhitelistedSender(address _sender) {
    address _whitelist = whitelist;
    require(_whitelist == address(0) || IWhitelist(_whitelist).isSenderAllowed(_sender), "sender not whitelisted");
    _;
  }

  /***************
   * Constructor *
   ***************/

  function _initialize(address _counterpart, address _feeVault) internal {
    OwnableUpgradeable.__Ownable_init();

    // initialize to a nonzero value
    xDomainMessageSender = ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER;

    counterpart = _counterpart;
    feeVault = _feeVault;
  }

  // allow others to send ether to messenger
  receive() external payable {}

  /************************
   * Restricted Functions *
   ************************/

  /// @notice Update whitelist contract.
  /// @dev This function can only called by contract owner.
  /// @param _newWhitelist The address of new whitelist contract.
  function updateWhitelist(address _newWhitelist) external onlyOwner {
    address _oldWhitelist = whitelist;

    whitelist = _newWhitelist;
    emit UpdateWhitelist(_oldWhitelist, _newWhitelist);
  }

  /// @notice Update fee vault contract.
  /// @dev This function can only called by contract owner.
  /// @param _newFeeVault The address of new fee vault contract.
  function updateFeeVault(address _newFeeVault) external onlyOwner {
    address _oldFeeVault = whitelist;

    feeVault = _newFeeVault;
    emit UpdateFeeVault(_oldFeeVault, _newFeeVault);
  }

  /**********************
   * Internal Functions *
   **********************/

  /// @dev Internal function to generate the correct cross domain calldata for a message.
  /// @param _sender Message sender address.
  /// @param _target Target contract address.
  /// @param _value The amount of ETH pass to the target.
  /// @param _messageNonce Nonce for the provided message.
  /// @param _message Message to send to the target.
  /// @return ABI encoded cross domain calldata.
  function _encodeXDomainCalldata(
    address _sender,
    address _target,
    uint256 _value,
    uint256 _messageNonce,
    bytes memory _message
  ) internal pure returns (bytes memory) {
    return
      abi.encodeWithSignature(
        "relayMessage(address,address,uint256,uint256,bytes)",
        _sender,
        _target,
        _value,
        _messageNonce,
        _message
      );
  }
}

// File: src/libraries/verifier/WithdrawTrieVerifier.sol



pragma solidity ^0.8.0;

library WithdrawTrieVerifier {
  function verifyMerkleProof(
    bytes32 _root,
    bytes32 _hash,
    uint256 _nonce,
    bytes memory _proof
  ) internal pure returns (bool) {
    require(_proof.length % 256 == 0, "Invalid proof");
    uint256 _length = _proof.length / 256;

    for (uint256 i = 0; i < _length; i++) {
      bytes32 item;
      assembly {
        item := mload(add(add(_proof, 0x20), mul(i, 0x20)))
      }
      if (_nonce % 2 == 0) {
        _hash = _efficientHash(_hash, item);
      } else {
        _hash = _efficientHash(item, _hash);
      }
      _nonce /= 2;
    }
    return _hash == _root;
  }

  function _efficientHash(bytes32 a, bytes32 b) private pure returns (bytes32 value) {
    // solhint-disable-next-line no-inline-assembly
    assembly {
      mstore(0x00, a)
      mstore(0x20, b)
      value := keccak256(0x00, 0x40)
    }
  }
}

// File: src/L1/L1ScrollMessenger.sol



pragma solidity ^0.8.0;







// solhint-disable avoid-low-level-calls

/// @title L1ScrollMessenger
/// @notice The `L1ScrollMessenger` contract can:
///
/// 1. send messages from layer 1 to layer 2;
/// 2. relay messages from layer 2 layer 1;
/// 3. replay failed message by replacing the gas limit;
/// 4. drop expired message due to sequencer problems.
///
/// @dev All deposited Ether (including `WETH` deposited throng `L1WETHGateway`) will locked in
/// this contract.
contract L1ScrollMessenger is PausableUpgradeable, ScrollMessengerBase, IL1ScrollMessenger {
  /*************
   * Variables *
   *************/

  /// @notice Mapping from relay id to relay status.
  mapping(bytes32 => bool) public isL1MessageRelayed;

  /// @notice Mapping from L1 message hash to sent status.
  mapping(bytes32 => bool) public isL1MessageSent;

  /// @notice Mapping from L2 message hash to a boolean value indicating if the message has been successfully executed.
  mapping(bytes32 => bool) public isL2MessageExecuted;

  /// @notice The address of Rollup contract.
  address public rollup;

  /// @notice The address of L1MessageQueue contract.
  address public messageQueue;

  /***************
   * Constructor *
   ***************/

  /// @notice Initialize the storage of L1ScrollMessenger.
  /// @param _counterpart The address of L2ScrollMessenger contract in L2.
  /// @param _feeVault The address of fee vault, which will be used to collect relayer fee.
  /// @param _rollup The address of ScrollChain contract.
  /// @param _messageQueue The address of L1MessageQueue contract.
  function initialize(
    address _counterpart,
    address _feeVault,
    address _rollup,
    address _messageQueue
  ) public initializer {
    PausableUpgradeable.__Pausable_init();
    ScrollMessengerBase._initialize(_counterpart, _feeVault);

    rollup = _rollup;
    messageQueue = _messageQueue;

    // initialize to a nonzero value
    xDomainMessageSender = ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER;
  }

  /****************************
   * Public Mutated Functions *
   ****************************/

  /// @inheritdoc IScrollMessenger
  function sendMessage(
    address _to,
    uint256 _value,
    bytes memory _message,
    uint256 _gasLimit
  ) external payable override whenNotPaused {
    address _messageQueue = messageQueue; // gas saving
    address _counterpart = counterpart; // gas saving

    // compute the actual cross domain message calldata.
    uint256 _messageNonce = IL1MessageQueue(_messageQueue).nextCrossDomainMessageIndex();
    bytes memory _xDomainCalldata = _encodeXDomainCalldata(msg.sender, _to, _value, _messageNonce, _message);

    // compute and deduct the messaging fee to fee vault.
    uint256 _fee = IL1MessageQueue(_messageQueue).estimateCrossDomainMessageFee(
      address(this),
      _counterpart,
      _xDomainCalldata,
      _gasLimit
    );
    require(msg.value >= _fee + _value, "Insufficient msg.value");
    if (_fee > 0) {
      (bool _success, ) = feeVault.call{ value: _fee }("");
      require(_success, "Failed to deduct the fee");
    }

    // append message to L1MessageQueue
    IL1MessageQueue(_messageQueue).appendCrossDomainMessage(_counterpart, _gasLimit, _xDomainCalldata);

    // record the message hash for future use.
    bytes32 _xDomainCalldataHash = keccak256(_xDomainCalldata);

    // normally this won't happen, since each message has different nonce, but just in case.
    require(!isL1MessageSent[_xDomainCalldataHash], "Duplicated message");
    isL1MessageSent[_xDomainCalldataHash] = true;

    emit SentMessage(msg.sender, _to, _value, _messageNonce, _gasLimit, _message);

    // refund fee to tx.origin
    unchecked {
      uint256 _refund = msg.value - _fee - _value;
      if (_refund > 0) {
        (bool _success, ) = tx.origin.call{ value: _refund }("");
        require(_success, "Failed to refund the fee");
      }
    }
  }

  /// @inheritdoc IL1ScrollMessenger
  function relayMessageWithProof(
    address _from,
    address _to,
    uint256 _value,
    uint256 _nonce,
    bytes memory _message,
    L2MessageProof memory _proof
  ) external override whenNotPaused onlyWhitelistedSender(msg.sender) {
    require(xDomainMessageSender == ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER, "Message is already in execution");

    bytes32 _xDomainCalldataHash = keccak256(_encodeXDomainCalldata(_from, _to, _value, _nonce, _message));
    require(!isL2MessageExecuted[_xDomainCalldataHash], "Message was already successfully executed");

    {
      address _rollup = rollup;
      require(IScrollChain(_rollup).isBatchFinalized(_proof.batchHash), "Batch is not finalized");
      // @note skip verify for now
      /*
      bytes32 _messageRoot = IScrollChain(_rollup).getL2MessageRoot(_proof.batchHash);
      require(
        WithdrawTrieVerifier.verifyMerkleProof(_messageRoot, _xDomainCalldataHash, _nonce, _proof.merkleProof),
        "Invalid proof"
      );
      */
    }

    // @note This usually will never happen, just in case.
    require(_from != xDomainMessageSender, "Invalid message sender");

    xDomainMessageSender = _from;
    (bool success, ) = _to.call{ value: _value }(_message);
    // reset value to refund gas.
    xDomainMessageSender = ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER;

    if (success) {
      isL2MessageExecuted[_xDomainCalldataHash] = true;
      emit RelayedMessage(_xDomainCalldataHash);
    } else {
      emit FailedRelayedMessage(_xDomainCalldataHash);
    }

    bytes32 _relayId = keccak256(abi.encodePacked(_xDomainCalldataHash, msg.sender, block.number));
    isL1MessageRelayed[_relayId] = true;
  }

  /// @inheritdoc IL1ScrollMessenger
  function replayMessage(
    address _from,
    address _to,
    uint256 _value,
    uint256 _queueIndex,
    bytes memory _message,
    uint32 _oldGasLimit,
    uint32 _newGasLimit
  ) external override whenNotPaused {
    // @todo
  }

  /************************
   * Restricted Functions *
   ************************/

  /// @notice Pause the contract
  /// @dev This function can only called by contract owner.
  function pause() external onlyOwner {
    _pause();
  }
}
