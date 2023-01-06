// File: @openzeppelin/contracts-upgradeable/utils/AddressUpgradeable.sol

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

// File: src/libraries/verifier/RollupVerifier.sol

// SPDX-License-Identifier: GPL-3.0
pragma solidity >=0.4.16 <0.9.0;

library RollupVerifier {
  function pairing(G1Point[] memory p1, G2Point[] memory p2) internal view returns (bool) {
    uint256 length = p1.length * 6;
    uint256[] memory input = new uint256[](length);
    uint256[1] memory result;
    bool ret;

    require(p1.length == p2.length);

    for (uint256 i = 0; i < p1.length; i++) {
      input[0 + i * 6] = p1[i].x;
      input[1 + i * 6] = p1[i].y;
      input[2 + i * 6] = p2[i].x[0];
      input[3 + i * 6] = p2[i].x[1];
      input[4 + i * 6] = p2[i].y[0];
      input[5 + i * 6] = p2[i].y[1];
    }

    assembly {
      ret := staticcall(gas(), 8, add(input, 0x20), mul(length, 0x20), result, 0x20)
    }
    require(ret);
    return result[0] != 0;
  }

  uint256 constant q_mod = 21888242871839275222246405745257275088548364400416034343698204186575808495617;

  function fr_invert(uint256 a) internal view returns (uint256) {
    return fr_pow(a, q_mod - 2);
  }

  function fr_pow(uint256 a, uint256 power) internal view returns (uint256) {
    uint256[6] memory input;
    uint256[1] memory result;
    bool ret;

    input[0] = 32;
    input[1] = 32;
    input[2] = 32;
    input[3] = a;
    input[4] = power;
    input[5] = q_mod;

    assembly {
      ret := staticcall(gas(), 0x05, input, 0xc0, result, 0x20)
    }
    require(ret);

    return result[0];
  }

  function fr_div(uint256 a, uint256 b) internal view returns (uint256) {
    require(b != 0);
    return mulmod(a, fr_invert(b), q_mod);
  }

  function fr_mul_add(
    uint256 a,
    uint256 b,
    uint256 c
  ) internal pure returns (uint256) {
    return addmod(mulmod(a, b, q_mod), c, q_mod);
  }

  function fr_mul_add_pm(
    uint256[84] memory m,
    uint256[] calldata proof,
    uint256 opcode,
    uint256 t
  ) internal pure returns (uint256) {
    for (uint256 i = 0; i < 32; i += 2) {
      uint256 a = opcode & 0xff;
      if (a != 0xff) {
        opcode >>= 8;
        uint256 b = opcode & 0xff;
        opcode >>= 8;
        t = addmod(mulmod(proof[a], m[b], q_mod), t, q_mod);
      } else {
        break;
      }
    }

    return t;
  }

  function fr_mul_add_mt(
    uint256[84] memory m,
    uint256 base,
    uint256 opcode,
    uint256 t
  ) internal pure returns (uint256) {
    for (uint256 i = 0; i < 32; i += 1) {
      uint256 a = opcode & 0xff;
      if (a != 0xff) {
        opcode >>= 8;
        t = addmod(mulmod(base, t, q_mod), m[a], q_mod);
      } else {
        break;
      }
    }

    return t;
  }

  function fr_reverse(uint256 input) internal pure returns (uint256 v) {
    v = input;

    // swap bytes
    v =
      ((v & 0xFF00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF00) >> 8) |
      ((v & 0x00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF) << 8);

    // swap 2-byte long pairs
    v =
      ((v & 0xFFFF0000FFFF0000FFFF0000FFFF0000FFFF0000FFFF0000FFFF0000FFFF0000) >> 16) |
      ((v & 0x0000FFFF0000FFFF0000FFFF0000FFFF0000FFFF0000FFFF0000FFFF0000FFFF) << 16);

    // swap 4-byte long pairs
    v =
      ((v & 0xFFFFFFFF00000000FFFFFFFF00000000FFFFFFFF00000000FFFFFFFF00000000) >> 32) |
      ((v & 0x00000000FFFFFFFF00000000FFFFFFFF00000000FFFFFFFF00000000FFFFFFFF) << 32);

    // swap 8-byte long pairs
    v =
      ((v & 0xFFFFFFFFFFFFFFFF0000000000000000FFFFFFFFFFFFFFFF0000000000000000) >> 64) |
      ((v & 0x0000000000000000FFFFFFFFFFFFFFFF0000000000000000FFFFFFFFFFFFFFFF) << 64);

    // swap 16-byte long pairs
    v = (v >> 128) | (v << 128);
  }

  uint256 constant p_mod = 21888242871839275222246405745257275088696311157297823662689037894645226208583;

  struct G1Point {
    uint256 x;
    uint256 y;
  }

  struct G2Point {
    uint256[2] x;
    uint256[2] y;
  }

  function ecc_from(uint256 x, uint256 y) internal pure returns (G1Point memory r) {
    r.x = x;
    r.y = y;
  }

  function ecc_add(
    uint256 ax,
    uint256 ay,
    uint256 bx,
    uint256 by
  ) internal view returns (uint256, uint256) {
    bool ret = false;
    G1Point memory r;
    uint256[4] memory input_points;

    input_points[0] = ax;
    input_points[1] = ay;
    input_points[2] = bx;
    input_points[3] = by;

    assembly {
      ret := staticcall(gas(), 6, input_points, 0x80, r, 0x40)
    }
    require(ret);

    return (r.x, r.y);
  }

  function ecc_sub(
    uint256 ax,
    uint256 ay,
    uint256 bx,
    uint256 by
  ) internal view returns (uint256, uint256) {
    return ecc_add(ax, ay, bx, p_mod - by);
  }

  function ecc_mul(
    uint256 px,
    uint256 py,
    uint256 s
  ) internal view returns (uint256, uint256) {
    uint256[3] memory input;
    bool ret = false;
    G1Point memory r;

    input[0] = px;
    input[1] = py;
    input[2] = s;

    assembly {
      ret := staticcall(gas(), 7, input, 0x60, r, 0x40)
    }
    require(ret);

    return (r.x, r.y);
  }

  function _ecc_mul_add(uint256[5] memory input) internal view {
    bool ret = false;

    assembly {
      ret := staticcall(gas(), 7, input, 0x60, add(input, 0x20), 0x40)
    }
    require(ret);

    assembly {
      ret := staticcall(gas(), 6, add(input, 0x20), 0x80, add(input, 0x60), 0x40)
    }
    require(ret);
  }

  function ecc_mul_add(
    uint256 px,
    uint256 py,
    uint256 s,
    uint256 qx,
    uint256 qy
  ) internal view returns (uint256, uint256) {
    uint256[5] memory input;
    input[0] = px;
    input[1] = py;
    input[2] = s;
    input[3] = qx;
    input[4] = qy;

    _ecc_mul_add(input);

    return (input[3], input[4]);
  }

  function ecc_mul_add_pm(
    uint256[84] memory m,
    uint256[] calldata proof,
    uint256 opcode,
    uint256 t0,
    uint256 t1
  ) internal view returns (uint256, uint256) {
    uint256[5] memory input;
    input[3] = t0;
    input[4] = t1;
    for (uint256 i = 0; i < 32; i += 2) {
      uint256 a = opcode & 0xff;
      if (a != 0xff) {
        opcode >>= 8;
        uint256 b = opcode & 0xff;
        opcode >>= 8;
        input[0] = proof[a];
        input[1] = proof[a + 1];
        input[2] = m[b];
        _ecc_mul_add(input);
      } else {
        break;
      }
    }

    return (input[3], input[4]);
  }

  function update_hash_scalar(
    uint256 v,
    uint256[144] memory absorbing,
    uint256 pos
  ) internal pure {
    absorbing[pos++] = 0x02;
    absorbing[pos++] = v;
  }

  function update_hash_point(
    uint256 x,
    uint256 y,
    uint256[144] memory absorbing,
    uint256 pos
  ) internal pure {
    absorbing[pos++] = 0x01;
    absorbing[pos++] = x;
    absorbing[pos++] = y;
  }

  function to_scalar(bytes32 r) private pure returns (uint256 v) {
    uint256 tmp = uint256(r);
    tmp = fr_reverse(tmp);
    v = tmp % 0x30644e72e131a029b85045b68181585d2833e84879b9709143e1f593f0000001;
  }

  function hash(uint256[144] memory absorbing, uint256 length) private view returns (bytes32[1] memory v) {
    bool success;
    assembly {
      success := staticcall(sub(gas(), 2000), 2, absorbing, length, v, 32)
      switch success
      case 0 {
        invalid()
      }
    }
    assert(success);
  }

  function squeeze_challenge(uint256[144] memory absorbing, uint32 length) internal view returns (uint256 v) {
    absorbing[length] = 0;
    bytes32 res = hash(absorbing, length * 32 + 1)[0];
    v = to_scalar(res);
    absorbing[0] = uint256(res);
    length = 1;
  }

  function get_verify_circuit_g2_s() internal pure returns (G2Point memory s) {
    s.x[0] = uint256(19996377281670978687180986182441301914718493784645870391946826878753710639456);
    s.x[1] = uint256(4287478848095488335912479212753150961411468232106701703291869721868407715111);
    s.y[0] = uint256(6995741485533723263267942814565501722132921805029874890336635619836737653877);
    s.y[1] = uint256(11126659726611658836425410744462014686753643655648740844565393330984713428953);
  }

  function get_verify_circuit_g2_n() internal pure returns (G2Point memory n) {
    n.x[0] = uint256(11559732032986387107991004021392285783925812861821192530917403151452391805634);
    n.x[1] = uint256(10857046999023057135944570762232829481370756359578518086990519993285655852781);
    n.y[0] = uint256(17805874995975841540914202342111839520379459829704422454583296818431106115052);
    n.y[1] = uint256(13392588948715843804641432497768002650278120570034223513918757245338268106653);
  }

  function get_target_circuit_g2_s() internal pure returns (G2Point memory s) {
    s.x[0] = uint256(19996377281670978687180986182441301914718493784645870391946826878753710639456);
    s.x[1] = uint256(4287478848095488335912479212753150961411468232106701703291869721868407715111);
    s.y[0] = uint256(6995741485533723263267942814565501722132921805029874890336635619836737653877);
    s.y[1] = uint256(11126659726611658836425410744462014686753643655648740844565393330984713428953);
  }

  function get_target_circuit_g2_n() internal pure returns (G2Point memory n) {
    n.x[0] = uint256(11559732032986387107991004021392285783925812861821192530917403151452391805634);
    n.x[1] = uint256(10857046999023057135944570762232829481370756359578518086990519993285655852781);
    n.y[0] = uint256(17805874995975841540914202342111839520379459829704422454583296818431106115052);
    n.y[1] = uint256(13392588948715843804641432497768002650278120570034223513918757245338268106653);
  }

  function get_wx_wg(uint256[] calldata proof, uint256[4] memory instances)
    internal
    view
    returns (
      uint256,
      uint256,
      uint256,
      uint256
    )
  {
    uint256[84] memory m;
    uint256[144] memory absorbing;
    uint256 t0 = 0;
    uint256 t1 = 0;

    (t0, t1) = (
      ecc_mul(
        13911018583007884881416842514661274050567796652031922980888952067142200734890,
        6304656948134906299141761906515211516376236447819044970320185642735642777036,
        instances[0]
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        10634526547038245645834822324032425487434811507756950001533785848774317018670,
        11025818855933089539342999945076144168100709119485154428833847826982360951459,
        instances[1],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        13485936455723319058155687139769502499697405985650416391707184524158646623799,
        16234009237501684544798205490615498675425737095147152991328466405207467143566,
        instances[2],
        t0,
        t1
      )
    );
    (m[0], m[1]) = (
      ecc_mul_add(
        21550585789286941025166870525096478397065943995678337623823808437877187678077,
        4447338868884713453743453617617291019986465683944733951178865127876671635659,
        instances[3],
        t0,
        t1
      )
    );
    update_hash_scalar(7326291674247555594112707886804937707847188185923070866278273345303869756280, absorbing, 0);
    update_hash_point(m[0], m[1], absorbing, 2);
    for (t0 = 0; t0 <= 4; t0++) {
      update_hash_point(proof[0 + t0 * 2], proof[1 + t0 * 2], absorbing, 5 + t0 * 3);
    }
    m[2] = (squeeze_challenge(absorbing, 20));
    for (t0 = 0; t0 <= 13; t0++) {
      update_hash_point(proof[10 + t0 * 2], proof[11 + t0 * 2], absorbing, 1 + t0 * 3);
    }
    m[3] = (squeeze_challenge(absorbing, 43));
    m[4] = (squeeze_challenge(absorbing, 1));
    for (t0 = 0; t0 <= 9; t0++) {
      update_hash_point(proof[38 + t0 * 2], proof[39 + t0 * 2], absorbing, 1 + t0 * 3);
    }
    m[5] = (squeeze_challenge(absorbing, 31));
    for (t0 = 0; t0 <= 3; t0++) {
      update_hash_point(proof[58 + t0 * 2], proof[59 + t0 * 2], absorbing, 1 + t0 * 3);
    }
    m[6] = (squeeze_challenge(absorbing, 13));
    for (t0 = 0; t0 <= 70; t0++) {
      update_hash_scalar(proof[66 + t0 * 1], absorbing, 1 + t0 * 2);
    }
    m[7] = (squeeze_challenge(absorbing, 143));
    for (t0 = 0; t0 <= 3; t0++) {
      update_hash_point(proof[137 + t0 * 2], proof[138 + t0 * 2], absorbing, 1 + t0 * 3);
    }
    m[8] = (squeeze_challenge(absorbing, 13));
    m[9] = (mulmod(m[6], 6143038923529407703646399695489445107254060255791852207908457597807435305312, q_mod));
    m[10] = (mulmod(m[6], 7358966525675286471217089135633860168646304224547606326237275077574224349359, q_mod));
    m[11] = (mulmod(m[6], 11377606117859914088982205826922132024839443553408109299929510653283289974216, q_mod));
    m[12] = (fr_pow(m[6], 33554432));
    m[13] = (addmod(m[12], q_mod - 1, q_mod));
    m[14] = (mulmod(21888242219518804655518433051623070663413851959604507555939307129453691614729, m[13], q_mod));
    t0 = (addmod(m[6], q_mod - 1, q_mod));
    m[14] = (fr_div(m[14], t0));
    m[15] = (mulmod(3814514741328848551622746860665626251343731549210296844380905280010844577811, m[13], q_mod));
    t0 = (addmod(m[6], q_mod - 11377606117859914088982205826922132024839443553408109299929510653283289974216, q_mod));
    m[15] = (fr_div(m[15], t0));
    m[16] = (mulmod(14167635312934689395373925807699824183296350635557349457928542208657273886961, m[13], q_mod));
    t0 = (addmod(m[6], q_mod - 17329448237240114492580865744088056414251735686965494637158808787419781175510, q_mod));
    m[16] = (fr_div(m[16], t0));
    m[17] = (mulmod(12609034248192017902501772617940356704925468750503023243291639149763830461639, m[13], q_mod));
    t0 = (addmod(m[6], q_mod - 16569469942529664681363945218228869388192121720036659574609237682362097667612, q_mod));
    m[17] = (fr_div(m[17], t0));
    m[18] = (mulmod(12805242257443675784492534138904933930037912868081131057088370227525924812579, m[13], q_mod));
    t0 = (addmod(m[6], q_mod - 9741553891420464328295280489650144566903017206473301385034033384879943874347, q_mod));
    m[18] = (fr_div(m[18], t0));
    m[19] = (mulmod(6559137297042406441428413756926584610543422337862324541665337888392460442551, m[13], q_mod));
    t0 = (addmod(m[6], q_mod - 5723528081196465413808013109680264505774289533922470433187916976440924869204, q_mod));
    m[19] = (fr_div(m[19], t0));
    m[20] = (mulmod(14811589476322888753142612645486192973009181596950146578897598212834285850868, m[13], q_mod));
    t0 = (addmod(m[6], q_mod - 7358966525675286471217089135633860168646304224547606326237275077574224349359, q_mod));
    m[20] = (fr_div(m[20], t0));
    t0 = (addmod(m[15], m[16], q_mod));
    t0 = (addmod(t0, m[17], q_mod));
    t0 = (addmod(t0, m[18], q_mod));
    m[15] = (addmod(t0, m[19], q_mod));
    t0 = (fr_mul_add(proof[74], proof[72], proof[73]));
    t0 = (fr_mul_add(proof[75], proof[67], t0));
    t0 = (fr_mul_add(proof[76], proof[68], t0));
    t0 = (fr_mul_add(proof[77], proof[69], t0));
    t0 = (fr_mul_add(proof[78], proof[70], t0));
    m[16] = (fr_mul_add(proof[79], proof[71], t0));
    t0 = (mulmod(proof[67], proof[68], q_mod));
    m[16] = (fr_mul_add(proof[80], t0, m[16]));
    t0 = (mulmod(proof[69], proof[70], q_mod));
    m[16] = (fr_mul_add(proof[81], t0, m[16]));
    t0 = (addmod(1, q_mod - proof[97], q_mod));
    m[17] = (mulmod(m[14], t0, q_mod));
    t0 = (mulmod(proof[100], proof[100], q_mod));
    t0 = (addmod(t0, q_mod - proof[100], q_mod));
    m[18] = (mulmod(m[20], t0, q_mod));
    t0 = (addmod(proof[100], q_mod - proof[99], q_mod));
    m[19] = (mulmod(t0, m[14], q_mod));
    m[21] = (mulmod(m[3], m[6], q_mod));
    t0 = (addmod(m[20], m[15], q_mod));
    m[15] = (addmod(1, q_mod - t0, q_mod));
    m[22] = (addmod(proof[67], m[4], q_mod));
    t0 = (fr_mul_add(proof[91], m[3], m[22]));
    m[23] = (mulmod(t0, proof[98], q_mod));
    t0 = (addmod(m[22], m[21], q_mod));
    m[22] = (mulmod(t0, proof[97], q_mod));
    m[24] = (mulmod(4131629893567559867359510883348571134090853742863529169391034518566172092834, m[21], q_mod));
    m[25] = (addmod(proof[68], m[4], q_mod));
    t0 = (fr_mul_add(proof[92], m[3], m[25]));
    m[23] = (mulmod(t0, m[23], q_mod));
    t0 = (addmod(m[25], m[24], q_mod));
    m[22] = (mulmod(t0, m[22], q_mod));
    m[24] = (mulmod(4131629893567559867359510883348571134090853742863529169391034518566172092834, m[24], q_mod));
    m[25] = (addmod(proof[69], m[4], q_mod));
    t0 = (fr_mul_add(proof[93], m[3], m[25]));
    m[23] = (mulmod(t0, m[23], q_mod));
    t0 = (addmod(m[25], m[24], q_mod));
    m[22] = (mulmod(t0, m[22], q_mod));
    m[24] = (mulmod(4131629893567559867359510883348571134090853742863529169391034518566172092834, m[24], q_mod));
    t0 = (addmod(m[23], q_mod - m[22], q_mod));
    m[22] = (mulmod(t0, m[15], q_mod));
    m[21] = (mulmod(m[21], 11166246659983828508719468090013646171463329086121580628794302409516816350802, q_mod));
    m[23] = (addmod(proof[70], m[4], q_mod));
    t0 = (fr_mul_add(proof[94], m[3], m[23]));
    m[24] = (mulmod(t0, proof[101], q_mod));
    t0 = (addmod(m[23], m[21], q_mod));
    m[23] = (mulmod(t0, proof[100], q_mod));
    m[21] = (mulmod(4131629893567559867359510883348571134090853742863529169391034518566172092834, m[21], q_mod));
    m[25] = (addmod(proof[71], m[4], q_mod));
    t0 = (fr_mul_add(proof[95], m[3], m[25]));
    m[24] = (mulmod(t0, m[24], q_mod));
    t0 = (addmod(m[25], m[21], q_mod));
    m[23] = (mulmod(t0, m[23], q_mod));
    m[21] = (mulmod(4131629893567559867359510883348571134090853742863529169391034518566172092834, m[21], q_mod));
    m[25] = (addmod(proof[66], m[4], q_mod));
    t0 = (fr_mul_add(proof[96], m[3], m[25]));
    m[24] = (mulmod(t0, m[24], q_mod));
    t0 = (addmod(m[25], m[21], q_mod));
    m[23] = (mulmod(t0, m[23], q_mod));
    m[21] = (mulmod(4131629893567559867359510883348571134090853742863529169391034518566172092834, m[21], q_mod));
    t0 = (addmod(m[24], q_mod - m[23], q_mod));
    m[21] = (mulmod(t0, m[15], q_mod));
    t0 = (addmod(proof[104], m[3], q_mod));
    m[23] = (mulmod(proof[103], t0, q_mod));
    t0 = (addmod(proof[106], m[4], q_mod));
    m[23] = (mulmod(m[23], t0, q_mod));
    m[24] = (mulmod(proof[67], proof[82], q_mod));
    m[2] = (mulmod(0, m[2], q_mod));
    m[24] = (addmod(m[2], m[24], q_mod));
    m[25] = (addmod(m[2], proof[83], q_mod));
    m[26] = (addmod(proof[104], q_mod - proof[106], q_mod));
    t0 = (addmod(1, q_mod - proof[102], q_mod));
    m[27] = (mulmod(m[14], t0, q_mod));
    t0 = (mulmod(proof[102], proof[102], q_mod));
    t0 = (addmod(t0, q_mod - proof[102], q_mod));
    m[28] = (mulmod(m[20], t0, q_mod));
    t0 = (addmod(m[24], m[3], q_mod));
    m[24] = (mulmod(proof[102], t0, q_mod));
    m[25] = (addmod(m[25], m[4], q_mod));
    t0 = (mulmod(m[24], m[25], q_mod));
    t0 = (addmod(m[23], q_mod - t0, q_mod));
    m[23] = (mulmod(t0, m[15], q_mod));
    m[24] = (mulmod(m[14], m[26], q_mod));
    t0 = (addmod(proof[104], q_mod - proof[105], q_mod));
    t0 = (mulmod(m[26], t0, q_mod));
    m[26] = (mulmod(t0, m[15], q_mod));
    t0 = (addmod(proof[109], m[3], q_mod));
    m[29] = (mulmod(proof[108], t0, q_mod));
    t0 = (addmod(proof[111], m[4], q_mod));
    m[29] = (mulmod(m[29], t0, q_mod));
    m[30] = (fr_mul_add(proof[82], proof[68], m[2]));
    m[31] = (addmod(proof[109], q_mod - proof[111], q_mod));
    t0 = (addmod(1, q_mod - proof[107], q_mod));
    m[32] = (mulmod(m[14], t0, q_mod));
    t0 = (mulmod(proof[107], proof[107], q_mod));
    t0 = (addmod(t0, q_mod - proof[107], q_mod));
    m[33] = (mulmod(m[20], t0, q_mod));
    t0 = (addmod(m[30], m[3], q_mod));
    t0 = (mulmod(proof[107], t0, q_mod));
    t0 = (mulmod(t0, m[25], q_mod));
    t0 = (addmod(m[29], q_mod - t0, q_mod));
    m[29] = (mulmod(t0, m[15], q_mod));
    m[30] = (mulmod(m[14], m[31], q_mod));
    t0 = (addmod(proof[109], q_mod - proof[110], q_mod));
    t0 = (mulmod(m[31], t0, q_mod));
    m[31] = (mulmod(t0, m[15], q_mod));
    t0 = (addmod(proof[114], m[3], q_mod));
    m[34] = (mulmod(proof[113], t0, q_mod));
    t0 = (addmod(proof[116], m[4], q_mod));
    m[34] = (mulmod(m[34], t0, q_mod));
    m[35] = (fr_mul_add(proof[82], proof[69], m[2]));
    m[36] = (addmod(proof[114], q_mod - proof[116], q_mod));
    t0 = (addmod(1, q_mod - proof[112], q_mod));
    m[37] = (mulmod(m[14], t0, q_mod));
    t0 = (mulmod(proof[112], proof[112], q_mod));
    t0 = (addmod(t0, q_mod - proof[112], q_mod));
    m[38] = (mulmod(m[20], t0, q_mod));
    t0 = (addmod(m[35], m[3], q_mod));
    t0 = (mulmod(proof[112], t0, q_mod));
    t0 = (mulmod(t0, m[25], q_mod));
    t0 = (addmod(m[34], q_mod - t0, q_mod));
    m[34] = (mulmod(t0, m[15], q_mod));
    m[35] = (mulmod(m[14], m[36], q_mod));
    t0 = (addmod(proof[114], q_mod - proof[115], q_mod));
    t0 = (mulmod(m[36], t0, q_mod));
    m[36] = (mulmod(t0, m[15], q_mod));
    t0 = (addmod(proof[119], m[3], q_mod));
    m[39] = (mulmod(proof[118], t0, q_mod));
    t0 = (addmod(proof[121], m[4], q_mod));
    m[39] = (mulmod(m[39], t0, q_mod));
    m[40] = (fr_mul_add(proof[82], proof[70], m[2]));
    m[41] = (addmod(proof[119], q_mod - proof[121], q_mod));
    t0 = (addmod(1, q_mod - proof[117], q_mod));
    m[42] = (mulmod(m[14], t0, q_mod));
    t0 = (mulmod(proof[117], proof[117], q_mod));
    t0 = (addmod(t0, q_mod - proof[117], q_mod));
    m[43] = (mulmod(m[20], t0, q_mod));
    t0 = (addmod(m[40], m[3], q_mod));
    t0 = (mulmod(proof[117], t0, q_mod));
    t0 = (mulmod(t0, m[25], q_mod));
    t0 = (addmod(m[39], q_mod - t0, q_mod));
    m[25] = (mulmod(t0, m[15], q_mod));
    m[39] = (mulmod(m[14], m[41], q_mod));
    t0 = (addmod(proof[119], q_mod - proof[120], q_mod));
    t0 = (mulmod(m[41], t0, q_mod));
    m[40] = (mulmod(t0, m[15], q_mod));
    t0 = (addmod(proof[124], m[3], q_mod));
    m[41] = (mulmod(proof[123], t0, q_mod));
    t0 = (addmod(proof[126], m[4], q_mod));
    m[41] = (mulmod(m[41], t0, q_mod));
    m[44] = (fr_mul_add(proof[84], proof[67], m[2]));
    m[45] = (addmod(m[2], proof[85], q_mod));
    m[46] = (addmod(proof[124], q_mod - proof[126], q_mod));
    t0 = (addmod(1, q_mod - proof[122], q_mod));
    m[47] = (mulmod(m[14], t0, q_mod));
    t0 = (mulmod(proof[122], proof[122], q_mod));
    t0 = (addmod(t0, q_mod - proof[122], q_mod));
    m[48] = (mulmod(m[20], t0, q_mod));
    t0 = (addmod(m[44], m[3], q_mod));
    m[44] = (mulmod(proof[122], t0, q_mod));
    t0 = (addmod(m[45], m[4], q_mod));
    t0 = (mulmod(m[44], t0, q_mod));
    t0 = (addmod(m[41], q_mod - t0, q_mod));
    m[41] = (mulmod(t0, m[15], q_mod));
    m[44] = (mulmod(m[14], m[46], q_mod));
    t0 = (addmod(proof[124], q_mod - proof[125], q_mod));
    t0 = (mulmod(m[46], t0, q_mod));
    m[45] = (mulmod(t0, m[15], q_mod));
    t0 = (addmod(proof[129], m[3], q_mod));
    m[46] = (mulmod(proof[128], t0, q_mod));
    t0 = (addmod(proof[131], m[4], q_mod));
    m[46] = (mulmod(m[46], t0, q_mod));
    m[49] = (fr_mul_add(proof[86], proof[67], m[2]));
    m[50] = (addmod(m[2], proof[87], q_mod));
    m[51] = (addmod(proof[129], q_mod - proof[131], q_mod));
    t0 = (addmod(1, q_mod - proof[127], q_mod));
    m[52] = (mulmod(m[14], t0, q_mod));
    t0 = (mulmod(proof[127], proof[127], q_mod));
    t0 = (addmod(t0, q_mod - proof[127], q_mod));
    m[53] = (mulmod(m[20], t0, q_mod));
    t0 = (addmod(m[49], m[3], q_mod));
    m[49] = (mulmod(proof[127], t0, q_mod));
    t0 = (addmod(m[50], m[4], q_mod));
    t0 = (mulmod(m[49], t0, q_mod));
    t0 = (addmod(m[46], q_mod - t0, q_mod));
    m[46] = (mulmod(t0, m[15], q_mod));
    m[49] = (mulmod(m[14], m[51], q_mod));
    t0 = (addmod(proof[129], q_mod - proof[130], q_mod));
    t0 = (mulmod(m[51], t0, q_mod));
    m[50] = (mulmod(t0, m[15], q_mod));
    t0 = (addmod(proof[134], m[3], q_mod));
    m[51] = (mulmod(proof[133], t0, q_mod));
    t0 = (addmod(proof[136], m[4], q_mod));
    m[51] = (mulmod(m[51], t0, q_mod));
    m[54] = (fr_mul_add(proof[88], proof[67], m[2]));
    m[2] = (addmod(m[2], proof[89], q_mod));
    m[55] = (addmod(proof[134], q_mod - proof[136], q_mod));
    t0 = (addmod(1, q_mod - proof[132], q_mod));
    m[56] = (mulmod(m[14], t0, q_mod));
    t0 = (mulmod(proof[132], proof[132], q_mod));
    t0 = (addmod(t0, q_mod - proof[132], q_mod));
    m[20] = (mulmod(m[20], t0, q_mod));
    t0 = (addmod(m[54], m[3], q_mod));
    m[3] = (mulmod(proof[132], t0, q_mod));
    t0 = (addmod(m[2], m[4], q_mod));
    t0 = (mulmod(m[3], t0, q_mod));
    t0 = (addmod(m[51], q_mod - t0, q_mod));
    m[2] = (mulmod(t0, m[15], q_mod));
    m[3] = (mulmod(m[14], m[55], q_mod));
    t0 = (addmod(proof[134], q_mod - proof[135], q_mod));
    t0 = (mulmod(m[55], t0, q_mod));
    m[4] = (mulmod(t0, m[15], q_mod));
    t0 = (fr_mul_add(m[5], 0, m[16]));
    t0 = (fr_mul_add_mt(m, m[5], 24064768791442479290152634096194013545513974547709823832001394403118888981009, t0));
    t0 = (fr_mul_add_mt(m, m[5], 4704208815882882920750, t0));
    m[2] = (fr_div(t0, m[13]));
    m[3] = (mulmod(m[8], m[8], q_mod));
    m[4] = (mulmod(m[3], m[8], q_mod));
    (t0, t1) = (ecc_mul(proof[143], proof[144], m[4]));
    (t0, t1) = (ecc_mul_add_pm(m, proof, 281470825071501, t0, t1));
    (m[14], m[15]) = (ecc_add(t0, t1, proof[137], proof[138]));
    m[5] = (mulmod(m[4], m[11], q_mod));
    m[11] = (mulmod(m[4], m[7], q_mod));
    m[13] = (mulmod(m[11], m[7], q_mod));
    m[16] = (mulmod(m[13], m[7], q_mod));
    m[17] = (mulmod(m[16], m[7], q_mod));
    m[18] = (mulmod(m[17], m[7], q_mod));
    m[19] = (mulmod(m[18], m[7], q_mod));
    t0 = (mulmod(m[19], proof[135], q_mod));
    t0 = (fr_mul_add_pm(m, proof, 79227007564587019091207590530, t0));
    m[20] = (fr_mul_add(proof[105], m[4], t0));
    m[10] = (mulmod(m[3], m[10], q_mod));
    m[20] = (fr_mul_add(proof[99], m[3], m[20]));
    m[9] = (mulmod(m[8], m[9], q_mod));
    m[21] = (mulmod(m[8], m[7], q_mod));
    for (t0 = 0; t0 < 8; t0++) {
      m[22 + t0 * 1] = (mulmod(m[21 + t0 * 1], m[7 + t0 * 0], q_mod));
    }
    t0 = (mulmod(m[29], proof[133], q_mod));
    t0 = (fr_mul_add_pm(m, proof, 1461480058012745347196003969984389955172320353408, t0));
    m[20] = (addmod(m[20], t0, q_mod));
    m[3] = (addmod(m[3], m[21], q_mod));
    m[21] = (mulmod(m[7], m[7], q_mod));
    m[30] = (mulmod(m[21], m[7], q_mod));
    for (t0 = 0; t0 < 50; t0++) {
      m[31 + t0 * 1] = (mulmod(m[30 + t0 * 1], m[7 + t0 * 0], q_mod));
    }
    m[81] = (mulmod(m[80], proof[90], q_mod));
    m[82] = (mulmod(m[79], m[12], q_mod));
    m[83] = (mulmod(m[82], m[12], q_mod));
    m[12] = (mulmod(m[83], m[12], q_mod));
    t0 = (fr_mul_add(m[79], m[2], m[81]));
    t0 = (fr_mul_add_pm(m, proof, 28637501128329066231612878461967933875285131620580756137874852300330784214624, t0));
    t0 = (fr_mul_add_pm(m, proof, 21474593857386732646168474467085622855647258609351047587832868301163767676495, t0));
    t0 = (fr_mul_add_pm(m, proof, 14145600374170319983429588659751245017860232382696106927048396310641433325177, t0));
    t0 = (fr_mul_add_pm(m, proof, 18446470583433829957, t0));
    t0 = (addmod(t0, proof[66], q_mod));
    m[2] = (addmod(m[20], t0, q_mod));
    m[19] = (addmod(m[19], m[54], q_mod));
    m[20] = (addmod(m[29], m[53], q_mod));
    m[18] = (addmod(m[18], m[51], q_mod));
    m[28] = (addmod(m[28], m[50], q_mod));
    m[17] = (addmod(m[17], m[48], q_mod));
    m[27] = (addmod(m[27], m[47], q_mod));
    m[16] = (addmod(m[16], m[45], q_mod));
    m[26] = (addmod(m[26], m[44], q_mod));
    m[13] = (addmod(m[13], m[42], q_mod));
    m[25] = (addmod(m[25], m[41], q_mod));
    m[11] = (addmod(m[11], m[39], q_mod));
    m[24] = (addmod(m[24], m[38], q_mod));
    m[4] = (addmod(m[4], m[36], q_mod));
    m[23] = (addmod(m[23], m[35], q_mod));
    m[22] = (addmod(m[22], m[34], q_mod));
    m[3] = (addmod(m[3], m[33], q_mod));
    m[8] = (addmod(m[8], m[32], q_mod));
    (t0, t1) = (ecc_mul(proof[143], proof[144], m[5]));
    (t0, t1) = (
      ecc_mul_add_pm(m, proof, 10933423423422768024429730621579321771439401845242250760130969989159573132066, t0, t1)
    );
    (t0, t1) = (ecc_mul_add_pm(m, proof, 1461486238301980199876269201563775120819706402602, t0, t1));
    (t0, t1) = (
      ecc_mul_add(
        18701609130775737229348071043080155034023979562517390395403433088802478899758,
        15966955543930185772599298905781740007968379271659670990460125132276790404701,
        m[78],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        10391672869328159104536012527288890078475214572275421477472198141744100604180,
        16383182967525077486800851500412772270268328143041811261940514978333847876450,
        m[77],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        1694121668121560366967381814358868176695875056710903754887787227675156636991,
        6288755472313871386012926867179622380057563139110460659328016508371672965822,
        m[76],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        8449090587209846475328734419746789925412190193479844231777165308243174237722,
        19620423218491500875965944829407986067794157844846402182805878618955604592848,
        m[75],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        5053208336959682582031156680199539869251745263409434673229644546747696847142,
        2515271708296970065769200367712058290268116287798438948140802173656220671206,
        m[74],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        14044565934581841113280816557133159251170886931106151374890478449607604267942,
        4516676687937794780030405510740994119381246893674971835541700695978704585552,
        m[73],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        8808629196631084710334110767449499515582902470045288549019060600095073238105,
        13294364470509711632739201553507258372326885785844949555702886281377427438475,
        m[72],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        5025513109896000321643874120256520860696240548707294083465215087271048364447,
        3512836639252013523316566987122028012000136443005216091303269685639094608348,
        m[71],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        20143075587083355112417414887372164250381042430441089145485481665404780784123,
        9674175910548207533970570126063643897609459066877075659644076646142886425503,
        m[70],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        15449875505347857882486479091299788291220259329814373554032711960946424724459,
        18962357525499685082729877436365914814836051345178637509857216081206536249101,
        m[69],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        8808629196631084710334110767449499515582902470045288549019060600095073238105,
        13294364470509711632739201553507258372326885785844949555702886281377427438475,
        m[68],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        4919836553908828082540426444868776555669883964231731088484431671272015675682,
        2534996469663628472218664436969797350677809756735321673130157881813913441609,
        m[67],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        11667150339256836494926506499230187360957884531183800528342644917396989453992,
        15540782144062394272475578831064080588044323224200171932910650185556553066875,
        m[66],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        7298741378311576950839968993357330108079245118485170808123459961337830256312,
        10327561179499117619949936626306234488421661318541529469701192193684736307992,
        m[65],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        19156320437354843782276382482504062704637529342417677454208679985931193905144,
        12513036134308417802230431028731202760516379532825961661396005403922128650283,
        m[64],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        21344975294019301064497004820288763682448968861642019035490416932201272957274,
        10527619823264344893410550194287064640208153251186939130321425213582959780489,
        m[63],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        8972742415650205333409282370033440562593431348747288268814492203356823531160,
        8116706321112691122771049432546166822575953322170688547310064134261753771143,
        m[62],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        2245383788954722547301665173770198299224442299145553661157120655982065376923,
        21429627532145565836455474503387893562363999035988060101286707048187310790834,
        m[61],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        6631831869726773976361406817204839637256208337970281843457872807848960103655,
        9564029493986604546558813596663080644256762699468834511701525072767927949801,
        m[60],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        11480433023546787855799302686493624232665854025790899812568432142639901048711,
        19408335616099148180409133533838326787843523379558500985213116784449716389602,
        m[59],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        17119009547436104907589161251911916154539209413889810725547125453954285498068,
        16196009614025712805558792610177918739658373559330006740051047693948800191562,
        m[58],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        4833170740960210126662783488087087210159995687268566750051519788650425720369,
        14321097009933429277686973550787181101481482473464521566076287626133354519061,
        m[57],
        t0,
        t1
      )
    );
    (t0, t1) = (
      ecc_mul_add(
        18650010323993268535055713787599480879302828622769515272251129462854128226895,
        11244246887388549559894193327128701737108444364011850111062992666532968469107,
        m[56],
        t0,
        t1
      )
    );
    (t0, t1) = (ecc_mul_add_pm(m, proof, 6277008573546246765208814532330797927747086570010716419876, t0, t1));
    (m[0], m[1]) = (ecc_add(t0, t1, m[0], m[1]));
    (t0, t1) = (ecc_mul(1, 2, m[2]));
    (m[0], m[1]) = (ecc_sub(m[0], m[1], t0, t1));
    return (m[14], m[15], m[0], m[1]);
  }

  function verify(uint256[] calldata proof, uint256[] calldata target_circuit_final_pair) public view {
    uint256[4] memory instances;
    instances[0] = target_circuit_final_pair[0] & ((1 << 136) - 1);
    instances[1] = (target_circuit_final_pair[0] >> 136) + ((target_circuit_final_pair[1] & 1) << 136);
    instances[2] = target_circuit_final_pair[2] & ((1 << 136) - 1);
    instances[3] = (target_circuit_final_pair[2] >> 136) + ((target_circuit_final_pair[3] & 1) << 136);

    uint256 x0 = 0;
    uint256 x1 = 0;
    uint256 y0 = 0;
    uint256 y1 = 0;

    G1Point[] memory g1_points = new G1Point[](2);
    G2Point[] memory g2_points = new G2Point[](2);
    bool checked = false;

    (x0, y0, x1, y1) = get_wx_wg(proof, instances);
    g1_points[0].x = x0;
    g1_points[0].y = y0;
    g1_points[1].x = x1;
    g1_points[1].y = y1;
    g2_points[0] = get_verify_circuit_g2_s();
    g2_points[1] = get_verify_circuit_g2_n();

    checked = pairing(g1_points, g2_points);
    require(checked, "verified failed");

    g1_points[0].x = target_circuit_final_pair[0];
    g1_points[0].y = target_circuit_final_pair[1];
    g1_points[1].x = target_circuit_final_pair[2];
    g1_points[1].y = target_circuit_final_pair[3];
    g2_points[0] = get_target_circuit_g2_s();
    g2_points[1] = get_target_circuit_g2_n();

    checked = pairing(g1_points, g2_points);
    require(checked, "verified failed");
  }
}

// File: src/L1/rollup/ZKRollup.sol



pragma solidity ^0.8.0;


// solhint-disable reason-string

/// @title ZKRollup
/// @notice This contract maintains essential data for zk rollup, including:
///
/// 1. a list of pending messages, which will be relayed to layer 2;
/// 2. the block tree generated by layer 2 and it's status.
///
/// @dev the message queue is not used yet, the offline relayer only use events in `L1ScrollMessenger`.
contract ZKRollup is OwnableUpgradeable, IZKRollup {
  /**************************************** Events ****************************************/

  /// @notice Emitted when owner updates address of operator
  /// @param _oldOperator The address of old operator.
  /// @param _newOperator The address of new operator.
  event UpdateOperator(address _oldOperator, address _newOperator);

  /// @notice Emitted when owner updates address of messenger
  /// @param _oldMesssenger The address of old messenger contract.
  /// @param _newMesssenger The address of new messenger contract.
  event UpdateMesssenger(address _oldMesssenger, address _newMesssenger);

  /**************************************** Variables ****************************************/

  struct Layer2BlockStored {
    bytes32 parentHash;
    bytes32 transactionRoot;
    uint64 blockHeight;
    uint64 batchIndex;
  }

  struct Layer2BatchStored {
    bytes32 batchHash;
    bytes32 parentHash;
    uint64 batchIndex;
    bool verified;
  }

  /// @notice The chain id of the corresponding layer 2 chain.
  uint256 public layer2ChainId;

  /// @notice The address of L1ScrollMessenger.
  address public messenger;

  /// @notice The address of operator.
  address public operator;

  /// @dev The index of the first queue element not yet executed.
  /// The operator should change this variable when new block is commited.
  uint256 private nextQueueIndex;

  /// @dev The list of appended message hash.
  bytes32[] private messageQueue;

  /// @notice The latest finalized batch id.
  bytes32 public lastFinalizedBatchID;

  /// @notice Mapping from block hash to block struct.
  mapping(bytes32 => Layer2BlockStored) public blocks;

  /// @notice Mapping from batch id to batch struct.
  mapping(bytes32 => Layer2BatchStored) public batches;

  /// @notice Mapping from batch index to finalized batch id.
  mapping(uint256 => bytes32) public finalizedBatches;

  modifier OnlyOperator() {
    // @todo In the decentralize mode, it should be only called by a list of validator.
    require(msg.sender == operator, "caller not operator");
    _;
  }

  /**************************************** Constructor ****************************************/

  function initialize(uint256 _chainId) public initializer {
    OwnableUpgradeable.__Ownable_init();

    layer2ChainId = _chainId;
  }

  /**************************************** View Functions ****************************************/

  /// @inheritdoc IZKRollup
  function isBlockFinalized(bytes32 _blockHash) external view returns (bool) {
    // block not commited
    if (blocks[_blockHash].transactionRoot == bytes32(0)) return false;

    uint256 _batchIndex = blocks[_blockHash].batchIndex;
    bytes32 _batchId = finalizedBatches[_batchIndex];
    return _batchId != bytes32(0);
  }

  /// @inheritdoc IZKRollup
  function isBlockFinalized(uint256 _blockHeight) external view returns (bool) {
    bytes32 _batchID = lastFinalizedBatchID;
    bytes32 _batchHash = batches[_batchID].batchHash;
    uint256 _maxHeight = blocks[_batchHash].blockHeight;
    return _blockHeight <= _maxHeight;
  }

  /// @inheritdoc IZKRollup
  function getMessageHashByIndex(uint256 _index) external view returns (bytes32) {
    return messageQueue[_index];
  }

  /// @inheritdoc IZKRollup
  function getNextQueueIndex() external view returns (uint256) {
    return nextQueueIndex;
  }

  /// @notice Return the total number of appended message.
  function getQeueuLength() external view returns (uint256) {
    return messageQueue.length;
  }

  /// @inheritdoc IZKRollup
  function layer2GasLimit(uint256) public view virtual returns (uint256) {
    // hardcode for now
    return 30000000;
  }

  /// @inheritdoc IZKRollup
  function verifyMessageStateProof(uint256 _batchIndex, uint256 _blockHeight) external view returns (bool) {
    bytes32 _batchId = finalizedBatches[_batchIndex];
    // check if batch is verified
    if (_batchId == bytes32(0)) return false;

    uint256 _maxBlockHeightInBatch = blocks[batches[_batchId].batchHash].blockHeight;
    // check block height is in batch range.
    if (_maxBlockHeightInBatch == 0) return _blockHeight == 0;
    else {
      uint256 _minBlockHeightInBatch = blocks[batches[_batchId].parentHash].blockHeight + 1;
      return _minBlockHeightInBatch <= _blockHeight && _blockHeight <= _maxBlockHeightInBatch;
    }
  }

  /**************************************** Mutated Functions ****************************************/

  /// @inheritdoc IZKRollup
  function appendMessage(
    address _sender,
    address _target,
    uint256 _value,
    uint256 _fee,
    uint256 _deadline,
    bytes memory _message,
    uint256 _gasLimit
  ) external override returns (uint256) {
    // currently make only messenger to call
    require(msg.sender == messenger, "caller not messenger");
    uint256 _nonce = messageQueue.length;

    // @todo may change it later
    bytes32 _messageHash = keccak256(
      abi.encodePacked(_sender, _target, _value, _fee, _deadline, _nonce, _message, _gasLimit)
    );
    messageQueue.push(_messageHash);

    return _nonce;
  }

  /// @notice Import layer 2 genesis block
  function importGenesisBlock(Layer2BlockHeader memory _genesis) external onlyOwner {
    require(lastFinalizedBatchID == bytes32(0), "Genesis block imported");
    require(_genesis.blockHash != bytes32(0), "Block hash is zero");
    require(_genesis.blockHeight == 0, "Block is not genesis");
    require(_genesis.parentHash == bytes32(0), "Parent hash not empty");

    require(_verifyBlockHash(_genesis), "Block hash verification failed");

    Layer2BlockStored storage _block = blocks[_genesis.blockHash];
    _block.transactionRoot = _computeTransactionRoot(_genesis.txs);

    bytes32 _batchId = _computeBatchId(_genesis.blockHash, bytes32(0), 0);
    Layer2BatchStored storage _batch = batches[_batchId];

    _batch.batchHash = _genesis.blockHash;
    _batch.verified = true;

    lastFinalizedBatchID = _batchId;
    finalizedBatches[0] = _batchId;

    emit CommitBatch(_batchId, _genesis.blockHash, 0, bytes32(0));
    emit FinalizeBatch(_batchId, _genesis.blockHash, 0, bytes32(0));
  }

  /// @inheritdoc IZKRollup
  function commitBatch(Layer2Batch memory _batch) external override OnlyOperator {
    // check whether the batch is empty
    require(_batch.blocks.length > 0, "Batch is empty");

    bytes32 _batchHash = _batch.blocks[_batch.blocks.length - 1].blockHash;
    bytes32 _batchId = _computeBatchId(_batchHash, _batch.parentHash, _batch.batchIndex);
    Layer2BatchStored storage _batchStored = batches[_batchId];

    // check whether the batch is commited before
    require(_batchStored.batchHash == bytes32(0), "Batch has been committed before");

    // make sure the parent batch is commited before
    Layer2BlockStored storage _parentBlock = blocks[_batch.parentHash];
    require(_parentBlock.transactionRoot != bytes32(0), "Parent batch hasn't been committed");
    require(_parentBlock.batchIndex + 1 == _batch.batchIndex, "Batch index and parent batch index mismatch");

    // check whether the blocks are correct.
    unchecked {
      uint256 _expectedBlockHeight = _parentBlock.blockHeight + 1;
      bytes32 _expectedParentHash = _batch.parentHash;
      for (uint256 i = 0; i < _batch.blocks.length; i++) {
        Layer2BlockHeader memory _block = _batch.blocks[i];
        require(_verifyBlockHash(_block), "Block hash verification failed");
        require(_block.parentHash == _expectedParentHash, "Block parent hash mismatch");
        require(_block.blockHeight == _expectedBlockHeight, "Block height mismatch");
        require(blocks[_block.blockHash].transactionRoot == bytes32(0), "Block has been commited before");

        _expectedBlockHeight += 1;
        _expectedParentHash = _block.blockHash;
      }
    }

    // do block commit
    for (uint256 i = 0; i < _batch.blocks.length; i++) {
      Layer2BlockHeader memory _block = _batch.blocks[i];
      Layer2BlockStored storage _blockStored = blocks[_block.blockHash];
      _blockStored.parentHash = _block.parentHash;
      _blockStored.transactionRoot = _computeTransactionRoot(_block.txs);
      _blockStored.blockHeight = _block.blockHeight;
      _blockStored.batchIndex = _batch.batchIndex;
    }

    _batchStored.batchHash = _batchHash;
    _batchStored.parentHash = _batch.parentHash;
    _batchStored.batchIndex = _batch.batchIndex;

    emit CommitBatch(_batchId, _batchHash, _batch.batchIndex, _batch.parentHash);
  }

  /// @inheritdoc IZKRollup
  function revertBatch(bytes32 _batchId) external override OnlyOperator {
    Layer2BatchStored storage _batch = batches[_batchId];

    require(_batch.batchHash != bytes32(0), "No such batch");
    require(!_batch.verified, "Unable to revert verified batch");

    bytes32 _blockHash = _batch.batchHash;
    bytes32 _parentHash = _batch.parentHash;

    // delete commited blocks
    while (_blockHash != _parentHash) {
      bytes32 _nextBlockHash = blocks[_blockHash].parentHash;
      delete blocks[_blockHash];

      _blockHash = _nextBlockHash;
    }

    // delete commited batch
    delete batches[_batchId];

    emit RevertBatch(_batchId);
  }

  /// @inheritdoc IZKRollup
  function finalizeBatchWithProof(
    bytes32 _batchId,
    uint256[] memory _proof,
    uint256[] memory _instances
  ) external override OnlyOperator {
    Layer2BatchStored storage _batch = batches[_batchId];
    require(_batch.batchHash != bytes32(0), "No such batch");
    require(!_batch.verified, "Batch already verified");

    // @note skip parent check for now, since we may not prove blocks in order.
    // bytes32 _parentHash = _block.header.parentHash;
    // require(lastFinalizedBlockHash == _parentHash, "parent not latest finalized");
    // this check below is not needed, just incase
    // require(blocks[_parentHash].verified, "parent not verified");

    // @todo add verification logic
    RollupVerifier.verify(_proof, _instances);

    uint256 _batchIndex = _batch.batchIndex;
    finalizedBatches[_batchIndex] = _batchId;
    _batch.verified = true;

    Layer2BatchStored storage _finalizedBatch = batches[lastFinalizedBatchID];
    if (_batchIndex > _finalizedBatch.batchIndex) {
      lastFinalizedBatchID = _batchId;
    }

    emit FinalizeBatch(_batchId, _batch.batchHash, _batchIndex, _batch.parentHash);
  }

  /**************************************** Restricted Functions ****************************************/

  /// @notice Update the address of operator.
  /// @dev This function can only called by contract owner.
  /// @param _newOperator The new operator address to update.
  function updateOperator(address _newOperator) external onlyOwner {
    address _oldOperator = operator;
    require(_oldOperator != _newOperator, "change to same operator");

    operator = _newOperator;

    emit UpdateOperator(_oldOperator, _newOperator);
  }

  /// @notice Update the address of messenger.
  /// @dev This function can only called by contract owner.
  /// @param _newMessenger The new messenger address to update.
  function updateMessenger(address _newMessenger) external onlyOwner {
    address _oldMessenger = messenger;
    require(_oldMessenger != _newMessenger, "change to same messenger");

    messenger = _newMessenger;

    emit UpdateMesssenger(_oldMessenger, _newMessenger);
  }

  /**************************************** Internal Functions ****************************************/

  function _verifyBlockHash(Layer2BlockHeader memory) internal pure returns (bool) {
    // @todo finish logic after more discussions
    return true;
  }

  /// @dev Internal function to compute a unique batch id for mapping.
  /// @param _batchHash The hash of the batch.
  /// @param _parentHash The hash of the batch.
  /// @param _batchIndex The index of the batch.
  /// @return Return the computed batch id.
  function _computeBatchId(
    bytes32 _batchHash,
    bytes32 _parentHash,
    uint256 _batchIndex
  ) internal pure returns (bytes32) {
    return keccak256(abi.encode(_batchHash, _parentHash, _batchIndex));
  }

  /// @dev Internal function to compute transaction root.
  /// @param _txn The list of transactions in the block.
  /// @return Return the hash of transaction root.
  function _computeTransactionRoot(Layer2Transaction[] memory _txn) internal pure returns (bytes32) {
    bytes32[] memory _hashes = new bytes32[](_txn.length);
    for (uint256 i = 0; i < _txn.length; i++) {
      // @todo use rlp
      _hashes[i] = keccak256(
        abi.encode(
          _txn[i].caller,
          _txn[i].nonce,
          _txn[i].target,
          _txn[i].gas,
          _txn[i].gasPrice,
          _txn[i].value,
          _txn[i].data,
          _txn[i].r,
          _txn[i].s,
          _txn[i].v
        )
      );
    }
    return keccak256(abi.encode(_hashes));
  }
}
