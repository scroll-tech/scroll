// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

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

    /// @notice The appropriate intrinsic gas for each byte.
    uint256 private constant APPROPRIATE_INTRINSIC_GAS_PER_BYTE = 16;

    /*************
     * Variables *
     *************/

    /// @inheritdoc IL1MessageQueueWithGasPriceOracle
    uint256 public override l2BaseFee;

    /// @inheritdoc IL1MessageQueueWithGasPriceOracle
    address public override whitelistChecker;

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
        whitelistChecker = IL2GasPriceOracle(gasOracle).whitelist();
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
            return INTRINSIC_GAS_TX + _calldata.length * APPROPRIATE_INTRINSIC_GAS_PER_BYTE;
        }
    }

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @notice Allows whitelistCheckered caller to modify the l2 base fee.
    /// @param _newL2BaseFee The new l2 base fee.
    function setL2BaseFee(uint256 _newL2BaseFee) external {
        if (!IWhitelist(whitelistChecker).isSenderAllowed(_msgSender())) {
            revert ErrorNotWhitelistedSender();
        }

        uint256 _oldL2BaseFee = l2BaseFee;
        l2BaseFee = _newL2BaseFee;

        emit UpdateL2BaseFee(_oldL2BaseFee, _newL2BaseFee);
    }

    /************************
     * Restricted Functions *
     ************************/

    /// @notice Update whitelist checker contract.
    /// @dev This function can only called by contract owner.
    /// @param _newWhitelistChecker The address of new whitelist checker contract.
    function updateWhitelistChecker(address _newWhitelistChecker) external onlyOwner {
        address _oldWhitelistChecker = whitelistChecker;
        whitelistChecker = _newWhitelistChecker;
        emit UpdateWhitelistChecker(_oldWhitelistChecker, _newWhitelistChecker);
    }
}
