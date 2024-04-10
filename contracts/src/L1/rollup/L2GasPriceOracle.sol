// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {OwnableUpgradeable} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";

import {IWhitelist} from "../../libraries/common/IWhitelist.sol";

import {IL2GasPriceOracle} from "./IL2GasPriceOracle.sol";

// solhint-disable reason-string

contract L2GasPriceOracle is OwnableUpgradeable, IL2GasPriceOracle {
    /**********
     * Events *
     **********/

    /// @notice Emitted when owner updates whitelist contract.
    /// @param _oldWhitelist The address of old whitelist contract.
    /// @param _newWhitelist The address of new whitelist contract.
    event UpdateWhitelist(address _oldWhitelist, address _newWhitelist);

    /// @notice Emitted when current l2 base fee is updated.
    /// @param oldL2BaseFee The original l2 base fee before update.
    /// @param newL2BaseFee The current l2 base fee updated.
    event L2BaseFeeUpdated(uint256 oldL2BaseFee, uint256 newL2BaseFee);

    /// @notice Emitted when intrinsic params are updated.
    /// @param txGas The intrinsic gas for transaction.
    /// @param txGasContractCreation The intrinsic gas for contract creation.
    /// @param zeroGas The intrinsic gas for each zero byte.
    /// @param nonZeroGas The intrinsic gas for each nonzero byte.
    event IntrinsicParamsUpdated(uint256 txGas, uint256 txGasContractCreation, uint256 zeroGas, uint256 nonZeroGas);

    /*************
     * Variables *
     *************/

    /// @notice The latest known l2 base fee.
    uint256 public l2BaseFee;

    /// @notice The address of whitelist contract.
    address public whitelist;

    struct IntrinsicParams {
        // The intrinsic gas for transaction.
        uint64 txGas;
        // The intrinsic gas for contract creation. It is reserved for future use.
        uint64 txGasContractCreation;
        // The intrinsic gas for each zero byte.
        uint64 zeroGas;
        // The intrinsic gas for each nonzero byte.
        uint64 nonZeroGas;
    }

    /// @notice The intrinsic params for transaction.
    IntrinsicParams public intrinsicParams;

    /***************
     * Constructor *
     ***************/

    constructor() {
        _disableInitializers();
    }

    function initialize(
        uint64 _txGas,
        uint64 _txGasContractCreation,
        uint64 _zeroGas,
        uint64 _nonZeroGas
    ) external initializer {
        OwnableUpgradeable.__Ownable_init();

        _setIntrinsicParams(_txGas, _txGasContractCreation, _zeroGas, _nonZeroGas);
    }

    /*************************
     * Public View Functions *
     *************************/

    /// @inheritdoc IL2GasPriceOracle
    function calculateIntrinsicGasFee(bytes memory _message) external view override returns (uint256) {
        // @note currently we don't support contract deployment via L1 messages.
        uint256 _txGas = uint256(intrinsicParams.txGas);
        uint256 _zeroGas = uint256(intrinsicParams.zeroGas);
        uint256 _nonZeroGas = uint256(intrinsicParams.nonZeroGas);

        uint256 gas = _txGas;
        if (_message.length > 0) {
            uint256 nz = 0;
            for (uint256 i = 0; i < _message.length; i++) {
                if (_message[i] != 0) {
                    nz++;
                }
            }
            gas += nz * _nonZeroGas + (_message.length - nz) * _zeroGas;
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

    /// @notice Allows whitelisted caller to modify the l2 base fee.
    /// @param _newL2BaseFee The new l2 base fee.
    function setL2BaseFee(uint256 _newL2BaseFee) external {
        require(IWhitelist(whitelist).isSenderAllowed(_msgSender()), "Not whitelisted sender");

        uint256 _oldL2BaseFee = l2BaseFee;
        l2BaseFee = _newL2BaseFee;

        emit L2BaseFeeUpdated(_oldL2BaseFee, _newL2BaseFee);
    }

    /************************
     * Restricted Functions *
     ************************/

    /// @notice Update whitelist contract.
    /// @dev This function can only called by contract owner.
    /// @param _newWhitelist The address of new whitelist contract.
    function updateWhitelist(address _newWhitelist) external onlyOwner {
        address _oldWhitelist = whitelist;

        whitelist = _newWhitelist;
        emit UpdateWhitelist(_oldWhitelist, _newWhitelist);
    }

    /// @notice Allows the owner to update parameters for intrinsic gas calculation.
    /// @param _txGas The intrinsic gas for transaction.
    /// @param _txGasContractCreation The intrinsic gas for contract creation.
    /// @param _zeroGas The intrinsic gas for each zero byte.
    /// @param _nonZeroGas The intrinsic gas for each nonzero byte.
    function setIntrinsicParams(
        uint64 _txGas,
        uint64 _txGasContractCreation,
        uint64 _zeroGas,
        uint64 _nonZeroGas
    ) external onlyOwner {
        _setIntrinsicParams(_txGas, _txGasContractCreation, _zeroGas, _nonZeroGas);
    }

    /**********************
     * Internal Functions *
     **********************/

    /// @dev Internal function to update parameters for intrinsic gas calculation.
    /// @param _txGas The intrinsic gas for transaction.
    /// @param _txGasContractCreation The intrinsic gas for contract creation.
    /// @param _zeroGas The intrinsic gas for each zero byte.
    /// @param _nonZeroGas The intrinsic gas for each nonzero byte.
    function _setIntrinsicParams(
        uint64 _txGas,
        uint64 _txGasContractCreation,
        uint64 _zeroGas,
        uint64 _nonZeroGas
    ) internal {
        require(_txGas > 0, "txGas is zero");
        require(_zeroGas > 0, "zeroGas is zero");
        require(_nonZeroGas > 0, "nonZeroGas is zero");
        require(_txGasContractCreation > _txGas, "txGasContractCreation is less than txGas");

        intrinsicParams = IntrinsicParams({
            txGas: _txGas,
            txGasContractCreation: _txGasContractCreation,
            zeroGas: _zeroGas,
            nonZeroGas: _nonZeroGas
        });

        emit IntrinsicParamsUpdated(_txGas, _txGasContractCreation, _zeroGas, _nonZeroGas);
    }
}
