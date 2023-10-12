// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

import {IL1ERC20Gateway} from "../L1/gateways/IL1ERC20Gateway.sol";
import {IL1ScrollMessenger} from "../L1/IL1ScrollMessenger.sol";
import {IL2ERC20Gateway} from "../L2/gateways/IL2ERC20Gateway.sol";

import {L1ERC20Gateway} from "../L1/gateways/L1ERC20Gateway.sol";
import {LidoBridgeableTokens} from "./LidoBridgeableTokens.sol";
import {LidoGatewayManager} from "./LidoGatewayManager.sol";

contract L1LidoGateway is L1ERC20Gateway, LidoBridgeableTokens, LidoGatewayManager {
    /**********
     * Errors *
     **********/

    /// @dev Thrown when deposit zero amount token.
    error ErrorDepositZeroAmount();

    /*************
     * Variables *
     *************/

    /// @dev The initial version of `L1LidoGateway` use `L1CustomERC20Gateway`. We keep the storage
    /// slot for `tokenMapping` for compatibility. It should no longer be used.
    mapping(address => address) private __tokenMapping;

    /***************
     * Constructor *
     ***************/

    /// @param _l1Token The address of the bridged token in the L1 chain
    /// @param _l2Token The address of the token minted on the L2 chain when token bridged
    constructor(address _l1Token, address _l2Token) LidoBridgeableTokens(_l1Token, _l2Token) {
        _disableInitializers();
    }

    /// @notice Initialize the storage of L1LidoGateway.
    function initializeV2() external reinitializer(2) {
        __LidoGatewayManager_init();
    }

    /*************************
     * Public View Functions *
     *************************/

    /// @inheritdoc IL1ERC20Gateway
    function getL2ERC20Address(address _l1Token)
        external
        view
        override
        onlySupportedL1Token(_l1Token)
        returns (address)
    {
        return l2Token;
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
        uint256,
        bytes calldata
    ) internal virtual override onlySupportedL1Token(_l1Token) onlySupportedL2Token(_l2Token) whenWithdrawalsEnabled {
        if (msg.value != 0) revert ErrorNonZeroMsgValue();
    }

    /// @inheritdoc L1ERC20Gateway
    function _beforeDropMessage(
        address _token,
        address,
        uint256
    ) internal virtual override onlySupportedL1Token(_token) {
        if (msg.value != 0) revert ErrorNonZeroMsgValue();
    }

    /// @inheritdoc L1ERC20Gateway
    function _deposit(
        address _token,
        address _to,
        uint256 _amount,
        bytes memory _data,
        uint256 _gasLimit
    ) internal virtual override nonReentrant onlySupportedL1Token(_token) onlyNonZeroAccount(_to) whenDepositsEnabled {
        if (_amount == 0) revert ErrorDepositZeroAmount();

        // 1. Transfer token into this contract.
        address _from;
        (_from, _amount, _data) = _transferERC20In(_token, _amount, _data);

        // 2. Generate message passed to L2LidoGateway.
        bytes memory _message = abi.encodeCall(
            IL2ERC20Gateway.finalizeDepositERC20,
            (_token, l2Token, _from, _to, _amount, _data)
        );

        // 3. Send message to L1ScrollMessenger.
        IL1ScrollMessenger(messenger).sendMessage{value: msg.value}(counterpart, 0, _message, _gasLimit, _from);

        emit DepositERC20(_token, l2Token, _from, _to, _amount, _data);
    }
}
