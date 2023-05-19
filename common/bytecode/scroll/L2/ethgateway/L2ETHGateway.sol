// File: @openzeppelin/contracts/utils/Address.sol

// SPDX-License-Identifier: MIT
// OpenZeppelin Contracts (last updated v4.5.0) (utils/Address.sol)

pragma solidity ^0.8.1;

/**
 * @dev Collection of functions related to the address type
 */
library Address {
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
     * @dev Same as {xref-Address-functionCall-address-bytes-}[`functionCall`],
     * but performing a delegate call.
     *
     * _Available since v3.4._
     */
    function functionDelegateCall(address target, bytes memory data) internal returns (bytes memory) {
        return functionDelegateCall(target, data, "Address: low-level delegate call failed");
    }

    /**
     * @dev Same as {xref-Address-functionCall-address-bytes-string-}[`functionCall`],
     * but performing a delegate call.
     *
     * _Available since v3.4._
     */
    function functionDelegateCall(
        address target,
        bytes memory data,
        string memory errorMessage
    ) internal returns (bytes memory) {
        require(isContract(target), "Address: delegate call to non-contract");

        (bool success, bytes memory returndata) = target.delegatecall(data);
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

// File: @openzeppelin/contracts/proxy/utils/Initializable.sol


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
        return !Address.isContract(address(this));
    }
}

// File: src/L1/gateways/IL1ETHGateway.sol



pragma solidity ^0.8.0;

interface IL1ETHGateway {
  /**********
   * Events *
   **********/

  /// @notice Emitted when ETH is withdrawn from L2 to L1 and transfer to recipient.
  /// @param from The address of sender in L2.
  /// @param to The address of recipient in L1.
  /// @param amount The amount of ETH withdrawn from L2 to L1.
  /// @param data The optional calldata passed to recipient in L1.
  event FinalizeWithdrawETH(address indexed from, address indexed to, uint256 amount, bytes data);

  /// @notice Emitted when someone deposit ETH from L1 to L2.
  /// @param from The address of sender in L1.
  /// @param to The address of recipient in L2.
  /// @param amount The amount of ETH will be deposited from L1 to L2.
  /// @param data The optional calldata passed to recipient in L2.
  event DepositETH(address indexed from, address indexed to, uint256 amount, bytes data);

  /****************************
   * Public Mutated Functions *
   ****************************/

  /// @notice Deposit ETH to caller's account in L2.
  /// @param amount The amount of ETH to be deposited.
  /// @param gasLimit Gas limit required to complete the deposit on L2.
  function depositETH(uint256 amount, uint256 gasLimit) external payable;

  /// @notice Deposit ETH to some recipient's account in L2.
  /// @param to The address of recipient's account on L2.
  /// @param amount The amount of ETH to be deposited.
  /// @param gasLimit Gas limit required to complete the deposit on L2.
  function depositETH(
    address to,
    uint256 amount,
    uint256 gasLimit
  ) external payable;

  /// @notice Deposit ETH to some recipient's account in L2 and call the target contract.
  /// @param to The address of recipient's account on L2.
  /// @param amount The amount of ETH to be deposited.
  /// @param data Optional data to forward to recipient's account.
  /// @param gasLimit Gas limit required to complete the deposit on L2.
  function depositETHAndCall(
    address to,
    uint256 amount,
    bytes calldata data,
    uint256 gasLimit
  ) external payable;

  /// @notice Complete ETH withdraw from L2 to L1 and send fund to recipient's account in L1.
  /// @dev This function should only be called by L1ScrollMessenger.
  ///      This function should also only be called by L1ETHGateway in L2.
  /// @param from The address of account who withdraw ETH in L2.
  /// @param to The address of recipient in L1 to receive ETH.
  /// @param amount The amount of ETH to withdraw.
  /// @param data Optional data to forward to recipient's account.
  function finalizeWithdrawETH(
    address from,
    address to,
    uint256 amount,
    bytes calldata data
  ) external payable;
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

// File: src/L2/gateways/IL2ETHGateway.sol



pragma solidity ^0.8.0;

interface IL2ETHGateway {
  /**********
   * Events *
   **********/

  /// @notice Emitted when someone withdraw ETH from L2 to L1.
  /// @param from The address of sender in L2.
  /// @param to The address of recipient in L1.
  /// @param amount The amount of ETH will be deposited from L2 to L1.
  /// @param data The optional calldata passed to recipient in L1.
  event WithdrawETH(address indexed from, address indexed to, uint256 amount, bytes data);

  /// @notice Emitted when ETH is deposited from L1 to L2 and transfer to recipient.
  /// @param from The address of sender in L1.
  /// @param to The address of recipient in L2.
  /// @param amount The amount of ETH deposited from L1 to L2.
  /// @param data The optional calldata passed to recipient in L2.
  event FinalizeDepositETH(address indexed from, address indexed to, uint256 amount, bytes data);

  /****************************
   * Public Mutated Functions *
   ****************************/

  /// @notice Withdraw ETH to caller's account in L1.
  /// @param amount The amount of ETH to be withdrawn.
  /// @param gasLimit Optional, gas limit used to complete the withdraw on L1.
  function withdrawETH(uint256 amount, uint256 gasLimit) external payable;

  /// @notice Withdraw ETH to caller's account in L1.
  /// @param to The address of recipient's account on L1.
  /// @param amount The amount of ETH to be withdrawn.
  /// @param gasLimit Optional, gas limit used to complete the withdraw on L1.
  function withdrawETH(
    address to,
    uint256 amount,
    uint256 gasLimit
  ) external payable;

  /// @notice Withdraw ETH to caller's account in L1.
  /// @param to The address of recipient's account on L1.
  /// @param amount The amount of ETH to be withdrawn.
  /// @param data Optional data to forward to recipient's account.
  /// @param gasLimit Optional, gas limit used to complete the withdraw on L1.
  function withdrawETHAndCall(
    address to,
    uint256 amount,
    bytes calldata data,
    uint256 gasLimit
  ) external payable;

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

// File: src/libraries/gateway/IScrollGateway.sol



pragma solidity ^0.8.0;

interface IScrollGateway {
  /// @notice The address of corresponding L1/L2 Gateway contract.
  function counterpart() external view returns (address);

  /// @notice The address of L1GatewayRouter/L2GatewayRouter contract.
  function router() external view returns (address);

  /// @notice The address of corresponding L1ScrollMessenger/L2ScrollMessenger contract.
  function messenger() external view returns (address);
}

// File: src/libraries/gateway/ScrollGatewayBase.sol



pragma solidity ^0.8.0;


abstract contract ScrollGatewayBase is IScrollGateway {
  /*************
   * Constants *
   *************/

  // https://github.com/OpenZeppelin/openzeppelin-contracts/blob/v4.5.0/contracts/security/ReentrancyGuard.sol
  uint256 private constant _NOT_ENTERED = 1;
  uint256 private constant _ENTERED = 2;

  /*************
   * Variables *
   *************/

  /// @inheritdoc IScrollGateway
  address public override counterpart;

  /// @inheritdoc IScrollGateway
  address public override router;

  /// @inheritdoc IScrollGateway
  address public override messenger;

  /// @dev The status of for non-reentrant check.
  uint256 private _status;

  /**********************
   * Function Modifiers *
   **********************/

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

  /***************
   * Constructor *
   ***************/

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

// File: src/L2/gateways/L2ETHGateway.sol



pragma solidity ^0.8.0;



/// @title L2ETHGateway
/// @notice The `L2ETHGateway` contract is used to withdraw ETH token in layer 2 and
/// finalize deposit ETH from layer 1.
/// @dev The ETH are not held in the gateway. The ETH will be sent to the `L2ScrollMessenger` contract.
/// On finalizing deposit, the Ether will be transfered from `L2ScrollMessenger`, then transfer to recipient.
contract L2ETHGateway is Initializable, ScrollGatewayBase, IL2ETHGateway {
  /***************
   * Constructor *
   ***************/

  /// @notice Initialize the storage of L2ETHGateway.
  /// @param _counterpart The address of L1ETHGateway in L2.
  /// @param _router The address of L2GatewayRouter.
  /// @param _messenger The address of L2ScrollMessenger.
  function initialize(
    address _counterpart,
    address _router,
    address _messenger
  ) external initializer {
    require(_router != address(0), "zero router address");
    ScrollGatewayBase._initialize(_counterpart, _router, _messenger);
  }

  /****************************
   * Public Mutated Functions *
   ****************************/

  /// @inheritdoc IL2ETHGateway
  function withdrawETH(uint256 _amount, uint256 _gasLimit) external payable override {
    _withdraw(msg.sender, _amount, new bytes(0), _gasLimit);
  }

  /// @inheritdoc IL2ETHGateway
  function withdrawETH(
    address _to,
    uint256 _amount,
    uint256 _gasLimit
  ) public payable override {
    _withdraw(_to, _amount, new bytes(0), _gasLimit);
  }

  /// @inheritdoc IL2ETHGateway
  function withdrawETHAndCall(
    address _to,
    uint256 _amount,
    bytes memory _data,
    uint256 _gasLimit
  ) public payable override {
    _withdraw(_to, _amount, _data, _gasLimit);
  }

  /// @inheritdoc IL2ETHGateway
  function finalizeDepositETH(
    address _from,
    address _to,
    uint256 _amount,
    bytes calldata _data
  ) external payable override onlyCallByCounterpart {
    // solhint-disable-next-line avoid-low-level-calls
    (bool _success, ) = _to.call{ value: _amount }("");
    require(_success, "ETH transfer failed");

    // @todo farward _data to `_to` in near future.

    emit FinalizeDepositETH(_from, _to, _amount, _data);
  }

  /**********************
   * Internal Functions *
   **********************/

  function _withdraw(
    address _to,
    uint256 _amount,
    bytes memory _data,
    uint256 _gasLimit
  ) internal nonReentrant {
    require(msg.value > 0, "withdraw zero eth");

    // 1. Extract real sender if this call is from L1GatewayRouter.
    address _from = msg.sender;
    if (router == msg.sender) {
      (_from, _data) = abi.decode(_data, (address, bytes));
    }

    bytes memory _message = abi.encodeWithSelector(
      IL1ETHGateway.finalizeWithdrawETH.selector,
      _from,
      _to,
      _amount,
      _data
    );
    IL2ScrollMessenger(messenger).sendMessage{ value: msg.value }(counterpart, _amount, _message, _gasLimit);

    emit WithdrawETH(_from, _to, _amount, _data);
  }
}
