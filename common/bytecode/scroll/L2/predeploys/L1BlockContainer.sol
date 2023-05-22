// File: src/L2/predeploys/IL1BlockContainer.sol

// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

interface IL1BlockContainer {
  /**********
   * Events *
   **********/

  /// @notice Emitted when a block is imported.
  /// @param blockHash The hash of the imported block.
  /// @param blockHeight The height of the imported block.
  /// @param blockTimestamp The timestamp of the imported block.
  /// @param baseFee The base fee of the imported block.
  /// @param stateRoot The state root of the imported block.
  event ImportBlock(
    bytes32 indexed blockHash,
    uint256 blockHeight,
    uint256 blockTimestamp,
    uint256 baseFee,
    bytes32 stateRoot
  );

  /*************************
   * Public View Functions *
   *************************/

  /// @notice Return the latest imported block hash
  function latestBlockHash() external view returns (bytes32);

  /// @notice Return the latest imported L1 base fee
  function latestBaseFee() external view returns (uint256);

  /// @notice Return the latest imported block number
  function latestBlockNumber() external view returns (uint256);

  /// @notice Return the latest imported block timestamp
  function latestBlockTimestamp() external view returns (uint256);

  /// @notice Return the state root of given block.
  /// @param blockHash The block hash to query.
  /// @return stateRoot The state root of the block.
  function getStateRoot(bytes32 blockHash) external view returns (bytes32 stateRoot);

  /// @notice Return the block timestamp of given block.
  /// @param blockHash The block hash to query.
  /// @return timestamp The corresponding block timestamp.
  function getBlockTimestamp(bytes32 blockHash) external view returns (uint256 timestamp);

  /****************************
   * Public Mutated Functions *
   ****************************/

  /// @notice Import L1 block header to this contract.
  /// @param blockHash The hash of block.
  /// @param blockHeaderRLP The RLP encoding of L1 block.
  /// @param updateGasPriceOracle Whether to update gas price oracle.
  function importBlockHeader(
    bytes32 blockHash,
    bytes calldata blockHeaderRLP,
    bool updateGasPriceOracle
  ) external;
}

// File: src/L2/predeploys/IL1GasPriceOracle.sol



pragma solidity ^0.8.0;

interface IL1GasPriceOracle {
  /**********
   * Events *
   **********/

  /// @notice Emitted when current fee overhead is updated.
  /// @param overhead The current fee overhead updated.
  event OverheadUpdated(uint256 overhead);

  /// @notice Emitted when current fee scalar is updated.
  /// @param scalar The current fee scalar updated.
  event ScalarUpdated(uint256 scalar);

  /// @notice Emitted when current l1 base fee is updated.
  /// @param l1BaseFee The current l1 base fee updated.
  event L1BaseFeeUpdated(uint256 l1BaseFee);

  /*************************
   * Public View Functions *
   *************************/

  /// @notice Return the current l1 fee overhead.
  function overhead() external view returns (uint256);

  /// @notice Return the current l1 fee scalar.
  function scalar() external view returns (uint256);

  /// @notice Return the latest known l1 base fee.
  function l1BaseFee() external view returns (uint256);

  /// @notice Computes the L1 portion of the fee based on the size of the rlp encoded input
  ///         transaction, the current L1 base fee, and the various dynamic parameters.
  /// @param data Unsigned fully RLP-encoded transaction to get the L1 fee for.
  /// @return L1 fee that should be paid for the tx
  function getL1Fee(bytes memory data) external view returns (uint256);

  /// @notice Computes the amount of L1 gas used for a transaction. Adds the overhead which
  ///         represents the per-transaction gas overhead of posting the transaction and state
  ///         roots to L1. Adds 68 bytes of padding to account for the fact that the input does
  ///         not have a signature.
  /// @param data Unsigned fully RLP-encoded transaction to get the L1 gas for.
  /// @return Amount of L1 gas used to publish the transaction.
  function getL1GasUsed(bytes memory data) external view returns (uint256);

  /****************************
   * Public Mutated Functions *
   ****************************/

  /// @notice Allows whitelisted caller to modify the l1 base fee.
  /// @param _l1BaseFee New l1 base fee.
  function setL1BaseFee(uint256 _l1BaseFee) external;
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

// File: src/libraries/common/IWhitelist.sol



pragma solidity ^0.8.0;

interface IWhitelist {
  /// @notice Check whether the sender is allowed to do something.
  /// @param _sender The address of sender.
  function isSenderAllowed(address _sender) external view returns (bool);
}

// File: src/libraries/constants/ScrollPredeploy.sol



pragma solidity ^0.8.0;

library ScrollPredeploy {
  address internal constant L1_MESSAGE_QUEUE = 0x5300000000000000000000000000000000000000;

  address internal constant L1_BLOCK_CONTAINER = 0x5300000000000000000000000000000000000001;

  address internal constant L1_GAS_PRICE_ORACLE = 0x5300000000000000000000000000000000000002;
}

// File: src/L2/predeploys/L1BlockContainer.sol



pragma solidity ^0.8.0;




/// @title L1BlockContainer
/// @notice This contract will maintain the list of blocks proposed in L1.
contract L1BlockContainer is OwnableBase, IL1BlockContainer {
  /**********
   * Events *
   **********/

  /// @notice Emitted when owner updates whitelist contract.
  /// @param _oldWhitelist The address of old whitelist contract.
  /// @param _newWhitelist The address of new whitelist contract.
  event UpdateWhitelist(address _oldWhitelist, address _newWhitelist);

  /***********
   * Structs *
   ***********/

  /// @dev Compiler will pack this into single `uint256`.
  struct BlockMetadata {
    // The block height.
    uint64 height;
    // The block timestamp.
    uint64 timestamp;
    // The base fee in the block.
    uint128 baseFee;
  }

  /*************
   * Variables *
   *************/

  /// @notice The address of whitelist contract.
  IWhitelist public whitelist;

  // @todo change to ring buffer to save gas usage.

  /// @inheritdoc IL1BlockContainer
  bytes32 public override latestBlockHash;

  /// @notice Mapping from block hash to corresponding state root.
  mapping(bytes32 => bytes32) public stateRoot;

  /// @notice Mapping from block hash to corresponding block metadata,
  /// including timestamp and height.
  mapping(bytes32 => BlockMetadata) public metadata;

  /***************
   * Constructor *
   ***************/

  constructor(address _owner) {
    _transferOwnership(_owner);
  }

  function initialize(
    bytes32 _startBlockHash,
    uint64 _startBlockHeight,
    uint64 _startBlockTimestamp,
    uint128 _startBlockBaseFee,
    bytes32 _startStateRoot
  ) external onlyOwner {
    require(latestBlockHash == bytes32(0), "already initialized");

    latestBlockHash = _startBlockHash;
    stateRoot[_startBlockHash] = _startStateRoot;
    metadata[_startBlockHash] = BlockMetadata(_startBlockHeight, _startBlockTimestamp, _startBlockBaseFee);

    emit ImportBlock(_startBlockHash, _startBlockHeight, _startBlockTimestamp, _startBlockBaseFee, _startStateRoot);
  }

  /*************************
   * Public View Functions *
   *************************/

  /// @inheritdoc IL1BlockContainer
  function latestBaseFee() external view override returns (uint256) {
    return metadata[latestBlockHash].baseFee;
  }

  /// @inheritdoc IL1BlockContainer
  function latestBlockNumber() external view override returns (uint256) {
    return metadata[latestBlockHash].height;
  }

  /// @inheritdoc IL1BlockContainer
  function latestBlockTimestamp() external view override returns (uint256) {
    return metadata[latestBlockHash].timestamp;
  }

  /// @inheritdoc IL1BlockContainer
  function getStateRoot(bytes32 _blockHash) external view returns (bytes32) {
    return stateRoot[_blockHash];
  }

  /// @inheritdoc IL1BlockContainer
  function getBlockTimestamp(bytes32 _blockHash) external view returns (uint256) {
    return metadata[_blockHash].timestamp;
  }

  /****************************
   * Public Mutated Functions *
   ****************************/

  /// @inheritdoc IL1BlockContainer
  function importBlockHeader(
    bytes32 _blockHash,
    bytes calldata _blockHeaderRLP,
    bool _updateGasPriceOracle
  ) external {
    // @todo remove this when ETH 2.0 signature verification is ready.
    {
      IWhitelist _whitelist = whitelist;
      require(address(_whitelist) == address(0) || _whitelist.isSenderAllowed(msg.sender), "Not whitelisted sender");
    }

    // The encoding order in block header is
    // 1. ParentHash: 32 bytes
    // 2. UncleHash: 32 bytes
    // 3. Coinbase: 20 bytes
    // 4. StateRoot: 32 bytes
    // 5. TransactionsRoot: 32 bytes
    // 6. ReceiptsRoot: 32 bytes
    // 7. LogsBloom: 256 bytes
    // 8. Difficulty: uint
    // 9. BlockHeight: uint
    // 10. GasLimit: uint64
    // 11. GasUsed: uint64
    // 12. BlockTimestamp: uint64
    // 13. ExtraData: several bytes
    // 14. MixHash: 32 bytes
    // 15. BlockNonce: 8 bytes
    // 16. BaseFee: uint // optional
    bytes32 _parentHash;
    bytes32 _stateRoot;
    uint64 _height;
    uint64 _timestamp;
    uint128 _baseFee;

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
      // load parent hash, 1-st entry
      _parentHash := mload(memPtr) 
      // load state root, 4-th entry
      _stateRoot := mload(add(memPtr, 0x60))
      // load block height, 9-th entry
      _height := mload(add(memPtr, 0x100))
      // load block timestamp, 12-th entry
      _timestamp := mload(add(memPtr, 0x160))
      // load base fee, 16-th entry
      _baseFee := mload(add(memPtr, 0x1e0))
    }
    require(stateRoot[_parentHash] != bytes32(0), "Parent not imported");
    BlockMetadata memory _parentMetadata = metadata[_parentHash];
    require(_parentMetadata.height + 1 == _height, "Block height mismatch");
    require(_parentMetadata.timestamp <= _timestamp, "Parent block has larger timestamp");

    latestBlockHash = _blockHash;
    stateRoot[_blockHash] = _stateRoot;
    metadata[_blockHash] = BlockMetadata(_height, _timestamp, _baseFee);

    emit ImportBlock(_blockHash, _height, _timestamp, _baseFee, _stateRoot);

    if (_updateGasPriceOracle) {
      IL1GasPriceOracle(ScrollPredeploy.L1_GAS_PRICE_ORACLE).setL1BaseFee(_baseFee);
    }
  }

  /************************
   * Restricted Functions *
   ************************/

  /// @notice Update whitelist contract.
  /// @dev This function can only called by contract owner.
  /// @param _newWhitelist The address of new whitelist contract.
  function updateWhitelist(address _newWhitelist) external onlyOwner {
    address _oldWhitelist = address(whitelist);

    whitelist = IWhitelist(_newWhitelist);
    emit UpdateWhitelist(_oldWhitelist, _newWhitelist);
  }
}
