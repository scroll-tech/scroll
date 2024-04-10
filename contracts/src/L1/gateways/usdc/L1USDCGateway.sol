// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {IFiatToken} from "../../../interfaces/IFiatToken.sol";
import {IUSDCBurnableSourceBridge} from "../../../interfaces/IUSDCBurnableSourceBridge.sol";
import {IL2ERC20Gateway} from "../../../L2/gateways/IL2ERC20Gateway.sol";
import {IL1ScrollMessenger} from "../../IL1ScrollMessenger.sol";
import {IL1ERC20Gateway} from "../IL1ERC20Gateway.sol";

import {ScrollGatewayBase} from "../../../libraries/gateway/ScrollGatewayBase.sol";
import {L1ERC20Gateway} from "../L1ERC20Gateway.sol";

/// @title L1USDCGateway
/// @notice The `L1USDCGateway` contract is used to deposit `USDC` token in layer 1 and
/// finalize withdraw `USDC` from layer 2, before USDC become native in layer 2.
contract L1USDCGateway is L1ERC20Gateway, IUSDCBurnableSourceBridge {
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

    /// @notice The address of caller from Circle.
    address public circleCaller;

    /// @notice The flag indicates whether USDC deposit is paused.
    bool public depositPaused;

    /// @notice The flag indicates whether USDC withdrawal is paused.
    /// @dev This is not necessary to be set `true` since we will set `L2USDCGateway.withdrawPaused` first.
    ///      This is kept just in case and will be set after all pending messages are relayed.
    bool public withdrawPaused;

    /// @notice The total amount of bridged USDC in this contract.
    /// @dev Only deposited USDC will count. Accidentally transferred USDC will be ignored.
    uint256 public totalBridgedUSDC;

    /***************
     * Constructor *
     ***************/

    /// @notice Constructor for `L1USDCGateway` implementation contract.
    ///
    /// @param _l1USDC The address of USDC in L1.
    /// @param _l2USDC The address of USDC in L2.
    /// @param _counterpart The address of `L2USDCGateway` contract in L2.
    /// @param _router The address of `L1GatewayRouter` contract in L1.
    /// @param _messenger The address of `L1ScrollMessenger` contract in L1.
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

    /// @notice Initialize the storage of L1USDCGateway.
    ///
    /// @dev The parameters `_counterpart`, `_router` and `_messenger` are no longer used.
    ///
    /// @param _counterpart The address of L2USDCGateway in L2.
    /// @param _router The address of L1GatewayRouter in L1.
    /// @param _messenger The address of L1ScrollMessenger in L1.
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

    /// @inheritdoc IL1ERC20Gateway
    function getL2ERC20Address(address) public view override returns (address) {
        return l2USDC;
    }

    /*******************************
     * Public Restricted Functions *
     *******************************/

    /// @inheritdoc IUSDCBurnableSourceBridge
    function burnAllLockedUSDC() external override {
        require(_msgSender() == circleCaller, "only circle caller");

        // @note Only bridged USDC will be burned. We may refund the rest if possible.
        uint256 _balance = totalBridgedUSDC;
        totalBridgedUSDC = 0;

        IFiatToken(l1USDC).burn(_balance);
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
    function _beforeFinalizeWithdrawERC20(
        address _l1Token,
        address _l2Token,
        address,
        address,
        uint256 _amount,
        bytes calldata
    ) internal virtual override {
        require(msg.value == 0, "nonzero msg.value");
        require(_l1Token == l1USDC, "l1 token not USDC");
        require(_l2Token == l2USDC, "l2 token not USDC");
        require(!withdrawPaused, "withdraw paused");

        totalBridgedUSDC -= _amount;
    }

    /// @inheritdoc L1ERC20Gateway
    function _beforeDropMessage(
        address,
        address,
        uint256 _amount
    ) internal virtual override {
        require(msg.value == 0, "nonzero msg.value");
        totalBridgedUSDC -= _amount;
    }

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

        // 1. Transfer token into this contract.
        address _from;
        (_from, _amount, _data) = _transferERC20In(_token, _amount, _data);
        require(_data.length == 0, "call is not allowed");
        totalBridgedUSDC += _amount;

        // 2. Generate message passed to L2USDCGateway.
        bytes memory _message = abi.encodeCall(
            IL2ERC20Gateway.finalizeDepositERC20,
            (_token, l2USDC, _from, _to, _amount, _data)
        );

        // 3. Send message to L1ScrollMessenger.
        IL1ScrollMessenger(messenger).sendMessage{value: msg.value}(counterpart, 0, _message, _gasLimit, _from);

        emit DepositERC20(_token, l2USDC, _from, _to, _amount, _data);
    }
}
