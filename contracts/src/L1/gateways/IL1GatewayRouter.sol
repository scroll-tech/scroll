// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { IL1ERC20Gateway } from "./IL1ERC20Gateway.sol";

import { IScrollGateway } from "../../libraries/gateway/IScrollGateway.sol";

interface IL1GatewayRouter is IL1ERC20Gateway, IScrollGateway {
  /**************************************** Events ****************************************/

  event FinalizeWithdrawETH(address indexed _from, address indexed _to, uint256 _amount, bytes _data);
  event DepositETH(address indexed _from, address indexed _to, uint256 _amount, bytes _data);

  /**************************************** Mutated Functions ****************************************/

  /// @notice Deposit ETH to call's account in L2.
  /// @param _gasLimit Gas limit required to complete the deposit on L2.
  function depositETH(uint256 _gasLimit) external payable;

  /// @notice Deposit ETH to recipient's account in L2.
  /// @param _to The address of recipient's account on L2.
  /// @param _gasLimit Gas limit required to complete the deposit on L2.
  function depositETH(address _to, uint256 _gasLimit) external payable;

  // @todo add depositETHAndCall;

  /// @notice Complete ETH withdraw from L2 to L1 and send fund to recipient's account in L1.
  /// @dev This function should only be called by L1ScrollMessenger.
  ///      This function should also only be called by L2GatewayRouter in L2.
  /// @param _from The address of account who withdraw ETH in L2.
  /// @param _to The address of recipient in L1 to receive ETH.
  /// @param _amount The amount of ETH to withdraw.
  /// @param _data Optional data to forward to recipient's account.
  function finalizeWithdrawETH(
    address _from,
    address _to,
    uint256 _amount,
    bytes calldata _data
  ) external payable;
}
