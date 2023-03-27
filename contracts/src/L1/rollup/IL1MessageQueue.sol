// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

interface IL1MessageQueue {
    /**********
     * Events *
     **********/

    /// @notice Emitted when a new L1 => L2 transaction is appended to the queue.
    /// @param sender The address of account who initiates the transaction.
    /// @param target The address of account who will recieve the transaction.
    /// @param value The value passed with the transaction.
    /// @param queueIndex The index of this transaction in the queue.
    /// @param gasLimit Gas limit required to complete the message relay on L2.
    /// @param data The calldata of the transaction.
    event QueueTransaction(
        address indexed sender,
        address indexed target,
        uint256 value,
        uint256 queueIndex,
        uint256 gasLimit,
        bytes data
    );

    /*************************
     * Public View Functions *
     *************************/

    /// @notice Return the index of next appended message.
    /// @dev Also the total number of appended messages.
    function nextCrossDomainMessageIndex() external view returns (uint256);

    /// @notice Return the message of in `queueIndex`.
    /// @param queueIndex The index to query.
    function getCrossDomainMessage(uint256 queueIndex) external view returns (bytes32);

    /// @notice Return the amount of ETH should pay for cross domain message.
    /// @param sender The address of account who initiates the message in L1.
    /// @param target The address of account who will recieve the message in L2.
    /// @param message The content of the message.
    /// @param gasLimit Gas limit required to complete the message relay on L2.
    function estimateCrossDomainMessageFee(
        address sender,
        address target,
        bytes memory message,
        uint256 gasLimit
    ) external view returns (uint256);

    /// @notice Return the hash of a L1 message.
    /// @param sender The address of sender.
    /// @param nonce The queue index of this message.
    /// @param value The amount of Ether transfer to target.
    /// @param target The address of target.
    /// @param gasLimit The gas limit provided.
    /// @param data The calldata passed to target address.
    function computeTransactionHash(
        address sender,
        uint256 nonce,
        uint256 value,
        address target,
        uint256 gasLimit,
        bytes calldata data
    ) external view returns (bytes32);

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @notice Append a L1 to L2 message into this contract.
    /// @param target The address of target contract to call in L2.
    /// @param gasLimit The maximum gas should be used for relay this message in L2.
    /// @param data The calldata passed to target contract.
    function appendCrossDomainMessage(
        address target,
        uint256 gasLimit,
        bytes calldata data
    ) external;

    /// @notice Append an enforced transaction to this contract.
    /// @dev The address of sender should be an EOA.
    /// @param sender The address of sender who will initiate this transaction in L2.
    /// @param target The address of target contract to call in L2.
    /// @param value The value passed
    /// @param gasLimit The maximum gas should be used for this transaction in L2.
    /// @param data The calldata passed to target contract.
    function appendEnforcedTransaction(
        address sender,
        address target,
        uint256 value,
        uint256 gasLimit,
        bytes calldata data
    ) external;
}
