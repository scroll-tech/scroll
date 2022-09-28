// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { IWhitelist } from "./common/IWhitelist.sol";
import { IScrollMessenger } from "./IScrollMessenger.sol";
import { ScrollConstants } from "./ScrollConstants.sol";

abstract contract ScrollMessengerBase is IScrollMessenger {
  /**************************************** Events ****************************************/

  /// @notice Emitted when owner updates gas oracle contract.
  /// @param _oldGasOracle The address of old gas oracle contract.
  /// @param _newGasOracle The address of new gas oracle contract.
  event UpdateGasOracle(address _oldGasOracle, address _newGasOracle);

  /// @notice Emitted when owner updates whitelist contract.
  /// @param _oldWhitelist The address of old whitelist contract.
  /// @param _newWhitelist The address of new whitelist contract.
  event UpdateWhitelist(address _oldWhitelist, address _newWhitelist);

  /// @notice Emitted when owner updates drop delay duration
  /// @param _oldDuration The old drop delay duration in seconds.
  /// @param _newDuration The new drop delay duration in seconds.
  event UpdateDropDelayDuration(uint256 _oldDuration, uint256 _newDuration);

  /**************************************** Variables ****************************************/

  /// @notice See {IScrollMessenger-xDomainMessageSender}
  address public override xDomainMessageSender;

  /// @notice The gas oracle used to estimate transaction fee on layer 2.
  address public gasOracle;

  /// @notice The whitelist contract to track the sender who can call `sendMessage` in ScrollMessenger.
  address public whitelist;

  /// @notice The amount of seconds needed to wait if we want to drop message.
  uint256 public dropDelayDuration;

  modifier onlyWhitelistedSender(address _sender) {
    address _whitelist = whitelist;
    require(_whitelist == address(0) || IWhitelist(_whitelist).isSenderAllowed(_sender), "sender not whitelisted");
    _;
  }

  function _initialize() internal {
    // initialize to a nonzero value
    xDomainMessageSender = ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER;

    dropDelayDuration = ScrollConstants.MIN_DROP_DELAY_DURATION;
  }

  // allow others to send ether to messenger
  receive() external payable {}
}
