// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

interface IScrollMessenger {
  /**************************************** Events ****************************************/

  event SentMessage(
    address indexed target,
    address sender,
    uint256 value,
    uint256 fee,
    uint256 deadline,
    bytes message,
    uint256 messageNonce,
    uint256 gasLimit
  );
  event MessageDropped(bytes32 indexed msgHash);
  event RelayedMessage(bytes32 indexed msgHash);
  event FailedRelayedMessage(bytes32 indexed msgHash);

  /**************************************** View Functions ****************************************/

  function xDomainMessageSender() external view returns (address);

  /**************************************** Mutated Functions ****************************************/

  /// @notice Send cross chain message (L1 => L2 or L2 => L1)
  /// @dev Currently, only privileged accounts can call this function for safty. And adding an extra
  /// `_fee` variable make it more easy to upgrade to decentralized version.
  /// @param _to The address of account who recieve the message.
  /// @param _fee The amount of fee in Ether the caller would like to pay to the relayer.
  /// @param _message The content of the message.
  /// @param _gasLimit Unused, but included for potential forward compatibility considerations.
  function sendMessage(
    address _to,
    uint256 _fee,
    bytes memory _message,
    uint256 _gasLimit
  ) external payable;

  // @todo add comments
  function dropMessage(
    address _from,
    address _to,
    uint256 _value,
    uint256 _fee,
    uint256 _deadline,
    uint256 _nonce,
    bytes memory _message,
    uint256 _gasLimit
  ) external;
}
