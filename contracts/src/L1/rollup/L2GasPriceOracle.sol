// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {OwnableUpgradeable} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";

import {IWhitelist} from "../../libraries/common/IWhitelist.sol";

import {IL2GasPriceOracle} from "./IL2GasPriceOracle.sol";

contract L2GasPriceOracle is OwnableUpgradeable, IL2GasPriceOracle {
    /**********
     * Events *
     **********/

    /// @notice Emitted when owner updates whitelist contract.
    /// @param _oldWhitelist The address of old whitelist contract.
    /// @param _newWhitelist The address of new whitelist contract.
    event UpdateWhitelist(address _oldWhitelist, address _newWhitelist);

    /// @notice Emitted when current l2 base fee is updated.
    /// @param l2BaseFee The current l2 base fee updated.
    event L2BaseFeeUpdated(uint256 l2BaseFee);

    /// @notice Emitted when intrinsic params are updated.
    /// @param txGas The intrinsic gas for transaction.
    event IntrinsicParamsUpdated(uint256 txGas, uint256 zeroGas, uint256 nonZeroGas);

    /*************
     * Variables *
     *************/

    /// @notice The latest known l2 base fee.
    uint256 public l2BaseFee;

    /// @notice The address of whitelist contract.
    IWhitelist public whitelist;


    // todo, initialize in constructor
    uint256 public txGas;
    uint256 public zeroGas;
    uint256 public nonZeroGas;
    
    uint256 immutable MAX_UINT_64 = 2**64-1;

    /***************
     * Constructor *
     ***************/

    function initialize(uint256 _txGas, uint256 _zeroGas, uint256 _nonZeroGas) external initializer {
        OwnableUpgradeable.__Ownable_init();

        txGas = _txGas;
        zeroGas = _zeroGas;
        nonZeroGas = _nonZeroGas;
    }

    /*************************
     * Public View Functions *
     *************************/

    /// @inheritdoc IL2GasPriceOracle
    function calculateIntrinsicGasFee(bytes memory _message) external view override returns (uint256) {
        uint256 gas = txGas;
        if (_message.length > 0) {
            uint256 nz = 0;
            for (uint256 i = 0; i < _message.length; i++) {
                if (_message[i] != 0) {
                    nz++;
                }
            }
            if (nonZeroGas > 0) {
                require((MAX_UINT_64 - txGas) / nonZeroGas > nz, "Intrinsic gas overflows from nonzero bytes cost");
            }

            gas += nz * nonZeroGas;
            
            uint256 z = uint256(_message.length) - nz;
            
            if (zeroGas > 0) {
                require((MAX_UINT_64 - txGas) / zeroGas > z, "Intrinsic gas overflows from zero bytes cost");
            }
            gas += z * zeroGas;
        }
        return uint256(gas);
    }

    /// @inheritdoc IL2GasPriceOracle
    function estimateCrossDomainMessageFee(uint256 _gasLimit) external view override returns (uint256) {
        return _gasLimit * l2BaseFee;
    }

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @notice Allows the owner to update parameters for intrinsic gas calculation.
    /// @param _txGas The intrinsic gas for transaction.
    /// @param _zeroGas The intrinsic gas for each zero byte.
    /// @param _nonZeroGas The intrinsic gas for each nonzero byte.
    function setIntrinsicParams(uint256 _txGas, uint256 _zeroGas, uint256 _nonZeroGas) public {
        require(whitelist.isSenderAllowed(msg.sender), "Not whitelisted sender");

        txGas = _txGas;
        zeroGas = _zeroGas;
        nonZeroGas = _nonZeroGas;

        emit IntrinsicParamsUpdated(_txGas, _zeroGas, _nonZeroGas);
    }

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

    /// @notice Update whitelist contract.
    /// @dev This function can only called by contract owner.
    /// @param _newWhitelist The address of new whitelist contract.
    function updateWhitelist(address _newWhitelist) external onlyOwner {
        address _oldWhitelist = address(whitelist);

        whitelist = IWhitelist(_newWhitelist);
        emit UpdateWhitelist(_oldWhitelist, _newWhitelist);
    }
}
