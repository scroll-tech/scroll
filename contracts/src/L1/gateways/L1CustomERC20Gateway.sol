// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { OwnableUpgradeable } from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import { IERC20Upgradeable } from "@openzeppelin/contracts-upgradeable/token/ERC20/IERC20Upgradeable.sol";
import { SafeERC20Upgradeable } from "@openzeppelin/contracts-upgradeable/token/ERC20/utils/SafeERC20Upgradeable.sol";

import { IL1ERC20Gateway, L1ERC20Gateway } from "./L1ERC20Gateway.sol";
import { IL1ScrollMessenger } from "../IL1ScrollMessenger.sol";
import { IL2ERC20Gateway } from "../../L2/gateways/IL2ERC20Gateway.sol";
import { ScrollGatewayBase, IScrollGateway } from "../../libraries/gateway/ScrollGatewayBase.sol";

/// @title L1CustomERC20Gateway
/// @notice The `L1CustomERC20Gateway` is used to deposit custom ERC20 compatible tokens in layer 1 and
/// finalize withdraw the tokens from layer 2.
/// @dev The deposited tokens are held in this gateway. On finalizing withdraw, the corresponding
/// tokens will be transfer to the recipient directly.
contract L1CustomERC20Gateway is OwnableUpgradeable, ScrollGatewayBase, L1ERC20Gateway {
  using SafeERC20Upgradeable for IERC20Upgradeable;

  /**************************************** Events ****************************************/

  /// @notice Emitted when token mapping for ERC20 token is updated.
  /// @param _l1Token The address of ERC20 token in layer 1.
  /// @param _l2Token The address of corresponding ERC20 token in layer 2.
  event UpdateTokenMapping(address _l1Token, address _l2Token);

  /**************************************** Variables ****************************************/

  /// @notice Mapping from l1 token address to l2 token address for ERC20 token.
  // solhint-disable-next-line var-name-mixedcase
  mapping(address => address) public tokenMapping;

  /**************************************** Constructor ****************************************/

  function initialize(
    address _counterpart,
    address _router,
    address _messenger
  ) external initializer {
    require(_router != address(0), "zero router address");

    OwnableUpgradeable.__Ownable_init();
    ScrollGatewayBase._initialize(_counterpart, _router, _messenger);
  }

  /**************************************** View Functions ****************************************/

  /// @inheritdoc IL1ERC20Gateway
  function getL2ERC20Address(address _l1Token) public view override returns (address) {
    return tokenMapping[_l1Token];
  }

  /**************************************** Mutate Functions ****************************************/

  /// @inheritdoc IL1ERC20Gateway
  function finalizeWithdrawERC20(
    address _l1Token,
    address _l2Token,
    address _from,
    address _to,
    uint256 _amount,
    bytes calldata _data
  ) external payable override onlyCallByCounterpart {
    require(msg.value == 0, "nonzero msg.value");

    // @note can possible trigger reentrant call to this contract or messenger,
    // but it seems not a big problem.
    IERC20Upgradeable(_l1Token).safeTransfer(_to, _amount);

    // @todo forward `_data` to `_to` in the near future

    emit FinalizeWithdrawERC20(_l1Token, _l2Token, _from, _to, _amount, _data);
  }

  /// @inheritdoc IScrollGateway
  function finalizeDropMessage() external payable {
    // @todo finish the logic later
  }

  /**************************************** Restricted Functions ****************************************/

  /// @notice Update layer 1 to layer 2 token mapping.
  /// @param _l1Token The address of ERC20 token in layer 1.
  /// @param _l2Token The address of corresponding ERC20 token in layer 2.
  function updateTokenMapping(address _l1Token, address _l2Token) external onlyOwner {
    require(_l2Token != address(0), "map to zero address");

    tokenMapping[_l1Token] = _l2Token;

    emit UpdateTokenMapping(_l1Token, _l2Token);
  }

  /**************************************** Internal Functions ****************************************/

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

    // 1. Extract real sender if this call is from L1GatewayRouter.
    address _from = msg.sender;
    if (router == msg.sender) {
      (_from, _data) = abi.decode(_data, (address, bytes));
    }

    // 2. Transfer token into this contract.
    {
      // common practice to handle fee on transfer token.
      uint256 _before = IERC20Upgradeable(_token).balanceOf(address(this));
      IERC20Upgradeable(_token).safeTransferFrom(_from, address(this), _amount);
      uint256 _after = IERC20Upgradeable(_token).balanceOf(address(this));
      // no unchecked here, since some weird token may return arbitrary balance.
      _amount = _after - _before;
      // ignore weird fee on transfer token
      require(_amount > 0, "deposit zero amount");
    }

    // 3. Generate message passed to L2StandardERC20Gateway.
    bytes memory _message = abi.encodeWithSelector(
      IL2ERC20Gateway.finalizeDepositERC20.selector,
      _token,
      _l2Token,
      _from,
      _to,
      _amount,
      _data
    );

    // 4. Send message to L1ScrollMessenger.
    IL1ScrollMessenger(messenger).sendMessage{ value: msg.value }(counterpart, msg.value, _message, _gasLimit);

    emit DepositERC20(_token, _l2Token, _from, _to, _amount, _data);
  }
}
