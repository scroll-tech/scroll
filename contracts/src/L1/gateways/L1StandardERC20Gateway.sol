// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

import {Initializable} from "@openzeppelin/contracts/proxy/utils/Initializable.sol";
import {IERC20} from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import {SafeERC20} from "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import {Clones} from "@openzeppelin/contracts/proxy/Clones.sol";

import {IERC20Metadata} from "../../interfaces/IERC20Metadata.sol";
import {IL2ERC20Gateway} from "../../L2/gateways/IL2ERC20Gateway.sol";
import {IL1ScrollMessenger} from "../IL1ScrollMessenger.sol";
import {IL1ERC20Gateway} from "./IL1ERC20Gateway.sol";

import {ScrollGatewayBase} from "../../libraries/gateway/ScrollGatewayBase.sol";
import {L1ERC20Gateway} from "./L1ERC20Gateway.sol";

/// @title L1StandardERC20Gateway
/// @notice The `L1StandardERC20Gateway` is used to deposit standard ERC20 tokens on layer 1 and
/// finalize withdraw the tokens from layer 2.
/// @dev The deposited ERC20 tokens are held in this gateway. On finalizing withdraw, the corresponding
/// token will be transfer to the recipient directly. Any ERC20 that requires non-standard functionality
/// should use a separate gateway.
contract L1StandardERC20Gateway is Initializable, ScrollGatewayBase, L1ERC20Gateway {
    using SafeERC20 for IERC20;

    /*************
     * Variables *
     *************/

    /// @notice The address of ScrollStandardERC20 implementation in L2.
    address public l2TokenImplementation;

    /// @notice The address of ScrollStandardERC20Factory contract in L2.
    address public l2TokenFactory;

    /// @notice Mapping from l1 token address to l2 token address.
    /// @dev This is not necessary, since we can compute the address directly. But, we use this mapping
    /// to keep track on whether we have deployed the token in L2 using the L2ScrollStandardERC20Factory and
    /// pass deploy data on first call to the token.
    mapping(address => address) private tokenMapping;

    /***************
     * Constructor *
     ***************/

    constructor() {
        _disableInitializers();
    }

    /// @notice Initialize the storage of L1StandardERC20Gateway.
    /// @param _counterpart The address of L2StandardERC20Gateway in L2.
    /// @param _router The address of L1GatewayRouter.
    /// @param _messenger The address of L1ScrollMessenger.
    /// @param _l2TokenImplementation The address of ScrollStandardERC20 implementation in L2.
    /// @param _l2TokenFactory The address of ScrollStandardERC20Factory contract in L2.
    function initialize(
        address _counterpart,
        address _router,
        address _messenger,
        address _l2TokenImplementation,
        address _l2TokenFactory
    ) external initializer {
        require(_router != address(0), "zero router address");
        ScrollGatewayBase._initialize(_counterpart, _router, _messenger);

        require(_l2TokenImplementation != address(0), "zero implementation hash");
        require(_l2TokenFactory != address(0), "zero factory address");

        l2TokenImplementation = _l2TokenImplementation;
        l2TokenFactory = _l2TokenFactory;
    }

    /*************************
     * Public View Functions *
     *************************/

    /// @inheritdoc IL1ERC20Gateway
    function getL2ERC20Address(address _l1Token) public view override returns (address) {
        // In StandardERC20Gateway, all corresponding l2 tokens are depoyed by Create2 with salt,
        // we can calculate the l2 address directly.
        bytes32 _salt = keccak256(abi.encodePacked(counterpart, keccak256(abi.encodePacked(_l1Token))));

        return Clones.predictDeterministicAddress(l2TokenImplementation, _salt, l2TokenFactory);
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
        require(getL2ERC20Address(_l1Token) == _l2Token, "l2 token mismatch");

        // update `tokenMapping` on first withdraw
        address _storedL2Token = tokenMapping[_l1Token];
        if (_storedL2Token == address(0)) {
            tokenMapping[_l1Token] = _l2Token;
        } else {
            require(_storedL2Token == _l2Token, "l2 token mismatch");
        }
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
        require(_amount > 0, "deposit zero amount");

        // 1. Transfer token into this contract.
        address _from;
        (_from, _amount, _data) = _transferERC20In(_token, _amount, _data);

        // 2. Generate message passed to L2StandardERC20Gateway.
        address _l2Token = tokenMapping[_token];
        bytes memory _l2Data;
        if (_l2Token == address(0)) {
            // @note we won't update `tokenMapping` here but update the `tokenMapping` on
            // first successful withdraw. This will prevent user to set arbitrary token
            // metadata by setting a very small `_gasLimit` on the first tx.
            _l2Token = getL2ERC20Address(_token);

            // passing symbol/name/decimal in order to deploy in L2.
            string memory _symbol = IERC20Metadata(_token).symbol();
            string memory _name = IERC20Metadata(_token).name();
            uint8 _decimals = IERC20Metadata(_token).decimals();
            _l2Data = abi.encode(true, abi.encode(_data, abi.encode(_symbol, _name, _decimals)));
        } else {
            _l2Data = abi.encode(false, _data);
        }
        bytes memory _message = abi.encodeCall(
            IL2ERC20Gateway.finalizeDepositERC20,
            (_token, _l2Token, _from, _to, _amount, _l2Data)
        );

        // 3. Send message to L1ScrollMessenger.
        IL1ScrollMessenger(messenger).sendMessage{value: msg.value}(counterpart, 0, _message, _gasLimit, _from);

        emit DepositERC20(_token, _l2Token, _from, _to, _amount, _data);
    }
}
