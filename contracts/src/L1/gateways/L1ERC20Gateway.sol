// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {IL1ERC20Gateway} from "./IL1ERC20Gateway.sol";

// solhint-disable no-empty-blocks

abstract contract L1ERC20Gateway is IL1ERC20Gateway {
    /*************
     * Variables *
     *************/

    /// @dev The storage slots for future usage.
    uint256[50] private __gap;

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @inheritdoc IL1ERC20Gateway
    function depositERC20(
        address _token,
        uint256 _amount,
        uint256 _gasLimit
    ) external payable override {
        _deposit(_token, msg.sender, _amount, new bytes(0), _gasLimit);
    }

    /// @inheritdoc IL1ERC20Gateway
    function depositERC20(
        address _token,
        address _to,
        uint256 _amount,
        uint256 _gasLimit
    ) external payable override {
        _deposit(_token, _to, _amount, new bytes(0), _gasLimit);
    }

    /// @inheritdoc IL1ERC20Gateway
    function depositERC20AndCall(
        address _token,
        address _to,
        uint256 _amount,
        bytes memory _data,
        uint256 _gasLimit
    ) external payable override {
        _deposit(_token, _to, _amount, _data, _gasLimit);
    }

    /**********************
     * Internal Functions *
     **********************/

    /// @dev Internal function to do all the deposit operations.
    ///
    /// @param _token The token to deposit.
    /// @param _to The recipient address to recieve the token in L2.
    /// @param _amount The amount of token to deposit.
    /// @param _data Optional data to forward to recipient's account.
    /// @param _gasLimit Gas limit required to complete the deposit on L2.
    function _deposit(
        address _token,
        address _to,
        uint256 _amount,
        bytes memory _data,
        uint256 _gasLimit
    ) internal virtual;
}
