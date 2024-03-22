// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {IL2ScrollMessenger} from "./IL2ScrollMessenger.sol";
import {L2MessageQueue} from "./predeploys/L2MessageQueue.sol";

import {PatriciaMerkleTrieVerifier} from "../libraries/verifier/PatriciaMerkleTrieVerifier.sol";
import {ScrollConstants} from "../libraries/constants/ScrollConstants.sol";
import {AddressAliasHelper} from "../libraries/common/AddressAliasHelper.sol";
import {IScrollMessenger} from "../libraries/IScrollMessenger.sol";
import {ScrollMessengerBase} from "../libraries/ScrollMessengerBase.sol";

// solhint-disable reason-string
// solhint-disable not-rely-on-time

/// @title L2ScrollMessenger
/// @notice The `L2ScrollMessenger` contract can:
///
/// 1. send messages from layer 2 to layer 1;
/// 2. relay messages from layer 1 layer 2;
/// 3. drop expired message due to sequencer problems.
///
/// @dev It should be a predeployed contract on layer 2 and should hold infinite amount
/// of Ether (Specifically, `uint256(-1)`), which can be initialized in Genesis Block.
contract L2ScrollMessenger is ScrollMessengerBase, IL2ScrollMessenger {
    /*************
     * Constants *
     *************/

    /// @notice The address of L2MessageQueue.
    address public immutable messageQueue;

    /*************
     * Variables *
     *************/

    /// @notice Mapping from L2 message hash to the timestamp when the message is sent.
    mapping(bytes32 => uint256) public messageSendTimestamp;

    /// @notice Mapping from L1 message hash to a boolean value indicating if the message has been successfully executed.
    mapping(bytes32 => bool) public isL1MessageExecuted;

    /// @dev The storage slots used by previous versions of this contract.
    uint256[2] private __used;

    /***************
     * Constructor *
     ***************/

    constructor(address _counterpart, address _messageQueue) ScrollMessengerBase(_counterpart) {
        if (_messageQueue == address(0)) {
            revert ErrorZeroAddress();
        }

        _disableInitializers();

        messageQueue = _messageQueue;
    }

    function initialize(address) external initializer {
        ScrollMessengerBase.__ScrollMessengerBase_init(address(0), address(0));
    }

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @inheritdoc IScrollMessenger
    function sendMessage(
        address _to,
        uint256 _value,
        bytes memory _message,
        uint256 _gasLimit
    ) external payable override whenNotPaused {
        _sendMessage(_to, _value, _message, _gasLimit);
    }

    /// @inheritdoc IScrollMessenger
    function sendMessage(
        address _to,
        uint256 _value,
        bytes calldata _message,
        uint256 _gasLimit,
        address
    ) external payable override whenNotPaused {
        _sendMessage(_to, _value, _message, _gasLimit);
    }

    /// @inheritdoc IL2ScrollMessenger
    function relayMessage(
        address _from,
        address _to,
        uint256 _value,
        uint256 _nonce,
        bytes memory _message
    ) external override whenNotPaused {
        // It is impossible to deploy a contract with the same address, reentrance is prevented in nature.
        require(AddressAliasHelper.undoL1ToL2Alias(_msgSender()) == counterpart, "Caller is not L1ScrollMessenger");

        bytes32 _xDomainCalldataHash = keccak256(_encodeXDomainCalldata(_from, _to, _value, _nonce, _message));

        require(!isL1MessageExecuted[_xDomainCalldataHash], "Message was already successfully executed");

        _executeMessage(_from, _to, _value, _message, _xDomainCalldataHash);
    }

    /**********************
     * Internal Functions *
     **********************/

    /// @dev Internal function to send cross domain message.
    /// @param _to The address of account who receive the message.
    /// @param _value The amount of ether passed when call target contract.
    /// @param _message The content of the message.
    /// @param _gasLimit Optional gas limit to complete the message relay on corresponding chain.
    function _sendMessage(
        address _to,
        uint256 _value,
        bytes memory _message,
        uint256 _gasLimit
    ) internal nonReentrant {
        require(msg.value == _value, "msg.value mismatch");

        uint256 _nonce = L2MessageQueue(messageQueue).nextMessageIndex();
        bytes32 _xDomainCalldataHash = keccak256(_encodeXDomainCalldata(_msgSender(), _to, _value, _nonce, _message));

        // normally this won't happen, since each message has different nonce, but just in case.
        require(messageSendTimestamp[_xDomainCalldataHash] == 0, "Duplicated message");
        messageSendTimestamp[_xDomainCalldataHash] = block.timestamp;

        L2MessageQueue(messageQueue).appendMessage(_xDomainCalldataHash);

        emit SentMessage(_msgSender(), _to, _value, _nonce, _gasLimit, _message);
    }

    /// @dev Internal function to execute a L1 => L2 message.
    /// @param _from The address of the sender of the message.
    /// @param _to The address of the recipient of the message.
    /// @param _value The msg.value passed to the message call.
    /// @param _message The content of the message.
    /// @param _xDomainCalldataHash The hash of the message.
    function _executeMessage(
        address _from,
        address _to,
        uint256 _value,
        bytes memory _message,
        bytes32 _xDomainCalldataHash
    ) internal {
        // @note check more `_to` address to avoid attack in the future when we add more gateways.
        require(_to != messageQueue, "Forbid to call message queue");
        _validateTargetAddress(_to);

        // @note This usually will never happen, just in case.
        require(_from != xDomainMessageSender, "Invalid message sender");

        xDomainMessageSender = _from;
        // solhint-disable-next-line avoid-low-level-calls
        (bool success, ) = _to.call{value: _value}(_message);
        // reset value to refund gas.
        xDomainMessageSender = ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER;

        if (success) {
            isL1MessageExecuted[_xDomainCalldataHash] = true;
            emit RelayedMessage(_xDomainCalldataHash);
        } else {
            emit FailedRelayedMessage(_xDomainCalldataHash);
        }
    }
}
