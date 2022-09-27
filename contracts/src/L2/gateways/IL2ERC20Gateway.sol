// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

interface IL2ERC20Gateway {
  /**************************************** Events ****************************************/

  event WithdrawERC20(
    address indexed _l1Token,
    address indexed _l2Token,
    address indexed _from,
    address _to,
    uint256 _amount,
    bytes _data
  );

  event FinalizeDepositERC20(
    address indexed _l1Token,
    address indexed _l2Token,
    address indexed _from,
    address _to,
    uint256 _amount,
    bytes _data
  );

  /**************************************** View Functions ****************************************/

  /// @notice Return the corresponding l1 token address given l2 token address.
  /// @param _l2Token The address of l2 token.
  function getL1ERC20Address(address _l2Token) external view returns (address);

  /// @notice Return the corresponding l2 token address given l1 token address.
  /// @param _l1Token The address of l1 token.
  function getL2ERC20Address(address _l1Token) external view returns (address);

  /**************************************** Mutated Functions ****************************************/

  /// @notice Withdraw of some token to a caller's account on L1.
  /// @dev Make this function payable to send relayer fee in Ether.
  /// @param _token The address of token in L2.
  /// @param _amount The amount of token to transfer.
  /// @param _gasLimit Unused, but included for potential forward compatibility considerations.
  function withdrawERC20(
    address _token,
    uint256 _amount,
    uint256 _gasLimit
  ) external payable;

  /// @notice Withdraw of some token to a recipient's account on L1.
  /// @dev Make this function payable to send relayer fee in Ether.
  /// @param _token The address of token in L2.
  /// @param _to The address of recipient's account on L1.
  /// @param _amount The amount of token to transfer.
  /// @param _gasLimit Unused, but included for potential forward compatibility considerations.
  function withdrawERC20(
    address _token,
    address _to,
    uint256 _amount,
    uint256 _gasLimit
  ) external payable;

  /// @notice Withdraw of some token to a recipient's account on L1 and call.
  /// @dev Make this function payable to send relayer fee in Ether.
  /// @param _token The address of token in L2.
  /// @param _to The address of recipient's account on L1.
  /// @param _amount The amount of token to transfer.
  /// @param _data Optional data to forward to recipient's account.
  /// @param _gasLimit Unused, but included for potential forward compatibility considerations.
  function withdrawERC20AndCall(
    address _token,
    address _to,
    uint256 _amount,
    bytes calldata _data,
    uint256 _gasLimit
  ) external payable;

  /// @notice Complete a deposit from L1 to L2 and send fund to recipient's account in L2.
  /// @dev Make this function payable to handle WETH deposit/withdraw.
  ///      The function should only be called by L2ScrollMessenger.
  ///      The function should also only be called by L1ERC20Gateway in L1.
  /// @param _l1Token The address of corresponding L1 token.
  /// @param _l2Token The address of corresponding L2 token.
  /// @param _from The address of account who deposits the token in L1.
  /// @param _to The address of recipient in L2 to receive the token.
  /// @param _amount The amount of the token to deposit.
  /// @param _data Optional data to forward to recipient's account.
  function finalizeDepositERC20(
    address _l1Token,
    address _l2Token,
    address _from,
    address _to,
    uint256 _amount,
    bytes calldata _data
  ) external payable;
}
