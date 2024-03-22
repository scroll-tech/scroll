// SPDX-License-Identifier: MIT

pragma solidity ^0.8.24;

import {IERC20Upgradeable} from "@openzeppelin/contracts-upgradeable/token/ERC20/IERC20Upgradeable.sol";
import {SafeERC20Upgradeable} from "@openzeppelin/contracts-upgradeable/token/ERC20/utils/SafeERC20Upgradeable.sol";

import {IL1ERC20Gateway} from "./IL1ERC20Gateway.sol";
import {IL1GatewayRouter} from "./IL1GatewayRouter.sol";

import {IL2ERC20Gateway} from "../../L2/gateways/IL2ERC20Gateway.sol";
import {ScrollGatewayBase} from "../../libraries/gateway/ScrollGatewayBase.sol";
import {IMessageDropCallback} from "../../libraries/callbacks/IMessageDropCallback.sol";

/// @title L1ERC20Gateway
/// @notice The `L1ERC20Gateway` as a base contract for ERC20 gateways in L1.
/// It has implementation of common used functions for ERC20 gateways.
abstract contract L1ERC20Gateway is IL1ERC20Gateway, IMessageDropCallback, ScrollGatewayBase {
    using SafeERC20Upgradeable for IERC20Upgradeable;

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
        _deposit(_token, _msgSender(), _amount, new bytes(0), _gasLimit);
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

    /// @inheritdoc IL1ERC20Gateway
    function finalizeWithdrawERC20(
        address _l1Token,
        address _l2Token,
        address _from,
        address _to,
        uint256 _amount,
        bytes calldata _data
    ) external payable virtual override onlyCallByCounterpart nonReentrant {
        _beforeFinalizeWithdrawERC20(_l1Token, _l2Token, _from, _to, _amount, _data);

        // @note can possible trigger reentrant call to this contract or messenger,
        // but it seems not a big problem.
        IERC20Upgradeable(_l1Token).safeTransfer(_to, _amount);

        _doCallback(_to, _data);

        emit FinalizeWithdrawERC20(_l1Token, _l2Token, _from, _to, _amount, _data);
    }

    /// @inheritdoc IMessageDropCallback
    function onDropMessage(bytes calldata _message) external payable virtual onlyInDropContext nonReentrant {
        // _message should start with 0x8431f5c1  =>  finalizeDepositERC20(address,address,address,address,uint256,bytes)
        require(bytes4(_message[0:4]) == IL2ERC20Gateway.finalizeDepositERC20.selector, "invalid selector");

        // decode (token, receiver, amount)
        (address _token, , address _receiver, , uint256 _amount, ) = abi.decode(
            _message[4:],
            (address, address, address, address, uint256, bytes)
        );

        // do dome check for each custom gateway
        _beforeDropMessage(_token, _receiver, _amount);

        IERC20Upgradeable(_token).safeTransfer(_receiver, _amount);

        emit RefundERC20(_token, _receiver, _amount);
    }

    /**********************
     * Internal Functions *
     **********************/

    /// @dev Internal function hook to perform checks and actions before finalizing the withdrawal.
    /// @param _l1Token The address of corresponding L1 token in L1.
    /// @param _l2Token The address of corresponding L2 token in L2.
    /// @param _from The address of account who withdraw the token in L2.
    /// @param _to The address of recipient in L1 to receive the token.
    /// @param _amount The amount of the token to withdraw.
    /// @param _data Optional data to forward to recipient's account.
    function _beforeFinalizeWithdrawERC20(
        address _l1Token,
        address _l2Token,
        address _from,
        address _to,
        uint256 _amount,
        bytes calldata _data
    ) internal virtual;

    /// @dev Internal function hook to perform checks and actions before dropping the message.
    /// @param _token The L1 token address.
    /// @param _receiver The recipient address on L1.
    /// @param _amount The amount of token to refund.
    function _beforeDropMessage(
        address _token,
        address _receiver,
        uint256 _amount
    ) internal virtual;

    /// @dev Internal function to transfer ERC20 token to this contract.
    /// @param _token The address of token to transfer.
    /// @param _amount The amount of token to transfer.
    /// @param _data The data passed by caller.
    function _transferERC20In(
        address _token,
        uint256 _amount,
        bytes memory _data
    )
        internal
        returns (
            address,
            uint256,
            bytes memory
        )
    {
        address _sender = _msgSender();
        address _from = _sender;
        if (router == _sender) {
            // Extract real sender if this call is from L1GatewayRouter.
            (_from, _data) = abi.decode(_data, (address, bytes));
            _amount = IL1GatewayRouter(_sender).requestERC20(_from, _token, _amount);
        } else {
            // common practice to handle fee on transfer token.
            uint256 _before = IERC20Upgradeable(_token).balanceOf(address(this));
            IERC20Upgradeable(_token).safeTransferFrom(_from, address(this), _amount);
            uint256 _after = IERC20Upgradeable(_token).balanceOf(address(this));
            // no unchecked here, since some weird token may return arbitrary balance.
            _amount = _after - _before;
        }
        // ignore weird fee on transfer token
        require(_amount > 0, "deposit zero amount");

        return (_from, _amount, _data);
    }

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
