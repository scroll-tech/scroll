// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

import {IWhitelist} from "../../libraries/common/IWhitelist.sol";
import {IL1MessageQueue} from "./IL1MessageQueue.sol";
import {IL1MessageQueueWithGasPriceOracle} from "./IL1MessageQueueWithGasPriceOracle.sol";
import {IL2GasPriceOracle} from "./IL2GasPriceOracle.sol";

import {L1MessageQueue} from "./L1MessageQueue.sol";

contract L1MessageQueueWithGasPriceOracle is L1MessageQueue, IL1MessageQueueWithGasPriceOracle {
    /*************
     * Constants *
     *************/

    /// @notice The intrinsic gas for transaction.
    uint256 private constant INTRINSIC_GAS_TX = 21000;

    /// @notice The intrinsic gas for each nonzero byte.
    uint256 private constant INTRINSIC_GAS_NONZERO_BYTE = 16;

    /*************
     * Variables *
     *************/

    /// @notice The latest known l2 base fee.
    uint256 public l2BaseFee;

    /// @notice The address of whitelist contract.
    address public whitelist;

    /***************
     * Constructor *
     ***************/

    /// @notice Constructor for `L1MessageQueueWithGasPriceOracle` implementation contract.
    ///
    /// @param _messenger The address of `L1ScrollMessenger` contract.
    /// @param _scrollChain The address of `ScrollChain` contract.
    /// @param _enforcedTxGateway The address of `EnforcedTxGateway` contract.
    constructor(
        address _messenger,
        address _scrollChain,
        address _enforcedTxGateway
    ) L1MessageQueue(_messenger, _scrollChain, _enforcedTxGateway) {}

    /// @notice Initialize the storage of L1MessageQueueWithGasPriceOracle.
    function initializeV2() external reinitializer(2) {
        l2BaseFee = IL2GasPriceOracle(gasOracle).l2BaseFee();
        whitelist = IL2GasPriceOracle(gasOracle).whitelist();
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
        pure
        override(IL1MessageQueue, L1MessageQueue)
        returns (uint256)
    {
        // no way this can overflow `uint256`
        unchecked {
            return INTRINSIC_GAS_TX + _calldata.length * INTRINSIC_GAS_NONZERO_BYTE;
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

        emit UpdateL2BaseFee(_oldL2BaseFee, _newL2BaseFee);
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
}
