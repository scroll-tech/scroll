// SPDX-License-Identifier: MIT

pragma solidity ^0.8.24;

/// @title ITokenMessenger
/// @notice The interface of `TokenMessenger` of Circle's Cross-Chain Transfer Protocol (CCTP).
interface ITokenMessenger {
    /**
     * @notice Deposits and burns tokens from sender to be minted on destination domain. The mint
     * on the destination domain must be called by `destinationCaller`.
     * WARNING: if the `destinationCaller` does not represent a valid address as bytes32, then it will not be possible
     * to broadcast the message on the destination domain. This is an advanced feature, and the standard
     * depositForBurn() should be preferred for use cases where a specific destination caller is not required.
     * Emits a `DepositForBurn` event.
     * @dev reverts if:
     * - given destinationCaller is zero address
     * - given burnToken is not supported
     * - given destinationDomain has no TokenMessenger registered
     * - transferFrom() reverts. For example, if sender's burnToken balance or approved allowance
     * to this contract is less than `amount`.
     * - burn() reverts. For example, if `amount` is 0.
     * - MessageTransmitter returns false or reverts.
     * @param amount amount of tokens to burn
     * @param destinationDomain destination domain
     * @param mintRecipient address of mint recipient on destination domain
     * @param burnToken address of contract to burn deposited tokens, on local domain
     * @param destinationCaller caller on the destination domain, as bytes32
     * @return nonce unique nonce reserved by message
     */
    function depositForBurnWithCaller(
        uint256 amount,
        uint32 destinationDomain,
        bytes32 mintRecipient,
        address burnToken,
        bytes32 destinationCaller
    ) external returns (uint64 nonce);

    /**
     * @notice Replace a BurnMessage to change the mint recipient and/or
     * destination caller. Allows the sender of a previous BurnMessage
     * (created by depositForBurn or depositForBurnWithCaller)
     * to send a new BurnMessage to replace the original.
     * The new BurnMessage will reuse the amount and burn token of the original,
     * without requiring a new deposit.
     * @dev The new message will reuse the original message's nonce. For a
     * given nonce, all replacement message(s) and the original message are
     * valid to broadcast on the destination domain, until the first message
     * at the nonce confirms, at which point all others are invalidated.
     * Note: The msg.sender of the replaced message must be the same as the
     * msg.sender of the original message.
     * @param originalMessage original message bytes (to replace)
     * @param originalAttestation original attestation bytes
     * @param newDestinationCaller the new destination caller, which may be the
     * same as the original destination caller, a new destination caller, or an empty
     * destination caller (bytes32(0), indicating that any destination caller is valid.)
     * @param newMintRecipient the new mint recipient, which may be the same as the
     * original mint recipient, or different.
     */
    function replaceDepositForBurn(
        bytes calldata originalMessage,
        bytes calldata originalAttestation,
        bytes32 newDestinationCaller,
        bytes32 newMintRecipient
    ) external;
}
