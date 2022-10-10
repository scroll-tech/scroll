// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

library ScrollConstants {
  /// @notice The address of default cross chain message sender.
  address internal constant DEFAULT_XDOMAIN_MESSAGE_SENDER = address(1);

  /// @notice The minimum seconds needed to wait if we want to drop message.
  uint256 internal constant MIN_DROP_DELAY_DURATION = 7 days;
}
