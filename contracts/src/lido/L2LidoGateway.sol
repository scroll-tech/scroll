// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

import {IL1ERC20Gateway} from "../L1/gateways/IL1ERC20Gateway.sol";
import {IL2ERC20Gateway} from "../L2/gateways/IL2ERC20Gateway.sol";
import {L2ERC20Gateway} from "../L2/gateways/L2ERC20Gateway.sol";
import {IL2ScrollMessenger} from "../L2/IL2ScrollMessenger.sol";
import {IScrollERC20Upgradeable} from "../libraries/token/IScrollERC20Upgradeable.sol";
import {ScrollGatewayBase} from "../libraries/gateway/ScrollGatewayBase.sol";

import {LidoBridgeableTokens} from "./LidoBridgeableTokens.sol";
import {LidoGatewayManager} from "./LidoGatewayManager.sol";

contract L2LidoGateway is L2ERC20Gateway, LidoBridgeableTokens, LidoGatewayManager {
    /**********
     * Errors *
     **********/

    /// @dev Thrown when withdraw zero amount token.
    error ErrorWithdrawZeroAmount();

    /*************
     * Variables *
     *************/

    /// @dev The initial version of `L2LidoGateway` use `L2CustomERC20Gateway`. We keep the storage
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

    /// @notice Initialize the storage of L2LidoGateway v1.
    function initialize(
        address _counterpart,
        address _router,
        address _messenger
    ) external initializer {
        require(_router != address(0), "zero router address");

        ScrollGatewayBase._initialize(_counterpart, _router, _messenger);
    }

    /// @notice Initialize the storage of L2LidoGateway v2.
    function initializeV2() external reinitializer(2) {
        __LidoGatewayManager_init();
    }

    /*************************
     * Public View Functions *
     *************************/

    /// @inheritdoc IL2ERC20Gateway
    function getL1ERC20Address(address _l2Token)
        external
        view
        override
        onlySupportedL2Token(_l2Token)
        returns (address)
    {
        return l1Token;
    }

    /// @inheritdoc IL2ERC20Gateway
    function getL2ERC20Address(address _l1Token)
        external
        view
        override
        onlySupportedL1Token(_l1Token)
        returns (address)
    {
        return l2Token;
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
    )
        external
        payable
        override
        onlyCallByCounterpart
        nonReentrant
        onlySupportedL1Token(_l1Token)
        onlySupportedL2Token(_l2Token)
        whenDepositsEnabled
    {
        if (msg.value != 0) revert ErrorNonZeroMsgValue();

        IScrollERC20Upgradeable(_l2Token).mint(_to, _amount);

        _doCallback(_to, _data);

        emit FinalizeDepositERC20(_l1Token, _l2Token, _from, _to, _amount, _data);
    }

    /**********************
     * Internal Functions *
     **********************/

    /// @inheritdoc L2ERC20Gateway
    function _withdraw(
        address _l2Token,
        address _to,
        uint256 _amount,
        bytes memory _data,
        uint256 _gasLimit
    )
        internal
        virtual
        override
        nonReentrant
        onlySupportedL2Token(_l2Token)
        onlyNonZeroAccount(_to)
        whenWithdrawalsEnabled
    {
        if (_amount == 0) revert ErrorWithdrawZeroAmount();

        // 1. Extract real sender if this call is from L2GatewayRouter.
        address _from = _msgSender();
        if (router == _from) {
            (_from, _data) = abi.decode(_data, (address, bytes));
        }

        // 2. Burn token.
        IScrollERC20Upgradeable(_l2Token).burn(_from, _amount);

        // 3. Generate message passed to L1LidoGateway.
        bytes memory _message = abi.encodeCall(
            IL1ERC20Gateway.finalizeWithdrawERC20,
            (l1Token, _l2Token, _from, _to, _amount, _data)
        );

        // 4. send message to L2ScrollMessenger
        IL2ScrollMessenger(messenger).sendMessage{value: msg.value}(counterpart, 0, _message, _gasLimit);

        emit WithdrawERC20(l1Token, _l2Token, _from, _to, _amount, _data);
    }
}
