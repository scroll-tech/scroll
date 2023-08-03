// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

contract MockBridgeL2 {
  /******************************
   * Events from L2MessageQueue *
   ******************************/

  /// @notice Emitted when a new message is added to the merkle tree.
  /// @param index The index of the corresponding message.
  /// @param messageHash The hash of the corresponding message.
  event AppendMessage(uint256 index, bytes32 messageHash);

  /*********************************
   * Events from L2ScrollMessenger *
   *********************************/

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

  /*************
   * Variables *
   *************/

  /// @notice Message nonce, used to avoid relay attack.
  uint256 public messageNonce;


  /***********************************
   * Functions from ScrollChain *
   ***********************************/

  function importGenesisBatch(bytes calldata _batchHeader, bytes32 _stateRoot) external {
  }

  /***********************************
   * Functions from L1GasPriceOracle *
   ***********************************/

  function setL1BaseFee(uint256) external {
  }

  /************************************
   * Functions from L2ScrollMessenger *
   ************************************/

  function sendMessage(
    address _to,
    uint256 _value,
    bytes memory _message,
    uint256 _gasLimit
  ) external payable {
    bytes memory _xDomainCalldata = _encodeXDomainCalldata(msg.sender, _to, _value, messageNonce, _message);
    bytes32 _xDomainCalldataHash = keccak256(_xDomainCalldata);

    emit AppendMessage(messageNonce, _xDomainCalldataHash);
    emit SentMessage(msg.sender, _to, _value, messageNonce, _gasLimit, _message);

    messageNonce += 1;
  }

  function relayMessage(
    address _from,
    address _to,
    uint256 _value,
    uint256 _nonce,
    bytes calldata _message
  ) external {
    bytes memory _xDomainCalldata = _encodeXDomainCalldata(_from, _to, _value, _nonce, _message);
    bytes32 _xDomainCalldataHash = keccak256(_xDomainCalldata);
    emit RelayedMessage(_xDomainCalldataHash);
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
