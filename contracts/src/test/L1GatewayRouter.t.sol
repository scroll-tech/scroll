// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { DSTestPlus } from "solmate/test/utils/DSTestPlus.sol";
import { MockERC20 } from "solmate/test/utils/mocks/MockERC20.sol";

import { L1GatewayRouter } from "../L1/gateways/L1GatewayRouter.sol";
import { L1StandardERC20Gateway } from "../L1/gateways/L1StandardERC20Gateway.sol";
import { L1ScrollMessenger } from "../L1/L1ScrollMessenger.sol";
import { ZKRollup } from "../L1/rollup/ZKRollup.sol";
import { ScrollStandardERC20 } from "../libraries/token/ScrollStandardERC20.sol";
import { ScrollStandardERC20Factory } from "../libraries/token/ScrollStandardERC20Factory.sol";

contract L1GatewayRouterTest is DSTestPlus {
  ScrollStandardERC20 private template;
  ScrollStandardERC20Factory private factory;

  ZKRollup private rollup;
  L1ScrollMessenger private messenger;
  L1StandardERC20Gateway private gateway;
  L1GatewayRouter private router;
  MockERC20 private token;

  function setUp() public {
    rollup = new ZKRollup();
    rollup.initialize(233);

    template = new ScrollStandardERC20();

    factory = new ScrollStandardERC20Factory(address(template));

    token = new MockERC20("Mock Token", "M", 18);
    messenger = new L1ScrollMessenger();
    messenger.initialize(address(rollup));

    rollup.updateMessenger(address(messenger));

    router = new L1GatewayRouter();
    router.initialize(address(0), address(1), address(messenger));

    gateway = new L1StandardERC20Gateway();
    gateway.initialize(address(1), address(router), address(messenger), address(template), address(factory));

    router.setDefaultERC20Gateway(address(gateway));

    token.mint(address(this), type(uint256).max);
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
    router.depositERC20(address(token), 1, 0);

    hevm.expectRevert("no gateway available");
    router.depositERC20(address(token), address(this), 1, 0);

    hevm.expectRevert("no gateway available");
    router.depositERC20AndCall(address(token), address(this), 1, "", 0);
  }

  function testDepositERC20ZeroAmount() public {
    hevm.expectRevert("deposit zero amount");
    router.depositERC20(address(token), 0, 0);

    hevm.expectRevert("deposit zero amount");
    router.depositERC20(address(token), address(this), 0, 0);

    hevm.expectRevert("deposit zero amount");
    router.depositERC20AndCall(address(token), address(this), 0, "", 0);
  }

  function testDepositERC20(uint256 amount) public {
    if (amount == 0) amount = 1;

    uint256 gatewayBalance = token.balanceOf(address(gateway));
    router.depositERC20(address(token), amount, 0);
    assertEq(amount + gatewayBalance, token.balanceOf(address(gateway)));
  }

  function testDepositERC20(uint256 amount, address to) public {
    if (amount == 0) amount = 1;

    uint256 gatewayBalance = token.balanceOf(address(gateway));
    router.depositERC20(address(token), to, amount, 0);
    assertEq(amount + gatewayBalance, token.balanceOf(address(gateway)));
  }

  function testDepositERC20AndCall(
    uint256 amount,
    address to,
    bytes calldata data
  ) public {
    if (amount == 0) amount = 1;

    uint256 gatewayBalance = token.balanceOf(address(gateway));
    router.depositERC20AndCall(address(token), to, amount, data, 0);
    assertEq(amount + gatewayBalance, token.balanceOf(address(gateway)));
  }

  function testDepositETH(uint256 amount) public {
    amount = bound(amount, 0, address(this).balance);

    if (amount == 0) {
      hevm.expectRevert("deposit zero eth");
      router.depositETH{ value: amount }(0);
    } else {
      uint256 messengerBalance = address(messenger).balance;
      router.depositETH{ value: amount }(0);
      assertEq(amount + messengerBalance, address(messenger).balance);
    }
  }

  function testDepositETH(uint256 amount, address to) public {
    amount = bound(amount, 0, address(this).balance);

    if (amount == 0) {
      hevm.expectRevert("deposit zero eth");
      router.depositETH{ value: amount }(to, 0);
    } else {
      uint256 messengerBalance = address(messenger).balance;
      router.depositETH{ value: amount }(to, 0);
      assertEq(amount + messengerBalance, address(messenger).balance);
    }
  }

  function testFinalizeWithdrawERC20() public {
    hevm.expectRevert("should never be called");
    router.finalizeWithdrawERC20(address(0), address(0), address(0), address(0), 0, "");
  }
}
