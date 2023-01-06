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

// File: src/L1/rollup/IZKRollup.sol



pragma solidity ^0.8.0;

interface IZKRollup {
  /**************************************** Events ****************************************/

  /// @notice Emitted when a new batch is commited.
  /// @param _batchHash The hash of the batch
  /// @param _batchIndex The index of the batch
  /// @param _parentHash The hash of parent batch
  event CommitBatch(bytes32 indexed _batchId, bytes32 _batchHash, uint256 _batchIndex, bytes32 _parentHash);

  /// @notice Emitted when a batch is reverted.
  /// @param _batchId The identification of the batch.
  event RevertBatch(bytes32 indexed _batchId);

  /// @notice Emitted when a batch is finalized.
  /// @param _batchHash The hash of the batch
  /// @param _batchIndex The index of the batch
  /// @param _parentHash The hash of parent batch
  event FinalizeBatch(bytes32 indexed _batchId, bytes32 _batchHash, uint256 _batchIndex, bytes32 _parentHash);

  /// @dev The transanction struct
  struct Layer2Transaction {
    address caller;
    uint64 nonce;
    address target;
    uint64 gas;
    uint256 gasPrice;
    uint256 value;
    bytes data;
    // signature
    uint256 r;
    uint256 s;
    uint64 v;
  }

  /// @dev The block header struct
  struct Layer2BlockHeader {
    bytes32 blockHash;
    bytes32 parentHash;
    uint256 baseFee;
    bytes32 stateRoot;
    uint64 blockHeight;
    uint64 gasUsed;
    uint64 timestamp;
    bytes extraData;
    Layer2Transaction[] txs;
  }

  /// @dev The batch struct, the batch hash is always the last block hash of `blocks`.
  struct Layer2Batch {
    uint64 batchIndex;
    // The hash of the last block in the parent batch
    bytes32 parentHash;
    Layer2BlockHeader[] blocks;
  }

  /**************************************** View Functions ****************************************/

  /// @notice Return whether the block is finalized by block hash.
  /// @param blockHash The hash of the block to query.
  function isBlockFinalized(bytes32 blockHash) external view returns (bool);

  /// @notice Return whether the block is finalized by block height.
  /// @param blockHeight The height of the block to query.
  function isBlockFinalized(uint256 blockHeight) external view returns (bool);

  /// @notice Return the message hash by index.
  /// @param _index The index to query.
  function getMessageHashByIndex(uint256 _index) external view returns (bytes32);

  /// @notice Return the index of the first queue element not yet executed.
  function getNextQueueIndex() external view returns (uint256);

  /// @notice Return the layer 2 block gas limit.
  /// @param _blockNumber The block number to query
  function layer2GasLimit(uint256 _blockNumber) external view returns (uint256);

  /// @notice Verify a state proof for message relay.
  /// @dev add more fields.
  function verifyMessageStateProof(uint256 _batchIndex, uint256 _blockHeight) external view returns (bool);

  /**************************************** Mutated Functions ****************************************/

  /// @notice Append a cross chain message to message queue.
  /// @dev This function should only be called by L1ScrollMessenger for safety.
  /// @param _sender The address of message sender in layer 1.
  /// @param _target The address of message recipient in layer 2.
  /// @param _value The amount of ether sent to recipient in layer 2.
  /// @param _fee The amount of ether paid to relayer in layer 2.
  /// @param _deadline The deadline of the message.
  /// @param _message The content of the message.
  /// @param _gasLimit Unused, but included for potential forward compatibility considerations.
  function appendMessage(
    address _sender,
    address _target,
    uint256 _value,
    uint256 _fee,
    uint256 _deadline,
    bytes memory _message,
    uint256 _gasLimit
  ) external returns (uint256);

  /// @notice commit a batch in layer 1
  /// @dev store in a more compacted form later.
  /// @param _batch The layer2 batch to commit.
  function commitBatch(Layer2Batch memory _batch) external;

  /// @notice revert a pending batch.
  /// @dev one can only revert unfinalized batches.
  /// @param _batchId The identification of the batch.
  function revertBatch(bytes32 _batchId) external;

  /// @notice finalize commited batch in layer 1
  /// @dev will add more parameters if needed.
  /// @param _batchId The identification of the commited batch.
  /// @param _proof The corresponding proof of the commited batch.
  /// @param _instances Instance used to verify, generated from batch.
  function finalizeBatchWithProof(
    bytes32 _batchId,
    uint256[] memory _proof,
    uint256[] memory _instances
  ) external;
}

// File: src/libraries/IScrollMessenger.sol



pragma solidity ^0.8.0;

interface IScrollMessenger {
  /**************************************** Events ****************************************/

  event SentMessage(
    address indexed target,
    address sender,
    uint256 value,
    uint256 fee,
    uint256 deadline,
    bytes message,
    uint256 messageNonce,
    uint256 gasLimit
  );
  event MessageDropped(bytes32 indexed msgHash);
  event RelayedMessage(bytes32 indexed msgHash);
  event FailedRelayedMessage(bytes32 indexed msgHash);

  /**************************************** View Functions ****************************************/

  function xDomainMessageSender() external view returns (address);

  /**************************************** Mutated Functions ****************************************/

  /// @notice Send cross chain message (L1 => L2 or L2 => L1)
  /// @dev Currently, only privileged accounts can call this function for safty. And adding an extra
  /// `_fee` variable make it more easy to upgrade to decentralized version.
  /// @param _to The address of account who recieve the message.
  /// @param _fee The amount of fee in Ether the caller would like to pay to the relayer.
  /// @param _message The content of the message.
  /// @param _gasLimit Unused, but included for potential forward compatibility considerations.
  function sendMessage(
    address _to,
    uint256 _fee,
    bytes memory _message,
    uint256 _gasLimit
  ) external payable;

  // @todo add comments
  function dropMessage(
    address _from,
    address _to,
    uint256 _value,
    uint256 _fee,
    uint256 _deadline,
    uint256 _nonce,
    bytes memory _message,
    uint256 _gasLimit
  ) external;
}

// File: src/L1/IL1ScrollMessenger.sol



pragma solidity ^0.8.0;

interface IL1ScrollMessenger is IScrollMessenger {
  struct L2MessageProof {
    // @todo add more fields
    uint256 batchIndex;
    uint256 blockHeight;
    bytes merkleProof;
  }

  /**************************************** Mutated Functions ****************************************/

  /// @notice execute L2 => L1 message
  /// @param _from The address of the sender of the message.
  /// @param _to The address of the recipient of the message.
  /// @param _value The msg.value passed to the message call.
  /// @param _fee The amount of fee in ETH to charge.
  /// @param _deadline The deadline of the message.
  /// @param _nonce The nonce of the message to avoid replay attack.
  /// @param _message The content of the message.
  /// @param _proof The proof used to verify the correctness of the transaction.
  function relayMessageWithProof(
    address _from,
    address _to,
    uint256 _value,
    uint256 _fee,
    uint256 _deadline,
    uint256 _nonce,
    bytes memory _message,
    L2MessageProof memory _proof
  ) external;

  /// @notice Replay an exsisting message.
  /// @param _from The address of the sender of the message.
  /// @param _to The address of the recipient of the message.
  /// @param _value The msg.value passed to the message call.
  /// @param _fee The amount of fee in ETH to charge.
  /// @param _deadline The deadline of the message.
  /// @param _message The content of the message.
  /// @param _queueIndex CTC Queue index for the message to replay.
  /// @param _oldGasLimit Original gas limit used to send the message.
  /// @param _newGasLimit New gas limit to be used for this message.
  function replayMessage(
    address _from,
    address _to,
    uint256 _value,
    uint256 _fee,
    uint256 _deadline,
    bytes memory _message,
    uint256 _queueIndex,
    uint32 _oldGasLimit,
    uint32 _newGasLimit
  ) external;
}

// File: src/libraries/oracle/IGasOracle.sol



pragma solidity ^0.8.0;

interface IGasOracle {
  /// @notice Estimate fee for cross chain message call.
  /// @param _sender The address of sender who invoke the call.
  /// @param _to The target address to receive the call.
  /// @param _message The message will be passed to the target address.
  function estimateMessageFee(
    address _sender,
    address _to,
    bytes memory _message
  ) external view returns (uint256);
}

// File: src/libraries/ScrollConstants.sol



pragma solidity ^0.8.0;

library ScrollConstants {
  /// @notice The address of default cross chain message sender.
  address internal constant DEFAULT_XDOMAIN_MESSAGE_SENDER = address(1);

  /// @notice The minimum seconds needed to wait if we want to drop message.
  uint256 internal constant MIN_DROP_DELAY_DURATION = 7 days;
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



abstract contract ScrollMessengerBase is IScrollMessenger {
  /**************************************** Events ****************************************/

  /// @notice Emitted when owner updates gas oracle contract.
  /// @param _oldGasOracle The address of old gas oracle contract.
  /// @param _newGasOracle The address of new gas oracle contract.
  event UpdateGasOracle(address _oldGasOracle, address _newGasOracle);

  /// @notice Emitted when owner updates whitelist contract.
  /// @param _oldWhitelist The address of old whitelist contract.
  /// @param _newWhitelist The address of new whitelist contract.
  event UpdateWhitelist(address _oldWhitelist, address _newWhitelist);

  /// @notice Emitted when owner updates drop delay duration
  /// @param _oldDuration The old drop delay duration in seconds.
  /// @param _newDuration The new drop delay duration in seconds.
  event UpdateDropDelayDuration(uint256 _oldDuration, uint256 _newDuration);

  /**************************************** Variables ****************************************/

  /// @notice See {IScrollMessenger-xDomainMessageSender}
  address public override xDomainMessageSender;

  /// @notice The gas oracle used to estimate transaction fee on layer 2.
  address public gasOracle;

  /// @notice The whitelist contract to track the sender who can call `sendMessage` in ScrollMessenger.
  address public whitelist;

  /// @notice The amount of seconds needed to wait if we want to drop message.
  uint256 public dropDelayDuration;

  modifier onlyWhitelistedSender(address _sender) {
    address _whitelist = whitelist;
    require(_whitelist == address(0) || IWhitelist(_whitelist).isSenderAllowed(_sender), "sender not whitelisted");
    _;
  }

  function _initialize() internal {
    // initialize to a nonzero value
    xDomainMessageSender = ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER;

    dropDelayDuration = ScrollConstants.MIN_DROP_DELAY_DURATION;
  }

  // allow others to send ether to messenger
  receive() external payable {}
}

// File: src/libraries/verifier/ZkTrieVerifier.sol



pragma solidity ^0.8.0;

library ZkTrieVerifier {
  function verifyMerkleProof(bytes memory) internal pure returns (bool) {
    return true;
  }
}

// File: src/L1/L1ScrollMessenger.sol



pragma solidity ^0.8.0;







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
contract L1ScrollMessenger is OwnableUpgradeable, PausableUpgradeable, ScrollMessengerBase, IL1ScrollMessenger {
  /**************************************** Variables ****************************************/

  /// @notice Mapping from relay id to relay status.
  mapping(bytes32 => bool) public isMessageRelayed;

  /// @notice Mapping from message hash to drop status.
  mapping(bytes32 => bool) public isMessageDropped;

  /// @notice Mapping from message hash to execution status.
  mapping(bytes32 => bool) public isMessageExecuted;

  /// @notice The address of Rollup contract.
  address public rollup;

  /**************************************** Constructor ****************************************/

  function initialize(address _rollup) public initializer {
    OwnableUpgradeable.__Ownable_init();
    PausableUpgradeable.__Pausable_init();
    ScrollMessengerBase._initialize();

    rollup = _rollup;
    // initialize to a nonzero value
    xDomainMessageSender = ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER;
  }

  /**************************************** Mutated Functions ****************************************/

  /// @inheritdoc IScrollMessenger
  function sendMessage(
    address _to,
    uint256 _fee,
    bytes memory _message,
    uint256 _gasLimit
  ) external payable override whenNotPaused onlyWhitelistedSender(msg.sender) {
    require(msg.value >= _fee, "cannot pay fee");

    // solhint-disable-next-line not-rely-on-time
    uint256 _deadline = block.timestamp + dropDelayDuration;
    // compute minimum fee required by GasOracle contract.
    uint256 _minFee = gasOracle == address(0) ? 0 : IGasOracle(gasOracle).estimateMessageFee(msg.sender, _to, _message);
    require(_fee >= _minFee, "fee too small");
    uint256 _value;
    unchecked {
      _value = msg.value - _fee;
    }

    uint256 _nonce = IZKRollup(rollup).appendMessage(msg.sender, _to, _value, _fee, _deadline, _message, _gasLimit);

    emit SentMessage(_to, msg.sender, _value, _fee, _deadline, _message, _nonce, _gasLimit);
  }

  /// @inheritdoc IL1ScrollMessenger
  function relayMessageWithProof(
    address _from,
    address _to,
    uint256 _value,
    uint256 _fee,
    uint256 _deadline,
    uint256 _nonce,
    bytes memory _message,
    L2MessageProof memory _proof
  ) external override whenNotPaused onlyWhitelistedSender(msg.sender) {
    require(xDomainMessageSender == ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER, "already in execution");

    // solhint-disable-next-line not-rely-on-time
    // @note disable for now since we cannot generate proof in time.
    // require(_deadline >= block.timestamp, "Message expired");

    bytes32 _msghash = keccak256(abi.encodePacked(_from, _to, _value, _fee, _deadline, _nonce, _message));

    require(!isMessageExecuted[_msghash], "Message successfully executed");

    // @todo check proof
    require(IZKRollup(rollup).verifyMessageStateProof(_proof.batchIndex, _proof.blockHeight), "invalid state proof");
    require(ZkTrieVerifier.verifyMerkleProof(_proof.merkleProof), "invalid proof");

    // @todo check `_to` address to avoid attack.

    // @todo take fee and distribute to relayer later.

    // @note This usually will never happen, just in case.
    require(_from != xDomainMessageSender, "invalid message sender");

    xDomainMessageSender = _from;
    // solhint-disable-next-line avoid-low-level-calls
    (bool success, ) = _to.call{ value: _value }(_message);
    // reset value to refund gas.
    xDomainMessageSender = ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER;

    if (success) {
      isMessageExecuted[_msghash] = true;
      emit RelayedMessage(_msghash);
    } else {
      emit FailedRelayedMessage(_msghash);
    }

    bytes32 _relayId = keccak256(abi.encodePacked(_msghash, msg.sender, block.number));

    isMessageRelayed[_relayId] = true;
  }

  /// @inheritdoc IL1ScrollMessenger
  function replayMessage(
    address _from,
    address _to,
    uint256 _value,
    uint256 _fee,
    uint256 _deadline,
    bytes memory _message,
    uint256 _queueIndex,
    uint32 _oldGasLimit,
    uint32 _newGasLimit
  ) external override whenNotPaused {
    // @todo
  }

  /// @inheritdoc IScrollMessenger
  function dropMessage(
    address _from,
    address _to,
    uint256 _value,
    uint256 _fee,
    uint256 _deadline,
    uint256 _nonce,
    bytes memory _message,
    uint256 _gasLimit
  ) external override whenNotPaused {
    // solhint-disable-next-line not-rely-on-time
    require(block.timestamp > _deadline, "message not expired");

    // @todo The `queueIndex` is acutally updated asynchronously, it's not a good practice to compare directly.
    address _rollup = rollup; // gas saving
    uint256 _queueIndex = IZKRollup(_rollup).getNextQueueIndex();
    require(_queueIndex <= _nonce, "message already executed");

    bytes32 _expectedMessageHash = IZKRollup(_rollup).getMessageHashByIndex(_nonce);
    bytes32 _messageHash = keccak256(
      abi.encodePacked(_from, _to, _value, _fee, _deadline, _nonce, _message, _gasLimit)
    );
    require(_messageHash == _expectedMessageHash, "message hash mismatched");

    require(!isMessageDropped[_messageHash], "message already dropped");
    isMessageDropped[_messageHash] = true;

    if (_from.code.length > 0) {
      // @todo call finalizeDropMessage of `_from`
    } else {
      // just do simple ether refund
      payable(_from).transfer(_value + _fee);
    }

    emit MessageDropped(_messageHash);
  }

  /**************************************** Restricted Functions ****************************************/

  /// @notice Pause the contract
  /// @dev This function can only called by contract owner.
  function pause() external onlyOwner {
    _pause();
  }

  /// @notice Update whitelist contract.
  /// @dev This function can only called by contract owner.
  /// @param _newWhitelist The address of new whitelist contract.
  function updateWhitelist(address _newWhitelist) external onlyOwner {
    address _oldWhitelist = whitelist;

    whitelist = _newWhitelist;
    emit UpdateWhitelist(_oldWhitelist, _newWhitelist);
  }

  /// @notice Update the address of gas oracle.
  /// @dev This function can only called by contract owner.
  /// @param _newGasOracle The address to update.
  function updateGasOracle(address _newGasOracle) external onlyOwner {
    address _oldGasOracle = gasOracle;
    gasOracle = _newGasOracle;

    emit UpdateGasOracle(_oldGasOracle, _newGasOracle);
  }

  /// @notice Update the drop delay duration.
  /// @dev This function can only called by contract owner.
  /// @param _newDuration The new delay duration to update.
  function updateDropDelayDuration(uint256 _newDuration) external onlyOwner {
    uint256 _oldDuration = dropDelayDuration;
    dropDelayDuration = _newDuration;

    emit UpdateDropDelayDuration(_oldDuration, _newDuration);
  }
}
