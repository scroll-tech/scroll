// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

import {OwnableUpgradeable} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";

import {IL2GatewayRouter} from "./IL2GatewayRouter.sol";
import {IL2ETHGateway} from "./IL2ETHGateway.sol";
import {IL2ERC20Gateway} from "./IL2ERC20Gateway.sol";
import {IL2ScrollMessenger} from "../IL2ScrollMessenger.sol";
import {IL1ETHGateway} from "../../L1/gateways/IL1ETHGateway.sol";
import {IScrollGateway} from "../../libraries/gateway/IScrollGateway.sol";

/// @title L2GatewayRouter
/// @notice The `L2GatewayRouter` is the main entry for withdrawing Ether and ERC20 tokens.
/// All deposited tokens are routed to corresponding gateways.
/// @dev One can also use this contract to query L1/L2 token address mapping.
/// In the future, ERC-721 and ERC-1155 tokens will be added to the router too.
contract L2GatewayRouter is OwnableUpgradeable, IL2GatewayRouter {
    /*************
     * Variables *
     *************/

    /// @notice The address of L2ETHGateway.
    address public ethGateway;

    /// @notice The addess of default L2 ERC20 gateway, normally the L2StandardERC20Gateway contract.
    address public defaultERC20Gateway;

    /// @notice Mapping from L2 ERC20 token address to corresponding L2ERC20Gateway.
    // solhint-disable-next-line var-name-mixedcase
    mapping(address => address) public ERC20Gateway;

    /***************
     * Constructor *
     ***************/

    constructor() {
        _disableInitializers();
    }

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

    /// @inheritdoc IL2ERC20Gateway
    function getL2ERC20Address(address) external pure override returns (address) {
        revert("unsupported");
    }

    /// @inheritdoc IL2ERC20Gateway
    function getL1ERC20Address(address _l2Address) external view override returns (address) {
        address _gateway = getERC20Gateway(_l2Address);
        if (_gateway == address(0)) {
            return address(0);
        }

        return IL2ERC20Gateway(_gateway).getL1ERC20Address(_l2Address);
    }

    /// @notice Return the corresponding gateway address for given token address.
    /// @param _token The address of token to query.
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

    /// @inheritdoc IL2ERC20Gateway
    function withdrawERC20(
        address _token,
        uint256 _amount,
        uint256 _gasLimit
    ) external payable override {
        withdrawERC20AndCall(_token, msg.sender, _amount, new bytes(0), _gasLimit);
    }

    /// @inheritdoc IL2ERC20Gateway
    function withdrawERC20(
        address _token,
        address _to,
        uint256 _amount,
        uint256 _gasLimit
    ) external payable override {
        withdrawERC20AndCall(_token, _to, _amount, new bytes(0), _gasLimit);
    }

    /// @inheritdoc IL2ERC20Gateway
    function withdrawERC20AndCall(
        address _token,
        address _to,
        uint256 _amount,
        bytes memory _data,
        uint256 _gasLimit
    ) public payable override {
        address _gateway = getERC20Gateway(_token);
        require(_gateway != address(0), "no gateway available");

        // encode msg.sender with _data
        bytes memory _routerData = abi.encode(msg.sender, _data);

        IL2ERC20Gateway(_gateway).withdrawERC20AndCall{value: msg.value}(_token, _to, _amount, _routerData, _gasLimit);
    }

    /// @inheritdoc IL2ETHGateway
    function withdrawETH(uint256 _amount, uint256 _gasLimit) external payable override {
        withdrawETHAndCall(msg.sender, _amount, new bytes(0), _gasLimit);
    }

    /// @inheritdoc IL2ETHGateway
    function withdrawETH(
        address _to,
        uint256 _amount,
        uint256 _gasLimit
    ) external payable override {
        withdrawETHAndCall(_to, _amount, new bytes(0), _gasLimit);
    }

    /// @inheritdoc IL2ETHGateway
    function withdrawETHAndCall(
        address _to,
        uint256 _amount,
        bytes memory _data,
        uint256 _gasLimit
    ) public payable override {
        address _gateway = ethGateway;
        require(_gateway != address(0), "eth gateway available");

        // encode msg.sender with _data
        bytes memory _routerData = abi.encode(msg.sender, _data);

        IL2ETHGateway(_gateway).withdrawETHAndCall{value: msg.value}(_to, _amount, _routerData, _gasLimit);
    }

    /// @inheritdoc IL2ETHGateway
    function finalizeDepositETH(
        address,
        address,
        uint256,
        bytes calldata
    ) external payable virtual override {
        revert("should never be called");
    }

    /// @inheritdoc IL2ERC20Gateway
    function finalizeDepositERC20(
        address,
        address,
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

    /// @notice Update the address of ETH gateway contract.
    /// @dev This function should only be called by contract owner.
    /// @param _ethGateway The address to update.
    function setETHGateway(address _ethGateway) external onlyOwner {
        ethGateway = _ethGateway;

        emit SetETHGateway(_ethGateway);
    }

    /// @notice Update the address of default ERC20 gateway contract.
    /// @dev This function should only be called by contract owner.
    /// @param _defaultERC20Gateway The address to update.
    function setDefaultERC20Gateway(address _defaultERC20Gateway) external onlyOwner {
        defaultERC20Gateway = _defaultERC20Gateway;

        emit SetDefaultERC20Gateway(_defaultERC20Gateway);
    }

    /// @notice Update the mapping from token address to gateway address.
    /// @dev This function should only be called by contract owner.
    /// @param _tokens The list of addresses of tokens to update.
    /// @param _gateways The list of addresses of gateways to update.
    function setERC20Gateway(address[] memory _tokens, address[] memory _gateways) external onlyOwner {
        require(_tokens.length == _gateways.length, "length mismatch");

        for (uint256 i = 0; i < _tokens.length; i++) {
            ERC20Gateway[_tokens[i]] = _gateways[i];

            emit SetERC20Gateway(_tokens[i], _gateways[i]);
        }
    }
}
