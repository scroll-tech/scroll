// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {OwnableUpgradeable} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import {IERC20Upgradeable} from "@openzeppelin/contracts-upgradeable/token/ERC20/IERC20Upgradeable.sol";
import {SafeERC20Upgradeable} from "@openzeppelin/contracts-upgradeable/token/ERC20/utils/SafeERC20Upgradeable.sol";

import {ITokenMessenger} from "../../../interfaces/ITokenMessenger.sol";
import {IL1ERC20Gateway} from "../../../L1/gateways/IL1ERC20Gateway.sol";
import {IL2ScrollMessenger} from "../../IL2ScrollMessenger.sol";
import {IL2ERC20Gateway} from "../IL2ERC20Gateway.sol";

import {CCTPGatewayBase} from "../../../libraries/gateway/CCTPGatewayBase.sol";
import {ScrollGatewayBase} from "../../../libraries/gateway/ScrollGatewayBase.sol";
import {L2ERC20Gateway} from "../L2ERC20Gateway.sol";

/// @title L2USDCGatewayCCTP
/// @notice The `L2USDCGatewayCCTP` contract is used to withdraw `USDC` token in layer 2 and
/// finalize deposit `USDC` from layer 1.
contract L2USDCGatewayCCTP is CCTPGatewayBase, L2ERC20Gateway {
    using SafeERC20Upgradeable for IERC20Upgradeable;

    /***************
     * Constructor *
     ***************/

    constructor(
        address _l1USDC,
        address _l2USDC,
        uint32 _destinationDomain,
        address _counterpart,
        address _router,
        address _messenger
    ) CCTPGatewayBase(_l1USDC, _l2USDC, _destinationDomain) ScrollGatewayBase(_counterpart, _router, _messenger) {
        if (_router == address(0)) revert ErrorZeroAddress();

        _disableInitializers();
    }

    /// @notice Initialize the storage of L2USDCGatewayCCTP.
    ///
    /// @dev The parameters `_counterpart`, `_router`, `_messenger` are no longer used.
    ///
    /// @param _counterpart The address of L1USDCGatewayCCTP in L1.
    /// @param _router The address of L2GatewayRouter.
    /// @param _messenger The address of L2ScrollMessenger.
    /// @param _cctpMessenger The address of TokenMessenger in local domain.
    /// @param _cctpTransmitter The address of MessageTransmitter in local domain.
    function initialize(
        address _counterpart,
        address _router,
        address _messenger,
        address _cctpMessenger,
        address _cctpTransmitter
    ) external initializer {
        ScrollGatewayBase._initialize(_counterpart, _router, _messenger);
        CCTPGatewayBase._initialize(_cctpMessenger, _cctpTransmitter);
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
    /// @dev The function will not mint the USDC, users need to call `claimUSDC` after this function is done.
    function finalizeDepositERC20(
        address _l1Token,
        address _l2Token,
        address _from,
        address _to,
        uint256 _amount,
        bytes memory _data
    ) external payable override onlyCallByCounterpart {
        require(msg.value == 0, "nonzero msg.value");
        require(_l1Token == l1USDC, "l1 token not USDC");
        require(_l2Token == l2USDC, "l2 token not USDC");

        uint256 _nonce;
        (_nonce, _data) = abi.decode(_data, (uint256, bytes));
        require(status[_nonce] == CCTPMessageStatus.None, "message relayed");
        status[_nonce] = CCTPMessageStatus.Pending;

        emit FinalizeDepositERC20(_l1Token, _l2Token, _from, _to, _amount, _data);
    }

    /*******************************
     * Public Restricted Functions *
     *******************************/

    /// @notice Update the CCTP contract addresses.
    /// @param _messenger The address of TokenMessenger in local domain.
    /// @param _transmitter The address of MessageTransmitter in local domain.
    function updateCCTPContracts(address _messenger, address _transmitter) external onlyOwner {
        cctpMessenger = _messenger;
        cctpTransmitter = _transmitter;
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
    ) internal virtual override {
        require(_amount > 0, "withdraw zero amount");
        require(_token == l2USDC, "only USDC is allowed");

        // 1. Extract real sender if this call is from L1GatewayRouter.
        address _from = _msgSender();
        if (router == _from) {
            (_from, _data) = abi.decode(_data, (address, bytes));
        }

        // 2. Transfer token into this contract.
        IERC20Upgradeable(_token).safeTransferFrom(_from, address(this), _amount);

        // 3. Burn token through CCTP TokenMessenger
        uint256 _nonce = ITokenMessenger(cctpMessenger).depositForBurnWithCaller(
            _amount,
            destinationDomain,
            bytes32(uint256(uint160(_to))),
            address(this),
            bytes32(uint256(uint160(counterpart)))
        );

        // 4. Generate message passed to L1USDCGateway.
        address _l1USDC = l1USDC;
        bytes memory _message = abi.encodeWithSelector(
            IL1ERC20Gateway.finalizeWithdrawERC20.selector,
            _l1USDC,
            _token,
            _from,
            _to,
            _amount,
            abi.encode(_nonce, _data)
        );

        // 4. Send message to L1ScrollMessenger.
        IL2ScrollMessenger(messenger).sendMessage{value: msg.value}(counterpart, 0, _message, _gasLimit);

        emit WithdrawERC20(_l1USDC, _token, _from, _to, _amount, _data);
    }
}
