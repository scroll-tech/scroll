// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { OwnableUpgradeable } from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";

import { IL2GatewayRouter } from "../../L2/gateways/IL2GatewayRouter.sol";
import { IScrollGateway } from "../../libraries/gateway/IScrollGateway.sol";
import { IL1ScrollMessenger } from "../IL1ScrollMessenger.sol";
import { IL1ETHGateway } from "./IL1ETHGateway.sol";
import { IL1ERC20Gateway } from "./IL1ERC20Gateway.sol";
import { IL1GatewayRouter } from "./IL1GatewayRouter.sol";

import { ScrollGatewayBase } from "../../libraries/gateway/ScrollGatewayBase.sol";

/// @title L1GatewayRouter
/// @notice The `L1GatewayRouter` is the main entry for depositing Ether and ERC20 tokens.
/// All deposited tokens are routed to corresponding gateways.
/// @dev One can also use this contract to query L1/L2 token address mapping.
/// In the future, ERC-721 and ERC-1155 tokens will be added to the router too.
contract L1GatewayRouter is OwnableUpgradeable, ScrollGatewayBase, IL1GatewayRouter {
  /**********
   * Events *
   **********/

  /// @notice Emitted when the address of ETH Gateway is updated.
  /// @param ethGateway The address of new ETH Gateway.
  event SetETHGateway(address indexed ethGateway);

  /// @notice Emitted when the address of default ERC20 Gateway is updated.
  /// @param defaultERC20Gateway The address of new default ERC20 Gateway.
  event SetDefaultERC20Gateway(address indexed defaultERC20Gateway);

  /// @notice Emitted when the `gateway` for `token` is updated.
  /// @param token The address of token updated.
  /// @param gateway The corresponding address of gateway updated.
  event SetERC20Gateway(address indexed token, address indexed gateway);

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

  // @todo: add ERC721/ERC1155 Gateway mapping.

  /***************
   * Constructor *
   ***************/

  /// @notice Initialize the storage of L1GatewayRouter.
  /// @param _ethGateway The address of L1ETHGateway contract.
  /// @param _defaultERC20Gateway The address of default ERC20 Gateway contract.
  /// @param _counterpart The address of L2GatewayRouter in L2.
  /// @param _messenger The address of L1ScrollMessenger.
  function initialize(
    address _ethGateway,
    address _defaultERC20Gateway,
    address _counterpart,
    address _messenger
  ) external initializer {
    OwnableUpgradeable.__Ownable_init();
    ScrollGatewayBase._initialize(_counterpart, address(0), _messenger);

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

  /// @notice Return the corresponding gateway address for given token address.
  /// @param _token The address of token to query.
  function getERC20Gateway(address _token) public view returns (address) {
    address _gateway = ERC20Gateway[_token];
    if (_gateway == address(0)) {
      _gateway = defaultERC20Gateway;
    }
    return _gateway;
  }

  /************************************************
   * Public Mutated Functions from L1ERC20Gateway *
   ************************************************/

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
  ) public payable override nonReentrant {
    address _gateway = getERC20Gateway(_token);
    require(_gateway != address(0), "no gateway available");

    // encode msg.sender with _data
    bytes memory _routerData = abi.encode(msg.sender, _data);

    IL1ERC20Gateway(_gateway).depositERC20AndCall{ value: msg.value }(_token, _to, _amount, _routerData, _gasLimit);
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

  /**********************************************
   * Public Mutated Functions from L1ETHGateway *
   **********************************************/

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
  ) public payable override nonReentrant {
    address _gateway = ethGateway;
    require(_gateway != address(0), "eth gateway available");

    // encode msg.sender with _data
    bytes memory _routerData = abi.encode(msg.sender, _data);

    IL1ETHGateway(_gateway).depositETHAndCall{ value: msg.value }(_to, _amount, _routerData, _gasLimit);
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
