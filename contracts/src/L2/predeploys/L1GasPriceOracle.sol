// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { OwnableBase } from "../../libraries/common/OwnableBase.sol";

import { IL1BlockContainer } from "./IL1BlockContainer.sol";
import { IL1GasPriceOracle } from "./IL1GasPriceOracle.sol";

contract L1GasPriceOracle is OwnableBase, IL1GasPriceOracle {
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

  /// @notice The address of L1BlockContainer contract.
  address public immutable blockContainer;

  /*************
   * Variables *
   *************/

  /// @inheritdoc IL1GasPriceOracle
  uint256 public override overhead;

  /// @inheritdoc IL1GasPriceOracle
  uint256 public override scalar;

  /***************
   * Constructor *
   ***************/

  constructor(address _owner, address _blockContainer) {
    _transferOwnership(_owner);

    blockContainer = _blockContainer;
  }

  /*************************
   * Public View Functions *
   *************************/

  /// @inheritdoc IL1GasPriceOracle
  function baseFee() external view override returns (uint256) {
    return block.basefee;
  }

  /// @inheritdoc IL1GasPriceOracle
  function gasPrice() external view override returns (uint256) {
    return block.basefee;
  }

  /// @notice Return the latest known l1 base fee.
  function l1BaseFee() public view override returns (uint256) {
    return IL1BlockContainer(blockContainer).latestBaseFee();
  }

  /// @inheritdoc IL1GasPriceOracle
  function getL1Fee(bytes memory _data) external view override returns (uint256) {
    unchecked {
      uint256 _l1GasUsed = getL1GasUsed(_data);
      uint256 _l1Fee = _l1GasUsed * l1BaseFee();
      return (_l1Fee * scalar) / PRECISION;
    }
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
}
