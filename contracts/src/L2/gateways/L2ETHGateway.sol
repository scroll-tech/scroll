// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

import {Initializable} from "@openzeppelin/contracts/proxy/utils/Initializable.sol";

import {IL1ETHGateway} from "../../L1/gateways/IL1ETHGateway.sol";
import {IL2ScrollMessenger} from "../IL2ScrollMessenger.sol";
import {IL2ETHGateway} from "./IL2ETHGateway.sol";

import {ScrollGatewayBase} from "../../libraries/gateway/ScrollGatewayBase.sol";

/// @title L2ETHGateway
/// @notice The `L2ETHGateway` contract is used to withdraw ETH token on layer 2 and
/// finalize deposit ETH from layer 1.
/// @dev The ETH are not held in the gateway. The ETH will be sent to the `L2ScrollMessenger` contract.
/// On finalizing deposit, the Ether will be transfered from `L2ScrollMessenger`, then transfer to recipient.
contract L2ETHGateway is Initializable, ScrollGatewayBase, IL2ETHGateway {
    /***************
     * Constructor *
     ***************/
    constructor() {
        _disableInitializers();
    }

    /// @notice Initialize the storage of L2ETHGateway.
    /// @param _counterpart The address of L1ETHGateway in L2.
    /// @param _router The address of L2GatewayRouter.
    /// @param _messenger The address of L2ScrollMessenger.
    function initialize(
        address _counterpart,
        address _router,
        address _messenger
    ) external initializer {
        require(_router != address(0), "zero router address");
        ScrollGatewayBase._initialize(_counterpart, _router, _messenger);
    }

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @inheritdoc IL2ETHGateway
    function withdrawETH(uint256 _amount, uint256 _gasLimit) external payable override {
        _withdraw(msg.sender, _amount, new bytes(0), _gasLimit);
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

    function _withdraw(
        address _to,
        uint256 _amount,
        bytes memory _data,
        uint256 _gasLimit
    ) internal virtual nonReentrant {
        require(msg.value > 0, "withdraw zero eth");

        // 1. Extract real sender if this call is from L1GatewayRouter.
        address _from = msg.sender;
        if (router == msg.sender) {
            (_from, _data) = abi.decode(_data, (address, bytes));
        }

        bytes memory _message = abi.encodeCall(IL1ETHGateway.finalizeWithdrawETH, (_from, _to, _amount, _data));
        IL2ScrollMessenger(messenger).sendMessage{value: msg.value}(counterpart, _amount, _message, _gasLimit);

        emit WithdrawETH(_from, _to, _amount, _data);
    }
}
