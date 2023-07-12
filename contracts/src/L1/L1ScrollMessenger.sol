// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {PausableUpgradeable} from "@openzeppelin/contracts-upgradeable/security/PausableUpgradeable.sol";

import {IScrollChain} from "./rollup/IScrollChain.sol";
import {IL1MessageQueue} from "./rollup/IL1MessageQueue.sol";
import {IL1ScrollMessenger} from "./IL1ScrollMessenger.sol";
import {ScrollConstants} from "../libraries/constants/ScrollConstants.sol";
import {IScrollMessenger} from "../libraries/IScrollMessenger.sol";
import {ScrollMessengerBase} from "../libraries/ScrollMessengerBase.sol";
import {WithdrawTrieVerifier} from "../libraries/verifier/WithdrawTrieVerifier.sol";

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

    // @note move to ScrollMessengerBase in next big refactor
    /// @dev The status of for non-reentrant check.
    uint256 private _lock_status;

    /**********************
     * Function Modifiers *
     **********************/

    modifier nonReentrant() {
        // On the first call to nonReentrant, _notEntered will be true
        require(_lock_status != _ENTERED, "ReentrancyGuard: reentrant call");

        // Any calls to nonReentrant after this point will fail
        _lock_status = _ENTERED;

        _;

        // By storing the original value once again, a refund is triggered (see
        // https://eips.ethereum.org/EIPS/eip-2200)
        _lock_status = _NOT_ENTERED;
    }

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
        _sendMessage(_to, _value, _message, _gasLimit, tx.origin);
    }

    /// @inheritdoc IScrollMessenger
    function sendMessage(
        address _to,
        uint256 _value,
        bytes calldata _message,
        uint256 _gasLimit,
        address _refundAddress
    ) external payable override whenNotPaused {
        _sendMessage(_to, _value, _message, _gasLimit, _refundAddress);
    }

    /// @inheritdoc IL1ScrollMessenger
    function relayMessageWithProof(
        address _from,
        address _to,
        uint256 _value,
        uint256 _nonce,
        bytes memory _message,
        L2MessageProof memory _proof
    ) external override whenNotPaused {
        require(
            xDomainMessageSender == ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER,
            "Message is already in execution"
        );

        bytes32 _xDomainCalldataHash = keccak256(_encodeXDomainCalldata(_from, _to, _value, _nonce, _message));
        require(!isL2MessageExecuted[_xDomainCalldataHash], "Message was already successfully executed");

        {
            address _rollup = rollup;
            require(IScrollChain(_rollup).isBatchFinalized(_proof.batchHash), "Batch is not finalized");
            bytes32 _messageRoot = IScrollChain(_rollup).getL2MessageRoot(_proof.batchHash);
            require(
                WithdrawTrieVerifier.verifyMerkleProof(_messageRoot, _xDomainCalldataHash, _nonce, _proof.merkleProof),
                "Invalid proof"
            );
        }

        // @todo check more `_to` address to avoid attack.
        require(_to != messageQueue, "Forbid to call message queue");
        require(_to != address(this), "Forbid to call self");

        // @note This usually will never happen, just in case.
        require(_from != xDomainMessageSender, "Invalid message sender");

        xDomainMessageSender = _from;
        (bool success, ) = _to.call{value: _value}(_message);
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
    /// @param _status The pause status to update.
    function setPause(bool _status) external onlyOwner {
        if (_status) {
            _pause();
        } else {
            _unpause();
        }
    }

    /**********************
     * Internal Functions *
     **********************/

    function _sendMessage(
        address _to,
        uint256 _value,
        bytes memory _message,
        uint256 _gasLimit,
        address _refundAddress
    ) internal nonReentrant {
        address _messageQueue = messageQueue; // gas saving
        address _counterpart = counterpart; // gas saving

        // compute the actual cross domain message calldata.
        uint256 _messageNonce = IL1MessageQueue(_messageQueue).nextCrossDomainMessageIndex();
        bytes memory _xDomainCalldata = _encodeXDomainCalldata(msg.sender, _to, _value, _messageNonce, _message);

        // compute and deduct the messaging fee to fee vault.
        uint256 _fee = IL1MessageQueue(_messageQueue).estimateCrossDomainMessageFee(_gasLimit);
        require(msg.value >= _fee + _value, "Insufficient msg.value");
        if (_fee > 0) {
            (bool _success, ) = feeVault.call{value: _fee}("");
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
                (bool _success, ) = _refundAddress.call{value: _refund}("");
                require(_success, "Failed to refund the fee");
            }
        }
    }
}
