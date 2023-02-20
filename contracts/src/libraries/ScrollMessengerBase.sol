// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { OwnableUpgradeable } from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";

import { IWhitelist } from "./common/IWhitelist.sol";
import { ScrollConstants } from "./constants/ScrollConstants.sol";
import { IScrollMessenger } from "./IScrollMessenger.sol";

abstract contract ScrollMessengerBase is OwnableUpgradeable, IScrollMessenger {
  /**********
   * Events *
   **********/

  /// @notice Emitted when owner updates whitelist contract.
  /// @param _oldWhitelist The address of old whitelist contract.
  /// @param _newWhitelist The address of new whitelist contract.
  event UpdateWhitelist(address _oldWhitelist, address _newWhitelist);

  /// @notice Emitted when owner updates fee vault contract.
  /// @param _oldFeeVault The address of old fee vault contract.
  /// @param _newFeeVault The address of new fee vault contract.
  event UpdateFeeVault(address _oldFeeVault, address _newFeeVault);

  /*************
   * Variables *
   *************/

  /// @notice See {IScrollMessenger-xDomainMessageSender}
  address public override xDomainMessageSender;

  /// @notice The whitelist contract to track the sender who can call `sendMessage` in ScrollMessenger.
  address public whitelist;

  /// @notice The address of counterpart ScrollMessenger contract in L1/L2.
  address public counterpart;

  /// @notice The address of fee vault, collecting cross domain messaging fee.
  address public feeVault;

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

  function _initialize(address _counterpart, address _feeVault) internal {
    OwnableUpgradeable.__Ownable_init();

    // initialize to a nonzero value
    xDomainMessageSender = ScrollConstants.DEFAULT_XDOMAIN_MESSAGE_SENDER;

    counterpart = _counterpart;
    feeVault = _feeVault;
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

  /// @notice Update fee vault contract.
  /// @dev This function can only called by contract owner.
  /// @param _newFeeVault The address of new fee vault contract.
  function updateFeeVault(address _newFeeVault) external onlyOwner {
    address _oldFeeVault = whitelist;

    feeVault = _newFeeVault;
    emit UpdateFeeVault(_oldFeeVault, _newFeeVault);
  }

  /**********************
   * Internal Functions *
   **********************/

  /// @dev Internal function to generate the correct cross domain calldata for a message.
  /// @param _sender Message sender address.
  /// @param _target Target contract address.
  /// @param _value The amount of ETH pass to the target.
  /// @param _messageNonce Nonce for the provided message.
  /// @param _message Message to send to the target.
  /// @return ABI encoded cross domain calldata.
  function _encodeXDomainCalldata(
    address _sender,
    address _target,
    uint256 _value,
    uint256 _messageNonce,
    bytes memory _message
  ) internal pure returns (bytes memory) {
    return
      abi.encodeWithSignature(
        "relayMessage(address,address,uint256,uint256,bytes)",
        _sender,
        _target,
        _value,
        _messageNonce,
        _message
      );
  }
}
