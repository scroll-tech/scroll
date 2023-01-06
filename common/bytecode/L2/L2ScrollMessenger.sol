// File: src/libraries/IScrollMessenger.sol

// SPDX-License-Identifier: MIT

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

// File: src/L2/predeploys/L2ToL1MessagePasser.sol



pragma solidity ^0.8.0;

/// @title L2ToL1MessagePasser
/// @notice The original idea is from Optimism, see [OVM_L2ToL1MessagePasser](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts/contracts/L2/predeploys/OVM_L2ToL1MessagePasser.sol).
/// The L2 to L1 Message Passer is a utility contract which facilitate an L1 proof of the
/// of a message on L2. The L1 Cross Domain Messenger performs this proof in its
/// _verifyStorageProof function, which verifies the existence of the transaction hash in this
/// contract's `sentMessages` mapping.
contract L2ToL1MessagePasser {
  address public immutable messenger;

  /// @notice Mapping from message hash to sent messages.
  mapping(bytes32 => bool) public sentMessages;

  constructor(address _messenger) {
    messenger = _messenger;
  }

  function passMessageToL1(bytes32 _messageHash) public {
    require(msg.sender == messenger, "only messenger");

    require(!sentMessages[_messageHash], "duplicated message");

    sentMessages[_messageHash] = true;
  }
}

// File: src/libraries/common/OwnableBase.sol



pragma solidity ^0.8.0;

abstract contract OwnableBase {
  /**************************************** Events ****************************************/

  /// @notice Emitted when owner is changed by current owner.
  /// @param _oldOwner The address of previous owner.
  /// @param _newOwner The address of new owner.
  event OwnershipTransferred(address indexed _oldOwner, address indexed _newOwner);

  /**************************************** Variables ****************************************/

  /// @notice The address of the current owner.
  address public owner;

  /// @dev Throws if called by any account other than the owner.
  modifier onlyOwner() {
    require(owner == msg.sender, "caller is not the owner");
    _;
  }

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

  /// @dev Transfers ownership of the contract to a new account (`newOwner`).
  /// Internal function without access restriction.
  function _transferOwnership(address _newOwner) internal {
    address _oldOwner = owner;
    owner = _newOwner;
    emit OwnershipTransferred(_oldOwner, _newOwner);
  }
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
contract L2ScrollMessenger is ScrollMessengerBase, OwnableBase, IL2ScrollMessenger {
  /**************************************** Variables ****************************************/

  /// @notice Mapping from relay id to relay status.
  mapping(bytes32 => bool) public isMessageRelayed;

  /// @notice Mapping from message hash to execution status.
  mapping(bytes32 => bool) public isMessageExecuted;

  /// @notice Message nonce, used to avoid relay attack.
  uint256 public messageNonce;

  /// @notice Contract to store the sent message.
  L2ToL1MessagePasser public messagePasser;

  constructor(address _owner) {
    ScrollMessengerBase._initialize();
    owner = _owner;

    // initialize to a nonzero value
    xDomainMessageSender = ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER;

    messagePasser = new L2ToL1MessagePasser(address(this));
  }

  /**************************************** Mutated Functions ****************************************/

  /// @inheritdoc IScrollMessenger
  function sendMessage(
    address _to,
    uint256 _fee,
    bytes memory _message,
    uint256 _gasLimit
  ) external payable override onlyWhitelistedSender(msg.sender) {
    require(msg.value >= _fee, "cannot pay fee");

    // solhint-disable-next-line not-rely-on-time
    uint256 _deadline = block.timestamp + dropDelayDuration;
    // compute fee by GasOracle contract.
    uint256 _minFee = gasOracle == address(0) ? 0 : IGasOracle(gasOracle).estimateMessageFee(msg.sender, _to, _message);
    require(_fee >= _minFee, "fee too small");

    uint256 _nonce = messageNonce;
    uint256 _value;
    unchecked {
      _value = msg.value - _fee;
    }

    bytes32 _msghash = keccak256(abi.encodePacked(msg.sender, _to, _value, _fee, _deadline, _nonce, _message));

    messagePasser.passMessageToL1(_msghash);

    emit SentMessage(_to, msg.sender, _value, _fee, _deadline, _message, _nonce, _gasLimit);

    unchecked {
      messageNonce = _nonce + 1;
    }
  }

  /// @inheritdoc IL2ScrollMessenger
  function relayMessage(
    address _from,
    address _to,
    uint256 _value,
    uint256 _fee,
    uint256 _deadline,
    uint256 _nonce,
    bytes memory _message
  ) external override {
    // anti reentrance
    require(xDomainMessageSender == ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER, "already in execution");

    // @todo only privileged accounts can call

    // solhint-disable-next-line not-rely-on-time
    require(_deadline >= block.timestamp, "Message expired");

    bytes32 _msghash = keccak256(abi.encodePacked(_from, _to, _value, _fee, _deadline, _nonce, _message));

    require(!isMessageExecuted[_msghash], "Message successfully executed");

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

  /// @inheritdoc IScrollMessenger
  function dropMessage(
    address,
    address,
    uint256,
    uint256,
    uint256,
    uint256,
    bytes memory,
    uint256
  ) external virtual override {
    revert("not supported");
  }

  /**************************************** Restricted Functions ****************************************/

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
