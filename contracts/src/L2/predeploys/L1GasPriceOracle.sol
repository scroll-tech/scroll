// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {OwnableBase} from "../../libraries/common/OwnableBase.sol";
import {IWhitelist} from "../../libraries/common/IWhitelist.sol";

import {IL1BlockContainer} from "./IL1BlockContainer.sol";
import {IL1GasPriceOracle} from "./IL1GasPriceOracle.sol";

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
    /// @dev The extra 74 bytes on top of the RLP encoded unsigned transaction data consist of
    ///   4 bytes   Add 4 bytes in the beginning for transaction data length
    ///   1 byte    RLP V prefix
    ///   3 bytes   V
    ///   1 bytes   RLP R prefix
    ///   32 bytes  R
    ///   1 bytes   RLP S prefix
    ///   32 bytes  S
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
            return _unsigned + (74 * 16);
        }
    }

    /*****************************
     * Public Mutating Functions *
     *****************************/

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
