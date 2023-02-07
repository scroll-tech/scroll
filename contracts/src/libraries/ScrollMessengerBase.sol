// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { OwnableUpgradeable } from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";

import { IWhitelist } from "./common/IWhitelist.sol";
import { IScrollMessenger } from "./IScrollMessenger.sol";
import { ScrollConstants } from "./ScrollConstants.sol";

abstract contract ScrollMessengerBase is OwnableUpgradeable, IScrollMessenger {
  /**********
   * Events *
   **********/

  /// @notice Emitted when owner updates whitelist contract.
  /// @param _oldWhitelist The address of old whitelist contract.
  /// @param _newWhitelist The address of new whitelist contract.
  event UpdateWhitelist(address _oldWhitelist, address _newWhitelist);

  /*************
   * Variables *
   *************/

  /// @notice See {IScrollMessenger-xDomainMessageSender}
  address public override xDomainMessageSender;

  /// @notice The whitelist contract to track the sender who can call `sendMessage` in ScrollMessenger.
  address public whitelist;

  /**********************
   * Function Modifiers *
   **********************/

  modifier onlyWhitelistedSender(address _sender) {
    address _whitelist = whitelist;
    require(_whitelist == address(0) || IWhitelist(_whitelist).isSenderAllowed(_sender), "sender not whitelisted");
    _;
  }

  /***************
   * Constructor *
   ***************/

  function _initialize() internal {
    OwnableUpgradeable.__Ownable_init();

    // initialize to a nonzero value
    xDomainMessageSender = ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER;
  }

  // allow others to send ether to messenger
  receive() external payable {}

  /************************
   * Restricted Functions *
   ************************/

  /// @notice Update whitelist contract.
  /// @dev This function can only called by contract owner.
  /// @param _newWhitelist The address of new whitelist contract.
  function updateWhitelist(address _newWhitelist) external onlyOwner {
    address _oldWhitelist = whitelist;

    whitelist = _newWhitelist;
    emit UpdateWhitelist(_oldWhitelist, _newWhitelist);
  }

  /**********************
   * Internal Functions *
   **********************/

  /// @dev Internal function to generate the correct cross domain calldata for a message.
  /// @param _sender Message sender address.
  /// @param _target Target contract address.
  /// @param _message Message to send to the target.
  /// @param _messageNonce Nonce for the provided message.
  /// @return ABI encoded cross domain calldata.
  function _encodeXDomainCalldata(
    address _sender,
    address _target,
    bytes memory _message,
    uint256 _messageNonce
  ) internal pure returns (bytes memory) {
    return
      abi.encodeWithSignature("relayMessage(address,address,bytes,uint256)", _sender, _target, _message, _messageNonce);
  }
}
