// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

contract MockBridgeL2 {
  /*********************************
   * Events from L2ScrollMessenger *
   *********************************/

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

  /******************************
   * Events from L2MessageQueue *
   ******************************/

  /// @notice Emitted when a new message is added to the merkle tree.
  /// @param index The index of the corresponding message.
  /// @param messageHash The hash of the corresponding message.
  event AppendMessage(uint256 index, bytes32 messageHash);

  /********************************
   * Events from L1BlockContainer *
   ********************************/

  /// @notice Emitted when a block is imported.
  /// @param blockHash The hash of the imported block.
  /// @param blockHeight The height of the imported block.
  /// @param blockTimestamp The timestamp of the imported block.
  /// @param stateRoot The state root of the imported block.
  event ImportBlock(bytes32 indexed blockHash, uint256 blockHeight, uint256 blockTimestamp, bytes32 stateRoot);

  /***********
   * Structs *
   ***********/

  struct L1MessageProof {
    bytes32 blockHash;
    bytes stateRootProof;
  }

  /*************
   * Variables *
   *************/

  /// @notice Message nonce, used to avoid relay attack.
  uint256 public messageNonce;

  /************************************
   * Functions from L2ScrollMessenger *
   ************************************/

  function sendMessage(
    address _to,
    uint256 _fee,
    bytes memory _message,
    uint256 _gasLimit
  ) external payable {
    // solhint-disable-next-line not-rely-on-time
    uint256 _deadline = block.timestamp + 1 days;
    uint256 _nonce = messageNonce;
    uint256 _value;
    unchecked {
      _value = msg.value - _fee;
    }
    bytes32 _msghash = keccak256(abi.encodePacked(msg.sender, _to, _value, _fee, _deadline, _nonce, _message));
    emit AppendMessage(_nonce, _msghash);
    emit SentMessage(_to, msg.sender, _value, _fee, _deadline, _message, _nonce, _gasLimit);
    messageNonce = _nonce + 1;
  }

  function relayMessageWithProof(
    address _from,
    address _to,
    uint256 _value,
    uint256 _fee,
    uint256 _deadline,
    uint256 _nonce,
    bytes memory _message,
    L1MessageProof calldata
  ) external {
    bytes32 _msghash = keccak256(abi.encodePacked(_from, _to, _value, _fee, _deadline, _nonce, _message));
    emit RelayedMessage(_msghash);
  }

  /***********************************
   * Functions from L1BlockContainer *
   ***********************************/

  function importBlockHeader(
    bytes32 _blockHash,
    bytes calldata _blockHeaderRLP,
    bytes calldata
  ) external {
    bytes32 _stateRoot;
    uint64 _height;
    uint64 _timestamp;

    assembly {
      // reverts with error `msg`.
      // make sure the length of error string <= 32
      function revertWith(msg) {
        // keccak("Error(string)")
        mstore(0x00, shl(224, 0x08c379a0))
        mstore(0x04, 0x20) // str.offset
        mstore(0x44, msg)
        let msgLen
        for {} msg {} {
          msg := shl(8, msg)
          msgLen := add(msgLen, 1)
        }
        mstore(0x24, msgLen) // str.length
        revert(0x00, 0x64)
      }
      // reverts with `msg` when condition is not matched.
      // make sure the length of error string <= 32
      function require(cond, msg) {
        if iszero(cond) {
          revertWith(msg)
        }
      }
      // returns the calldata offset of the value and the length in bytes
      // for the RLP encoded data item at `ptr`. used in `decodeFlat`
      function decodeValue(ptr) -> dataLen, valueOffset {
        let b0 := byte(0, calldataload(ptr))

        // 0x00 - 0x7f, single byte
        if lt(b0, 0x80) {
          // for a single byte whose value is in the [0x00, 0x7f] range,
          // that byte is its own RLP encoding.
          dataLen := 1
          valueOffset := ptr
          leave
        }

        // 0x80 - 0xb7, short string/bytes, length <= 55
        if lt(b0, 0xb8) {
          // the RLP encoding consists of a single byte with value 0x80
          // plus the length of the string followed by the string.
          dataLen := sub(b0, 0x80)
          valueOffset := add(ptr, 1)
          leave
        }

        // 0xb8 - 0xbf, long string/bytes, length > 55
        if lt(b0, 0xc0) {
          // the RLP encoding consists of a single byte with value 0xb7
          // plus the length in bytes of the length of the string in binary form,
          // followed by the length of the string, followed by the string.
          let lengthBytes := sub(b0, 0xb7)
          if gt(lengthBytes, 4) {
            invalid()
          }

          // load the extended length
          valueOffset := add(ptr, 1)
          let extendedLen := calldataload(valueOffset)
          let bits := sub(256, mul(lengthBytes, 8))
          extendedLen := shr(bits, extendedLen)

          dataLen := extendedLen
          valueOffset := add(valueOffset, lengthBytes)
          leave
        }

        revertWith("Not value")
      }

      let ptr := _blockHeaderRLP.offset
      let headerPayloadLength
      {
        let b0 := byte(0, calldataload(ptr))
        // the input should be a long list
        if lt(b0, 0xf8) {
          invalid()
        }
        let lengthBytes := sub(b0, 0xf7)
        if gt(lengthBytes, 32) {
          invalid()
        }
        // load the extended length
        ptr := add(ptr, 1)
        headerPayloadLength := calldataload(ptr)
        let bits := sub(256, mul(lengthBytes, 8))
        // compute payload length: extended length + length bytes + 1
        headerPayloadLength := shr(bits, headerPayloadLength)
        headerPayloadLength := add(headerPayloadLength, lengthBytes)
        headerPayloadLength := add(headerPayloadLength, 1)
        ptr := add(ptr, lengthBytes)
      }

      let memPtr := mload(0x40)
      calldatacopy(memPtr, _blockHeaderRLP.offset, headerPayloadLength)
      let _computedBlockHash := keccak256(memPtr, headerPayloadLength)
      require(eq(_blockHash, _computedBlockHash), "Block hash mismatch")
      
      // load 16 vaules
      for { let i := 0 } lt(i, 16) { i := add(i, 1) } {
        let len, offset := decodeValue(ptr)
        // the value we care must have at most 32 bytes
        if lt(len, 33) {
          let bits := mul( sub(32, len), 8)
          let value := calldataload(offset)
          value := shr(bits, value)
          mstore(memPtr, value)
        }
        memPtr := add(memPtr, 0x20)
        ptr := add(len, offset)
      }
      require(eq(ptr, add(_blockHeaderRLP.offset, _blockHeaderRLP.length)), "Header RLP length mismatch")

      memPtr := mload(0x40)
      // load state root, 4-th entry
      _stateRoot := mload(add(memPtr, 0x60))
      // load block height, 9-th entry
      _height := mload(add(memPtr, 0x100))
      // load block timestamp, 12-th entry
      _timestamp := mload(add(memPtr, 0x160))
    }

    emit ImportBlock(_blockHash, _height, _timestamp, _stateRoot);
  }
}
