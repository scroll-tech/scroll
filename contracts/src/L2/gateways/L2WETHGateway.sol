// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {IERC20Upgradeable} from "@openzeppelin/contracts-upgradeable/token/ERC20/IERC20Upgradeable.sol";
import {SafeERC20Upgradeable} from "@openzeppelin/contracts-upgradeable/token/ERC20/utils/SafeERC20Upgradeable.sol";

import {IL2ERC20Gateway, L2ERC20Gateway} from "./L2ERC20Gateway.sol";
import {IL2ScrollMessenger} from "../IL2ScrollMessenger.sol";
import {IWETH} from "../../interfaces/IWETH.sol";
import {IL1ERC20Gateway} from "../../L1/gateways/IL1ERC20Gateway.sol";
import {ScrollGatewayBase} from "../../libraries/gateway/ScrollGatewayBase.sol";

/// @title L2WETHGateway
/// @notice The `L2WETHGateway` contract is used to withdraw `WETH` token on layer 2 and
/// finalize deposit `WETH` from layer 1.
/// @dev The WETH tokens are not held in the gateway. It will first be unwrapped as Ether and
/// then the Ether will be sent to the `L2ScrollMessenger` contract.
/// On finalizing deposit, the Ether will be transferred from `L2ScrollMessenger`, then
/// wrapped as WETH and finally transfer to recipient.
contract L2WETHGateway is L2ERC20Gateway {
    using SafeERC20Upgradeable for IERC20Upgradeable;

    /*************
     * Constants *
     *************/

    /// @notice The address of L1 WETH address.
    address public immutable l1WETH;

    /// @notice The address of L2 WETH address.
    // solhint-disable-next-line var-name-mixedcase
    address public immutable WETH;

    /***************
     * Constructor *
     ***************/

    /// @notice Constructor for `L2WETHGateway` implementation contract.
    ///
    /// @param _WETH The address of WETH in L2.
    /// @param _l1WETH The address of WETH in L1.
    /// @param _counterpart The address of `L1WETHGateway` contract in L1.
    /// @param _router The address of `L2GatewayRouter` contract.
    /// @param _messenger The address of `L2ScrollMessenger` contract.
    constructor(
        address _WETH,
        address _l1WETH,
        address _counterpart,
        address _router,
        address _messenger
    ) ScrollGatewayBase(_counterpart, _router, _messenger) {
        if (_WETH == address(0) || _l1WETH == address(0) || _router == address(0)) {
            revert ErrorZeroAddress();
        }

        _disableInitializers();

        WETH = _WETH;
        l1WETH = _l1WETH;
    }

    /// @notice Initialize the storage of `L2WETHGateway`.
    ///
    /// @dev The parameters `_counterpart`, `_router` and `_messenger` are no longer used.
    ///
    /// @param _counterpart The address of `L1WETHGateway` contract in L1.
    /// @param _router The address of `L2GatewayRouter` contract in L2.
    /// @param _messenger The address of `L2ScrollMessenger` contract in L2.
    function initialize(
        address _counterpart,
        address _router,
        address _messenger
    ) external initializer {
        ScrollGatewayBase._initialize(_counterpart, _router, _messenger);
    }

    receive() external payable {
        require(_msgSender() == WETH, "only WETH");
    }

    /*************************
     * Public View Functions *
     *************************/

    /// @inheritdoc IL2ERC20Gateway
    function getL1ERC20Address(address) external view override returns (address) {
        return l1WETH;
    }

    /// @inheritdoc IL2ERC20Gateway
    function getL2ERC20Address(address) public view override returns (address) {
        return WETH;
    }

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @inheritdoc IL2ERC20Gateway
    function finalizeDepositERC20(
        address _l1Token,
        address _l2Token,
        address _from,
        address _to,
        uint256 _amount,
        bytes calldata _data
    ) external payable override onlyCallByCounterpart nonReentrant {
        require(_l1Token == l1WETH, "l1 token not WETH");
        require(_l2Token == WETH, "l2 token not WETH");
        require(_amount == msg.value, "msg.value mismatch");

        IWETH(_l2Token).deposit{value: _amount}();
        IERC20Upgradeable(_l2Token).safeTransfer(_to, _amount);

        _doCallback(_to, _data);

        emit FinalizeDepositERC20(_l1Token, _l2Token, _from, _to, _amount, _data);
    }

    /**********************
     * Internal Functions *
     **********************/

    /// @inheritdoc L2ERC20Gateway
    function _withdraw(
        address _token,
        address _to,
        uint256 _amount,
        bytes memory _data,
        uint256 _gasLimit
    ) internal virtual override nonReentrant {
        require(_amount > 0, "withdraw zero amount");
        require(_token == WETH, "only WETH is allowed");

        // 1. Extract real sender if this call is from L1GatewayRouter.
        address _from = _msgSender();
        if (router == _from) {
            (_from, _data) = abi.decode(_data, (address, bytes));
        }

        // 2. Transfer token into this contract.
        IERC20Upgradeable(_token).safeTransferFrom(_from, address(this), _amount);
        IWETH(_token).withdraw(_amount);

        // 3. Generate message passed to L2StandardERC20Gateway.
        address _l1WETH = l1WETH;
        bytes memory _message = abi.encodeCall(
            IL1ERC20Gateway.finalizeWithdrawERC20,
            (_l1WETH, _token, _from, _to, _amount, _data)
        );

        // 4. Send message to L1ScrollMessenger.
        IL2ScrollMessenger(messenger).sendMessage{value: _amount + msg.value}(
            counterpart,
            _amount,
            _message,
            _gasLimit
        );

        emit WithdrawERC20(_l1WETH, _token, _from, _to, _amount, _data);
    }
}
