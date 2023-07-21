// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

import {OwnableUpgradeable} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import {IERC20Upgradeable} from "@openzeppelin/contracts-upgradeable/token/ERC20/IERC20Upgradeable.sol";
import {SafeERC20Upgradeable} from "@openzeppelin/contracts-upgradeable/token/ERC20/utils/SafeERC20Upgradeable.sol";

import {IScrollGateway} from "../../libraries/gateway/IScrollGateway.sol";
import {IL1ScrollMessenger} from "../IL1ScrollMessenger.sol";
import {IL1ETHGateway} from "./IL1ETHGateway.sol";
import {IL1ERC20Gateway} from "./IL1ERC20Gateway.sol";
import {IL1GatewayRouter} from "./IL1GatewayRouter.sol";

/// @title L1GatewayRouter
/// @notice The `L1GatewayRouter` is the main entry for depositing Ether and ERC20 tokens.
/// All deposited tokens are routed to corresponding gateways.
/// @dev One can also use this contract to query L1/L2 token address mapping.
/// In the future, ERC-721 and ERC-1155 tokens will be added to the router too.
contract L1GatewayRouter is OwnableUpgradeable, IL1GatewayRouter {
    using SafeERC20Upgradeable for IERC20Upgradeable;

    /*************
     * Variables *
     *************/

    /// @notice The address of L1ETHGateway.
    address public ethGateway;

    /// @notice The addess of default ERC20 gateway, normally the L1StandardERC20Gateway contract.
    address public defaultERC20Gateway;

    /// @notice Mapping from ERC20 token address to corresponding L1ERC20Gateway.
    // solhint-disable-next-line var-name-mixedcase
    mapping(address => address) public ERC20Gateway;

    /// @notice The address of gateway in current execution context.
    address public gatewayInContext;

    /**********************
     * Function Modifiers *
     **********************/

    modifier onlyNotInContext() {
        require(gatewayInContext == address(0), "Only not in context");
        _;
    }

    modifier onlyInContext() {
        require(msg.sender == gatewayInContext, "Only in deposit context");
        _;
    }

    /***************
     * Constructor *
     ***************/

    constructor() {
        _disableInitializers();
    }

    /// @notice Initialize the storage of L1GatewayRouter.
    /// @param _ethGateway The address of L1ETHGateway contract.
    /// @param _defaultERC20Gateway The address of default ERC20 Gateway contract.
    function initialize(address _ethGateway, address _defaultERC20Gateway) external initializer {
        OwnableUpgradeable.__Ownable_init();

        // it can be zero during initialization
        if (_defaultERC20Gateway != address(0)) {
            defaultERC20Gateway = _defaultERC20Gateway;
            emit SetDefaultERC20Gateway(_defaultERC20Gateway);
        }

        // it can be zero during initialization
        if (_ethGateway != address(0)) {
            ethGateway = _ethGateway;
            emit SetETHGateway(_ethGateway);
        }
    }

    /*************************
     * Public View Functions *
     *************************/

    /// @inheritdoc IL1ERC20Gateway
    function getL2ERC20Address(address _l1Address) external view override returns (address) {
        address _gateway = getERC20Gateway(_l1Address);
        if (_gateway == address(0)) {
            return address(0);
        }

        return IL1ERC20Gateway(_gateway).getL2ERC20Address(_l1Address);
    }

    /// @inheritdoc IL1GatewayRouter
    function getERC20Gateway(address _token) public view returns (address) {
        address _gateway = ERC20Gateway[_token];
        if (_gateway == address(0)) {
            _gateway = defaultERC20Gateway;
        }
        return _gateway;
    }

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @inheritdoc IL1GatewayRouter
    /// @dev All the gateways should have reentrancy guard to prevent potential attack though this function.
    function requestERC20(
        address _sender,
        address _token,
        uint256 _amount
    ) external onlyInContext returns (uint256) {
        uint256 _balance = IERC20Upgradeable(_token).balanceOf(msg.sender);
        IERC20Upgradeable(_token).safeTransferFrom(_sender, msg.sender, _amount);
        _amount = IERC20Upgradeable(_token).balanceOf(msg.sender) - _balance;
        return _amount;
    }

    /*************************************************
     * Public Mutating Functions from L1ERC20Gateway *
     *************************************************/

    /// @inheritdoc IL1ERC20Gateway
    function depositERC20(
        address _token,
        uint256 _amount,
        uint256 _gasLimit
    ) external payable override {
        depositERC20AndCall(_token, msg.sender, _amount, new bytes(0), _gasLimit);
    }

    /// @inheritdoc IL1ERC20Gateway
    function depositERC20(
        address _token,
        address _to,
        uint256 _amount,
        uint256 _gasLimit
    ) external payable override {
        depositERC20AndCall(_token, _to, _amount, new bytes(0), _gasLimit);
    }

    /// @inheritdoc IL1ERC20Gateway
    function depositERC20AndCall(
        address _token,
        address _to,
        uint256 _amount,
        bytes memory _data,
        uint256 _gasLimit
    ) public payable override onlyNotInContext {
        address _gateway = getERC20Gateway(_token);
        require(_gateway != address(0), "no gateway available");

        // enter deposit context
        gatewayInContext = _gateway;

        // encode msg.sender with _data
        bytes memory _routerData = abi.encode(msg.sender, _data);

        IL1ERC20Gateway(_gateway).depositERC20AndCall{value: msg.value}(_token, _to, _amount, _routerData, _gasLimit);

        // leave deposit context
        gatewayInContext = address(0);
    }

    /// @inheritdoc IL1ERC20Gateway
    function finalizeWithdrawERC20(
        address,
        address,
        address,
        address,
        uint256,
        bytes calldata
    ) external payable virtual override {
        revert("should never be called");
    }

    /***********************************************
     * Public Mutating Functions from L1ETHGateway *
     ***********************************************/

    /// @inheritdoc IL1ETHGateway
    function depositETH(uint256 _amount, uint256 _gasLimit) external payable override {
        depositETHAndCall(msg.sender, _amount, new bytes(0), _gasLimit);
    }

    /// @inheritdoc IL1ETHGateway
    function depositETH(
        address _to,
        uint256 _amount,
        uint256 _gasLimit
    ) external payable override {
        depositETHAndCall(_to, _amount, new bytes(0), _gasLimit);
    }

    /// @inheritdoc IL1ETHGateway
    function depositETHAndCall(
        address _to,
        uint256 _amount,
        bytes memory _data,
        uint256 _gasLimit
    ) public payable override onlyNotInContext {
        address _gateway = ethGateway;
        require(_gateway != address(0), "eth gateway available");

        // encode msg.sender with _data
        bytes memory _routerData = abi.encode(msg.sender, _data);

        IL1ETHGateway(_gateway).depositETHAndCall{value: msg.value}(_to, _amount, _routerData, _gasLimit);
    }

    /// @inheritdoc IL1ETHGateway
    function finalizeWithdrawETH(
        address,
        address,
        uint256,
        bytes calldata
    ) external payable virtual override {
        revert("should never be called");
    }

    /************************
     * Restricted Functions *
     ************************/

    /// @inheritdoc IL1GatewayRouter
    function setETHGateway(address _ethGateway) external onlyOwner {
        ethGateway = _ethGateway;

        emit SetETHGateway(_ethGateway);
    }

    /// @inheritdoc IL1GatewayRouter
    function setDefaultERC20Gateway(address _defaultERC20Gateway) external onlyOwner {
        defaultERC20Gateway = _defaultERC20Gateway;

        emit SetDefaultERC20Gateway(_defaultERC20Gateway);
    }

    /// @inheritdoc IL1GatewayRouter
    function setERC20Gateway(address[] memory _tokens, address[] memory _gateways) external onlyOwner {
        require(_tokens.length == _gateways.length, "length mismatch");

        for (uint256 i = 0; i < _tokens.length; i++) {
            ERC20Gateway[_tokens[i]] = _gateways[i];

            emit SetERC20Gateway(_tokens[i], _gateways[i]);
        }
    }
}
