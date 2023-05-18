// File: src/libraries/IScrollMessenger.sol

// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

interface IScrollMessenger {
  /**********
   * Events *
   **********/

  /// @notice Emitted when a cross domain message is sent.
  /// @param sender The address of the sender who initiates the message.
  /// @param target The address of target contract to call.
  /// @param value The amount of value passed to the target contract.
  /// @param messageNonce The nonce of the message.
  /// @param gasLimit The optional gas limit passed to L1 or L2.
  /// @param message The calldata passed to the target contract.
  event SentMessage(
    address indexed sender,
    address indexed target,
    uint256 value,
    uint256 messageNonce,
    uint256 gasLimit,
    bytes message
  );

  /// @notice Emitted when a cross domain message is relayed successfully.
  /// @param messageHash The hash of the message.
  event RelayedMessage(bytes32 indexed messageHash);

  /// @notice Emitted when a cross domain message is failed to relay.
  /// @param messageHash The hash of the message.
  event FailedRelayedMessage(bytes32 indexed messageHash);

  /*************************
   * Public View Functions *
   *************************/

  /// @notice Return the sender of a cross domain message.
  function xDomainMessageSender() external view returns (address);

  /****************************
   * Public Mutated Functions *
   ****************************/

  /// @notice Send cross chain message from L1 to L2 or L2 to L1.
  /// @param target The address of account who recieve the message.
  /// @param value The amount of ether passed when call target contract.
  /// @param message The content of the message.
  /// @param gasLimit Gas limit required to complete the message relay on corresponding chain.
  function sendMessage(
    address target,
    uint256 value,
    bytes calldata message,
    uint256 gasLimit
  ) external payable;
}

// File: src/L2/IL2ScrollMessenger.sol



pragma solidity ^0.8.0;

interface IL2ScrollMessenger is IScrollMessenger {
  /***********
   * Structs *
   ***********/

  struct L1MessageProof {
    bytes32 blockHash;
    bytes stateRootProof;
  }

  /****************************
   * Public Mutated Functions *
   ****************************/

  /// @notice execute L1 => L2 message
  /// @dev Make sure this is only called by privileged accounts.
  /// @param from The address of the sender of the message.
  /// @param to The address of the recipient of the message.
  /// @param value The msg.value passed to the message call.
  /// @param nonce The nonce of the message to avoid replay attack.
  /// @param message The content of the message.
  function relayMessage(
    address from,
    address to,
    uint256 value,
    uint256 nonce,
    bytes calldata message
  ) external;

  /// @notice execute L1 => L2 message with proof
  /// @param from The address of the sender of the message.
  /// @param to The address of the recipient of the message.
  /// @param value The msg.value passed to the message call.
  /// @param nonce The nonce of the message to avoid replay attack.
  /// @param message The content of the message.
  /// @param proof The message proof.
  function retryMessageWithProof(
    address from,
    address to,
    uint256 value,
    uint256 nonce,
    bytes calldata message,
    L1MessageProof calldata proof
  ) external;
}

// File: src/libraries/common/OwnableBase.sol



pragma solidity ^0.8.0;

abstract contract OwnableBase {
  /**********
   * Events *
   **********/

  /// @notice Emitted when owner is changed by current owner.
  /// @param _oldOwner The address of previous owner.
  /// @param _newOwner The address of new owner.
  event OwnershipTransferred(address indexed _oldOwner, address indexed _newOwner);

  /*************
   * Variables *
   *************/

  /// @notice The address of the current owner.
  address public owner;

  /**********************
   * Function Modifiers *
   **********************/

  /// @dev Throws if called by any account other than the owner.
  modifier onlyOwner() {
    require(owner == msg.sender, "caller is not the owner");
    _;
  }

  /************************
   * Restricted Functions *
   ************************/

  /// @notice Leaves the contract without owner. It will not be possible to call
  /// `onlyOwner` functions anymore. Can only be called by the current owner.
  ///
  /// @dev Renouncing ownership will leave the contract without an owner,
  /// thereby removing any functionality that is only available to the owner.
  function renounceOwnership() public onlyOwner {
    _transferOwnership(address(0));
  }

  /// @notice Transfers ownership of the contract to a new account (`newOwner`).
  /// Can only be called by the current owner.
  function transferOwnership(address _newOwner) public onlyOwner {
    require(_newOwner != address(0), "new owner is the zero address");
    _transferOwnership(_newOwner);
  }

  /**********************
   * Internal Functions *
   **********************/

  /// @dev Transfers ownership of the contract to a new account (`newOwner`).
  /// Internal function without access restriction.
  function _transferOwnership(address _newOwner) internal {
    address _oldOwner = owner;
    owner = _newOwner;
    emit OwnershipTransferred(_oldOwner, _newOwner);
  }
}

// File: src/libraries/FeeVault.sol



// MIT License

// Copyright (c) 2022 Optimism
// Copyright (c) 2022 Scroll

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

pragma solidity ^0.8.0;


/// @title FeeVault
/// @notice The FeeVault contract contains the basic logic for the various different vault contracts
///         used to hold fee revenue generated by the L2 system.
abstract contract FeeVault is OwnableBase {
  /// @notice Emits each time that a withdrawal occurs.
  ///
  /// @param value Amount that was withdrawn (in wei).
  /// @param to    Address that the funds were sent to.
  /// @param from  Address that triggered the withdrawal.
  event Withdrawal(uint256 value, address to, address from);

  /// @notice Minimum balance before a withdrawal can be triggered.
  uint256 public minWithdrawAmount;

  /// @notice Scroll L2 messenger address.
  address public messenger;

  /// @notice Wallet that will receive the fees on L1.
  address public recipient;

  /// @notice Total amount of wei processed by the contract.
  uint256 public totalProcessed;

  /// @param _owner               The owner of the contract.
  /// @param _recipient           Wallet that will receive the fees on L1.
  /// @param _minWithdrawalAmount Minimum balance before a withdrawal can be triggered.
  constructor(
    address _owner,
    address _recipient,
    uint256 _minWithdrawalAmount
  ) {
    _transferOwnership(_owner);

    minWithdrawAmount = _minWithdrawalAmount;
    recipient = _recipient;
  }

  /// @notice Allow the contract to receive ETH.
  receive() external payable {}

  /// @notice Triggers a withdrawal of funds to the L1 fee wallet.
  function withdraw() external {
    uint256 value = address(this).balance;

    require(value >= minWithdrawAmount, "FeeVault: withdrawal amount must be greater than minimum withdrawal amount");

    unchecked {
      totalProcessed += value;
    }

    emit Withdrawal(value, recipient, msg.sender);

    // no fee provided
    IL2ScrollMessenger(messenger).sendMessage{ value: value }(
      recipient,
      value,
      bytes(""), // no message (simple eth transfer)
      0 // _gasLimit can be zero for fee vault.
    );
  }

  /// @notice Update the address of messenger.
  /// @param _messenger The address of messenger to update.
  function updateMessenger(address _messenger) external onlyOwner {
    messenger = _messenger;
  }

  /// @notice Update the address of recipient.
  /// @param _recipient The address of recipient to update.
  function updateRecipient(address _recipient) external onlyOwner {
    recipient = _recipient;
  }

  /// @notice Update the minimum withdraw amount.
  /// @param _minWithdrawAmount The minimum withdraw amount to update.
  function updateMinWithdrawAmount(uint256 _minWithdrawAmount) external onlyOwner {
    minWithdrawAmount = _minWithdrawAmount;
  }
}

// File: src/L2/predeploys/L2TxFeeVault.sol



pragma solidity ^0.8.0;

/// @title L2TxFeeVault
/// @notice The `L2TxFeeVault` contract collects all L2 transaction fees and allows withdrawing these fees to a predefined L1 address.
/// The minimum withdrawal amount is 10 ether.
contract L2TxFeeVault is FeeVault {
  /// @param _owner The owner of the contract.
  /// @param _recipient The fee recipient address on L1.
  constructor(address _owner, address _recipient) FeeVault(_owner, _recipient, 10 ether) {}
}
