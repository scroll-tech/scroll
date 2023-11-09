// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

import {IWhitelist} from "../../libraries/common/IWhitelist.sol";
import {IL1MessageQueue} from "./IL1MessageQueue.sol";
import {IL1MessageQueueWithOracle} from "./IL1MessageQueueWithOracle.sol";
import {IL2GasPriceOracle} from "./IL2GasPriceOracle.sol";

import {L1MessageQueue} from "./L1MessageQueue.sol";

contract L1MessageQueueWithGasPriceOracle is L1MessageQueue, IL1MessageQueueWithOracle {
    /***********
     * Structs *
     ***********/

    /// @dev The struct for intrinsic gas parameters.
    /// @param txGas The intrinsic gas for transaction.
    /// @param txGasContractCreation The intrinsic gas for contract creation. It is reserved for future use.
    /// @param zeroGas The intrinsic gas for each zero byte.
    /// @param nonZeroGas The intrinsic gas for each nonzero byte.
    struct IntrinsicParams {
        uint64 txGas;
        uint64 txGasContractCreation;
        uint64 zeroGas;
        uint64 nonZeroGas;
    }

    /*************
     * Variables *
     *************/

    /// @notice The latest known l2 base fee.
    uint256 public l2BaseFee;

    /// @notice The address of whitelist contract.
    address public whitelist;

    /// @notice The intrinsic params for transaction.
    IntrinsicParams public intrinsicParams;

    /***************
     * Constructor *
     ***************/

    /// @notice Constructor for `L1MessageQueueWithGasPriceOracle` implementation contract.
    ///
    /// @param _messenger The address of `L1ScrollMessenger` contract.
    /// @param _scrollChain The address of `ScrollChain` contract.
    constructor(address _messenger, address _scrollChain) L1MessageQueue(_messenger, _scrollChain) {}

    /// @notice Initialize the storage of L1MessageQueueWithGasPriceOracle.
    function initializeV2() external reinitializer(2) {
        l2BaseFee = IL2GasPriceOracle(gasOracle).l2BaseFee();
        whitelist = IL2GasPriceOracle(gasOracle).whitelist();
        intrinsicParams = IntrinsicParams({txGas: 21000, txGasContractCreation: 53000, zeroGas: 4, nonZeroGas: 16});
    }

    /*************************
     * Public View Functions *
     *************************/

    /// @inheritdoc IL1MessageQueue
    function estimateCrossDomainMessageFee(uint256 _gasLimit)
        external
        view
        override(IL1MessageQueue, L1MessageQueue)
        returns (uint256)
    {
        return _gasLimit * l2BaseFee;
    }

    /// @inheritdoc IL1MessageQueue
    function calculateIntrinsicGasFee(bytes calldata _calldata)
        public
        view
        override(IL1MessageQueue, L1MessageQueue)
        returns (uint256)
    {
        IntrinsicParams memory _cachedIntrinsicParams = intrinsicParams;
        // no way this can overflow `uint256`
        unchecked {
            return
                uint256(_cachedIntrinsicParams.txGas) + _calldata.length * uint256(_cachedIntrinsicParams.nonZeroGas);
        }
    }

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @notice Allows whitelisted caller to modify the l2 base fee.
    /// @param _newL2BaseFee The new l2 base fee.
    function setL2BaseFee(uint256 _newL2BaseFee) external {
        if (!IWhitelist(whitelist).isSenderAllowed(_msgSender())) {
            revert ErrorNotWhitelistedSender();
        }

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
        if (_txGas == 0) revert ErrorTxGasIsZero();
        if (_zeroGas == 0) revert ErrorZeroGasIsZero();
        if (_nonZeroGas == 0) revert ErrorNonZeroGasIsZero();
        if (_txGasContractCreation <= _txGas) revert ErrorTxGasContractCreationLessThanTxGas();

        intrinsicParams = IntrinsicParams({
            txGas: _txGas,
            txGasContractCreation: _txGasContractCreation,
            zeroGas: _zeroGas,
            nonZeroGas: _nonZeroGas
        });

        emit IntrinsicParamsUpdated(_txGas, _txGasContractCreation, _zeroGas, _nonZeroGas);
    }
}
