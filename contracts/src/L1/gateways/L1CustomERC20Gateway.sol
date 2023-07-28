// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

import {OwnableUpgradeable} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import {IERC20Upgradeable} from "@openzeppelin/contracts-upgradeable/token/ERC20/IERC20Upgradeable.sol";
import {SafeERC20Upgradeable} from "@openzeppelin/contracts-upgradeable/token/ERC20/utils/SafeERC20Upgradeable.sol";

import {IL2ERC20Gateway} from "../../L2/gateways/IL2ERC20Gateway.sol";
import {IL1ScrollMessenger} from "../IL1ScrollMessenger.sol";
import {IL1ERC20Gateway} from "./IL1ERC20Gateway.sol";

import {ScrollGatewayBase} from "../../libraries/gateway/ScrollGatewayBase.sol";
import {L1ERC20Gateway} from "./L1ERC20Gateway.sol";

/// @title L1CustomERC20Gateway
/// @notice The `L1CustomERC20Gateway` is used to deposit custom ERC20 compatible tokens on layer 1 and
/// finalize withdraw the tokens from layer 2.
/// @dev The deposited tokens are held in this gateway. On finalizing withdraw, the corresponding
/// tokens will be transfer to the recipient directly.
contract L1CustomERC20Gateway is OwnableUpgradeable, ScrollGatewayBase, L1ERC20Gateway {
    using SafeERC20Upgradeable for IERC20Upgradeable;

    /**********
     * Events *
     **********/

    /// @notice Emitted when token mapping for ERC20 token is updated.
    /// @param _l1Token The address of ERC20 token on layer 1.
    /// @param _l2Token The address of corresponding ERC20 token on layer 2.
    event UpdateTokenMapping(address _l1Token, address _l2Token);

    /*************
     * Variables *
     *************/

    /// @notice Mapping from l1 token address to l2 token address for ERC20 token.
    mapping(address => address) public tokenMapping;

    /***************
     * Constructor *
     ***************/

    constructor() {
        _disableInitializers();
    }

    /// @notice Initialize the storage of L1CustomERC20Gateway.
    /// @param _counterpart The address of L2CustomERC20Gateway in L2.
    /// @param _router The address of L1GatewayRouter.
    /// @param _messenger The address of L1ScrollMessenger.
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

    /// @inheritdoc IL1ERC20Gateway
    function getL2ERC20Address(address _l1Token) public view override returns (address) {
        return tokenMapping[_l1Token];
    }

    /************************
     * Restricted Functions *
     ************************/

    /// @notice Update layer 1 to layer 2 token mapping.
    /// @param _l1Token The address of ERC20 token on layer 1.
    /// @param _l2Token The address of corresponding ERC20 token on layer 2.
    function updateTokenMapping(address _l1Token, address _l2Token) external onlyOwner {
        require(_l2Token != address(0), "token address cannot be 0");

        tokenMapping[_l1Token] = _l2Token;

        emit UpdateTokenMapping(_l1Token, _l2Token);
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
    ) internal virtual override {
        require(msg.value == 0, "nonzero msg.value");
        require(_l2Token != address(0), "token address cannot be 0");
        require(_l2Token == tokenMapping[_l1Token], "l2 token mismatch");
    }

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
        address _l2Token = tokenMapping[_token];
        require(_l2Token != address(0), "no corresponding l2 token");

        // 1. Transfer token into this contract.
        address _from;
        (_from, _amount, _data) = _transferERC20In(_token, _amount, _data);

        // 2. Generate message passed to L2CustomERC20Gateway.
        bytes memory _message = abi.encodeCall(
            IL2ERC20Gateway.finalizeDepositERC20,
            (_token, _l2Token, _from, _to, _amount, _data)
        );

        // 3. Send message to L1ScrollMessenger.
        IL1ScrollMessenger(messenger).sendMessage{value: msg.value}(counterpart, 0, _message, _gasLimit, _from);

        emit DepositERC20(_token, _l2Token, _from, _to, _amount, _data);
    }
}
