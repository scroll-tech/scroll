// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { OwnableUpgradeable } from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";

import { IL1GatewayRouter } from "./IL1GatewayRouter.sol";
import { IL1ERC20Gateway } from "./IL1ERC20Gateway.sol";
import { IL1ScrollMessenger } from "../IL1ScrollMessenger.sol";
import { IL2GatewayRouter } from "../../L2/gateways/IL2GatewayRouter.sol";
import { IScrollGateway } from "../../libraries/gateway/IScrollGateway.sol";
import { ScrollGatewayBase } from "../../libraries/gateway/ScrollGatewayBase.sol";

/// @title L1GatewayRouter
/// @notice The `L1GatewayRouter` is the main entry for depositing Ether and ERC20 tokens.
/// All deposited tokens are routed to corresponding gateways.
/// @dev One can also use this contract to query L1/L2 token address mapping.
/// In the future, ERC-721 and ERC-1155 tokens will be added to the router too.
contract L1GatewayRouter is OwnableUpgradeable, ScrollGatewayBase, IL1GatewayRouter {
  /**************************************** Events ****************************************/

  event SetDefaultERC20Gateway(address indexed _defaultERC20Gateway);
  event SetERC20Gateway(address indexed _token, address indexed _gateway);

  /**************************************** Variables ****************************************/

  /// @notice The addess of default ERC20 gateway, normally the L1StandardERC20Gateway contract.
  address public defaultERC20Gateway;
  /// @notice Mapping from ERC20 token address to corresponding L1ERC20Gateway.
  // solhint-disable-next-line var-name-mixedcase
  mapping(address => address) public ERC20Gateway;

  // @todo: add ERC721/ERC1155 Gateway mapping.

  /**************************************** Constructor ****************************************/

  function initialize(
    address _defaultERC20Gateway,
    address _counterpart,
    address _messenger
  ) external initializer {
    OwnableUpgradeable.__Ownable_init();
    ScrollGatewayBase._initialize(_counterpart, address(0), _messenger);

    // it can be zero during initialization
    if (_defaultERC20Gateway != address(0)) {
      defaultERC20Gateway = _defaultERC20Gateway;
    }
  }

  /**************************************** View Functions ****************************************/

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

  /**************************************** Mutate Functions ****************************************/

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

  /// @inheritdoc IL1GatewayRouter
  function depositETH(uint256 _gasLimit) external payable override {
    depositETH(msg.sender, _gasLimit);
  }

  /// @inheritdoc IL1GatewayRouter
  function depositETH(address _to, uint256 _gasLimit) public payable override nonReentrant {
    require(msg.value > 0, "deposit zero eth");

    bytes memory _message = abi.encodeWithSelector(
      IL2GatewayRouter.finalizeDepositETH.selector,
      msg.sender,
      _to,
      msg.value,
      new bytes(0)
    );
    IL1ScrollMessenger(messenger).sendMessage{ value: msg.value }(counterpart, 0, _message, _gasLimit);

    emit DepositETH(msg.sender, _to, msg.value, "");
  }

  /// @inheritdoc IL1GatewayRouter
  function finalizeWithdrawETH(
    address _from,
    address _to,
    uint256 _amount,
    bytes calldata _data
  ) external payable override onlyCallByCounterpart {
    require(msg.value == _amount, "msg.value mismatch");

    // @note can possible trigger reentrant call to this contract or messenger,
    // but it seems not a big problem.
    // solhint-disable-next-line avoid-low-level-calls
    (bool _success, ) = _to.call{ value: _amount }("");
    require(_success, "ETH transfer failed");

    // @todo farward _data to `_to` in near future.

    emit FinalizeWithdrawETH(_from, _to, _amount, _data);
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

  /// @inheritdoc IScrollGateway
  function finalizeDropMessage() external payable virtual override onlyMessenger {
    // @todo should refund ETH back to sender.
  }

  /**************************************** Restricted Functions ****************************************/

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
