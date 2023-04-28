// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {Initializable} from "@openzeppelin/contracts/proxy/utils/Initializable.sol";
import {IERC20} from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import {SafeERC20} from "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";

import {IWETH} from "../../interfaces/IWETH.sol";
import {IL2ERC20Gateway} from "../../L2/gateways/IL2ERC20Gateway.sol";
import {IL1ScrollMessenger} from "../IL1ScrollMessenger.sol";
import {IL1ERC20Gateway} from "./IL1ERC20Gateway.sol";

import {ScrollGatewayBase} from "../../libraries/gateway/ScrollGatewayBase.sol";
import {L1ERC20Gateway} from "./L1ERC20Gateway.sol";

/// @title L1WETHGateway
/// @notice The `L1WETHGateway` contract is used to deposit `WETH` token in layer 1 and
/// finalize withdraw `WETH` from layer 2.
/// @dev The deposited WETH tokens are not held in the gateway. It will first be unwrapped
/// as Ether and then the Ether will be sent to the `L1ScrollMessenger` contract.
/// On finalizing withdraw, the Ether will be transfered from `L1ScrollMessenger`, then
/// wrapped as WETH and finally transfer to recipient.
contract L1WETHGateway is Initializable, ScrollGatewayBase, L1ERC20Gateway {
    using SafeERC20 for IERC20;

    /*************
     * Constants *
     *************/

    /// @notice The address of L2 WETH address.
    address public immutable l2WETH;

    /// @notice The address of L1 WETH address.
    // solhint-disable-next-line var-name-mixedcase
    address public immutable WETH;

    /***************
     * Constructor *
     ***************/

    constructor(address _WETH, address _l2WETH) {
        WETH = _WETH;
        l2WETH = _l2WETH;
    }

    /// @notice Initialize the storage of L1WETHGateway.
    /// @param _counterpart The address of L2ETHGateway in L2.
    /// @param _router The address of L1GatewayRouter.
    /// @param _messenger The address of L1ScrollMessenger.
    function initialize(
        address _counterpart,
        address _router,
        address _messenger
    ) external initializer {
        require(_router != address(0), "zero router address");
        ScrollGatewayBase._initialize(_counterpart, _router, _messenger);
    }

    receive() external payable {
        require(msg.sender == WETH, "only WETH");
    }

    /*************************
     * Public View Functions *
     *************************/

    /// @inheritdoc IL1ERC20Gateway
    function getL2ERC20Address(address) public view override returns (address) {
        return l2WETH;
    }

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @inheritdoc IL1ERC20Gateway
    function finalizeWithdrawERC20(
        address _l1Token,
        address _l2Token,
        address _from,
        address _to,
        uint256 _amount,
        bytes calldata _data
    ) external payable override onlyCallByCounterpart {
        require(_l1Token == WETH, "l1 token not WETH");
        require(_l2Token == l2WETH, "l2 token not WETH");
        require(_amount == msg.value, "msg.value mismatch");

        IWETH(_l1Token).deposit{value: _amount}();
        IERC20(_l1Token).safeTransfer(_to, _amount);

        _doCallback(_to, _data);

        emit FinalizeWithdrawERC20(_l1Token, _l2Token, _from, _to, _amount, _data);
    }

    /**********************
     * Internal Functions *
     **********************/

    /// @inheritdoc L1ERC20Gateway
    function _deposit(
        address _token,
        address _to,
        uint256 _amount,
        bytes memory _data,
        uint256 _gasLimit
    ) internal virtual override nonReentrant {
        require(_amount > 0, "deposit zero amount");
        require(_token == WETH, "only WETH is allowed");

        // 1. Extract real sender if this call is from L1GatewayRouter.
        address _from = msg.sender;
        if (router == msg.sender) {
            (_from, _data) = abi.decode(_data, (address, bytes));
        }

        // 2. Transfer token into this contract.
        IERC20(_token).safeTransferFrom(_from, address(this), _amount);
        IWETH(_token).withdraw(_amount);

        // 3. Generate message passed to L2StandardERC20Gateway.
        bytes memory _message = abi.encodeWithSelector(
            IL2ERC20Gateway.finalizeDepositERC20.selector,
            _token,
            l2WETH,
            _from,
            _to,
            _amount,
            _data
        );

        // 4. Send message to L1ScrollMessenger.
        IL1ScrollMessenger(messenger).sendMessage{value: _amount + msg.value}(
            counterpart,
            _amount,
            _message,
            _gasLimit
        );

        emit DepositERC20(_token, l2WETH, _from, _to, _amount, _data);
    }
}
