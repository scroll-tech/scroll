// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { IScrollGateway } from "./IScrollGateway.sol";
import { IScrollMessenger } from "../IScrollMessenger.sol";

abstract contract ScrollGatewayBase is IScrollGateway {
  /*************
   * Constants *
   *************/

  // https://github.com/OpenZeppelin/openzeppelin-contracts/blob/v4.5.0/contracts/security/ReentrancyGuard.sol
  uint256 private constant _NOT_ENTERED = 1;
  uint256 private constant _ENTERED = 2;

  /*************
   * Variables *
   *************/

  /// @inheritdoc IScrollGateway
  address public override counterpart;

  /// @inheritdoc IScrollGateway
  address public override router;

  /// @inheritdoc IScrollGateway
  address public override messenger;

  /// @dev The status of for non-reentrant check.
  uint256 private _status;

  /**********************
   * Function Modifiers *
   **********************/

  modifier nonReentrant() {
    // On the first call to nonReentrant, _notEntered will be true
    require(_status != _ENTERED, "ReentrancyGuard: reentrant call");

    // Any calls to nonReentrant after this point will fail
    _status = _ENTERED;

    _;

    // By storing the original value once again, a refund is triggered (see
    // https://eips.ethereum.org/EIPS/eip-2200)
    _status = _NOT_ENTERED;
  }

  modifier onlyMessenger() {
    require(msg.sender == messenger, "only messenger can call");
    _;
  }

  modifier onlyCallByCounterpart() {
    address _messenger = messenger; // gas saving
    require(msg.sender == _messenger, "only messenger can call");
    require(counterpart == IScrollMessenger(_messenger).xDomainMessageSender(), "only call by conterpart");
    _;
  }

  /***************
   * Constructor *
   ***************/

  function _initialize(
    address _counterpart,
    address _router,
    address _messenger
  ) internal {
    require(_counterpart != address(0), "zero counterpart address");
    require(_messenger != address(0), "zero messenger address");

    counterpart = _counterpart;
    messenger = _messenger;

    // @note: the address of router could be zero, if this contract is GatewayRouter.
    if (_router != address(0)) {
      router = _router;
    }

    // for reentrancy guard
    _status = _NOT_ENTERED;
  }
}
