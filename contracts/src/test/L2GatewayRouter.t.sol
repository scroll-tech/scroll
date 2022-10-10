// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { DSTestPlus } from "solmate/test/utils/DSTestPlus.sol";
import { MockERC20 } from "solmate/test/utils/mocks/MockERC20.sol";

import { L2GatewayRouter } from "../L2/gateways/L2GatewayRouter.sol";
import { L2StandardERC20Gateway } from "../L2/gateways/L2StandardERC20Gateway.sol";
import { L2ScrollMessenger } from "../L2/L2ScrollMessenger.sol";
import { ScrollStandardERC20 } from "../libraries/token/ScrollStandardERC20.sol";
import { ScrollStandardERC20Factory } from "../libraries/token/ScrollStandardERC20Factory.sol";
import { MockScrollMessenger } from "./mocks/MockScrollMessenger.sol";

contract L2GatewayRouterTest is DSTestPlus {
  ScrollStandardERC20 private template;
  ScrollStandardERC20Factory private factory;

  MockScrollMessenger private messenger;
  L2StandardERC20Gateway private gateway;
  L2GatewayRouter private router;
  ScrollStandardERC20 private token;

  function setUp() public {
    template = new ScrollStandardERC20();

    factory = new ScrollStandardERC20Factory(address(template));

    messenger = new MockScrollMessenger();
    router = new L2GatewayRouter();
    router.initialize(address(0), address(1), address(messenger));

    gateway = new L2StandardERC20Gateway();
    gateway.initialize(address(1), address(router), address(messenger), address(factory));

    router.setDefaultERC20Gateway(address(gateway));
    factory.transferOwnership(address(gateway));

    // deploy l2 token
    MockERC20 l1Token = new MockERC20("L1", "L1", 18);
    token = ScrollStandardERC20(gateway.getL2ERC20Address(address(l1Token)));
    assertEq(gateway.getL1ERC20Address(address(token)), address(0));
    messenger.setXDomainMessageSender(address(1));
    messenger.callTarget(
      address(gateway),
      abi.encodeWithSelector(
        L2StandardERC20Gateway.finalizeDepositERC20.selector,
        address(l1Token),
        address(token),
        address(this),
        address(this),
        type(uint256).max,
        abi.encode("", abi.encode("symbol", "name", 18))
      )
    );
    messenger.setXDomainMessageSender(address(0));
    token.approve(address(gateway), type(uint256).max);
  }

  function testOwnership() public {
    assertEq(address(this), router.owner());
  }

  function testReinitilize() public {
    hevm.expectRevert("Initializable: contract is already initialized");
    router.initialize(address(0), address(1), address(messenger));
  }

  function testSetDefaultERC20Gateway() public {
    router.setDefaultERC20Gateway(address(0));

    // set by non-owner, should revert
    hevm.startPrank(address(1));
    hevm.expectRevert("Ownable: caller is not the owner");
    router.setDefaultERC20Gateway(address(gateway));
    hevm.stopPrank();

    // set by owner, should succeed
    assertEq(address(0), router.getERC20Gateway(address(token)));
    assertEq(address(0), router.defaultERC20Gateway());
    router.setDefaultERC20Gateway(address(gateway));
    assertEq(address(gateway), router.defaultERC20Gateway());
    assertEq(address(gateway), router.getERC20Gateway(address(token)));
  }

  function testSetERC20Gateway() public {
    router.setDefaultERC20Gateway(address(0));

    // set by non-owner, should revert
    hevm.startPrank(address(1));
    hevm.expectRevert("Ownable: caller is not the owner");
    router.setDefaultERC20Gateway(address(gateway));
    hevm.stopPrank();

    // length mismatch, should revert
    address[] memory empty = new address[](0);
    address[] memory single = new address[](1);
    hevm.expectRevert("length mismatch");
    router.setERC20Gateway(empty, single);
    hevm.expectRevert("length mismatch");
    router.setERC20Gateway(single, empty);

    // set by owner, should succeed
    address[] memory _tokens = new address[](1);
    address[] memory _gateways = new address[](1);
    _tokens[0] = address(token);
    _gateways[0] = address(gateway);
    assertEq(address(0), router.getERC20Gateway(address(token)));
    router.setERC20Gateway(_tokens, _gateways);
    assertEq(address(gateway), router.getERC20Gateway(address(token)));
  }

  function testDepositERC20WhenNoGateway() public {
    router.setDefaultERC20Gateway(address(0));

    hevm.expectRevert("no gateway available");
    router.withdrawERC20(address(token), 1, 0);

    hevm.expectRevert("no gateway available");
    router.withdrawERC20(address(token), address(this), 1, 0);

    hevm.expectRevert("no gateway available");
    router.withdrawERC20AndCall(address(token), address(this), 1, "", 0);
  }

  function testDepositERC20ZeroAmount() public {
    hevm.expectRevert("withdraw zero amount");
    router.withdrawERC20(address(token), 0, 0);

    hevm.expectRevert("withdraw zero amount");
    router.withdrawERC20(address(token), address(this), 0, 0);

    hevm.expectRevert("withdraw zero amount");
    router.withdrawERC20AndCall(address(token), address(this), 0, "", 0);
  }

  function testDepositERC20(uint256 amount) public {
    amount = bound(amount, 1, token.balanceOf(address(this)));

    uint256 myBalance = token.balanceOf(address(this));
    assertEq(token.balanceOf(address(gateway)), 0);
    router.withdrawERC20(address(token), amount, 0);
    assertEq(myBalance - amount, token.balanceOf(address(this)));
    assertEq(token.balanceOf(address(gateway)), 0);
  }

  function testDepositERC20(uint256 amount, address to) public {
    amount = bound(amount, 1, token.balanceOf(address(this)));

    uint256 myBalance = token.balanceOf(address(this));
    assertEq(token.balanceOf(address(gateway)), 0);
    router.withdrawERC20(address(token), to, amount, 0);
    assertEq(myBalance - amount, token.balanceOf(address(this)));
    assertEq(token.balanceOf(address(gateway)), 0);
  }

  function testDepositERC20AndCall(
    uint256 amount,
    address to,
    bytes calldata data
  ) public {
    amount = bound(amount, 1, token.balanceOf(address(this)));

    uint256 myBalance = token.balanceOf(address(this));
    assertEq(token.balanceOf(address(gateway)), 0);
    router.withdrawERC20AndCall(address(token), to, amount, data, 0);
    assertEq(myBalance - amount, token.balanceOf(address(this)));
    assertEq(token.balanceOf(address(gateway)), 0);
  }

  function testWithdrawETH(uint256 amount) public {
    amount = bound(amount, 0, address(this).balance);

    if (amount == 0) {
      hevm.expectRevert("withdraw zero eth");
      router.withdrawETH{ value: amount }(0);
    } else {
      uint256 messengerBalance = address(messenger).balance;
      router.withdrawETH{ value: amount }(0);
      assertEq(amount + messengerBalance, address(messenger).balance);
    }
  }

  function testWithdrawETH(uint256 amount, address to) public {
    amount = bound(amount, 0, address(this).balance);

    if (amount == 0) {
      hevm.expectRevert("withdraw zero eth");
      router.withdrawETH{ value: amount }(to, 0);
    } else {
      uint256 messengerBalance = address(messenger).balance;
      router.withdrawETH{ value: amount }(to, 0);
      assertEq(amount + messengerBalance, address(messenger).balance);
    }
  }

  function testFinalizeDepositERC20() public {
    hevm.expectRevert("should never be called");
    router.finalizeDepositERC20(address(0), address(0), address(0), address(0), 0, "");
  }
}
