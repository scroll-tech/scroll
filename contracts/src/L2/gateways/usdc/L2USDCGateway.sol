// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {OwnableUpgradeable} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import {IERC20Upgradeable} from "@openzeppelin/contracts-upgradeable/token/ERC20/IERC20Upgradeable.sol";
import {SafeERC20Upgradeable} from "@openzeppelin/contracts-upgradeable/token/ERC20/utils/SafeERC20Upgradeable.sol";

import {IFiatToken} from "../../../interfaces/IFiatToken.sol";
import {IUSDCDestinationBridge} from "../../../interfaces/IUSDCDestinationBridge.sol";
import {IL1ERC20Gateway} from "../../../L1/gateways/IL1ERC20Gateway.sol";
import {IL2ScrollMessenger} from "../../IL2ScrollMessenger.sol";
import {IL2ERC20Gateway} from "../IL2ERC20Gateway.sol";

import {ScrollGatewayBase} from "../../../libraries/gateway/ScrollGatewayBase.sol";
import {L2ERC20Gateway} from "../L2ERC20Gateway.sol";

/// @title L2USDCGateway
/// @notice The `L2USDCGateway` contract is used to withdraw `USDC` token on layer 2 and
/// finalize deposit `USDC` from layer 1.
contract L2USDCGateway is L2ERC20Gateway, IUSDCDestinationBridge {
    using SafeERC20Upgradeable for IERC20Upgradeable;

    /*************
     * Constants *
     *************/

    /// @notice The address of L1 USDC address.
    address public immutable l1USDC;

    /// @notice The address of L2 USDC address.
    address public immutable l2USDC;

    /*************
     * Variables *
     *************/

    /// @notice The address of caller from Circle.
    address public circleCaller;

    /// @notice The flag indicates whether USDC deposit is paused.
    /// @dev This is not necessary to be set `true` since we will set `L1USDCGateway.depositPaused` first.
    ///      This is kept just in case and will be set after all pending messages are relayed.
    bool public depositPaused;

    /// @notice The flag indicates whether USDC withdrawal is paused.
    bool public withdrawPaused;

    /***************
     * Constructor *
     ***************/

    /// @notice Constructor for `L2USDCGateway` implementation contract.
    ///
    /// @param _l1USDC The address of USDC in L1.
    /// @param _l2USDC The address of USDC in L2.
    /// @param _counterpart The address of `L1USDCGateway` contract in L1.
    /// @param _router The address of `L2GatewayRouter` contract in L2.
    /// @param _messenger The address of `L2ScrollMessenger` contract in L2.
    constructor(
        address _l1USDC,
        address _l2USDC,
        address _counterpart,
        address _router,
        address _messenger
    ) ScrollGatewayBase(_counterpart, _router, _messenger) {
        if (_l1USDC == address(0) || _l2USDC == address(0) || _router == address(0)) {
            revert ErrorZeroAddress();
        }

        _disableInitializers();

        l1USDC = _l1USDC;
        l2USDC = _l2USDC;
    }

    /// @notice Initialize the storage of `L2USDCGateway`.
    ///
    /// @dev The parameters `_counterpart`, `_router` and `_messenger` are no longer used.
    ///
    /// @param _counterpart The address of `L1USDCGateway` contract in L1.
    /// @param _router The address of `L2GatewayRouter` contract in L2.
    /// @param _messenger The address of `L2ScrollMessenger` contract in L2.
    function initialize(
        address _counterpart,
        address _router,
        address _messenger
    ) external initializer {
        ScrollGatewayBase._initialize(_counterpart, _router, _messenger);
    }

    /*************************
     * Public View Functions *
     *************************/

    /// @inheritdoc IL2ERC20Gateway
    function getL1ERC20Address(address) external view override returns (address) {
        return l1USDC;
    }

    /// @inheritdoc IL2ERC20Gateway
    function getL2ERC20Address(address) public view override returns (address) {
        return l2USDC;
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
        require(msg.value == 0, "nonzero msg.value");
        require(_l1Token == l1USDC, "l1 token not USDC");
        require(_l2Token == l2USDC, "l2 token not USDC");
        require(!depositPaused, "deposit paused");

        require(IFiatToken(_l2Token).mint(_to, _amount), "mint USDC failed");

        // disable call for USDC
        // _doCallback(_to, _data);

        emit FinalizeDepositERC20(_l1Token, _l2Token, _from, _to, _amount, _data);
    }

    /*******************************
     * Public Restricted Functions *
     *******************************/

    /// @inheritdoc IUSDCDestinationBridge
    function transferUSDCRoles(address _owner) external {
        require(_msgSender() == circleCaller, "only circle caller");

        OwnableUpgradeable(l2USDC).transferOwnership(_owner);
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

    /// @inheritdoc L2ERC20Gateway
    function _withdraw(
        address _token,
        address _to,
        uint256 _amount,
        bytes memory _data,
        uint256 _gasLimit
    ) internal virtual override nonReentrant {
        require(_amount > 0, "withdraw zero amount");
        require(_token == l2USDC, "only USDC is allowed");
        require(!withdrawPaused, "withdraw paused");

        // 1. Extract real sender if this call is from L2GatewayRouter.
        address _from = _msgSender();
        if (router == _from) {
            (_from, _data) = abi.decode(_data, (address, bytes));
        }
        require(_data.length == 0, "call is not allowed");

        // 2. Transfer token into this contract.
        IERC20Upgradeable(_token).safeTransferFrom(_from, address(this), _amount);
        IFiatToken(_token).burn(_amount);

        // 3. Generate message passed to L1USDCGateway.
        address _l1USDC = l1USDC;
        bytes memory _message = abi.encodeCall(
            IL1ERC20Gateway.finalizeWithdrawERC20,
            (_l1USDC, _token, _from, _to, _amount, _data)
        );

        // 4. Send message to L2ScrollMessenger.
        IL2ScrollMessenger(messenger).sendMessage{value: msg.value}(counterpart, 0, _message, _gasLimit);

        emit WithdrawERC20(_l1USDC, _token, _from, _to, _amount, _data);
    }
}
