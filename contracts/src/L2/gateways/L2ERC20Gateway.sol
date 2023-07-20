// SPDX-License-Identifier: MIT

pragma solidity ^0.8.16;

import {IL2ERC20Gateway} from "./IL2ERC20Gateway.sol";

// solhint-disable no-empty-blocks

abstract contract L2ERC20Gateway is IL2ERC20Gateway {
    /*************
     * Variables *
     *************/

    /// @dev The storage slots for future usage.
    uint256[50] private __gap;

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @inheritdoc IL2ERC20Gateway
    function withdrawERC20(
        address _token,
        uint256 _amount,
        uint256 _gasLimit
    ) external payable override {
        _withdraw(_token, msg.sender, _amount, new bytes(0), _gasLimit);
    }

    /// @inheritdoc IL2ERC20Gateway
    function withdrawERC20(
        address _token,
        address _to,
        uint256 _amount,
        uint256 _gasLimit
    ) external payable override {
        _withdraw(_token, _to, _amount, new bytes(0), _gasLimit);
    }

    /// @inheritdoc IL2ERC20Gateway
    function withdrawERC20AndCall(
        address _token,
        address _to,
        uint256 _amount,
        bytes calldata _data,
        uint256 _gasLimit
    ) external payable override {
        _withdraw(_token, _to, _amount, _data, _gasLimit);
    }

    /**********************
     * Internal Functions *
     **********************/

    function _withdraw(
        address _token,
        address _to,
        uint256 _amount,
        bytes memory _data,
        uint256 _gasLimit
    ) internal virtual;
}
