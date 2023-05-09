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

    /*************
     * Variables *
     *************/

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

    /// @inheritdoc IL2GasPriceOracle
    function estimateCrossDomainMessageFee(uint256 _gasLimit) external view override returns (uint256) {
        return _gasLimit * l2BaseFee;
    }

    /*****************************
     * Public Mutating Functions *
     *****************************/

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
