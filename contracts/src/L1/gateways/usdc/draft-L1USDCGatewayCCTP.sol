// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {OwnableUpgradeable} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";

import {ITokenMessenger} from "../../../interfaces/ITokenMessenger.sol";
import {IL2ERC20Gateway} from "../../../L2/gateways/IL2ERC20Gateway.sol";
import {IL1ScrollMessenger} from "../../IL1ScrollMessenger.sol";
import {IL1ERC20Gateway} from "../IL1ERC20Gateway.sol";

import {CCTPGatewayBase} from "../../../libraries/gateway/CCTPGatewayBase.sol";
import {ScrollGatewayBase} from "../../../libraries/gateway/ScrollGatewayBase.sol";
import {L1ERC20Gateway} from "../L1ERC20Gateway.sol";

/// @title L1USDCGatewayCCTP
/// @notice The `L1USDCGateway` contract is used to deposit `USDC` token in layer 1 and
/// finalize withdraw `USDC` from layer 2, after USDC become native in layer 2.
contract L1USDCGatewayCCTP is CCTPGatewayBase, L1ERC20Gateway {
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

    /// @notice Initialize the storage of L1USDCGatewayCCTP.
    ///
    /// @dev The parameters `_counterpart`, `_router`, `_messenger` are no longer used.
    ///
    /// @param _counterpart The address of L2USDCGatewayCCTP in L2.
    /// @param _router The address of L1GatewayRouter.
    /// @param _messenger The address of L1ScrollMessenger.
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

    /// @inheritdoc IL1ERC20Gateway
    function getL2ERC20Address(address) public view override returns (address) {
        return l2USDC;
    }

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @notice Relay cross chain message and claim USDC that has been cross chained.
    /// @dev The `_scrollMessage` is actually encoded calldata for `L1ScrollMessenger.relayMessageWithProof`.
    ///
    /// @dev This helper function is aimed to claim USDC in single transaction.
    ///      Normally, an user should call `L1ScrollMessenger.relayMessageWithProof` first,
    ///      then `L1USDCGatewayCCTP.claimUSDC`.
    ///
    /// @param _nonce The nonce of the message from CCTP.
    /// @param _cctpMessage The message passed to MessageTransmitter contract in CCTP.
    /// @param _cctpSignature The message passed to MessageTransmitter contract in CCTP.
    /// @param _scrollMessage The message passed to L1ScrollMessenger contract.
    function relayAndClaimUSDC(
        uint256 _nonce,
        bytes calldata _cctpMessage,
        bytes calldata _cctpSignature,
        bytes calldata _scrollMessage
    ) external {
        require(status[_nonce] == CCTPMessageStatus.None, "message relayed");
        // call messenger to set `status[_nonce]` to `CCTPMessageStatus.Pending`.
        (bool _success, ) = messenger.call(_scrollMessage);
        require(_success, "call messenger failed");

        claimUSDC(_nonce, _cctpMessage, _cctpSignature);
    }

    /// @inheritdoc IL1ERC20Gateway
    /// @dev The function will not mint the USDC, users need to call `claimUSDC` after this function is done.
    function finalizeWithdrawERC20(
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

        emit FinalizeWithdrawERC20(_l1Token, _l2Token, _from, _to, _amount, _data);
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

    /// @inheritdoc L1ERC20Gateway
    function _beforeFinalizeWithdrawERC20(
        address,
        address,
        address,
        address,
        uint256,
        bytes calldata
    ) internal virtual override {}

    /// @inheritdoc L1ERC20Gateway
    function _beforeDropMessage(
        address,
        address,
        uint256
    ) internal virtual override {
        require(msg.value == 0, "nonzero msg.value");
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

        // 1. Extract real sender if this call is from L1GatewayRouter.
        address _from;
        (_from, _amount, _data) = _transferERC20In(_token, _amount, _data);

        // 2. Burn token through CCTP TokenMessenger
        uint256 _nonce = ITokenMessenger(cctpMessenger).depositForBurnWithCaller(
            _amount,
            destinationDomain,
            bytes32(uint256(uint160(_to))),
            address(this),
            bytes32(uint256(uint160(counterpart)))
        );

        // 3. Generate message passed to L2USDCGatewayCCTP.
        bytes memory _message = abi.encodeCall(
            IL2ERC20Gateway.finalizeDepositERC20,
            (_token, l2USDC, _from, _to, _amount, abi.encode(_nonce, _data))
        );

        // 4. Send message to L1ScrollMessenger.
        IL1ScrollMessenger(messenger).sendMessage{value: msg.value}(counterpart, 0, _message, _gasLimit);

        emit DepositERC20(_token, l2USDC, _from, _to, _amount, _data);
    }
}
