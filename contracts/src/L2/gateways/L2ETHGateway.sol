// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {IL1ETHGateway} from "../../L1/gateways/IL1ETHGateway.sol";
import {IL2ScrollMessenger} from "../IL2ScrollMessenger.sol";
import {IL2ETHGateway} from "./IL2ETHGateway.sol";

import {ScrollGatewayBase} from "../../libraries/gateway/ScrollGatewayBase.sol";

/// @title L2ETHGateway
/// @notice The `L2ETHGateway` contract is used to withdraw ETH token on layer 2 and
/// finalize deposit ETH from layer 1.
/// @dev The ETH are not held in the gateway. The ETH will be sent to the `L2ScrollMessenger` contract.
/// On finalizing deposit, the Ether will be transferred from `L2ScrollMessenger`, then transfer to recipient.
contract L2ETHGateway is ScrollGatewayBase, IL2ETHGateway {
    /***************
     * Constructor *
     ***************/

    /// @notice Constructor for `L2ETHGateway` implementation contract.
    ///
    /// @param _counterpart The address of `L1ETHGateway` contract in L1.
    /// @param _router The address of `L1GatewayRouter` contract.
    /// @param _messenger The address of `L1ScrollMessenger` contract.
    constructor(
        address _counterpart,
        address _router,
        address _messenger
    ) ScrollGatewayBase(_counterpart, _router, _messenger) {
        if (_router == address(0)) revert ErrorZeroAddress();

        _disableInitializers();
    }

    /// @notice Initialize the storage of L2ETHGateway.
    ///
    /// @dev The parameters `_counterpart`, `_router` and `_messenger` are no longer used.
    ///
    /// @param _counterpart The address of L1ETHGateway in L1.
    /// @param _router The address of L2GatewayRouter in L2.
    /// @param _messenger The address of L2ScrollMessenger in L2.
    function initialize(
        address _counterpart,
        address _router,
        address _messenger
    ) external initializer {
        ScrollGatewayBase._initialize(_counterpart, _router, _messenger);
    }

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @inheritdoc IL2ETHGateway
    function withdrawETH(uint256 _amount, uint256 _gasLimit) external payable override {
        _withdraw(_msgSender(), _amount, new bytes(0), _gasLimit);
    }

    /// @inheritdoc IL2ETHGateway
    function withdrawETH(
        address _to,
        uint256 _amount,
        uint256 _gasLimit
    ) public payable override {
        _withdraw(_to, _amount, new bytes(0), _gasLimit);
    }

    /// @inheritdoc IL2ETHGateway
    function withdrawETHAndCall(
        address _to,
        uint256 _amount,
        bytes memory _data,
        uint256 _gasLimit
    ) public payable override {
        _withdraw(_to, _amount, _data, _gasLimit);
    }

    /// @inheritdoc IL2ETHGateway
    function finalizeDepositETH(
        address _from,
        address _to,
        uint256 _amount,
        bytes calldata _data
    ) external payable override onlyCallByCounterpart nonReentrant {
        require(msg.value == _amount, "msg.value mismatch");

        // solhint-disable-next-line avoid-low-level-calls
        (bool _success, ) = _to.call{value: _amount}("");
        require(_success, "ETH transfer failed");

        _doCallback(_to, _data);

        emit FinalizeDepositETH(_from, _to, _amount, _data);
    }

    /**********************
     * Internal Functions *
     **********************/

    /// @dev The internal ETH withdraw implementation.
    /// @param _to The address of recipient's account on L1.
    /// @param _amount The amount of ETH to be withdrawn.
    /// @param _data Optional data to forward to recipient's account.
    /// @param _gasLimit Optional gas limit to complete the deposit on L1.
    function _withdraw(
        address _to,
        uint256 _amount,
        bytes memory _data,
        uint256 _gasLimit
    ) internal virtual nonReentrant {
        require(msg.value > 0, "withdraw zero eth");

        // 1. Extract real sender if this call is from L1GatewayRouter.
        address _from = _msgSender();

        if (router == _from) {
            (_from, _data) = abi.decode(_data, (address, bytes));
        }

        // @note no rate limit here, since ETH is limited in messenger

        bytes memory _message = abi.encodeCall(IL1ETHGateway.finalizeWithdrawETH, (_from, _to, _amount, _data));
        IL2ScrollMessenger(messenger).sendMessage{value: msg.value}(counterpart, _amount, _message, _gasLimit);

        emit WithdrawETH(_from, _to, _amount, _data);
    }
}
