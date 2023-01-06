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

// File: @openzeppelin/contracts-upgradeable/utils/introspection/IERC165Upgradeable.sol


// OpenZeppelin Contracts v4.4.1 (utils/introspection/IERC165.sol)

pragma solidity ^0.8.0;

/**
 * @dev Interface of the ERC165 standard, as defined in the
 * https://eips.ethereum.org/EIPS/eip-165[EIP].
 *
 * Implementers can declare support of contract interfaces, which can then be
 * queried by others ({ERC165Checker}).
 *
 * For an implementation, see {ERC165}.
 */
interface IERC165Upgradeable {
    /**
     * @dev Returns true if this contract implements the interface defined by
     * `interfaceId`. See the corresponding
     * https://eips.ethereum.org/EIPS/eip-165#how-interfaces-are-identified[EIP section]
     * to learn more about how these ids are created.
     *
     * This function call must use less than 30 000 gas.
     */
    function supportsInterface(bytes4 interfaceId) external view returns (bool);
}

// File: @openzeppelin/contracts-upgradeable/token/ERC1155/IERC1155Upgradeable.sol


// OpenZeppelin Contracts v4.4.1 (token/ERC1155/IERC1155.sol)

pragma solidity ^0.8.0;

/**
 * @dev Required interface of an ERC1155 compliant contract, as defined in the
 * https://eips.ethereum.org/EIPS/eip-1155[EIP].
 *
 * _Available since v3.1._
 */
interface IERC1155Upgradeable is IERC165Upgradeable {
    /**
     * @dev Emitted when `value` tokens of token type `id` are transferred from `from` to `to` by `operator`.
     */
    event TransferSingle(address indexed operator, address indexed from, address indexed to, uint256 id, uint256 value);

    /**
     * @dev Equivalent to multiple {TransferSingle} events, where `operator`, `from` and `to` are the same for all
     * transfers.
     */
    event TransferBatch(
        address indexed operator,
        address indexed from,
        address indexed to,
        uint256[] ids,
        uint256[] values
    );

    /**
     * @dev Emitted when `account` grants or revokes permission to `operator` to transfer their tokens, according to
     * `approved`.
     */
    event ApprovalForAll(address indexed account, address indexed operator, bool approved);

    /**
     * @dev Emitted when the URI for token type `id` changes to `value`, if it is a non-programmatic URI.
     *
     * If an {URI} event was emitted for `id`, the standard
     * https://eips.ethereum.org/EIPS/eip-1155#metadata-extensions[guarantees] that `value` will equal the value
     * returned by {IERC1155MetadataURI-uri}.
     */
    event URI(string value, uint256 indexed id);

    /**
     * @dev Returns the amount of tokens of token type `id` owned by `account`.
     *
     * Requirements:
     *
     * - `account` cannot be the zero address.
     */
    function balanceOf(address account, uint256 id) external view returns (uint256);

    /**
     * @dev xref:ROOT:erc1155.adoc#batch-operations[Batched] version of {balanceOf}.
     *
     * Requirements:
     *
     * - `accounts` and `ids` must have the same length.
     */
    function balanceOfBatch(address[] calldata accounts, uint256[] calldata ids)
        external
        view
        returns (uint256[] memory);

    /**
     * @dev Grants or revokes permission to `operator` to transfer the caller's tokens, according to `approved`,
     *
     * Emits an {ApprovalForAll} event.
     *
     * Requirements:
     *
     * - `operator` cannot be the caller.
     */
    function setApprovalForAll(address operator, bool approved) external;

    /**
     * @dev Returns true if `operator` is approved to transfer ``account``'s tokens.
     *
     * See {setApprovalForAll}.
     */
    function isApprovedForAll(address account, address operator) external view returns (bool);

    /**
     * @dev Transfers `amount` tokens of token type `id` from `from` to `to`.
     *
     * Emits a {TransferSingle} event.
     *
     * Requirements:
     *
     * - `to` cannot be the zero address.
     * - If the caller is not `from`, it must be have been approved to spend ``from``'s tokens via {setApprovalForAll}.
     * - `from` must have a balance of tokens of type `id` of at least `amount`.
     * - If `to` refers to a smart contract, it must implement {IERC1155Receiver-onERC1155Received} and return the
     * acceptance magic value.
     */
    function safeTransferFrom(
        address from,
        address to,
        uint256 id,
        uint256 amount,
        bytes calldata data
    ) external;

    /**
     * @dev xref:ROOT:erc1155.adoc#batch-operations[Batched] version of {safeTransferFrom}.
     *
     * Emits a {TransferBatch} event.
     *
     * Requirements:
     *
     * - `ids` and `amounts` must have the same length.
     * - If `to` refers to a smart contract, it must implement {IERC1155Receiver-onERC1155BatchReceived} and return the
     * acceptance magic value.
     */
    function safeBatchTransferFrom(
        address from,
        address to,
        uint256[] calldata ids,
        uint256[] calldata amounts,
        bytes calldata data
    ) external;
}

// File: @openzeppelin/contracts-upgradeable/token/ERC1155/IERC1155ReceiverUpgradeable.sol


// OpenZeppelin Contracts (last updated v4.5.0) (token/ERC1155/IERC1155Receiver.sol)

pragma solidity ^0.8.0;

/**
 * @dev _Available since v3.1._
 */
interface IERC1155ReceiverUpgradeable is IERC165Upgradeable {
    /**
     * @dev Handles the receipt of a single ERC1155 token type. This function is
     * called at the end of a `safeTransferFrom` after the balance has been updated.
     *
     * NOTE: To accept the transfer, this must return
     * `bytes4(keccak256("onERC1155Received(address,address,uint256,uint256,bytes)"))`
     * (i.e. 0xf23a6e61, or its own function selector).
     *
     * @param operator The address which initiated the transfer (i.e. msg.sender)
     * @param from The address which previously owned the token
     * @param id The ID of the token being transferred
     * @param value The amount of tokens being transferred
     * @param data Additional data with no specified format
     * @return `bytes4(keccak256("onERC1155Received(address,address,uint256,uint256,bytes)"))` if transfer is allowed
     */
    function onERC1155Received(
        address operator,
        address from,
        uint256 id,
        uint256 value,
        bytes calldata data
    ) external returns (bytes4);

    /**
     * @dev Handles the receipt of a multiple ERC1155 token types. This function
     * is called at the end of a `safeBatchTransferFrom` after the balances have
     * been updated.
     *
     * NOTE: To accept the transfer(s), this must return
     * `bytes4(keccak256("onERC1155BatchReceived(address,address,uint256[],uint256[],bytes)"))`
     * (i.e. 0xbc197c81, or its own function selector).
     *
     * @param operator The address which initiated the batch transfer (i.e. msg.sender)
     * @param from The address which previously owned the token
     * @param ids An array containing ids of each token being transferred (order and length must match values array)
     * @param values An array containing amounts of each token being transferred (order and length must match ids array)
     * @param data Additional data with no specified format
     * @return `bytes4(keccak256("onERC1155BatchReceived(address,address,uint256[],uint256[],bytes)"))` if transfer is allowed
     */
    function onERC1155BatchReceived(
        address operator,
        address from,
        uint256[] calldata ids,
        uint256[] calldata values,
        bytes calldata data
    ) external returns (bytes4);
}

// File: @openzeppelin/contracts-upgradeable/utils/introspection/ERC165Upgradeable.sol


// OpenZeppelin Contracts v4.4.1 (utils/introspection/ERC165.sol)

pragma solidity ^0.8.0;


/**
 * @dev Implementation of the {IERC165} interface.
 *
 * Contracts that want to implement ERC165 should inherit from this contract and override {supportsInterface} to check
 * for the additional interface id that will be supported. For example:
 *
 * ```solidity
 * function supportsInterface(bytes4 interfaceId) public view virtual override returns (bool) {
 *     return interfaceId == type(MyInterface).interfaceId || super.supportsInterface(interfaceId);
 * }
 * ```
 *
 * Alternatively, {ERC165Storage} provides an easier to use but more expensive implementation.
 */
abstract contract ERC165Upgradeable is Initializable, IERC165Upgradeable {
    function __ERC165_init() internal onlyInitializing {
    }

    function __ERC165_init_unchained() internal onlyInitializing {
    }
    /**
     * @dev See {IERC165-supportsInterface}.
     */
    function supportsInterface(bytes4 interfaceId) public view virtual override returns (bool) {
        return interfaceId == type(IERC165Upgradeable).interfaceId;
    }

    /**
     * @dev This empty reserved space is put in place to allow future versions to add new
     * variables without shifting down storage in the inheritance chain.
     * See https://docs.openzeppelin.com/contracts/4.x/upgradeable#storage_gaps
     */
    uint256[50] private __gap;
}

// File: @openzeppelin/contracts-upgradeable/token/ERC1155/utils/ERC1155ReceiverUpgradeable.sol


// OpenZeppelin Contracts v4.4.1 (token/ERC1155/utils/ERC1155Receiver.sol)

pragma solidity ^0.8.0;



/**
 * @dev _Available since v3.1._
 */
abstract contract ERC1155ReceiverUpgradeable is Initializable, ERC165Upgradeable, IERC1155ReceiverUpgradeable {
    function __ERC1155Receiver_init() internal onlyInitializing {
    }

    function __ERC1155Receiver_init_unchained() internal onlyInitializing {
    }
    /**
     * @dev See {IERC165-supportsInterface}.
     */
    function supportsInterface(bytes4 interfaceId) public view virtual override(ERC165Upgradeable, IERC165Upgradeable) returns (bool) {
        return interfaceId == type(IERC1155ReceiverUpgradeable).interfaceId || super.supportsInterface(interfaceId);
    }

    /**
     * @dev This empty reserved space is put in place to allow future versions to add new
     * variables without shifting down storage in the inheritance chain.
     * See https://docs.openzeppelin.com/contracts/4.x/upgradeable#storage_gaps
     */
    uint256[50] private __gap;
}

// File: @openzeppelin/contracts-upgradeable/token/ERC1155/utils/ERC1155HolderUpgradeable.sol


// OpenZeppelin Contracts (last updated v4.5.0) (token/ERC1155/utils/ERC1155Holder.sol)

pragma solidity ^0.8.0;


/**
 * Simple implementation of `ERC1155Receiver` that will allow a contract to hold ERC1155 tokens.
 *
 * IMPORTANT: When inheriting this contract, you must include a way to use the received tokens, otherwise they will be
 * stuck.
 *
 * @dev _Available since v3.1._
 */
contract ERC1155HolderUpgradeable is Initializable, ERC1155ReceiverUpgradeable {
    function __ERC1155Holder_init() internal onlyInitializing {
    }

    function __ERC1155Holder_init_unchained() internal onlyInitializing {
    }
    function onERC1155Received(
        address,
        address,
        uint256,
        uint256,
        bytes memory
    ) public virtual override returns (bytes4) {
        return this.onERC1155Received.selector;
    }

    function onERC1155BatchReceived(
        address,
        address,
        uint256[] memory,
        uint256[] memory,
        bytes memory
    ) public virtual override returns (bytes4) {
        return this.onERC1155BatchReceived.selector;
    }

    /**
     * @dev This empty reserved space is put in place to allow future versions to add new
     * variables without shifting down storage in the inheritance chain.
     * See https://docs.openzeppelin.com/contracts/4.x/upgradeable#storage_gaps
     */
    uint256[50] private __gap;
}

// File: src/L2/gateways/IL2ERC1155Gateway.sol



pragma solidity ^0.8.0;

/// @title The interface for the ERC1155 cross chain gateway in layer 2.
interface IL2ERC1155Gateway {
  /**************************************** Events ****************************************/

  event FinalizeDepositERC1155(
    address indexed _l1Token,
    address indexed _l2Token,
    address indexed _from,
    address _to,
    uint256 _tokenId,
    uint256 _amount
  );

  event FinalizeBatchDepositERC1155(
    address indexed _l1Token,
    address indexed _l2Token,
    address indexed _from,
    address _to,
    uint256[] _tokenIds,
    uint256[] _amounts
  );

  event WithdrawERC1155(
    address indexed _l1Token,
    address indexed _l2Token,
    address indexed _from,
    address _to,
    uint256 _tokenId,
    uint256 _amount
  );

  event BatchWithdrawERC1155(
    address indexed _l1Token,
    address indexed _l2Token,
    address indexed _from,
    address _to,
    uint256[] _tokenIds,
    uint256[] _amounts
  );

  /**************************************** Mutated Funtions ****************************************/

  function withdrawERC1155(
    address _token,
    uint256 _tokenId,
    uint256 _amount,
    uint256 _gasLimit
  ) external;

  function withdrawERC1155(
    address _token,
    address _to,
    uint256 _tokenId,
    uint256 _amount,
    uint256 _gasLimit
  ) external;

  function batchWithdrawERC1155(
    address _token,
    uint256[] memory _tokenIds,
    uint256[] memory _amounts,
    uint256 _gasLimit
  ) external;

  function batchWithdrawERC1155(
    address _token,
    address _to,
    uint256[] memory _tokenIds,
    uint256[] memory _amounts,
    uint256 _gasLimit
  ) external;

  function finalizeDepositERC1155(
    address _l1Token,
    address _l2Token,
    address _from,
    address _to,
    uint256 _tokenId,
    uint256 _amount
  ) external;

  function finalizeBatchDepositERC1155(
    address _l1Token,
    address _l2Token,
    address _from,
    address _to,
    uint256[] calldata _tokenIds,
    uint256[] calldata _amounts
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

// File: src/L2/IL2ScrollMessenger.sol



pragma solidity ^0.8.0;

interface IL2ScrollMessenger is IScrollMessenger {
  /**************************************** Mutate Functions ****************************************/

  /// @notice execute L1 => L2 message
  /// @dev Make sure this is only called by privileged accounts.
  /// @param _from The address of the sender of the message.
  /// @param _to The address of the recipient of the message.
  /// @param _value The msg.value passed to the message call.
  /// @param _fee The amount of fee in ETH to charge.
  /// @param _deadline The deadline of the message.
  /// @param _nonce The nonce of the message to avoid replay attack.
  /// @param _message The content of the message.
  function relayMessage(
    address _from,
    address _to,
    uint256 _value,
    uint256 _fee,
    uint256 _deadline,
    uint256 _nonce,
    bytes memory _message
  ) external;
}

// File: src/L1/gateways/IL1ERC1155Gateway.sol



pragma solidity ^0.8.0;

/// @title The interface for the ERC1155 cross chain gateway in layer 1.
interface IL1ERC1155Gateway {
  /**************************************** Events ****************************************/

  /// @notice Emitted when the ERC1155 NFT is transfered to recipient in layer 1.
  /// @param _l1Token The address of ERC1155 NFT in layer 1.
  /// @param _l2Token The address of ERC1155 NFT in layer 2.
  /// @param _from The address of sender in layer 2.
  /// @param _to The address of recipient in layer 1.
  /// @param _tokenId The token id of the ERC1155 NFT to withdraw from layer 2.
  /// @param _amount The number of token to withdraw from layer 2.
  event FinalizeWithdrawERC1155(
    address indexed _l1Token,
    address indexed _l2Token,
    address indexed _from,
    address _to,
    uint256 _tokenId,
    uint256 _amount
  );

  /// @notice Emitted when the ERC1155 NFT is batch transfered to recipient in layer 1.
  /// @param _l1Token The address of ERC1155 NFT in layer 1.
  /// @param _l2Token The address of ERC1155 NFT in layer 2.
  /// @param _from The address of sender in layer 2.
  /// @param _to The address of recipient in layer 1.
  /// @param _tokenIds The list of token ids of the ERC1155 NFT to withdraw from layer 2.
  /// @param _amounts The list of corresponding number of token to withdraw from layer 2.
  event FinalizeBatchWithdrawERC1155(
    address indexed _l1Token,
    address indexed _l2Token,
    address indexed _from,
    address _to,
    uint256[] _tokenIds,
    uint256[] _amounts
  );

  /// @notice Emitted when the ERC1155 NFT is deposited to gateway in layer 1.
  /// @param _l1Token The address of ERC1155 NFT in layer 1.
  /// @param _l2Token The address of ERC1155 NFT in layer 2.
  /// @param _from The address of sender in layer 1.
  /// @param _to The address of recipient in layer 2.
  /// @param _tokenId The token id of the ERC1155 NFT to deposit in layer 1.
  /// @param _amount The number of token to deposit in layer 1.
  event DepositERC1155(
    address indexed _l1Token,
    address indexed _l2Token,
    address indexed _from,
    address _to,
    uint256 _tokenId,
    uint256 _amount
  );

  /// @notice Emitted when the ERC1155 NFT is batch deposited to gateway in layer 1.
  /// @param _l1Token The address of ERC1155 NFT in layer 1.
  /// @param _l2Token The address of ERC1155 NFT in layer 2.
  /// @param _from The address of sender in layer 1.
  /// @param _to The address of recipient in layer 2.
  /// @param _tokenIds The list of token ids of the ERC1155 NFT to deposit in layer 1.
  /// @param _amounts The list of corresponding number of token to deposit in layer 1.
  event BatchDepositERC1155(
    address indexed _l1Token,
    address indexed _l2Token,
    address indexed _from,
    address _to,
    uint256[] _tokenIds,
    uint256[] _amounts
  );

  /**************************************** Mutated Funtions ****************************************/

  /// @notice Deposit some ERC1155 NFT to caller's account on layer 2.
  /// @param _token The address of ERC1155 NFT in layer 1.
  /// @param _tokenId The token id to deposit.
  /// @param _amount The amount of token to deposit.
  /// @param _gasLimit Estimated gas limit required to complete the deposit on layer 2.
  function depositERC1155(
    address _token,
    uint256 _tokenId,
    uint256 _amount,
    uint256 _gasLimit
  ) external;

  /// @notice Deposit some ERC1155 NFT to a recipient's account on layer 2.
  /// @param _token The address of ERC1155 NFT in layer 1.
  /// @param _to The address of recipient in layer 2.
  /// @param _tokenId The token id to deposit.
  /// @param _amount The amount of token to deposit.
  /// @param _gasLimit Estimated gas limit required to complete the deposit on layer 2.
  function depositERC1155(
    address _token,
    address _to,
    uint256 _tokenId,
    uint256 _amount,
    uint256 _gasLimit
  ) external;

  /// @notice Deposit a list of some ERC1155 NFT to caller's account on layer 2.
  /// @param _token The address of ERC1155 NFT in layer 1.
  /// @param _tokenIds The list of token ids to deposit.
  /// @param _amounts The list of corresponding number of token to deposit.
  /// @param _gasLimit Estimated gas limit required to complete the deposit on layer 2.
  function batchDepositERC1155(
    address _token,
    uint256[] calldata _tokenIds,
    uint256[] calldata _amounts,
    uint256 _gasLimit
  ) external;

  /// @notice Deposit a list of some ERC1155 NFT to a recipient's account on layer 2.
  /// @param _token The address of ERC1155 NFT in layer 1.
  /// @param _to The address of recipient in layer 2.
  /// @param _tokenIds The list of token ids to deposit.
  /// @param _amounts The list of corresponding number of token to deposit.
  /// @param _gasLimit Estimated gas limit required to complete the deposit on layer 2.
  function batchDepositERC1155(
    address _token,
    address _to,
    uint256[] calldata _tokenIds,
    uint256[] calldata _amounts,
    uint256 _gasLimit
  ) external;

  /// @notice Complete ERC1155 withdraw from layer 2 to layer 1 and send fund to recipient's account in layer 1.
  ///      The function should only be called by L1ScrollMessenger.
  ///      The function should also only be called by L2ERC1155Gateway in layer 2.
  /// @param _l1Token The address of corresponding layer 1 token.
  /// @param _l2Token The address of corresponding layer 2 token.
  /// @param _from The address of account who withdraw the token in layer 2.
  /// @param _to The address of recipient in layer 1 to receive the token.
  /// @param _tokenId The token id to withdraw.
  /// @param _amount The amount of token to withdraw.
  function finalizeWithdrawERC1155(
    address _l1Token,
    address _l2Token,
    address _from,
    address _to,
    uint256 _tokenId,
    uint256 _amount
  ) external;

  /// @notice Complete ERC1155 batch withdraw from layer 2 to layer 1 and send fund to recipient's account in layer 1.
  ///      The function should only be called by L1ScrollMessenger.
  ///      The function should also only be called by L2ERC1155Gateway in layer 2.
  /// @param _l1Token The address of corresponding layer 1 token.
  /// @param _l2Token The address of corresponding layer 2 token.
  /// @param _from The address of account who withdraw the token in layer 2.
  /// @param _to The address of recipient in layer 1 to receive the token.
  /// @param _tokenIds The list of token ids to withdraw.
  /// @param _amounts The list of corresponding number of token to withdraw.
  function finalizeBatchWithdrawERC1155(
    address _l1Token,
    address _l2Token,
    address _from,
    address _to,
    uint256[] calldata _tokenIds,
    uint256[] calldata _amounts
  ) external;
}

// File: src/libraries/gateway/IScrollGateway.sol



pragma solidity ^0.8.0;

interface IScrollGateway {
  function counterpart() external view returns (address);

  function finalizeDropMessage() external payable;
}

// File: src/libraries/gateway/ScrollGatewayBase.sol



pragma solidity ^0.8.0;


abstract contract ScrollGatewayBase is IScrollGateway {
  /// @notice The address of corresponding L1/L2 Gateway contract.
  address public override counterpart;
  /// @notice The address of L1GatewayRouter/L2GatewayRouter contract.
  address public router;
  /// @notice The address of L1ScrollMessenger/L2ScrollMessenger contract.
  address public messenger;

  // start of inline reentrancy guard
  // https://github.com/OpenZeppelin/openzeppelin-contracts/blob/v4.5.0/contracts/security/ReentrancyGuard.sol
  uint256 private constant _NOT_ENTERED = 1;
  uint256 private constant _ENTERED = 2;
  uint256 private _status;

  modifier nonReentrant() {
    // On the first call to nonReentrant, _notEntered will be true
    require(_status != _ENTERED, "ReentrancyGuard: reentrant call");

    // Any calls to nonReentrant after this point will fail
    _status = _ENTERED;

    _;

    // By storing the original value once again, a refund is triggered (see
    // https://eips.ethereum.org/EIPS/eip-2200)
    _status = _NOT_ENTERED;
  }

  modifier onlyMessenger() {
    require(msg.sender == messenger, "only messenger can call");
    _;
  }

  modifier onlyCallByCounterpart() {
    address _messenger = messenger; // gas saving
    require(msg.sender == _messenger, "only messenger can call");
    require(counterpart == IScrollMessenger(_messenger).xDomainMessageSender(), "only call by conterpart");
    _;
  }

  function _initialize(
    address _counterpart,
    address _router,
    address _messenger
  ) internal {
    require(_counterpart != address(0), "zero counterpart address");
    require(_messenger != address(0), "zero messenger address");

    counterpart = _counterpart;
    messenger = _messenger;

    // @note: the address of router could be zero, if this contract is GatewayRouter.
    if (_router != address(0)) {
      router = _router;
    }

    // for reentrancy guard
    _status = _NOT_ENTERED;
  }
}

// File: @openzeppelin/contracts/utils/introspection/IERC165.sol


// OpenZeppelin Contracts v4.4.1 (utils/introspection/IERC165.sol)

pragma solidity ^0.8.0;

/**
 * @dev Interface of the ERC165 standard, as defined in the
 * https://eips.ethereum.org/EIPS/eip-165[EIP].
 *
 * Implementers can declare support of contract interfaces, which can then be
 * queried by others ({ERC165Checker}).
 *
 * For an implementation, see {ERC165}.
 */
interface IERC165 {
    /**
     * @dev Returns true if this contract implements the interface defined by
     * `interfaceId`. See the corresponding
     * https://eips.ethereum.org/EIPS/eip-165#how-interfaces-are-identified[EIP section]
     * to learn more about how these ids are created.
     *
     * This function call must use less than 30 000 gas.
     */
    function supportsInterface(bytes4 interfaceId) external view returns (bool);
}

// File: @openzeppelin/contracts/token/ERC1155/IERC1155.sol


// OpenZeppelin Contracts v4.4.1 (token/ERC1155/IERC1155.sol)

pragma solidity ^0.8.0;

/**
 * @dev Required interface of an ERC1155 compliant contract, as defined in the
 * https://eips.ethereum.org/EIPS/eip-1155[EIP].
 *
 * _Available since v3.1._
 */
interface IERC1155 is IERC165 {
    /**
     * @dev Emitted when `value` tokens of token type `id` are transferred from `from` to `to` by `operator`.
     */
    event TransferSingle(address indexed operator, address indexed from, address indexed to, uint256 id, uint256 value);

    /**
     * @dev Equivalent to multiple {TransferSingle} events, where `operator`, `from` and `to` are the same for all
     * transfers.
     */
    event TransferBatch(
        address indexed operator,
        address indexed from,
        address indexed to,
        uint256[] ids,
        uint256[] values
    );

    /**
     * @dev Emitted when `account` grants or revokes permission to `operator` to transfer their tokens, according to
     * `approved`.
     */
    event ApprovalForAll(address indexed account, address indexed operator, bool approved);

    /**
     * @dev Emitted when the URI for token type `id` changes to `value`, if it is a non-programmatic URI.
     *
     * If an {URI} event was emitted for `id`, the standard
     * https://eips.ethereum.org/EIPS/eip-1155#metadata-extensions[guarantees] that `value` will equal the value
     * returned by {IERC1155MetadataURI-uri}.
     */
    event URI(string value, uint256 indexed id);

    /**
     * @dev Returns the amount of tokens of token type `id` owned by `account`.
     *
     * Requirements:
     *
     * - `account` cannot be the zero address.
     */
    function balanceOf(address account, uint256 id) external view returns (uint256);

    /**
     * @dev xref:ROOT:erc1155.adoc#batch-operations[Batched] version of {balanceOf}.
     *
     * Requirements:
     *
     * - `accounts` and `ids` must have the same length.
     */
    function balanceOfBatch(address[] calldata accounts, uint256[] calldata ids)
        external
        view
        returns (uint256[] memory);

    /**
     * @dev Grants or revokes permission to `operator` to transfer the caller's tokens, according to `approved`,
     *
     * Emits an {ApprovalForAll} event.
     *
     * Requirements:
     *
     * - `operator` cannot be the caller.
     */
    function setApprovalForAll(address operator, bool approved) external;

    /**
     * @dev Returns true if `operator` is approved to transfer ``account``'s tokens.
     *
     * See {setApprovalForAll}.
     */
    function isApprovedForAll(address account, address operator) external view returns (bool);

    /**
     * @dev Transfers `amount` tokens of token type `id` from `from` to `to`.
     *
     * Emits a {TransferSingle} event.
     *
     * Requirements:
     *
     * - `to` cannot be the zero address.
     * - If the caller is not `from`, it must be have been approved to spend ``from``'s tokens via {setApprovalForAll}.
     * - `from` must have a balance of tokens of type `id` of at least `amount`.
     * - If `to` refers to a smart contract, it must implement {IERC1155Receiver-onERC1155Received} and return the
     * acceptance magic value.
     */
    function safeTransferFrom(
        address from,
        address to,
        uint256 id,
        uint256 amount,
        bytes calldata data
    ) external;

    /**
     * @dev xref:ROOT:erc1155.adoc#batch-operations[Batched] version of {safeTransferFrom}.
     *
     * Emits a {TransferBatch} event.
     *
     * Requirements:
     *
     * - `ids` and `amounts` must have the same length.
     * - If `to` refers to a smart contract, it must implement {IERC1155Receiver-onERC1155BatchReceived} and return the
     * acceptance magic value.
     */
    function safeBatchTransferFrom(
        address from,
        address to,
        uint256[] calldata ids,
        uint256[] calldata amounts,
        bytes calldata data
    ) external;
}

// File: src/libraries/token/IScrollERC1155.sol



pragma solidity ^0.8.0;

interface IScrollERC1155 is IERC1155 {
  /// @notice Return the address of Gateway the token belongs to.
  function gateway() external view returns (address);

  /// @notice Return the address of counterpart token.
  function counterpart() external view returns (address);

  /// @notice Mint some token to recipient's account.
  /// @dev Gateway Utilities, only gateway contract can call
  /// @param _to The address of recipient.
  /// @param _tokenId The token id to mint.
  /// @param _amount The amount of token to mint.
  /// @param _data The data passed to recipient
  function mint(
    address _to,
    uint256 _tokenId,
    uint256 _amount,
    bytes memory _data
  ) external;

  /// @notice Burn some token from account.
  /// @dev Gateway Utilities, only gateway contract can call
  /// @param _from The address of account to burn token.
  /// @param _tokenId The token id to burn.
  /// @param _amount The amount of token to burn.
  function burn(
    address _from,
    uint256 _tokenId,
    uint256 _amount
  ) external;

  /// @notice Batch mint some token to recipient's account.
  /// @dev Gateway Utilities, only gateway contract can call
  /// @param _to The address of recipient.
  /// @param _tokenIds The token id to mint.
  /// @param _amounts The list of corresponding amount of token to mint.
  /// @param _data The data passed to recipient
  function batchMint(
    address _to,
    uint256[] calldata _tokenIds,
    uint256[] calldata _amounts,
    bytes calldata _data
  ) external;

  /// @notice Batch burn some token from account.
  /// @dev Gateway Utilities, only gateway contract can call
  /// @param _from The address of account to burn token.
  /// @param _tokenIds The list of token ids to burn.
  /// @param _amounts The list of corresponding amount of token to burn.
  function batchBurn(
    address _from,
    uint256[] calldata _tokenIds,
    uint256[] calldata _amounts
  ) external;
}

// File: src/L2/gateways/L2ERC1155Gateway.sol



pragma solidity ^0.8.0;







/// @title L2ERC1155Gateway
/// @notice The `L2ERC1155Gateway` is used to withdraw ERC1155 compatible NFTs in layer 2 and
/// finalize deposit the NFTs from layer 1.
/// @dev The withdrawn NFTs tokens will be burned directly. On finalizing deposit, the corresponding
/// NFT will be minted and transfered to the recipient.
///
/// This will be changed if we have more specific scenarios.
// @todo Current implementation doesn't support calling from `L2GatewayRouter`.
contract L2ERC1155Gateway is OwnableUpgradeable, ERC1155HolderUpgradeable, ScrollGatewayBase, IL2ERC1155Gateway {
  /**************************************** Events ****************************************/

  /// @notice Emitted when token mapping for ERC1155 token is updated.
  /// @param _l1Token The address of corresponding ERC1155 token in layer 2.
  /// @param _l1Token The address of ERC1155 token in layer 1.
  event UpdateTokenMapping(address _l2Token, address _l1Token);

  /**************************************** Variables ****************************************/

  /// @notice Mapping from layer 2 token address to layer 1 token address for ERC1155 NFT.
  // solhint-disable-next-line var-name-mixedcase
  mapping(address => address) public tokenMapping;

  /**************************************** Constructor ****************************************/

  function initialize(address _counterpart, address _messenger) external initializer {
    OwnableUpgradeable.__Ownable_init();
    ScrollGatewayBase._initialize(_counterpart, address(0), _messenger);
  }

  /**************************************** Mutate Funtions ****************************************/

  /// @inheritdoc IL2ERC1155Gateway
  function withdrawERC1155(
    address _token,
    uint256 _tokenId,
    uint256 _amount,
    uint256 _gasLimit
  ) external override {
    _withdrawERC1155(_token, msg.sender, _tokenId, _amount, _gasLimit);
  }

  /// @inheritdoc IL2ERC1155Gateway
  function withdrawERC1155(
    address _token,
    address _to,
    uint256 _tokenId,
    uint256 _amount,
    uint256 _gasLimit
  ) external override {
    _withdrawERC1155(_token, _to, _tokenId, _amount, _gasLimit);
  }

  /// @inheritdoc IL2ERC1155Gateway
  function batchWithdrawERC1155(
    address _token,
    uint256[] calldata _tokenIds,
    uint256[] calldata _amounts,
    uint256 _gasLimit
  ) external override {
    _batchWithdrawERC1155(_token, msg.sender, _tokenIds, _amounts, _gasLimit);
  }

  /// @inheritdoc IL2ERC1155Gateway
  function batchWithdrawERC1155(
    address _token,
    address _to,
    uint256[] calldata _tokenIds,
    uint256[] calldata _amounts,
    uint256 _gasLimit
  ) external override {
    _batchWithdrawERC1155(_token, _to, _tokenIds, _amounts, _gasLimit);
  }

  /// @inheritdoc IL2ERC1155Gateway
  function finalizeDepositERC1155(
    address _l1Token,
    address _l2Token,
    address _from,
    address _to,
    uint256 _tokenId,
    uint256 _amount
  ) external override nonReentrant onlyCallByCounterpart {
    IScrollERC1155(_l2Token).mint(_to, _tokenId, _amount, "");

    emit FinalizeDepositERC1155(_l1Token, _l2Token, _from, _to, _tokenId, _amount);
  }

  /// @inheritdoc IL2ERC1155Gateway
  function finalizeBatchDepositERC1155(
    address _l1Token,
    address _l2Token,
    address _from,
    address _to,
    uint256[] calldata _tokenIds,
    uint256[] calldata _amounts
  ) external override nonReentrant onlyCallByCounterpart {
    IScrollERC1155(_l2Token).batchMint(_to, _tokenIds, _amounts, "");

    emit FinalizeBatchDepositERC1155(_l1Token, _l2Token, _from, _to, _tokenIds, _amounts);
  }

  /// @inheritdoc IScrollGateway
  function finalizeDropMessage() external payable {
    // @todo finish the logic later
  }

  /**************************************** Restricted Funtions ****************************************/

  /// @notice Update layer 2 to layer 1 token mapping.
  /// @param _l1Token The address of corresponding ERC1155 token in layer 2.
  /// @param _l1Token The address of ERC1155 token in layer 1.
  function updateTokenMapping(address _l2Token, address _l1Token) external onlyOwner {
    require(_l1Token != address(0), "map to zero address");

    tokenMapping[_l2Token] = _l1Token;

    emit UpdateTokenMapping(_l2Token, _l1Token);
  }

  /**************************************** Internal Funtions ****************************************/

  /// @dev Internal function to withdraw ERC1155 NFT to layer 2.
  /// @param _token The address of ERC1155 NFT in layer 1.
  /// @param _to The address of recipient in layer 2.
  /// @param _tokenId The token id to withdraw.
  /// @param _amount The amount of token to withdraw.
  /// @param _gasLimit Estimated gas limit required to complete the withdraw on layer 2.
  function _withdrawERC1155(
    address _token,
    address _to,
    uint256 _tokenId,
    uint256 _amount,
    uint256 _gasLimit
  ) internal nonReentrant {
    require(_amount > 0, "withdraw zero amount");

    address _l1Token = tokenMapping[_token];
    require(_l1Token != address(0), "token not supported");

    // 1. burn token
    IScrollERC1155(_token).burn(msg.sender, _tokenId, _amount);

    // 2. Generate message passed to L1ERC1155Gateway.
    bytes memory _message = abi.encodeWithSelector(
      IL1ERC1155Gateway.finalizeWithdrawERC1155.selector,
      _l1Token,
      _token,
      msg.sender,
      _to,
      _tokenId,
      _amount
    );

    // 3. Send message to L2ScrollMessenger.
    IL2ScrollMessenger(messenger).sendMessage(counterpart, msg.value, _message, _gasLimit);

    emit WithdrawERC1155(_l1Token, _token, msg.sender, _to, _tokenId, _amount);
  }

  /// @dev Internal function to batch withdraw ERC1155 NFT to layer 2.
  /// @param _token The address of ERC1155 NFT in layer 1.
  /// @param _to The address of recipient in layer 2.
  /// @param _tokenIds The list of token ids to withdraw.
  /// @param _amounts The list of corresponding number of token to withdraw.
  /// @param _gasLimit Estimated gas limit required to complete the withdraw on layer 1.
  function _batchWithdrawERC1155(
    address _token,
    address _to,
    uint256[] calldata _tokenIds,
    uint256[] calldata _amounts,
    uint256 _gasLimit
  ) internal nonReentrant {
    require(_tokenIds.length > 0, "no token to withdraw");
    require(_tokenIds.length == _amounts.length, "length mismatch");

    for (uint256 i = 0; i < _amounts.length; i++) {
      require(_amounts[i] > 0, "withdraw zero amount");
    }

    address _l1Token = tokenMapping[_token];
    require(_l1Token != address(0), "token not supported");

    // 1. transfer token to this contract
    IScrollERC1155(_token).batchBurn(msg.sender, _tokenIds, _amounts);

    // 2. Generate message passed to L1ERC1155Gateway.
    bytes memory _message = abi.encodeWithSelector(
      IL1ERC1155Gateway.finalizeBatchWithdrawERC1155.selector,
      _l1Token,
      _token,
      msg.sender,
      _to,
      _tokenIds,
      _amounts
    );

    // 3. Send message to L2ScrollMessenger.
    IL2ScrollMessenger(messenger).sendMessage{ value: msg.value }(counterpart, msg.value, _message, _gasLimit);

    emit BatchWithdrawERC1155(_l1Token, _token, msg.sender, _to, _tokenIds, _amounts);
  }
}
