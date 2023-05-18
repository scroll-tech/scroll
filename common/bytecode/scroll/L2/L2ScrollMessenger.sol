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

// File: src/L2/IL2ScrollMessenger.sol



pragma solidity ^0.8.0;

interface IL2ScrollMessenger is IScrollMessenger {
  /***********
   * Structs *
   ***********/

  struct L1MessageProof {
    bytes32 blockHash;
    bytes stateRootProof;
  }

  /****************************
   * Public Mutated Functions *
   ****************************/

  /// @notice execute L1 => L2 message
  /// @dev Make sure this is only called by privileged accounts.
  /// @param from The address of the sender of the message.
  /// @param to The address of the recipient of the message.
  /// @param value The msg.value passed to the message call.
  /// @param nonce The nonce of the message to avoid replay attack.
  /// @param message The content of the message.
  function relayMessage(
    address from,
    address to,
    uint256 value,
    uint256 nonce,
    bytes calldata message
  ) external;

  /// @notice execute L1 => L2 message with proof
  /// @param from The address of the sender of the message.
  /// @param to The address of the recipient of the message.
  /// @param value The msg.value passed to the message call.
  /// @param nonce The nonce of the message to avoid replay attack.
  /// @param message The content of the message.
  /// @param proof The message proof.
  function retryMessageWithProof(
    address from,
    address to,
    uint256 value,
    uint256 nonce,
    bytes calldata message,
    L1MessageProof calldata proof
  ) external;
}

// File: src/libraries/common/AppendOnlyMerkleTree.sol



pragma solidity ^0.8.0;

abstract contract AppendOnlyMerkleTree {
  /// @dev The maximum height of the withdraw merkle tree.
  uint256 private constant MAX_TREE_HEIGHT = 40;

  /// @notice The merkle root of the current merkle tree.
  /// @dev This is actual equal to `branches[n]`.
  bytes32 public messageRoot;

  /// @notice The next unused message index.
  uint256 public nextMessageIndex;

  /// @notice The list of zero hash in each height.
  bytes32[MAX_TREE_HEIGHT] private zeroHashes;

  /// @notice The list of minimum merkle proofs needed to compute next root.
  /// @dev Only first `n` elements are used, where `n` is the minimum value that `2^{n-1} >= currentMaxNonce + 1`.
  /// It means we only use `currentMaxNonce + 1` leaf nodes to construct the merkle tree.
  bytes32[MAX_TREE_HEIGHT] public branches;

  function _initializeMerkleTree() internal {
    // Compute hashes in empty sparse Merkle tree
    for (uint256 height = 0; height + 1 < MAX_TREE_HEIGHT; height++) {
      zeroHashes[height + 1] = _efficientHash(zeroHashes[height], zeroHashes[height]);
    }
  }

  function _appendMessageHash(bytes32 _messageHash) internal returns (uint256, bytes32) {
    uint256 _currentMessageIndex = nextMessageIndex;
    bytes32 _hash = _messageHash;
    uint256 _height = 0;
    // @todo it can be optimized, since we only need the newly added branch.
    while (_currentMessageIndex != 0) {
      if (_currentMessageIndex % 2 == 0) {
        // it may be used in next round.
        branches[_height] = _hash;
        // it's a left child, the right child must be null
        _hash = _efficientHash(_hash, zeroHashes[_height]);
      } else {
        // it's a right child, use previously computed hash
        _hash = _efficientHash(branches[_height], _hash);
      }
      unchecked {
        _height += 1;
      }
      _currentMessageIndex >>= 1;
    }

    branches[_height] = _hash;
    messageRoot = _hash;

    _currentMessageIndex = nextMessageIndex;
    unchecked {
      nextMessageIndex = _currentMessageIndex + 1;
    }

    return (_currentMessageIndex, _hash);
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

// File: src/libraries/common/OwnableBase.sol



pragma solidity ^0.8.0;

abstract contract OwnableBase {
  /**********
   * Events *
   **********/

  /// @notice Emitted when owner is changed by current owner.
  /// @param _oldOwner The address of previous owner.
  /// @param _newOwner The address of new owner.
  event OwnershipTransferred(address indexed _oldOwner, address indexed _newOwner);

  /*************
   * Variables *
   *************/

  /// @notice The address of the current owner.
  address public owner;

  /**********************
   * Function Modifiers *
   **********************/

  /// @dev Throws if called by any account other than the owner.
  modifier onlyOwner() {
    require(owner == msg.sender, "caller is not the owner");
    _;
  }

  /************************
   * Restricted Functions *
   ************************/

  /// @notice Leaves the contract without owner. It will not be possible to call
  /// `onlyOwner` functions anymore. Can only be called by the current owner.
  ///
  /// @dev Renouncing ownership will leave the contract without an owner,
  /// thereby removing any functionality that is only available to the owner.
  function renounceOwnership() public onlyOwner {
    _transferOwnership(address(0));
  }

  /// @notice Transfers ownership of the contract to a new account (`newOwner`).
  /// Can only be called by the current owner.
  function transferOwnership(address _newOwner) public onlyOwner {
    require(_newOwner != address(0), "new owner is the zero address");
    _transferOwnership(_newOwner);
  }

  /**********************
   * Internal Functions *
   **********************/

  /// @dev Transfers ownership of the contract to a new account (`newOwner`).
  /// Internal function without access restriction.
  function _transferOwnership(address _newOwner) internal {
    address _oldOwner = owner;
    owner = _newOwner;
    emit OwnershipTransferred(_oldOwner, _newOwner);
  }
}

// File: src/L2/predeploys/L2MessageQueue.sol



pragma solidity ^0.8.0;


/// @title L2MessageQueue
/// @notice The original idea is from Optimism, see [OVM_L2ToL1MessagePasser](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts/contracts/L2/predeploys/OVM_L2ToL1MessagePasser.sol).
/// The L2 to L1 Message Passer is a utility contract which facilitate an L1 proof of the
/// of a message on L2. The L1 Cross Domain Messenger performs this proof in its
/// _verifyStorageProof function, which verifies the existence of the transaction hash in this
/// contract's `sentMessages` mapping.
contract L2MessageQueue is AppendOnlyMerkleTree, OwnableBase {
  /// @notice Emitted when a new message is added to the merkle tree.
  /// @param index The index of the corresponding message.
  /// @param messageHash The hash of the corresponding message.
  event AppendMessage(uint256 index, bytes32 messageHash);

  /// @notice The address of L2ScrollMessenger contract.
  address public messenger;

  constructor(address _owner) {
    _transferOwnership(_owner);
  }

  function initialize() external {
    _initializeMerkleTree();
  }

  /// @notice record the message to merkle tree and compute the new root.
  /// @param _messageHash The hash of the new added message.
  function appendMessage(bytes32 _messageHash) external returns (bytes32) {
    require(msg.sender == messenger, "only messenger");

    (uint256 _currentNonce, bytes32 _currentRoot) = _appendMessageHash(_messageHash);

    // We can use the event to compute the merkle tree locally.
    emit AppendMessage(_currentNonce, _messageHash);

    return _currentRoot;
  }

  /// @notice Update the address of messenger.
  /// @dev You are not allowed to update messenger when there are some messages appended.
  /// @param _messenger The address of messenger to update.
  function updateMessenger(address _messenger) external onlyOwner {
    require(nextMessageIndex == 0, "cannot update messenger");

    messenger = _messenger;
  }
}

// File: src/L2/predeploys/IL1BlockContainer.sol



pragma solidity ^0.8.0;

interface IL1BlockContainer {
  /**********
   * Events *
   **********/

  /// @notice Emitted when a block is imported.
  /// @param blockHash The hash of the imported block.
  /// @param blockHeight The height of the imported block.
  /// @param blockTimestamp The timestamp of the imported block.
  /// @param baseFee The base fee of the imported block.
  /// @param stateRoot The state root of the imported block.
  event ImportBlock(
    bytes32 indexed blockHash,
    uint256 blockHeight,
    uint256 blockTimestamp,
    uint256 baseFee,
    bytes32 stateRoot
  );

  /*************************
   * Public View Functions *
   *************************/

  /// @notice Return the latest imported block hash
  function latestBlockHash() external view returns (bytes32);

  /// @notice Return the latest imported L1 base fee
  function latestBaseFee() external view returns (uint256);

  /// @notice Return the latest imported block number
  function latestBlockNumber() external view returns (uint256);

  /// @notice Return the latest imported block timestamp
  function latestBlockTimestamp() external view returns (uint256);

  /// @notice Return the state root of given block.
  /// @param blockHash The block hash to query.
  /// @return stateRoot The state root of the block.
  function getStateRoot(bytes32 blockHash) external view returns (bytes32 stateRoot);

  /// @notice Return the block timestamp of given block.
  /// @param blockHash The block hash to query.
  /// @return timestamp The corresponding block timestamp.
  function getBlockTimestamp(bytes32 blockHash) external view returns (uint256 timestamp);

  /****************************
   * Public Mutated Functions *
   ****************************/

  /// @notice Import L1 block header to this contract.
  /// @param blockHash The hash of block.
  /// @param blockHeaderRLP The RLP encoding of L1 block.
  /// @param updateGasPriceOracle Whether to update gas price oracle.
  function importBlockHeader(
    bytes32 blockHash,
    bytes calldata blockHeaderRLP,
    bool updateGasPriceOracle
  ) external;
}

// File: src/L2/predeploys/IL1GasPriceOracle.sol



pragma solidity ^0.8.0;

interface IL1GasPriceOracle {
  /**********
   * Events *
   **********/

  /// @notice Emitted when current fee overhead is updated.
  /// @param overhead The current fee overhead updated.
  event OverheadUpdated(uint256 overhead);

  /// @notice Emitted when current fee scalar is updated.
  /// @param scalar The current fee scalar updated.
  event ScalarUpdated(uint256 scalar);

  /// @notice Emitted when current l1 base fee is updated.
  /// @param l1BaseFee The current l1 base fee updated.
  event L1BaseFeeUpdated(uint256 l1BaseFee);

  /*************************
   * Public View Functions *
   *************************/

  /// @notice Return the current l1 fee overhead.
  function overhead() external view returns (uint256);

  /// @notice Return the current l1 fee scalar.
  function scalar() external view returns (uint256);

  /// @notice Return the latest known l1 base fee.
  function l1BaseFee() external view returns (uint256);

  /// @notice Computes the L1 portion of the fee based on the size of the rlp encoded input
  ///         transaction, the current L1 base fee, and the various dynamic parameters.
  /// @param data Unsigned fully RLP-encoded transaction to get the L1 fee for.
  /// @return L1 fee that should be paid for the tx
  function getL1Fee(bytes memory data) external view returns (uint256);

  /// @notice Computes the amount of L1 gas used for a transaction. Adds the overhead which
  ///         represents the per-transaction gas overhead of posting the transaction and state
  ///         roots to L1. Adds 68 bytes of padding to account for the fact that the input does
  ///         not have a signature.
  /// @param data Unsigned fully RLP-encoded transaction to get the L1 gas for.
  /// @return Amount of L1 gas used to publish the transaction.
  function getL1GasUsed(bytes memory data) external view returns (uint256);

  /****************************
   * Public Mutated Functions *
   ****************************/

  /// @notice Allows whitelisted caller to modify the l1 base fee.
  /// @param _l1BaseFee New l1 base fee.
  function setL1BaseFee(uint256 _l1BaseFee) external;
}

// File: src/libraries/verifier/PatriciaMerkleTrieVerifier.sol



pragma solidity ^0.8.0;

library PatriciaMerkleTrieVerifier {
  /// @notice Internal function to validates a proof from eth_getProof.
  /// @param account The address of the contract.
  /// @param storageKey The storage slot to verify.
  /// @param proof The rlp encoding result of eth_getProof.
  /// @return stateRoot The computed state root. Must be checked by the caller.
  /// @return storageValue The value of `storageKey`.
  ///
  /// @dev The code is based on
  /// 1. https://eips.ethereum.org/EIPS/eip-1186
  /// 2. https://ethereum.org/en/developers/docs/data-structures-and-encoding/rlp/
  /// 3. https://github.com/ethereum/go-ethereum/blob/master/trie/proof.go#L114
  /// 4. https://github.com/privacy-scaling-explorations/zkevm-chain/blob/master/contracts/templates/PatriciaValidator.sol
  ///
  /// The encoding order of `proof` is
  /// ```text
  /// |        1 byte        |      ...      |        1 byte        |      ...      |
  /// | account proof length | account proof | storage proof length | storage proof |
  /// ```
  function verifyPatriciaProof(
    address account,
    bytes32 storageKey,
    bytes calldata proof
  ) internal pure returns (bytes32 stateRoot, bytes32 storageValue) {
    assembly {
      // hashes 32 bytes of `v`
      function keccak_32(v) -> r {
        mstore(0x00, v)
        r := keccak256(0x00, 0x20)
      }
      // hashes the last 20 bytes of `v`
      function keccak_20(v) -> r {
        mstore(0x00, v)
        r := keccak256(0x0c, 0x14)
      }
      // reverts with error `msg`.
      // make sure the length of error string <= 32
      function revertWith(msg) {
        // keccak("Error(string)")
        mstore(0x00, shl(224, 0x08c379a0))
        mstore(0x04, 0x20) // str.offset
        mstore(0x44, msg)
        let msgLen
        for {} msg {} {
          msg := shl(8, msg)
          msgLen := add(msgLen, 1)
        }
        mstore(0x24, msgLen) // str.length
        revert(0x00, 0x64)
      }
      // reverts with `msg` when condition is not matched.
      // make sure the length of error string <= 32
      function require(cond, msg) {
        if iszero(cond) {
          revertWith(msg)
        }
      }

      // special function for decoding the storage value
      // because of the prefix truncation if value > 31 bytes
      // see `loadValue`
      function decodeItem(word, len) -> ret {
        // default
        ret := word

        // RLP single byte
        if lt(word, 0x80) {
          leave
        }

        // truncated
        if gt(len, 32) {
          leave
        }

        // value is >= 0x80 and <= 32 bytes.
        // `len` should be at least 2 (prefix byte + value)
        // otherwise the RLP is malformed.
        let bits := mul(len, 8)
        // sub 8 bits - the prefix
        bits := sub(bits, 8)
        let mask := shl(bits, 0xff)
        // invert the mask
        mask := not(mask)
        // should hold the value - prefix byte
        ret := and(ret, mask)
      }

      // returns the `len` of the whole RLP list at `ptr`
      // and the offset for the first value inside the list.
      function decodeListLength(ptr) -> len, startOffset {
        let b0 := byte(0, calldataload(ptr))
        // In most cases, it is a long list. So we reorder the branch to reduce branch prediction miss.

        // 0xf8 - 0xff, long list, length > 55
        if gt(b0, 0xf7) {
          // the RLP encoding consists of a single byte with value 0xf7 
          // plus the length in bytes of the length of the payload in binary form,
          // followed by the length of the payload, followed by the concatenation 
          // of the RLP encodings of the items.
          // the extended length is ignored
          let lengthBytes := sub(b0, 0xf7)
          if gt(lengthBytes, 32) {
            invalid()
          }

          // load the extended length
          startOffset := add(ptr, 1)
          let extendedLen := calldataload(startOffset)
          let bits := sub(256, mul(lengthBytes, 8))
          extendedLen := shr(bits, extendedLen)

          len := add(extendedLen, lengthBytes)
          len := add(len, 1)
          startOffset := add(startOffset, lengthBytes)
          leave
        }
        // 0xc0 - 0xf7, short list, length <= 55
        if gt(b0, 0xbf) {
          // the RLP encoding consists of a single byte with value 0xc0
          // plus the length of the list followed by the concatenation of
          // the RLP encodings of the items.
          len := sub(b0, 0xbf)
          startOffset := add(ptr, 1)
          leave
        }
        revertWith("Not list")
      }

      // returns the kind, calldata offset of the value and the length in bytes
      // for the RLP encoded data item at `ptr`. used in `decodeFlat`
      // kind = 0 means string/bytes, kind = 1 means list.
      function decodeValue(ptr) -> kind, dataLen, valueOffset {
        let b0 := byte(0, calldataload(ptr))

        // 0x00 - 0x7f, single byte
        if lt(b0, 0x80) {
          // for a single byte whose value is in the [0x00, 0x7f] range,
          // that byte is its own RLP encoding.
          dataLen := 1
          valueOffset := ptr
          leave
        }

        // 0x80 - 0xb7, short string/bytes, length <= 55
        if lt(b0, 0xb8) {
          // the RLP encoding consists of a single byte with value 0x80
          // plus the length of the string followed by the string.
          dataLen := sub(b0, 0x80)
          valueOffset := add(ptr, 1)
          leave
        }

        // 0xb8 - 0xbf, long string/bytes, length > 55
        if lt(b0, 0xc0) {
          // the RLP encoding consists of a single byte with value 0xb7
          // plus the length in bytes of the length of the string in binary form,
          // followed by the length of the string, followed by the string.
          let lengthBytes := sub(b0, 0xb7)
          if gt(lengthBytes, 4) {
            invalid()
          }

          // load the extended length
          valueOffset := add(ptr, 1)
          let extendedLen := calldataload(valueOffset)
          let bits := sub(256, mul(lengthBytes, 8))
          extendedLen := shr(bits, extendedLen)

          dataLen := extendedLen
          valueOffset := add(valueOffset, lengthBytes)
          leave
        }

        kind := 1
        // 0xc0 - 0xf7, short list, length <= 55
        if lt(b0, 0xf8) {
          // intentionally ignored
          // dataLen := sub(firstByte, 0xc0)
          valueOffset := add(ptr, 1)
          leave
        }

        // 0xf8 - 0xff, long list, length > 55
        {
          // the extended length is ignored
          dataLen := sub(b0, 0xf7)
          valueOffset := add(ptr, 1)
          leave
        }
      }

      // decodes all RLP encoded data and stores their DATA items
      // [length - 128 bits | calldata offset - 128 bits] in a continous memory region.
      // Expects that the RLP starts with a list that defines the length
      // of the whole RLP region.
      function decodeFlat(_ptr) -> ptr, memStart, nItems, hash {
        ptr := _ptr

        // load free memory ptr
        // doesn't update the ptr and leaves the memory region dirty
        memStart := mload(0x40)

        let payloadLen, startOffset := decodeListLength(ptr)
        // reuse memStart region and hash
        calldatacopy(memStart, ptr, payloadLen)
        hash := keccak256(memStart, payloadLen)

        let memPtr := memStart
        let ptrStop := add(ptr, payloadLen)
        ptr := startOffset

        // decode until the end of the list
        for {} lt(ptr, ptrStop) {} {
          let kind, len, valuePtr := decodeValue(ptr)
          ptr := add(len, valuePtr)

          if iszero(kind) {
            // store the length of the data and the calldata offset
            // low -------> high
            // |     128 bits    |   128 bits   |
            // | calldata offset | value length |
            mstore(memPtr, or(shl(128, len), valuePtr))
            memPtr := add(memPtr, 0x20)
          }
        }

        if iszero(eq(ptr, ptrStop)) {
          invalid()
        }

        nItems := div( sub(memPtr, memStart), 32 )
      }

      // prefix gets truncated to 256 bits
      // `depth` is untrusted and can lead to bogus
      // shifts/masks. In that case, the remaining verification
      // steps must fail or lead to an invalid stateRoot hash
      // if the proof data is 'spoofed but valid'
      function derivePath(key, depth) -> path {
        path := key

        let bits := mul(depth, 4)
        {
          let mask := not(0)
          mask := shr(bits, mask)
          path := and(path, mask)
        }

        // even prefix
        let prefix := 0x20
        if mod(depth, 2) {
          // odd
          prefix := 0x3
        }

        // the prefix may be shifted outside bounds
        // this is intended, see `loadValue`
        bits := sub(256, bits)
        prefix := shl(bits, prefix)
        path := or(prefix, path)
      }

      // loads and aligns a value from calldata
      // given the `len|offset` stored at `memPtr`
      function loadValue(memPtr, idx) -> value {
        let tmp := mload(add(memPtr, mul(32, idx)))
        // assuming 0xffffff is sufficient for storing calldata offset
        let offset := and(tmp, 0xffffff)
        let len := shr(128, tmp)

        if gt(len, 31) {
          // special case - truncating the value is intended.
          // this matches the behavior in `derivePath` that truncates to 256 bits.
          offset := add(offset, sub(len, 32))
          value := calldataload(offset)
          leave
        }

        // everything else is
        // < 32 bytes - align the value
        let bits := mul( sub(32, len), 8)
        value := calldataload(offset)
        value := shr(bits, value)
      }

      // loads and aligns a value from calldata
      // given the `len|offset` stored at `memPtr`
      // Same as `loadValue` except it returns also the size
      // of the value.
      function loadValueLen(memPtr, idx) -> value, len {
        let tmp := mload(add(memPtr, mul(32, idx)))
        // assuming 0xffffff is sufficient for storing calldata offset
        let offset := and(tmp, 0xffffff)
        len := shr(128, tmp)

        if gt(len, 31) {
          // special case - truncating the value is intended.
          // this matches the behavior in `derivePath` that truncates to 256 bits.
          offset := add(offset, sub(len, 32))
          value := calldataload(offset)
          leave
        }

        // everything else is
        // < 32 bytes - align the value
        let bits := mul( sub(32, len), 8)
        value := calldataload(offset)
        value := shr(bits, value)
      }

      function loadPair(memPtr, idx) -> offset, len {
        let tmp := mload(add(memPtr, mul(32, idx)))
        // assuming 0xffffff is sufficient for storing calldata offset
        offset := and(tmp, 0xffffff)
        len := shr(128, tmp)
      }

      // decodes RLP at `_ptr`.
      // reverts if the number of DATA items doesn't match `nValues`.
      // returns the RLP data items at pos `v0`, `v1`
      // and the size of `v1out`
      function hashCompareSelect(_ptr, nValues, v0, v1) -> ptr, hash, v0out, v1out, v1outlen {
        ptr := _ptr

        let memStart, nItems
        ptr, memStart, nItems, hash := decodeFlat(ptr)

        if iszero( eq(nItems, nValues) ) {
          revertWith('Node items mismatch')
        }

        v0out, v1outlen := loadValueLen(memStart, v0)
        v1out, v1outlen := loadValueLen(memStart, v1)
      }

      // traverses the tree from the root to the node before the leaf.
      // based on https://github.com/ethereum/go-ethereum/blob/master/trie/proof.go#L114
      function walkTree(key, _ptr) -> ptr, rootHash, expectedHash, path {
        ptr := _ptr

        // the first byte is the number of nodes
        let nodes := byte(0, calldataload(ptr))
        ptr := add(ptr, 1)

        // keeps track of ascend/descend - however you may look at a tree
        let depth

        // treat the leaf node with different logic
        for { let i := 1 } lt(i, nodes) { i := add(i, 1) } {
          let memStart, nItems, hash
          ptr, memStart, nItems, hash := decodeFlat(ptr)

          // first item is considered the root node.
          // Otherwise verifies that the hash of the current node
          // is the same as the previous choosen one.
          switch i
          case 1 {
            rootHash := hash
          } default {
            require(eq(hash, expectedHash), 'Hash mismatch')
          }

          switch nItems
          case 2 {
            // extension node
            // load the second item.
            // this is the hash of the next node.
            let value, len := loadValueLen(memStart, 1)
            expectedHash := value

            // get the byte length of the first item
            // Note: the value itself is not validated
            // and it is instead assumed that any invalid
            // value is invalidated by comparing the root hash.
            let prefixLen := shr(128, mload(memStart))
            depth := add(depth, prefixLen)
          }
          case 17 {
            let bits := sub(252, mul(depth, 4))
            let nibble := and(shr(bits, key), 0xf)

            // load the value at pos `nibble`
            let value, len := loadValueLen(memStart, nibble)

            expectedHash := value
            depth := add(depth, 1)
          }
          default {
            // everything else is unexpected
            revertWith('Invalid node')
          }
        }

        // lastly, derive the path of the choosen one (TM)
        path := derivePath(key, depth)
      }
      
      // shared variable names
      let storageHash
      let encodedPath
      let path
      let hash
      let vlen
      // starting point
      let ptr := proof.offset

      {
        // account proof
        // Note: this doesn't work if there are no intermediate nodes before the leaf.
        // This is not possible in practice because of the fact that there must be at least
        // 2 accounts in the tree to make a transaction to a existing contract possible.
        // Thus, 2 leaves.
        let prevHash
        let key := keccak_20(account)
        // `stateRoot` is a return value and must be checked by the caller
        ptr, stateRoot, prevHash, path := walkTree(key, ptr)

        let memStart, nItems
        ptr, memStart, nItems, hash := decodeFlat(ptr)

        // the hash of the leaf must match the previous hash from the node
        require(eq(hash, prevHash), 'Account leaf hash mismatch')

        // 2 items
        // - encoded path
        // - account leaf RLP (4 items)
        require(eq(nItems, 2), "Account leaf node mismatch")

        encodedPath := loadValue(memStart, 0)
        // the calculated path must match the encoded path in the leaf
        require(eq(path, encodedPath), 'Account encoded path mismatch')

        // Load the position, length of the second element (RLP encoded)
        let leafPtr, leafLen := loadPair(memStart, 1)
        leafPtr, memStart, nItems, hash := decodeFlat(leafPtr)

        // the account leaf should contain 4 values,
        // we want:
        // - storageHash @ 2
        require(eq(nItems, 4), "Account leaf items mismatch")
        storageHash := loadValue(memStart, 2)
      }

      {
        // storage proof
        let rootHash
        let key := keccak_32(storageKey)
        ptr, rootHash, hash, path := walkTree(key, ptr)

        // leaf should contain 2 values
        // - encoded path @ 0
        // - storageValue @ 1
        ptr, hash, encodedPath, storageValue, vlen := hashCompareSelect(ptr, 2, 0, 1)
        // the calculated path must match the encoded path in the leaf
        require(eq(path, encodedPath), 'Storage encoded path mismatch')

        switch rootHash
        case 0 {
          // in the case that the leaf is the only element, then
          // the hash of the leaf must match the value from the account leaf
          require(eq(hash, storageHash), 'Storage root mismatch')
        }
        default {
          // otherwise the root hash of the storage tree
          // must match the value from the account leaf
          require(eq(rootHash, storageHash), 'Storage root mismatch')
        }

        // storageValue is a return value
        storageValue := decodeItem(storageValue, vlen)
      }

      // the one and only boundary check
      // in case an attacker crafted a malicous payload
      // and succeeds in the prior verification steps
      // then this should catch any bogus accesses
      if iszero( eq(ptr, add(proof.offset, proof.length)) ) {
        revertWith('Proof length mismatch')
      }
    }
  }
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

// File: src/L2/L2ScrollMessenger.sol



pragma solidity ^0.8.0;







/// @title L2ScrollMessenger
/// @notice The `L2ScrollMessenger` contract can:
///
/// 1. send messages from layer 2 to layer 1;
/// 2. relay messages from layer 1 layer 2;
/// 3. drop expired message due to sequencer problems.
///
/// @dev It should be a predeployed contract in layer 2 and should hold infinite amount
/// of Ether (Specifically, `uint256(-1)`), which can be initialized in Genesis Block.
contract L2ScrollMessenger is ScrollMessengerBase, PausableUpgradeable, IL2ScrollMessenger {
  /**********
   * Events *
   **********/

  /// @notice Emitted when the maximum number of times each message can fail in L2 is updated.
  /// @param maxFailedExecutionTimes The new maximum number of times each message can fail in L2.
  event UpdateMaxFailedExecutionTimes(uint256 maxFailedExecutionTimes);

  /*************
   * Constants *
   *************/

  uint256 private constant MIN_GAS_LIMIT = 21000;

  /// @notice The contract contains the list of L1 blocks.
  address public immutable blockContainer;

  /// @notice The address of L2MessageQueue.
  address public immutable gasOracle;

  /// @notice The address of L2MessageQueue.
  address public immutable messageQueue;

  /*************
   * Variables *
   *************/

  /// @notice Mapping from L2 message hash to sent status.
  mapping(bytes32 => bool) public isL2MessageSent;

  /// @notice Mapping from L1 message hash to a boolean value indicating if the message has been successfully executed.
  mapping(bytes32 => bool) public isL1MessageExecuted;

  /// @notice Mapping from L1 message hash to the number of failure times.
  mapping(bytes32 => uint256) public l1MessageFailedTimes;

  /// @notice The maximum number of times each L1 message can fail on L2.
  uint256 public maxFailedExecutionTimes;

  /***************
   * Constructor *
   ***************/

  constructor(
    address _blockContainer,
    address _gasOracle,
    address _messageQueue
  ) {
    blockContainer = _blockContainer;
    gasOracle = _gasOracle;
    messageQueue = _messageQueue;
  }

  function initialize(address _counterpart, address _feeVault) external initializer {
    PausableUpgradeable.__Pausable_init();
    ScrollMessengerBase._initialize(_counterpart, _feeVault);

    maxFailedExecutionTimes = 3;

    // initialize to a nonzero value
    xDomainMessageSender = ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER;
  }

  /*************************
   * Public View Functions *
   *************************/

  /// @notice Check whether the l1 message is included in the corresponding L1 block.
  /// @param _blockHash The block hash where the message should in.
  /// @param _msgHash The hash of the message to check.
  /// @param _proof The encoded storage proof from eth_getProof.
  /// @return bool Return true is the message is included in L1, otherwise return false.
  function verifyMessageInclusionStatus(
    bytes32 _blockHash,
    bytes32 _msgHash,
    bytes calldata _proof
  ) public view returns (bool) {
    bytes32 _expectedStateRoot = IL1BlockContainer(blockContainer).getStateRoot(_blockHash);
    require(_expectedStateRoot != bytes32(0), "Block is not imported");

    // @todo fix the actual slot later.
    bytes32 _storageKey;
    // `mapping(bytes32 => bool) public isL1MessageSent` is the 105-nd slot of contract `L1ScrollMessenger`.
    assembly {
      mstore(0x00, _msgHash)
      mstore(0x20, 105)
      _storageKey := keccak256(0x00, 0x40)
    }

    (bytes32 _computedStateRoot, bytes32 _storageValue) = PatriciaMerkleTrieVerifier.verifyPatriciaProof(
      counterpart,
      _storageKey,
      _proof
    );
    require(_computedStateRoot == _expectedStateRoot, "State roots mismatch");

    return uint256(_storageValue) == 1;
  }

  /// @notice Check whether the message is executed in the corresponding L1 block.
  /// @param _blockHash The block hash where the message should in.
  /// @param _msgHash The hash of the message to check.
  /// @param _proof The encoded storage proof from eth_getProof.
  /// @return bool Return true is the message is executed in L1, otherwise return false.
  function verifyMessageExecutionStatus(
    bytes32 _blockHash,
    bytes32 _msgHash,
    bytes calldata _proof
  ) external view returns (bool) {
    bytes32 _expectedStateRoot = IL1BlockContainer(blockContainer).getStateRoot(_blockHash);
    require(_expectedStateRoot != bytes32(0), "Block not imported");

    // @todo fix the actual slot later.
    bytes32 _storageKey;
    // `mapping(bytes32 => bool) public isL2MessageExecuted` is the 106-th slot of contract `L1ScrollMessenger`.
    assembly {
      mstore(0x00, _msgHash)
      mstore(0x20, 106)
      _storageKey := keccak256(0x00, 0x40)
    }

    (bytes32 _computedStateRoot, bytes32 _storageValue) = PatriciaMerkleTrieVerifier.verifyPatriciaProof(
      counterpart,
      _storageKey,
      _proof
    );
    require(_computedStateRoot == _expectedStateRoot, "State root mismatch");

    return uint256(_storageValue) == 1;
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
    // by pass fee vault relay
    if (feeVault != msg.sender) {
      require(_gasLimit >= MIN_GAS_LIMIT, "gas limit too small");
    }

    // compute and deduct the messaging fee to fee vault.
    uint256 _fee = _gasLimit * IL1GasPriceOracle(gasOracle).l1BaseFee();
    require(msg.value >= _value + _fee, "Insufficient msg.value");
    if (_fee > 0) {
      (bool _success, ) = feeVault.call{ value: _fee }("");
      require(_success, "Failed to deduct the fee");
    }

    uint256 _nonce = L2MessageQueue(messageQueue).nextMessageIndex();
    bytes32 _xDomainCalldataHash = keccak256(_encodeXDomainCalldata(msg.sender, _to, _value, _nonce, _message));

    // normally this won't happen, since each message has different nonce, but just in case.
    require(!isL2MessageSent[_xDomainCalldataHash], "Duplicated message");
    isL2MessageSent[_xDomainCalldataHash] = true;

    L2MessageQueue(messageQueue).appendMessage(_xDomainCalldataHash);

    emit SentMessage(msg.sender, _to, _value, _nonce, _gasLimit, _message);

    // refund fee to tx.origin
    unchecked {
      uint256 _refund = msg.value - _fee - _value;
      if (_refund > 0) {
        (bool _success, ) = tx.origin.call{ value: _refund }("");
        require(_success, "Failed to refund the fee");
      }
    }
  }

  /// @inheritdoc IL2ScrollMessenger
  function relayMessage(
    address _from,
    address _to,
    uint256 _value,
    uint256 _nonce,
    bytes memory _message
  ) external override whenNotPaused onlyWhitelistedSender(msg.sender) {
    // anti reentrance
    require(xDomainMessageSender == ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER, "Message is already in execution");

    // @todo address unalis to check sender is L1ScrollMessenger

    bytes32 _xDomainCalldataHash = keccak256(_encodeXDomainCalldata(_from, _to, _value, _nonce, _message));

    require(!isL1MessageExecuted[_xDomainCalldataHash], "Message was already successfully executed");

    _executeMessage(_from, _to, _value, _message, _xDomainCalldataHash);
  }

  /// @inheritdoc IL2ScrollMessenger
  function retryMessageWithProof(
    address _from,
    address _to,
    uint256 _value,
    uint256 _nonce,
    bytes memory _message,
    L1MessageProof calldata _proof
  ) external override whenNotPaused {
    // anti reentrance
    require(xDomainMessageSender == ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER, "Already in execution");

    // check message status
    bytes32 _xDomainCalldataHash = keccak256(_encodeXDomainCalldata(_from, _to, _value, _nonce, _message));
    require(!isL1MessageExecuted[_xDomainCalldataHash], "Message successfully executed");
    require(l1MessageFailedTimes[_xDomainCalldataHash] > 0, "Message not relayed before");

    require(
      verifyMessageInclusionStatus(_proof.blockHash, _xDomainCalldataHash, _proof.stateRootProof),
      "Message not included"
    );

    _executeMessage(_from, _to, _value, _message, _xDomainCalldataHash);
  }

  /************************
   * Restricted Functions *
   ************************/

  /// @notice Pause the contract
  /// @dev This function can only called by contract owner.
  function pause() external onlyOwner {
    _pause();
  }

  function updateMaxFailedExecutionTimes(uint256 _maxFailedExecutionTimes) external onlyOwner {
    maxFailedExecutionTimes = _maxFailedExecutionTimes;

    emit UpdateMaxFailedExecutionTimes(_maxFailedExecutionTimes);
  }

  /**********************
   * Internal Functions *
   **********************/

  function _executeMessage(
    address _from,
    address _to,
    uint256 _value,
    bytes memory _message,
    bytes32 _xDomainCalldataHash
  ) internal {
    // @todo check more `_to` address to avoid attack.
    require(_to != messageQueue, "Forbid to call message queue");
    require(_to != address(this), "Forbid to call self");

    // @note This usually will never happen, just in case.
    require(_from != xDomainMessageSender, "Invalid message sender");

    xDomainMessageSender = _from;
    // solhint-disable-next-line avoid-low-level-calls
    (bool success, ) = _to.call{ value: _value }(_message);
    // reset value to refund gas.
    xDomainMessageSender = ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER;

    if (success) {
      isL1MessageExecuted[_xDomainCalldataHash] = true;
      emit RelayedMessage(_xDomainCalldataHash);
    } else {
      unchecked {
        uint256 _failedTimes = l1MessageFailedTimes[_xDomainCalldataHash] + 1;
        require(_failedTimes <= maxFailedExecutionTimes, "Exceed maximum failure times");
        l1MessageFailedTimes[_xDomainCalldataHash] = _failedTimes;
      }
      emit FailedRelayedMessage(_xDomainCalldataHash);
    }
  }
}
