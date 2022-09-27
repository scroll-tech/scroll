// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { IL2ERC20Gateway } from "./IL2ERC20Gateway.sol";

import { IScrollGateway } from "../../libraries/gateway/IScrollGateway.sol";

interface IL2GatewayRouter is IL2ERC20Gateway, IScrollGateway {
  /**************************************** Events ****************************************/

  event WithdrawETH(address indexed _from, address indexed _to, uint256 _amount, bytes _data);
  event FinalizeDepositETH(address indexed _from, address indexed _to, uint256 _amount, bytes _data);

  /**************************************** Mutated Functions ****************************************/
  /// @notice Withdraw ETH to caller's account in L1.
  /// @param _gasLimit Gas limit required to complete the withdraw on L1.
  function withdrawETH(uint256 _gasLimit) external payable;

  /// @notice Withdraw ETH to caller's account in L1.
  /// @param _to The address of recipient's account on L1.
  /// @param _gasLimit Gas limit required to complete the withdraw on L1.
  function withdrawETH(address _to, uint256 _gasLimit) external payable;

  // @todo add withdrawETHAndCall;

  /// @notice Complete ETH deposit from L1 to L2 and send fund to recipient's account in L2.
  /// @dev This function should only be called by L2ScrollMessenger.
  ///      This function should also only be called by L1GatewayRouter in L1.
  /// @param _from The address of account who deposit ETH in L1.
  /// @param _to The address of recipient in L2 to receive ETH.
  /// @param _amount The amount of ETH to deposit.
  /// @param _data Optional data to forward to recipient's account.
  function finalizeDepositETH(
    address _from,
    address _to,
    uint256 _amount,
    bytes calldata _data
  ) external payable;
}
