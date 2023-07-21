// SPDX-License-Identifier: MIT

pragma solidity ^0.8.16;

library ScrollConstants {
    /// @notice The address of default cross chain message sender.
    address internal constant DEFAULT_XDOMAIN_MESSAGE_SENDER = address(1);

    /// @notice The address for dropping message.
    /// @dev The first 20 bytes of keccak("drop")
    address internal constant DROP_XDOMAIN_MESSAGE_SENDER = 0x6f297C61B5C92eF107fFD30CD56AFFE5A273e841;
}
