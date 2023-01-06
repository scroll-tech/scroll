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

// File: src/L1/gateways/IL1ERC20Gateway.sol



pragma solidity ^0.8.0;

interface IL1ERC20Gateway {
  /**************************************** Events ****************************************/

  event FinalizeWithdrawERC20(
    address indexed _l1Token,
    address indexed _l2Token,
    address indexed _from,
    address _to,
    uint256 _amount,
    bytes _data
  );

  event DepositERC20(
    address indexed _l1Token,
    address indexed _l2Token,
    address indexed _from,
    address _to,
    uint256 _amount,
    bytes _data
  );

  /**************************************** View Functions ****************************************/

  /// @notice Return the corresponding l2 token address given l1 token address.
  /// @param _l1Token The address of l1 token.
  function getL2ERC20Address(address _l1Token) external view returns (address);

  /**************************************** Mutated Functions ****************************************/

  /// @notice Deposit some token to a caller's account on L2.
  /// @dev Make this function payable to send relayer fee in Ether.
  /// @param _token The address of token in L1.
  /// @param _amount The amount of token to transfer.
  /// @param _gasLimit Gas limit required to complete the deposit on L2.
  function depositERC20(
    address _token,
    uint256 _amount,
    uint256 _gasLimit
  ) external payable;

  /// @notice Deposit some token to a recipient's account on L2.
  /// @dev Make this function payable to send relayer fee in Ether.
  /// @param _token The address of token in L1.
  /// @param _to The address of recipient's account on L2.
  /// @param _amount The amount of token to transfer.
  /// @param _gasLimit Gas limit required to complete the deposit on L2.
  function depositERC20(
    address _token,
    address _to,
    uint256 _amount,
    uint256 _gasLimit
  ) external payable;

  /// @notice Deposit some token to a recipient's account on L2 and call.
  /// @dev Make this function payable to send relayer fee in Ether.
  /// @param _token The address of token in L1.
  /// @param _to The address of recipient's account on L2.
  /// @param _amount The amount of token to transfer.
  /// @param _data Optional data to forward to recipient's account.
  /// @param _gasLimit Gas limit required to complete the deposit on L2.
  function depositERC20AndCall(
    address _token,
    address _to,
    uint256 _amount,
    bytes memory _data,
    uint256 _gasLimit
  ) external payable;

  /// @notice Complete ERC20 withdraw from L2 to L1 and send fund to recipient's account in L1.
  /// @dev Make this function payable to handle WETH deposit/withdraw.
  ///      The function should only be called by L1ScrollMessenger.
  ///      The function should also only be called by L2ERC20Gateway in L2.
  /// @param _l1Token The address of corresponding L1 token.
  /// @param _l2Token The address of corresponding L2 token.
  /// @param _from The address of account who withdraw the token in L2.
  /// @param _to The address of recipient in L1 to receive the token.
  /// @param _amount The amount of the token to withdraw.
  /// @param _data Optional data to forward to recipient's account.
  function finalizeWithdrawERC20(
    address _l1Token,
    address _l2Token,
    address _from,
    address _to,
    uint256 _amount,
    bytes calldata _data
  ) external payable;
}

// File: src/libraries/gateway/IScrollGateway.sol



pragma solidity ^0.8.0;

interface IScrollGateway {
  function counterpart() external view returns (address);

  function finalizeDropMessage() external payable;
}

// File: src/L1/gateways/IL1GatewayRouter.sol



pragma solidity ^0.8.0;

interface IL1GatewayRouter is IL1ERC20Gateway, IScrollGateway {
  /**************************************** Events ****************************************/

  event FinalizeWithdrawETH(address indexed _from, address indexed _to, uint256 _amount, bytes _data);
  event DepositETH(address indexed _from, address indexed _to, uint256 _amount, bytes _data);

  /**************************************** Mutated Functions ****************************************/

  /// @notice Deposit ETH to call's account in L2.
  /// @param _gasLimit Gas limit required to complete the deposit on L2.
  function depositETH(uint256 _gasLimit) external payable;

  /// @notice Deposit ETH to recipient's account in L2.
  /// @param _to The address of recipient's account on L2.
  /// @param _gasLimit Gas limit required to complete the deposit on L2.
  function depositETH(address _to, uint256 _gasLimit) external payable;

  // @todo add depositETHAndCall;

  /// @notice Complete ETH withdraw from L2 to L1 and send fund to recipient's account in L1.
  /// @dev This function should only be called by L1ScrollMessenger.
  ///      This function should also only be called by L2GatewayRouter in L2.
  /// @param _from The address of account who withdraw ETH in L2.
  /// @param _to The address of recipient in L1 to receive ETH.
  /// @param _amount The amount of ETH to withdraw.
  /// @param _data Optional data to forward to recipient's account.
  function finalizeWithdrawETH(
    address _from,
    address _to,
    uint256 _amount,
    bytes calldata _data
  ) external payable;
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

// File: src/L2/gateways/IL2ERC20Gateway.sol



pragma solidity ^0.8.0;

interface IL2ERC20Gateway {
  /**************************************** Events ****************************************/

  event WithdrawERC20(
    address indexed _l1Token,
    address indexed _l2Token,
    address indexed _from,
    address _to,
    uint256 _amount,
    bytes _data
  );

  event FinalizeDepositERC20(
    address indexed _l1Token,
    address indexed _l2Token,
    address indexed _from,
    address _to,
    uint256 _amount,
    bytes _data
  );

  /**************************************** View Functions ****************************************/

  /// @notice Return the corresponding l1 token address given l2 token address.
  /// @param _l2Token The address of l2 token.
  function getL1ERC20Address(address _l2Token) external view returns (address);

  /// @notice Return the corresponding l2 token address given l1 token address.
  /// @param _l1Token The address of l1 token.
  function getL2ERC20Address(address _l1Token) external view returns (address);

  /**************************************** Mutated Functions ****************************************/

  /// @notice Withdraw of some token to a caller's account on L1.
  /// @dev Make this function payable to send relayer fee in Ether.
  /// @param _token The address of token in L2.
  /// @param _amount The amount of token to transfer.
  /// @param _gasLimit Unused, but included for potential forward compatibility considerations.
  function withdrawERC20(
    address _token,
    uint256 _amount,
    uint256 _gasLimit
  ) external payable;

  /// @notice Withdraw of some token to a recipient's account on L1.
  /// @dev Make this function payable to send relayer fee in Ether.
  /// @param _token The address of token in L2.
  /// @param _to The address of recipient's account on L1.
  /// @param _amount The amount of token to transfer.
  /// @param _gasLimit Unused, but included for potential forward compatibility considerations.
  function withdrawERC20(
    address _token,
    address _to,
    uint256 _amount,
    uint256 _gasLimit
  ) external payable;

  /// @notice Withdraw of some token to a recipient's account on L1 and call.
  /// @dev Make this function payable to send relayer fee in Ether.
  /// @param _token The address of token in L2.
  /// @param _to The address of recipient's account on L1.
  /// @param _amount The amount of token to transfer.
  /// @param _data Optional data to forward to recipient's account.
  /// @param _gasLimit Unused, but included for potential forward compatibility considerations.
  function withdrawERC20AndCall(
    address _token,
    address _to,
    uint256 _amount,
    bytes calldata _data,
    uint256 _gasLimit
  ) external payable;

  /// @notice Complete a deposit from L1 to L2 and send fund to recipient's account in L2.
  /// @dev Make this function payable to handle WETH deposit/withdraw.
  ///      The function should only be called by L2ScrollMessenger.
  ///      The function should also only be called by L1ERC20Gateway in L1.
  /// @param _l1Token The address of corresponding L1 token.
  /// @param _l2Token The address of corresponding L2 token.
  /// @param _from The address of account who deposits the token in L1.
  /// @param _to The address of recipient in L2 to receive the token.
  /// @param _amount The amount of the token to deposit.
  /// @param _data Optional data to forward to recipient's account.
  function finalizeDepositERC20(
    address _l1Token,
    address _l2Token,
    address _from,
    address _to,
    uint256 _amount,
    bytes calldata _data
  ) external payable;
}

// File: src/L2/gateways/IL2GatewayRouter.sol



pragma solidity ^0.8.0;

interface IL2GatewayRouter is IL2ERC20Gateway, IScrollGateway {
  /**************************************** Events ****************************************/

  event WithdrawETH(address indexed _from, address indexed _to, uint256 _amount, bytes _data);
  event FinalizeDepositETH(address indexed _from, address indexed _to, uint256 _amount, bytes _data);

  /**************************************** Mutated Functions ****************************************/
  /// @notice Withdraw ETH to caller's account in L1.
  /// @param _gasLimit Gas limit required to complete the withdraw on L1.
  function withdrawETH(uint256 _gasLimit) external payable;

  /// @notice Withdraw ETH to caller's account in L1.
  /// @param _to The address of recipient's account on L1.
  /// @param _gasLimit Gas limit required to complete the withdraw on L1.
  function withdrawETH(address _to, uint256 _gasLimit) external payable;

  // @todo add withdrawETHAndCall;

  /// @notice Complete ETH deposit from L1 to L2 and send fund to recipient's account in L2.
  /// @dev This function should only be called by L2ScrollMessenger.
  ///      This function should also only be called by L1GatewayRouter in L1.
  /// @param _from The address of account who deposit ETH in L1.
  /// @param _to The address of recipient in L2 to receive ETH.
  /// @param _amount The amount of ETH to deposit.
  /// @param _data Optional data to forward to recipient's account.
  function finalizeDepositETH(
    address _from,
    address _to,
    uint256 _amount,
    bytes calldata _data
  ) external payable;
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

// File: src/L1/gateways/L1GatewayRouter.sol



pragma solidity ^0.8.0;






/// @title L1GatewayRouter
/// @notice The `L1GatewayRouter` is the main entry for depositing Ether and ERC20 tokens.
/// All deposited tokens are routed to corresponding gateways.
/// @dev One can also use this contract to query L1/L2 token address mapping.
/// In the future, ERC-721 and ERC-1155 tokens will be added to the router too.
contract L1GatewayRouter is OwnableUpgradeable, ScrollGatewayBase, IL1GatewayRouter {
  /**************************************** Events ****************************************/

  event SetDefaultERC20Gateway(address indexed _defaultERC20Gateway);
  event SetERC20Gateway(address indexed _token, address indexed _gateway);

  /**************************************** Variables ****************************************/

  /// @notice The addess of default ERC20 gateway, normally the L1StandardERC20Gateway contract.
  address public defaultERC20Gateway;
  /// @notice Mapping from ERC20 token address to corresponding L1ERC20Gateway.
  // solhint-disable-next-line var-name-mixedcase
  mapping(address => address) public ERC20Gateway;

  // @todo: add ERC721/ERC1155 Gateway mapping.

  /**************************************** Constructor ****************************************/

  function initialize(
    address _defaultERC20Gateway,
    address _counterpart,
    address _messenger
  ) external initializer {
    OwnableUpgradeable.__Ownable_init();
    ScrollGatewayBase._initialize(_counterpart, address(0), _messenger);

    // it can be zero during initialization
    if (_defaultERC20Gateway != address(0)) {
      defaultERC20Gateway = _defaultERC20Gateway;
    }
  }

  /**************************************** View Functions ****************************************/

  /// @inheritdoc IL1ERC20Gateway
  function getL2ERC20Address(address _l1Address) external view override returns (address) {
    address _gateway = getERC20Gateway(_l1Address);
    if (_gateway == address(0)) {
      return address(0);
    }

    return IL1ERC20Gateway(_gateway).getL2ERC20Address(_l1Address);
  }

  /// @notice Return the corresponding gateway address for given token address.
  /// @param _token The address of token to query.
  function getERC20Gateway(address _token) public view returns (address) {
    address _gateway = ERC20Gateway[_token];
    if (_gateway == address(0)) {
      _gateway = defaultERC20Gateway;
    }
    return _gateway;
  }

  /**************************************** Mutate Functions ****************************************/

  /// @inheritdoc IL1ERC20Gateway
  function depositERC20(
    address _token,
    uint256 _amount,
    uint256 _gasLimit
  ) external payable override {
    depositERC20AndCall(_token, msg.sender, _amount, new bytes(0), _gasLimit);
  }

  /// @inheritdoc IL1ERC20Gateway
  function depositERC20(
    address _token,
    address _to,
    uint256 _amount,
    uint256 _gasLimit
  ) external payable override {
    depositERC20AndCall(_token, _to, _amount, new bytes(0), _gasLimit);
  }

  /// @inheritdoc IL1ERC20Gateway
  function depositERC20AndCall(
    address _token,
    address _to,
    uint256 _amount,
    bytes memory _data,
    uint256 _gasLimit
  ) public payable override nonReentrant {
    address _gateway = getERC20Gateway(_token);
    require(_gateway != address(0), "no gateway available");

    // encode msg.sender with _data
    bytes memory _routerData = abi.encode(msg.sender, _data);

    IL1ERC20Gateway(_gateway).depositERC20AndCall{ value: msg.value }(_token, _to, _amount, _routerData, _gasLimit);
  }

  /// @inheritdoc IL1GatewayRouter
  function depositETH(uint256 _gasLimit) external payable override {
    depositETH(msg.sender, _gasLimit);
  }

  /// @inheritdoc IL1GatewayRouter
  function depositETH(address _to, uint256 _gasLimit) public payable override nonReentrant {
    require(msg.value > 0, "deposit zero eth");

    bytes memory _message = abi.encodeWithSelector(
      IL2GatewayRouter.finalizeDepositETH.selector,
      msg.sender,
      _to,
      msg.value,
      new bytes(0)
    );
    IL1ScrollMessenger(messenger).sendMessage{ value: msg.value }(counterpart, 0, _message, _gasLimit);

    emit DepositETH(msg.sender, _to, msg.value, "");
  }

  /// @inheritdoc IL1GatewayRouter
  function finalizeWithdrawETH(
    address _from,
    address _to,
    uint256 _amount,
    bytes calldata _data
  ) external payable override onlyCallByCounterpart {
    require(msg.value == _amount, "msg.value mismatch");

    // @note can possible trigger reentrant call to this contract or messenger,
    // but it seems not a big problem.
    // solhint-disable-next-line avoid-low-level-calls
    (bool _success, ) = _to.call{ value: _amount }("");
    require(_success, "ETH transfer failed");

    // @todo farward _data to `_to` in near future.

    emit FinalizeWithdrawETH(_from, _to, _amount, _data);
  }

  /// @inheritdoc IL1ERC20Gateway
  function finalizeWithdrawERC20(
    address,
    address,
    address,
    address,
    uint256,
    bytes calldata
  ) external payable virtual override {
    revert("should never be called");
  }

  /// @inheritdoc IScrollGateway
  function finalizeDropMessage() external payable virtual override onlyMessenger {
    // @todo should refund ETH back to sender.
  }

  /**************************************** Restricted Functions ****************************************/

  /// @notice Update the address of default ERC20 gateway contract.
  /// @dev This function should only be called by contract owner.
  /// @param _defaultERC20Gateway The address to update.
  function setDefaultERC20Gateway(address _defaultERC20Gateway) external onlyOwner {
    defaultERC20Gateway = _defaultERC20Gateway;

    emit SetDefaultERC20Gateway(_defaultERC20Gateway);
  }

  /// @notice Update the mapping from token address to gateway address.
  /// @dev This function should only be called by contract owner.
  /// @param _tokens The list of addresses of tokens to update.
  /// @param _gateways The list of addresses of gateways to update.
  function setERC20Gateway(address[] memory _tokens, address[] memory _gateways) external onlyOwner {
    require(_tokens.length == _gateways.length, "length mismatch");

    for (uint256 i = 0; i < _tokens.length; i++) {
      ERC20Gateway[_tokens[i]] = _gateways[i];

      emit SetERC20Gateway(_tokens[i], _gateways[i]);
    }
  }
}
