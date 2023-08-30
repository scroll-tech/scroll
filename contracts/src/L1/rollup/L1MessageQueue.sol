// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

import {OwnableUpgradeable} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import {BitMapsUpgradeable} from "@openzeppelin/contracts-upgradeable/utils/structs/BitMapsUpgradeable.sol";

import {IL2GasPriceOracle} from "./IL2GasPriceOracle.sol";
import {IL1MessageQueue} from "./IL1MessageQueue.sol";

import {AddressAliasHelper} from "../../libraries/common/AddressAliasHelper.sol";

// solhint-disable no-empty-blocks
// solhint-disable no-inline-assembly
// solhint-disable reason-string

/// @title L1MessageQueue
/// @notice This contract will hold all L1 to L2 messages.
/// Each appended message is assigned with a unique and increasing `uint256` index.
contract L1MessageQueue is OwnableUpgradeable, IL1MessageQueue {
    using BitMapsUpgradeable for BitMapsUpgradeable.BitMap;

    /**********
     * Events *
     **********/

    /// @notice Emitted when owner updates gas oracle contract.
    /// @param _oldGasOracle The address of old gas oracle contract.
    /// @param _newGasOracle The address of new gas oracle contract.
    event UpdateGasOracle(address indexed _oldGasOracle, address indexed _newGasOracle);

    /// @notice Emitted when owner updates EnforcedTxGateway contract.
    /// @param _oldGateway The address of old EnforcedTxGateway contract.
    /// @param _newGateway The address of new EnforcedTxGateway contract.
    event UpdateEnforcedTxGateway(address indexed _oldGateway, address indexed _newGateway);

    /// @notice Emitted when owner updates max gas limit.
    /// @param _oldMaxGasLimit The old max gas limit.
    /// @param _newMaxGasLimit The new max gas limit.
    event UpdateMaxGasLimit(uint256 _oldMaxGasLimit, uint256 _newMaxGasLimit);

    /*************
     * Variables *
     *************/

    /// @notice The address of L1ScrollMessenger contract.
    address public messenger;

    /// @notice The address of ScrollChain contract.
    address public scrollChain;

    /// @notice The address EnforcedTxGateway contract.
    address public enforcedTxGateway;

    /// @notice The address of GasOracle contract.
    address public gasOracle;

    /// @notice The list of queued cross domain messages.
    bytes32[] public messageQueue;

    /// @inheritdoc IL1MessageQueue
    uint256 public pendingQueueIndex;

    /// @notice The max gas limit of L1 transactions.
    uint256 public maxGasLimit;

    /// @dev The bitmap for skipped messages.
    BitMapsUpgradeable.BitMap private droppedMessageBitmap;

    /// @dev The bitmap for skipped messages, where `skippedMessageBitmap[i]` keeps the bits from `[i*256, (i+1)*256)`.
    mapping(uint256 => uint256) private skippedMessageBitmap;

    /// @notice The start queue index when we enable the `skippedMessageBitmap`.
    uint256 public bitmapEnabledQueueIndex;

    /**********************
     * Function Modifiers *
     **********************/

    modifier onlyMessenger() {
        require(msg.sender == messenger, "Only callable by the L1ScrollMessenger");
        _;
    }

    /***************
     * Constructor *
     ***************/

    constructor() {
        _disableInitializers();
    }

    function initialize(
        address _messenger,
        address _scrollChain,
        address _enforcedTxGateway,
        address _gasOracle,
        uint256 _maxGasLimit
    ) external initializer {
        OwnableUpgradeable.__Ownable_init();

        messenger = _messenger;
        scrollChain = _scrollChain;
        enforcedTxGateway = _enforcedTxGateway;
        gasOracle = _gasOracle;
        maxGasLimit = _maxGasLimit;
    }

    function initializeV2() external reinitializer(2) {
        bitmapEnabledQueueIndex = pendingQueueIndex;
    }

    /*************************
     * Public View Functions *
     *************************/

    /// @inheritdoc IL1MessageQueue
    function nextCrossDomainMessageIndex() external view returns (uint256) {
        return messageQueue.length;
    }

    /// @inheritdoc IL1MessageQueue
    function getCrossDomainMessage(uint256 _queueIndex) external view returns (bytes32) {
        return messageQueue[_queueIndex];
    }

    /// @inheritdoc IL1MessageQueue
    function estimateCrossDomainMessageFee(uint256 _gasLimit) external view override returns (uint256) {
        address _oracle = gasOracle;
        if (_oracle == address(0)) return 0;
        return IL2GasPriceOracle(_oracle).estimateCrossDomainMessageFee(_gasLimit);
    }

    /// @inheritdoc IL1MessageQueue
    function calculateIntrinsicGasFee(bytes memory _calldata) public view override returns (uint256) {
        address _oracle = gasOracle;
        if (_oracle == address(0)) return 0;
        return IL2GasPriceOracle(_oracle).calculateIntrinsicGasFee(_calldata);
    }

    /// @inheritdoc IL1MessageQueue
    function computeTransactionHash(
        address _sender,
        uint256 _queueIndex,
        uint256 _value,
        address _target,
        uint256 _gasLimit,
        bytes calldata _data
    ) public pure override returns (bytes32) {
        // We use EIP-2718 to encode the L1 message, and the encoding of the message is
        //      `TransactionType || TransactionPayload`
        // where
        //  1. `TransactionType` is 0x7E
        //  2. `TransactionPayload` is `rlp([queueIndex, gasLimit, to, value, data, sender])`
        //
        // The spec of rlp: https://ethereum.org/en/developers/docs/data-structures-and-encoding/rlp/
        uint256 transactionType = 0x7E;
        bytes32 hash;
        assembly {
            function get_uint_bytes(v) -> len {
                if eq(v, 0) {
                    len := 1
                    leave
                }
                for {

                } gt(v, 0) {

                } {
                    len := add(len, 1)
                    v := shr(8, v)
                }
            }

            // This is used for both store uint and single byte.
            // Integer zero is special handled by geth to encode as `0x80`
            function store_uint_or_byte(_ptr, v, is_uint) -> ptr {
                ptr := _ptr
                switch lt(v, 128)
                case 1 {
                    switch and(iszero(v), is_uint)
                    case 1 {
                        // integer 0
                        mstore8(ptr, 0x80)
                    }
                    default {
                        // single byte in the [0x00, 0x7f]
                        mstore8(ptr, v)
                    }
                    ptr := add(ptr, 1)
                }
                default {
                    // 1-32 bytes long
                    let len := get_uint_bytes(v)
                    mstore8(ptr, add(len, 0x80))
                    ptr := add(ptr, 1)
                    mstore(ptr, shl(mul(8, sub(32, len)), v))
                    ptr := add(ptr, len)
                }
            }

            function store_address(_ptr, v) -> ptr {
                ptr := _ptr
                // 20 bytes long
                mstore8(ptr, 0x94) // 0x80 + 0x14
                ptr := add(ptr, 1)
                mstore(ptr, shl(96, v))
                ptr := add(ptr, 0x14)
            }

            // 1 byte for TransactionType
            // 4 byte for list payload length
            let start_ptr := add(mload(0x40), 5)
            let ptr := start_ptr
            ptr := store_uint_or_byte(ptr, _queueIndex, 1)
            ptr := store_uint_or_byte(ptr, _gasLimit, 1)
            ptr := store_address(ptr, _target)
            ptr := store_uint_or_byte(ptr, _value, 1)

            switch eq(_data.length, 1)
            case 1 {
                // single byte
                ptr := store_uint_or_byte(ptr, byte(0, calldataload(_data.offset)), 0)
            }
            default {
                switch lt(_data.length, 56)
                case 1 {
                    // a string is 0-55 bytes long
                    mstore8(ptr, add(0x80, _data.length))
                    ptr := add(ptr, 1)
                    calldatacopy(ptr, _data.offset, _data.length)
                    ptr := add(ptr, _data.length)
                }
                default {
                    // a string is more than 55 bytes long
                    let len_bytes := get_uint_bytes(_data.length)
                    mstore8(ptr, add(0xb7, len_bytes))
                    ptr := add(ptr, 1)
                    mstore(ptr, shl(mul(8, sub(32, len_bytes)), _data.length))
                    ptr := add(ptr, len_bytes)
                    calldatacopy(ptr, _data.offset, _data.length)
                    ptr := add(ptr, _data.length)
                }
            }
            ptr := store_address(ptr, _sender)

            let payload_len := sub(ptr, start_ptr)
            let value
            let value_bytes
            switch lt(payload_len, 56)
            case 1 {
                // the total payload of a list is 0-55 bytes long
                value := add(0xc0, payload_len)
                value_bytes := 1
            }
            default {
                // If the total payload of a list is more than 55 bytes long
                let len_bytes := get_uint_bytes(payload_len)
                value_bytes := add(len_bytes, 1)
                value := add(0xf7, len_bytes)
                value := shl(mul(len_bytes, 8), value)
                value := or(value, payload_len)
            }
            value := or(value, shl(mul(8, value_bytes), transactionType))
            value_bytes := add(value_bytes, 1)
            let value_bits := mul(8, value_bytes)
            value := or(shl(sub(256, value_bits), value), shr(value_bits, mload(start_ptr)))
            start_ptr := sub(start_ptr, value_bytes)
            mstore(start_ptr, value)
            hash := keccak256(start_ptr, sub(ptr, start_ptr))
        }
        return hash;
    }

    /// @inheritdoc IL1MessageQueue
    /// @dev Before the `bitmapEnabledQueueIndex`, if the message is dropped, the function will return `false`.
    function isMessageSkipped(uint256 _queueIndex) external view returns (bool) {
        if (_queueIndex >= pendingQueueIndex) return false;

        if (_queueIndex < bitmapEnabledQueueIndex) {
            return messageQueue[_queueIndex] != bytes32(0);
        } else {
            return _getBitmap(_queueIndex);
        }
    }

    /// @inheritdoc IL1MessageQueue
    function isMessageDropped(uint256 _queueIndex) external view returns (bool) {
        if (_queueIndex < bitmapEnabledQueueIndex) {
            // @note This will also include the executed messages.
            return messageQueue[_queueIndex] == bytes32(0);
        } else {
            // it should be a skipped message first.
            return _getBitmap(_queueIndex) && droppedMessageBitmap.get(_queueIndex);
        }
    }

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @inheritdoc IL1MessageQueue
    function appendCrossDomainMessage(
        address _target,
        uint256 _gasLimit,
        bytes calldata _data
    ) external override onlyMessenger {
        // validate gas limit
        _validateGasLimit(_gasLimit, _data);

        // do address alias to avoid replay attack in L2.
        address _sender = AddressAliasHelper.applyL1ToL2Alias(msg.sender);

        _queueTransaction(_sender, _target, 0, _gasLimit, _data);
    }

    /// @inheritdoc IL1MessageQueue
    function appendEnforcedTransaction(
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        bytes calldata _data
    ) external override {
        require(msg.sender == enforcedTxGateway, "Only callable by the EnforcedTxGateway");
        // We will check it in EnforcedTxGateway, just in case.
        require(_sender.code.length == 0, "only EOA");

        // validate gas limit
        _validateGasLimit(_gasLimit, _data);

        _queueTransaction(_sender, _target, _value, _gasLimit, _data);
    }

    /// @inheritdoc IL1MessageQueue
    function popCrossDomainMessage(
        uint256 _startIndex,
        uint256 _count,
        uint256 _skippedBitmap
    ) external {
        require(msg.sender == scrollChain, "Only callable by the ScrollChain");

        require(_count <= 256, "pop too many messages");
        require(pendingQueueIndex == _startIndex, "start index mismatch");

        unchecked {
            // clear extra bits in `_skippedBitmap`, and if _count = 256, the overflow is designed
            uint256 mask = (1 << _count) - 1;
            _skippedBitmap &= mask;

            uint256 bucket = _startIndex >> 8;
            uint256 offset = _startIndex & 0xff;
            skippedMessageBitmap[bucket] |= _skippedBitmap << offset;
            if (offset + _count > 256) {
                skippedMessageBitmap[bucket + 1] = _skippedBitmap >> (256 - offset);
            }

            pendingQueueIndex = _startIndex + _count;
        }

        emit DequeueTransaction(_startIndex, _count, _skippedBitmap);
    }

    /// @inheritdoc IL1MessageQueue
    function dropCrossDomainMessage(uint256 _index) external onlyMessenger {
        require(_index < pendingQueueIndex, "cannot drop pending message");

        require(_getBitmap(_index), "drop non-skipped message");
        require(!droppedMessageBitmap.get(_index), "message already dropped");
        droppedMessageBitmap.set(_index);

        emit DropTransaction(_index);
    }

    /************************
     * Restricted Functions *
     ************************/

    /// @notice Update the address of gas oracle.
    /// @dev This function can only called by contract owner.
    /// @param _newGasOracle The address to update.
    function updateGasOracle(address _newGasOracle) external onlyOwner {
        address _oldGasOracle = gasOracle;
        gasOracle = _newGasOracle;

        emit UpdateGasOracle(_oldGasOracle, _newGasOracle);
    }

    /// @notice Update the address of EnforcedTxGateway.
    /// @dev This function can only called by contract owner.
    /// @param _newGateway The address to update.
    function updateEnforcedTxGateway(address _newGateway) external onlyOwner {
        address _oldGateway = enforcedTxGateway;
        enforcedTxGateway = _newGateway;

        emit UpdateEnforcedTxGateway(_oldGateway, _newGateway);
    }

    /// @notice Update the max gas limit.
    /// @dev This function can only called by contract owner.
    /// @param _newMaxGasLimit The new max gas limit.
    function updateMaxGasLimit(uint256 _newMaxGasLimit) external onlyOwner {
        uint256 _oldMaxGasLimit = maxGasLimit;
        maxGasLimit = _newMaxGasLimit;

        emit UpdateMaxGasLimit(_oldMaxGasLimit, _newMaxGasLimit);
    }

    /**********************
     * Internal Functions *
     **********************/

    /// @dev Internal function to queue a L1 transaction.
    /// @param _sender The address of sender who will initiate this transaction in L2.
    /// @param _target The address of target contract to call in L2.
    /// @param _value The value passed
    /// @param _gasLimit The maximum gas should be used for this transaction in L2.
    /// @param _data The calldata passed to target contract.
    function _queueTransaction(
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        bytes calldata _data
    ) internal {
        // compute transaction hash
        uint256 _queueIndex = messageQueue.length;
        bytes32 _hash = computeTransactionHash(_sender, _queueIndex, _value, _target, _gasLimit, _data);
        messageQueue.push(_hash);

        // emit event
        emit QueueTransaction(_sender, _target, _value, uint64(_queueIndex), _gasLimit, _data);
    }

    function _validateGasLimit(uint256 _gasLimit, bytes memory _calldata) internal view {
        require(_gasLimit <= maxGasLimit, "Gas limit must not exceed maxGasLimit");
        // check if the gas limit is above intrinsic gas
        uint256 intrinsicGas = calculateIntrinsicGasFee(_calldata);
        require(_gasLimit >= intrinsicGas, "Insufficient gas limit, must be above intrinsic gas");
    }

    /// @dev Returns whether the bit at `index` is set.
    function _getBitmap(uint256 index) internal view returns (bool) {
        uint256 bucket = index >> 8;
        uint256 mask = 1 << (index & 0xff);
        return skippedMessageBitmap[bucket] & mask != 0;
    }
}
