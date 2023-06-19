// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {PausableUpgradeable} from "@openzeppelin/contracts-upgradeable/security/PausableUpgradeable.sol";

import {IScrollChain} from "./rollup/IScrollChain.sol";
import {IL1MessageQueue} from "./rollup/IL1MessageQueue.sol";
import {IL1ScrollMessenger} from "./IL1ScrollMessenger.sol";
import {ScrollConstants} from "../libraries/constants/ScrollConstants.sol";
import {IScrollMessenger} from "../libraries/IScrollMessenger.sol";
import {ScrollMessengerBase} from "../libraries/ScrollMessengerBase.sol";
import {AddressAliasHelper} from "../libraries/common/AddressAliasHelper.sol";
import {WithdrawTrieVerifier} from "../libraries/verifier/WithdrawTrieVerifier.sol";

import {IMessageDropCallback} from "../libraries/callbacks/IMessageDropCallback.sol";

// solhint-disable avoid-low-level-calls
// solhint-disable reason-string

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
    /**********
     * Events *
     **********/

    /// @notice Emitted when the maximum number of times each message can be replayed is updated.
    /// @param maxReplayTimes The new maximum number of times each message can be replayed.
    event UpdateMaxReplayTimes(uint256 maxReplayTimes);

    /***********
     * Structs *
     ***********/

    struct ReplayState {
        // The number of replayed times.
        uint128 times;
        // The queue index of lastest replayed one. If it is zero, it means the message has not been replayed.
        uint256 lastIndex;
    }

    /*************
     * Variables *
     *************/

    /// @notice Mapping from relay id to relay status.
    mapping(bytes32 => bool) public isL1MessageRelayed;

    /// @notice Mapping from L1 message hash to sent status.
    mapping(bytes32 => bool) public isL1MessageSent;

    /// @notice Mapping from L2 message hash to a boolean value indicating if the message has been successfully executed.
    mapping(bytes32 => bool) public isL2MessageExecuted;

    /// @notice Mappging from L1 message hash to drop status.
    mapping(bytes32 => bool) public isL1MessageDropped;

    /// @notice The address of Rollup contract.
    address public rollup;

    /// @notice The address of L1MessageQueue contract.
    address public messageQueue;

    /// @notice The maximum number of times each L1 message can be replayed.
    uint256 public maxReplayTimes;

    /// @notice Mapping from L1 message hash to replay state.
    mapping(bytes32 => ReplayState) public replayStates;

    /// @notice Mapping from queue index to previous replay queue index.
    /// @dev If a message `x` was replayed 3 times with index `q1`, `q2` and `q3`, the
    /// value of `prevReplayIndex` and `replayStates` will be `replayStates[hash(x)].lastIndex = q3`,
    /// `replayStates[hash(x)].times = 3`, `prevReplayIndex[q3] = q2`, `prevReplayIndex[q2] = q1`
    /// and `prevReplayIndex[q1] = x`.
    mapping(uint256 => uint256) public prevReplayIndex;

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
            require(IScrollChain(_rollup).isBatchFinalized(_proof.batchIndex), "Batch is not finalized");
            bytes32 _messageRoot = IScrollChain(_rollup).withdrawRoots(_proof.batchIndex);
            require(
                WithdrawTrieVerifier.verifyMerkleProof(_messageRoot, _xDomainCalldataHash, _nonce, _proof.merkleProof),
                "Invalid proof"
            );
        }

        // @note check more `_to` address to avoid attack in the future when we add more gateways.
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
        uint32 _newGasLimit,
        address _refundAddress
    ) external payable override whenNotPaused {
        // only replay when no other malicious action is ongoing
        require(xDomainMessageSender == ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER, "message in some context");

        // We will use a different `queueIndex` for the replaced message. However, the original `queueIndex` or `nonce`
        // is encoded in the `_message`. We will check the `xDomainCalldata` in layer 2 to avoid duplicated execution.
        // So, only one message will succeed in layer 2. If one of the message is executed successfully, the other one
        // will revert with "Message was already successfully executed".
        address _messageQueue = messageQueue;
        address _counterpart = counterpart;
        bytes memory _xDomainCalldata = _encodeXDomainCalldata(_from, _to, _value, _queueIndex, _message);
        bytes32 _xDomainCalldataHash = keccak256(_xDomainCalldata);

        require(isL1MessageSent[_xDomainCalldataHash], "Provided message has not been enqueued");
        // cannot replay dropped message
        require(!isL1MessageDropped[_xDomainCalldataHash], "Message already dropped");

        // compute and deduct the messaging fee to fee vault.
        uint256 _fee = IL1MessageQueue(_messageQueue).estimateCrossDomainMessageFee(_newGasLimit);

        // charge relayer fee
        require(msg.value >= _fee, "Insufficient msg.value for fee");
        if (_fee > 0) {
            (bool _success, ) = feeVault.call{value: _fee}("");
            require(_success, "Failed to deduct the fee");
        }

        // enqueue the new transaction
        uint256 _nextQueueIndex = IL1MessageQueue(_messageQueue).nextCrossDomainMessageIndex();
        IL1MessageQueue(_messageQueue).appendCrossDomainMessage(_counterpart, _newGasLimit, _xDomainCalldata);

        ReplayState memory _replayState = replayStates[_xDomainCalldataHash];
        // update the replayed message chain.
        if (_replayState.lastIndex == 0) {
            // the message has not been replayed before.
            prevReplayIndex[_nextQueueIndex] = _queueIndex;
        } else {
            prevReplayIndex[_nextQueueIndex] = _replayState.lastIndex;
        }
        _replayState.lastIndex = uint128(_nextQueueIndex);

        // update replay times
        require(_replayState.times < maxReplayTimes, "Exceed maximum replay times");
        unchecked {
            _replayState.times += 1;
        }
        replayStates[_xDomainCalldataHash] = _replayState;

        // refund fee to `_refundAddress`
        unchecked {
            uint256 _refund = msg.value - _fee;
            if (_refund > 0) {
                (bool _success, ) = _refundAddress.call{value: _refund}("");
                require(_success, "Failed to refund the fee");
            }
        }
    }

    /// @inheritdoc IL1ScrollMessenger
    function dropMessage(
        address _from,
        address _to,
        uint256 _value,
        uint256 _queueIndex,
        bytes memory _message
    ) external override whenNotPaused {
        // The criteria for dropping a message:
        // 1. The message is a L1 message.
        // 2. The message has not been dropped before.
        // 3. the message and all of its replacement are finalized in L1.
        // 4. the message and all of its replacement are skipped.
        //
        // Possible denial of service attack:
        // + replayMessage is called every time someone want to drop the message.
        // + replayMessage is called so many times for a skipped message, thus results a long list.
        //
        // We limit the number of `replayMessage` calls of each message, which may solve the above problem.

        address _messageQueue = messageQueue;

        // check message exists
        bytes memory _xDomainCalldata = _encodeXDomainCalldata(_from, _to, _value, _queueIndex, _message);
        bytes32 _xDomainCalldataHash = keccak256(_xDomainCalldata);
        require(isL1MessageSent[_xDomainCalldataHash], "Provided message has not been enqueued");

        // check message not dropped
        require(!isL1MessageDropped[_xDomainCalldataHash], "Message already dropped");

        // check message is finalized
        uint256 _lastIndex = replayStates[_xDomainCalldataHash].lastIndex;
        if (_lastIndex == 0) _lastIndex = _queueIndex;
        require(IL1MessageQueue(_messageQueue).pendingQueueIndex() > _lastIndex, "Message not finalized");

        // check message is skipped and drop it.
        // @note If the list is very long, the message may never be dropped.
        while (true) {
            IL1MessageQueue(_messageQueue).dropCrossDomainMessage(_lastIndex);
            _lastIndex = prevReplayIndex[_lastIndex];
            if (_lastIndex == 0) break;
        }

        isL1MessageDropped[_xDomainCalldataHash] = true;

        // set execution context
        xDomainMessageSender = ScrollConstants.DROP_XDOMAIN_MESSAGE_SENDER;
        IMessageDropCallback(_from).onDropMessage{value: _value}(_message);
        // clear execution context
        xDomainMessageSender = ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER;
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

    /// @notice Update max replay times.
    /// @dev This function can only called by contract owner.
    /// @param _maxReplayTimes The new max replay times.
    function updateMaxReplayTimes(uint256 _maxReplayTimes) external onlyOwner {
        maxReplayTimes = _maxReplayTimes;

        emit UpdateMaxReplayTimes(_maxReplayTimes);
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

        // refund fee to `_refundAddress`
        unchecked {
            uint256 _refund = msg.value - _fee - _value;
            if (_refund > 0) {
                (bool _success, ) = _refundAddress.call{value: _refund}("");
                require(_success, "Failed to refund the fee");
            }
        }
    }
}
