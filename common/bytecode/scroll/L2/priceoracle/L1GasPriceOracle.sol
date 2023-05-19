// File: src/libraries/common/OwnableBase.sol

// SPDX-License-Identifier: MIT

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

// File: src/L2/predeploys/IL1BlockContainer.sol



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

// File: src/L2/predeploys/L1GasPriceOracle.sol



pragma solidity ^0.8.0;



contract L1GasPriceOracle is OwnableBase, IL1GasPriceOracle {
  /**********
   * Events *
   **********/

  /// @notice Emitted when owner updates whitelist contract.
  /// @param _oldWhitelist The address of old whitelist contract.
  /// @param _newWhitelist The address of new whitelist contract.
  event UpdateWhitelist(address _oldWhitelist, address _newWhitelist);

  /*************
   * Constants *
   *************/

  /// @dev The precision used in the scalar.
  uint256 private constant PRECISION = 1e9;

  /// @dev The maximum possible l1 fee overhead.
  ///      Computed based on current l1 block gas limit.
  uint256 private constant MAX_OVERHEAD = 30000000 / 16;

  /// @dev The maximum possible l1 fee scale.
  ///      x1000 should be enough.
  uint256 private constant MAX_SCALE = 1000 * PRECISION;

  /*************
   * Variables *
   *************/

  /// @inheritdoc IL1GasPriceOracle
  uint256 public l1BaseFee;

  /// @inheritdoc IL1GasPriceOracle
  uint256 public override overhead;

  /// @inheritdoc IL1GasPriceOracle
  uint256 public override scalar;

  /// @notice The address of whitelist contract.
  IWhitelist public whitelist;

  /***************
   * Constructor *
   ***************/

  constructor(address _owner) {
    _transferOwnership(_owner);
  }

  /*************************
   * Public View Functions *
   *************************/

  /// @inheritdoc IL1GasPriceOracle
  function getL1Fee(bytes memory _data) external view override returns (uint256) {
    uint256 _l1GasUsed = getL1GasUsed(_data);
    uint256 _l1Fee = _l1GasUsed * l1BaseFee;
    return (_l1Fee * scalar) / PRECISION;
  }

  /// @inheritdoc IL1GasPriceOracle
  /// @dev See the comments in `OVM_GasPriceOracle1` for more details
  ///      https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts/contracts/L2/predeploys/OVM_GasPriceOracle.sol
  function getL1GasUsed(bytes memory _data) public view override returns (uint256) {
    uint256 _total = 0;
    uint256 _length = _data.length;
    unchecked {
      for (uint256 i = 0; i < _length; i++) {
        if (_data[i] == 0) {
          _total += 4;
        } else {
          _total += 16;
        }
      }
      uint256 _unsigned = _total + overhead;
      return _unsigned + (68 * 16);
    }
  }

  /****************************
   * Public Mutated Functions *
   ****************************/

  /// @inheritdoc IL1GasPriceOracle
  function setL1BaseFee(uint256 _l1BaseFee) external override {
    require(whitelist.isSenderAllowed(msg.sender), "Not whitelisted sender");

    l1BaseFee = _l1BaseFee;

    emit L1BaseFeeUpdated(_l1BaseFee);
  }

  /************************
   * Restricted Functions *
   ************************/

  /// @notice Allows the owner to modify the overhead.
  /// @param _overhead New overhead
  function setOverhead(uint256 _overhead) external onlyOwner {
    require(_overhead <= MAX_OVERHEAD, "exceed maximum overhead");

    overhead = _overhead;
    emit OverheadUpdated(_overhead);
  }

  /// Allows the owner to modify the scalar.
  /// @param _scalar New scalar
  function setScalar(uint256 _scalar) external onlyOwner {
    require(_scalar <= MAX_SCALE, "exceed maximum scale");

    scalar = _scalar;
    emit ScalarUpdated(_scalar);
  }

  /// @notice Update whitelist contract.
  /// @dev This function can only called by contract owner.
  /// @param _newWhitelist The address of new whitelist contract.
  function updateWhitelist(address _newWhitelist) external onlyOwner {
    address _oldWhitelist = address(whitelist);

    whitelist = IWhitelist(_newWhitelist);
    emit UpdateWhitelist(_oldWhitelist, _newWhitelist);
  }
}
