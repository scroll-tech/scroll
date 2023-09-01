// SPDX-License-Identifier: MIT

pragma solidity ^0.8.16;

interface IMessageTransmitter {
    function usedNonces(bytes32 _sourceAndNonce) external view returns (uint256);

    /**
     * @notice Receives an incoming message, validating the header and passing
     * the body to application-specific handler.
     * @param message The message raw bytes
     * @param signature The message signature
     * @return success bool, true if successful
     */
    function receiveMessage(bytes calldata message, bytes calldata signature) external returns (bool success);
}
