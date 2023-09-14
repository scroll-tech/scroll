// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {IMessageTransmitter} from "../../interfaces/IMessageTransmitter.sol";

import {ScrollGatewayBase} from "./ScrollGatewayBase.sol";

/// @title CCTPGatewayBase
/// @notice The `CCTPGatewayBase` is a base contract for USDC gateways with CCTP supports.
abstract contract CCTPGatewayBase is ScrollGatewayBase {
    /*********
     * Enums *
     *********/

    enum CCTPMessageStatus {
        None,
        Pending,
        Relayed
    }

    /*************
     * Constants *
     *************/

    /// @notice The address of L1 USDC address.
    address public immutable l1USDC;

    /// @notice The address of L2 USDC address.
    address public immutable l2USDC;

    /// @notice The destination domain for layer2.
    uint32 public immutable destinationDomain;

    /*************
     * Variables *
     *************/

    /// @notice The address of TokenMessenger in local domain.
    address public cctpMessenger;

    /// @notice The address of MessageTransmitter in local domain.
    address public cctpTransmitter;

    /// @notice Mapping from destination domain CCTP nonce to status.
    mapping(uint256 => CCTPMessageStatus) public status;

    /// @dev The storage slots for future usage.
    uint256[47] private __gap;

    /***************
     * Constructor *
     ***************/

    constructor(
        address _l1USDC,
        address _l2USDC,
        uint32 _destinationDomain
    ) {
        l1USDC = _l1USDC;
        l2USDC = _l2USDC;
        destinationDomain = _destinationDomain;
    }

    function _initialize(address _cctpMessenger, address _cctpTransmitter) internal {
        cctpMessenger = _cctpMessenger;
        cctpTransmitter = _cctpTransmitter;
    }

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @notice Claim USDC that has been cross chained.
    /// @param _nonce The nonce of the message from CCTP.
    /// @param _cctpMessage The message passed to MessageTransmitter contract in CCTP.
    /// @param _cctpSignature The message passed to MessageTransmitter contract in CCTP.
    function claimUSDC(
        uint256 _nonce,
        bytes calldata _cctpMessage,
        bytes calldata _cctpSignature
    ) public {
        // Check `_nonce` match with `_cctpMessage`.
        // According to the encoding of `_cctpMessage`, the nonce is in bytes 12 to 16.
        // See here: https://github.com/circlefin/evm-cctp-contracts/blob/master/src/messages/Message.sol#L29
        uint256 _expectedMessageNonce;
        assembly {
            _expectedMessageNonce := and(shr(96, calldataload(_cctpMessage.offset)), 0xffffffffffffffff)
        }
        require(_expectedMessageNonce == _nonce, "nonce mismatch");

        require(status[_nonce] == CCTPMessageStatus.Pending, "message not relayed");

        // call transmitter to mint USDC
        bool _success = IMessageTransmitter(cctpTransmitter).receiveMessage(_cctpMessage, _cctpSignature);
        require(_success, "call transmitter failed");

        status[_nonce] = CCTPMessageStatus.Relayed;
    }
}
