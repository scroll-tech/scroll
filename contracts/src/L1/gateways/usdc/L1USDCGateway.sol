// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {OwnableUpgradeable} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import {IERC20Upgradeable} from "@openzeppelin/contracts-upgradeable/token/ERC20/IERC20Upgradeable.sol";
import {SafeERC20Upgradeable} from "@openzeppelin/contracts-upgradeable/token/ERC20/utils/SafeERC20Upgradeable.sol";

import {IFiatToken} from "../../../interfaces/IFiatToken.sol";
import {IL2ERC20Gateway} from "../../../L2/gateways/IL2ERC20Gateway.sol";
import {IL1ScrollMessenger} from "../../IL1ScrollMessenger.sol";
import {IL1ERC20Gateway} from "../IL1ERC20Gateway.sol";

import {ScrollGatewayBase} from "../../../libraries/gateway/ScrollGatewayBase.sol";
import {L1ERC20Gateway} from "../L1ERC20Gateway.sol";

/// @title L1USDCGateway
/// @notice The `L1USDCGateway` contract is used to deposit `USDC` token in layer 1 and
/// finalize withdraw `USDC` from layer 2, before USDC become native in layer 2.
contract L1USDCGateway is OwnableUpgradeable, ScrollGatewayBase, L1ERC20Gateway {
    using SafeERC20Upgradeable for IERC20Upgradeable;

    /*************
     * Constants *
     *************/

    /// @notice The address of L1 USDC address.
    // solhint-disable-next-line var-name-mixedcase
    address public immutable l1USDC;

    /// @notice The address of L2 USDC address.
    address public immutable l2USDC;

    /*************
     * Variables *
     *************/

    address public circleCaller;

    bool public depositPaused;

    bool public withdrawPaused;

    /***************
     * Constructor *
     ***************/

    constructor(address _l1USDC, address _l2USDC) {
        l1USDC = _l1USDC;
        l2USDC = _l2USDC;
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
        OwnableUpgradeable.__Ownable_init();
    }

    /*************************
     * Public View Functions *
     *************************/

    /// @inheritdoc IL1ERC20Gateway
    function getL2ERC20Address(address) public view override returns (address) {
        return l2USDC;
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
        require(msg.value == 0, "nonzero msg.value");
        require(_l1Token == l1USDC, "l1 token not USDC");
        require(_l2Token == l2USDC, "l2 token not USDC");
        require(!withdrawPaused, "withdraw paused");

        IERC20Upgradeable(_l1Token).safeTransfer(_to, _amount);

        _doCallback(_to, _data);

        emit FinalizeWithdrawERC20(_l1Token, _l2Token, _from, _to, _amount, _data);
    }

    /*******************************
     * Public Restricted Functions *
     *******************************/

    /// @notice Burn all USDC in this contract.
    /// @dev The function should only be called by a EOA controlled by Circle.
    function burnAllHeldUsdc() external {
        require(msg.sender == circleCaller, "only circle caller");

        uint256 _balance = IERC20Upgradeable(l1USDC).balanceOf(address(this));
        require(IFiatToken(l1USDC).burn(_balance), "burn USDC failed");
    }

    /// @notice Update the Circle EOA address.
    /// @param _caller The address to update.
    function updateCircleCaller(address _caller) external onlyOwner {
        circleCaller = _caller;
    }

    /// @notice Change the deposit pause status of this contract.
    /// @param _paused The new status, `true` means paused and `false` means not paused.
    function pauseDeposit(bool _paused) external onlyOwner {
        depositPaused = _paused;
    }

    /// @notice Change the withdraw pause status of this contract.
    /// @param _paused The new status, `true` means paused and `false` means not paused.
    function pauseWithdraw(bool _paused) external onlyOwner {
        withdrawPaused = _paused;
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
        require(_token == l1USDC, "only USDC is allowed");
        require(!depositPaused, "deposit paused");

        // 1. Extract real sender if this call is from L1GatewayRouter.
        address _from = msg.sender;
        if (router == msg.sender) {
            (_from, _data) = abi.decode(_data, (address, bytes));
        }

        // 2. Transfer token into this contract.
        IERC20Upgradeable(_token).safeTransferFrom(_from, address(this), _amount);

        // 3. Generate message passed to L2USDCGateway.
        bytes memory _message = abi.encodeWithSelector(
            IL2ERC20Gateway.finalizeDepositERC20.selector,
            _token,
            l2USDC,
            _from,
            _to,
            _amount,
            _data
        );

        // 4. Send message to L1ScrollMessenger.
        IL1ScrollMessenger(messenger).sendMessage{value: msg.value}(counterpart, 0, _message, _gasLimit);

        emit DepositERC20(_token, l2USDC, _from, _to, _amount, _data);
    }
}
