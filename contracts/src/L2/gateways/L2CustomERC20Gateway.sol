// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

import {OwnableUpgradeable} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import {IERC20Upgradeable} from "@openzeppelin/contracts-upgradeable/token/ERC20/IERC20Upgradeable.sol";

import {IL2ERC20Gateway, L2ERC20Gateway} from "./L2ERC20Gateway.sol";
import {IL2ScrollMessenger} from "../IL2ScrollMessenger.sol";
import {IL1ERC20Gateway} from "../../L1/gateways/IL1ERC20Gateway.sol";
import {ScrollGatewayBase, IScrollGateway} from "../../libraries/gateway/ScrollGatewayBase.sol";
import {IScrollERC20} from "../../libraries/token/IScrollERC20.sol";

/// @title L2ERC20Gateway
/// @notice The `L2ERC20Gateway` is used to withdraw custom ERC20 compatible tokens on layer 2 and
/// finalize deposit the tokens from layer 1.
/// @dev The withdrawn tokens tokens will be burned directly. On finalizing deposit, the corresponding
/// tokens will be minted and transfered to the recipient.
contract L2CustomERC20Gateway is OwnableUpgradeable, ScrollGatewayBase, L2ERC20Gateway {
    /**********
     * Events *
     **********/

    /// @notice Emitted when token mapping for ERC20 token is updated.
    /// @param _l2Token The address of corresponding ERC20 token on layer 2.
    /// @param _l1Token The address of ERC20 token on layer 1.
    event UpdateTokenMapping(address _l2Token, address _l1Token);

    /*************
     * Variables *
     *************/

    /// @notice Mapping from layer 2 token address to layer 1 token address for ERC20 token.
    // solhint-disable-next-line var-name-mixedcase
    mapping(address => address) public tokenMapping;

    /***************
     * Constructor *
     ***************/
    constructor() {
        _disableInitializers();
    }

    function initialize(
        address _counterpart,
        address _router,
        address _messenger
    ) external initializer {
        require(_router != address(0), "zero router address");
        OwnableUpgradeable.__Ownable_init();

        ScrollGatewayBase._initialize(_counterpart, _router, _messenger);
    }

    /*************************
     * Public View Functions *
     *************************/

    /// @inheritdoc IL2ERC20Gateway
    function getL1ERC20Address(address _l2Token) external view override returns (address) {
        return tokenMapping[_l2Token];
    }

    /// @inheritdoc IL2ERC20Gateway
    function getL2ERC20Address(address) public pure override returns (address) {
        revert("unimplemented");
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
        require(_l1Token != address(0), "token address cannot be 0");
        require(_l1Token == tokenMapping[_l2Token], "l1 token mismatch");

        IScrollERC20(_l2Token).mint(_to, _amount);

        _doCallback(_to, _data);

        emit FinalizeDepositERC20(_l1Token, _l2Token, _from, _to, _amount, _data);
    }

    /************************
     * Restricted Functions *
     ************************/

    /// @notice Update layer 2 to layer 1 token mapping.
    /// @param _l2Token The address of corresponding ERC20 token on layer 2.
    /// @param _l1Token The address of ERC20 token on layer 1.
    function updateTokenMapping(address _l2Token, address _l1Token) external onlyOwner {
        require(_l1Token != address(0), "token address cannot be 0");

        tokenMapping[_l2Token] = _l1Token;

        emit UpdateTokenMapping(_l2Token, _l1Token);
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
        address _l1Token = tokenMapping[_token];
        require(_l1Token != address(0), "no corresponding l1 token");

        require(_amount > 0, "withdraw zero amount");

        // 1. Extract real sender if this call is from L2GatewayRouter.
        address _from = msg.sender;
        if (router == msg.sender) {
            (_from, _data) = abi.decode(_data, (address, bytes));
        }

        // 2. Burn token.
        IScrollERC20(_token).burn(_from, _amount);

        // 3. Generate message passed to L1StandardERC20Gateway.
        bytes memory _message = abi.encodeCall(
            IL1ERC20Gateway.finalizeWithdrawERC20,
            (_l1Token, _token, _from, _to, _amount, _data)
        );

        // 4. send message to L2ScrollMessenger
        IL2ScrollMessenger(messenger).sendMessage{value: msg.value}(counterpart, 0, _message, _gasLimit);

        emit WithdrawERC20(_l1Token, _token, _from, _to, _amount, _data);
    }
}
