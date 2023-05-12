// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {OwnableUpgradeable} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";

import {IL2GasPriceOracle} from "./IL2GasPriceOracle.sol";
import {IL1MessageQueue} from "./IL1MessageQueue.sol";

import {AddressAliasHelper} from "../../libraries/common/AddressAliasHelper.sol";

/// @title L1MessageQueue
/// @notice This contract will hold all L1 to L2 messages.
/// Each appended message is assigned with a unique and increasing `uint256` index.
contract L1MessageQueue is OwnableUpgradeable, IL1MessageQueue {
    /**********
     * Events *
     **********/

    /// @notice Emitted when owner updates gas oracle contract.
    /// @param _oldGasOracle The address of old gas oracle contract.
    /// @param _newGasOracle The address of new gas oracle contract.
    event UpdateGasOracle(address _oldGasOracle, address _newGasOracle);

    /// @notice Emitted when owner updates max gas limit.
    /// @param _oldMaxGasLimit The old max gas limit.
    /// @param _newMaxGasLimit The new max gas limit.
    event UpdateMaxGasLimit(uint256 _oldMaxGasLimit, uint256 _newMaxGasLimit);

    /*************
     * Variables *
     *************/

    /// @notice The address of L1ScrollMessenger contract.
    address public messenger;

    /// @notice The address of GasOracle contract.
    address public gasOracle;

    /// @notice The list of queued cross domain messages.
    bytes32[] public messageQueue;

    /// @notice The max gas limit of L1 transactions.
    uint256 public maxGasLimit;

    /***************
     * Constructor *
     ***************/

    function initialize(address _messenger, address _gasOracle, uint256 _maxGasLimit) external initializer {
        OwnableUpgradeable.__Ownable_init();

        messenger = _messenger;
        gasOracle = _gasOracle;
        maxGasLimit = _maxGasLimit;
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

            function store_uint(_ptr, v) -> ptr {
                ptr := _ptr
                switch lt(v, 128)
                case 1 {
                    // single byte in the [0x00, 0x7f]
                    mstore(ptr, shl(248, v))
                    ptr := add(ptr, 1)
                }
                default {
                    // 1-32 bytes long
                    let len := get_uint_bytes(v)
                    mstore(ptr, shl(248, add(len, 0x80)))
                    ptr := add(ptr, 1)
                    mstore(ptr, shl(mul(8, sub(32, len)), v))
                    ptr := add(ptr, len)
                }
            }

            function store_address(_ptr, v) -> ptr {
                ptr := _ptr
                // 20 bytes long
                mstore(ptr, shl(248, 0x94)) // 0x80 + 0x14
                ptr := add(ptr, 1)
                mstore(ptr, shl(96, v))
                ptr := add(ptr, 0x14)
            }

            // 1 byte for TransactionType
            // 4 byte for list payload length
            let start_ptr := add(mload(0x40), 5)
            let ptr := start_ptr
            ptr := store_uint(ptr, _queueIndex)
            ptr := store_uint(ptr, _gasLimit)
            ptr := store_address(ptr, _target)
            ptr := store_uint(ptr, _value)

            switch eq(_data.length, 1)
            case 1 {
                // single byte
                ptr := store_uint(ptr, shr(248, calldataload(_data.offset)))
            }
            default {
                switch lt(_data.length, 56)
                case 1 {
                    // a string is 0-55 bytes long
                    mstore(ptr, shl(248, add(0x80, _data.length)))
                    ptr := add(ptr, 1)
                    calldatacopy(ptr, _data.offset, _data.length)
                    ptr := add(ptr, _data.length)
                }
                default {
                    // a string is more than 55 bytes long
                    let len_bytes := get_uint_bytes(_data.length)
                    mstore(ptr, shl(248, add(0xb7, len_bytes)))
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

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @inheritdoc IL1MessageQueue
    function appendCrossDomainMessage(address _target, uint256 _gasLimit, bytes calldata _data) external override {
        require(msg.sender == messenger, "Only callable by the L1ScrollMessenger");

        // validate gas limit
        _validateGasLimit(_gasLimit, _data);

        // do address alias to avoid replay attack in L2.
        address _sender = AddressAliasHelper.applyL1ToL2Alias(msg.sender);

        // compute transaction hash
        uint256 _queueIndex = messageQueue.length;
        bytes32 _hash = computeTransactionHash(_sender, _queueIndex, 0, _target, _gasLimit, _data);
        messageQueue.push(_hash);

        emit QueueTransaction(_sender, _target, 0, _queueIndex, _gasLimit, _data);
    }

    /// @inheritdoc IL1MessageQueue
    function appendEnforcedTransaction(address, address, uint256, uint256, bytes calldata) external override {
        // @todo
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

    function _validateGasLimit(uint256 _gasLimit, bytes memory _calldata) internal {
        require(_gasLimit <= maxGasLimit, "Gas limit must not exceed maxGasLimit");
        // check if the gas limit is above intrinsic gas
        uint256 intrinsicGas = calculateIntrinsicGasFee(_calldata);
        require(_gasLimit >= intrinsicGas, "Insufficient gas limit, must be above intrinsic gas");
    }
}
