// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { OwnableUpgradeable } from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";

import { IWhitelist } from "../../libraries/common/IWhitelist.sol";

import { IL2GasPriceOracle } from "./IL2GasPriceOracle.sol";

contract L2GasPriceOracle is OwnableUpgradeable, IL2GasPriceOracle {
  /**********
   * Events *
   **********/

  /// @notice Emitted when owner updates whitelist contract.
  /// @param _oldWhitelist The address of old whitelist contract.
  /// @param _newWhitelist The address of new whitelist contract.
  event UpdateWhitelist(address _oldWhitelist, address _newWhitelist);

  /// @notice Emitted when current fee overhead is updated.
  /// @param overhead The current fee overhead updated.
  event OverheadUpdated(uint256 overhead);

  /// @notice Emitted when current fee scalar is updated.
  /// @param scalar The current fee scalar updated.
  event ScalarUpdated(uint256 scalar);

  /// @notice Emitted when current l2 base fee is updated.
  /// @param l2BaseFee The current l2 base fee updated.
  event L2BaseFeeUpdated(uint256 l2BaseFee);

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

  /// @notice The current l1 fee overhead.
  uint256 public overhead;

  /// @notice The current l1 fee scalar.
  uint256 public scalar;

  /// @notice The latest known l2 base fee.
  uint256 public l2BaseFee;

  /// @notice The address of whitelist contract.
  IWhitelist public whitelist;

  /***************
   * Constructor *
   ***************/

  function initialize() external initializer {
    OwnableUpgradeable.__Ownable_init();
  }

  /*************************
   * Public View Functions *
   *************************/

  /// @notice Return the current l1 base fee.
  function l1BaseFee() public view returns (uint256) {
    return block.basefee;
  }

  /// @inheritdoc IL2GasPriceOracle
  function estimateCrossDomainMessageFee(
    address,
    address,
    bytes memory _message,
    uint256 _gasLimit
  ) external view override returns (uint256) {
    unchecked {
      uint256 _l1GasUsed = getL1GasUsed(_message);
      uint256 _rollupFee = (_l1GasUsed * l1BaseFee() * scalar) / PRECISION;
      uint256 _l2Fee = _gasLimit * l2BaseFee;
      return _l2Fee + _rollupFee;
    }
  }

  /// @notice Computes the amount of L1 gas used for a transaction. Adds the overhead which
  ///         represents the per-transaction gas overhead of posting the transaction and state
  ///         roots to L1. Adds 68 bytes of padding to account for the fact that the input does
  ///         not have a signature.
  /// @param _data Unsigned fully RLP-encoded transaction to get the L1 gas for.
  /// @return Amount of L1 gas used to publish the transaction.
  function getL1GasUsed(bytes memory _data) public view returns (uint256) {
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

  /// @notice Allows the owner to modify the l2 base fee.
  /// @param _l2BaseFee The new l2 base fee.
  function setL2BaseFee(uint256 _l2BaseFee) external {
    require(whitelist.isSenderAllowed(msg.sender), "Not whitelisted sender");

    l2BaseFee = _l2BaseFee;

    emit L2BaseFeeUpdated(_l2BaseFee);
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
  /// @param _scalar The new scalar
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
