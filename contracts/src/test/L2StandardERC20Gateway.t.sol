// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { console2 } from "forge-std/console2.sol";
import { DSTestPlus } from "solmate/test/utils/DSTestPlus.sol";
import { MockERC20 } from "solmate/test/utils/mocks/MockERC20.sol";
import { WETH } from "solmate/tokens/WETH.sol";

import { L1StandardERC20Gateway } from "../L1/gateways/L1StandardERC20Gateway.sol";
import { L2GatewayRouter } from "../L2/gateways/L2GatewayRouter.sol";
import { L2StandardERC20Gateway } from "../L2/gateways/L2StandardERC20Gateway.sol";
import { ScrollStandardERC20 } from "../libraries/token/ScrollStandardERC20.sol";
import { ScrollStandardERC20Factory } from "../libraries/token/ScrollStandardERC20Factory.sol";
import { MockScrollMessenger } from "./mocks/MockScrollMessenger.sol";

contract L2StandardERC20GatewayTest is DSTestPlus {
  ScrollStandardERC20 private template;
  ScrollStandardERC20Factory private factory;

  MockScrollMessenger private messenger;
  L1StandardERC20Gateway private counterpart;
  L2StandardERC20Gateway private gateway;
  L2GatewayRouter private router;

  MockERC20 private badToken;
  ScrollStandardERC20 private token;

  function setUp() public {
    template = new ScrollStandardERC20();
    factory = new ScrollStandardERC20Factory(address(template));

    messenger = new MockScrollMessenger();
    router = new L2GatewayRouter();
    router.initialize(address(0), address(1), address(messenger));

    counterpart = new L1StandardERC20Gateway();
    gateway = new L2StandardERC20Gateway();
    gateway.initialize(address(counterpart), address(router), address(messenger), address(factory));

    router.setDefaultERC20Gateway(address(gateway));
    factory.transferOwnership(address(gateway));

    badToken = new MockERC20("Mock Bad", "M", 18);

    // deploy l2 token
    MockERC20 l1Token = new MockERC20("L1", "L1", 18);
    token = ScrollStandardERC20(gateway.getL2ERC20Address(address(l1Token)));
    assertEq(gateway.getL1ERC20Address(address(token)), address(0));
    messenger.setXDomainMessageSender(address(counterpart));
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

  function testReinitilize() public {
    hevm.expectRevert("Initializable: contract is already initialized");
    gateway.initialize(address(1), address(router), address(messenger), address(factory));
  }

  function testGetL2ERC20Address(address l1Address) public {
    assertEq(gateway.getL2ERC20Address(l1Address), factory.computeL2TokenAddress(address(gateway), l1Address));
  }

  function testWithdrawERC20WithFailed() public {
    hevm.expectRevert("no corresponding l1 token");
    gateway.withdrawERC20(address(badToken), 1, 0);
  }

  function testWithdrawERC20WithRouter(uint256 amount) public {
    amount = bound(amount, 0, token.balanceOf(address(this)));

    if (amount == 0) {
      hevm.expectRevert("withdraw zero amount");
      router.withdrawERC20(address(token), amount, 0);
    } else {
      uint256 myBalance = token.balanceOf(address(this));
      assertEq(token.balanceOf(address(gateway)), 0);
      router.withdrawERC20(address(token), amount, 0);
      assertEq(myBalance - amount, token.balanceOf(address(this)));
      assertEq(token.balanceOf(address(gateway)), 0);

      // @todo check event
    }
  }

  function testWithdrawERC20WithRouter(uint256 amount, address to) public {
    amount = bound(amount, 0, token.balanceOf(address(this)));
    if (to == address(0)) to = address(1);

    if (amount == 0) {
      hevm.expectRevert("withdraw zero amount");
      router.withdrawERC20(address(token), to, amount, 0);
    } else {
      uint256 myBalance = token.balanceOf(address(this));
      assertEq(token.balanceOf(address(gateway)), 0);
      router.withdrawERC20(address(token), to, amount, 0);
      assertEq(myBalance - amount, token.balanceOf(address(this)));
      assertEq(token.balanceOf(address(gateway)), 0);

      // @todo check event
    }
  }

  function testWithdrawERC20AndCallWithRouter(uint256 amount, address to) public {
    amount = bound(amount, 0, token.balanceOf(address(this)));
    if (to == address(0)) to = address(1);

    if (amount == 0) {
      hevm.expectRevert("withdraw zero amount");
      router.withdrawERC20AndCall(address(token), to, amount, "", 0);
    } else {
      uint256 myBalance = token.balanceOf(address(this));
      assertEq(token.balanceOf(address(gateway)), 0);
      router.withdrawERC20AndCall(address(token), to, amount, "", 0);
      assertEq(myBalance - amount, token.balanceOf(address(this)));
      assertEq(token.balanceOf(address(gateway)), 0);

      // @todo check event
    }
  }

  function testWithdrawERC20WithGateway(uint256 amount) public {
    amount = bound(amount, 0, token.balanceOf(address(this)));

    if (amount == 0) {
      hevm.expectRevert("withdraw zero amount");
      gateway.withdrawERC20(address(token), amount, 0);
    } else {
      uint256 myBalance = token.balanceOf(address(this));
      assertEq(token.balanceOf(address(gateway)), 0);
      gateway.withdrawERC20(address(token), amount, 0);
      assertEq(myBalance - amount, token.balanceOf(address(this)));
      assertEq(token.balanceOf(address(gateway)), 0);

      // @todo check event
    }
  }

  function testWithdrawERC20WithGateway(uint256 amount, address to) public {
    amount = bound(amount, 0, token.balanceOf(address(this)));
    if (to == address(0)) to = address(1);

    if (amount == 0) {
      hevm.expectRevert("withdraw zero amount");
      gateway.withdrawERC20(address(token), to, amount, 0);
    } else {
      uint256 myBalance = token.balanceOf(address(this));
      assertEq(token.balanceOf(address(gateway)), 0);
      gateway.withdrawERC20(address(token), to, amount, 0);
      assertEq(myBalance - amount, token.balanceOf(address(this)));
      assertEq(token.balanceOf(address(gateway)), 0);

      // @todo check event
    }
  }

  function testWithdrawERC20AndCallWithGateway(uint256 amount, address to) public {
    amount = bound(amount, 0, token.balanceOf(address(this)));
    if (to == address(0)) to = address(1);

    if (amount == 0) {
      // should revert, when amount is zero
      hevm.expectRevert("withdraw zero amount");
      gateway.withdrawERC20AndCall(address(token), to, amount, "", 0);
    } else {
      // should succeed, for valid amount
      uint256 myBalance = token.balanceOf(address(this));
      assertEq(token.balanceOf(address(gateway)), 0);
      gateway.withdrawERC20AndCall(address(token), to, amount, "", 0);
      assertEq(myBalance - amount, token.balanceOf(address(this)));
      assertEq(token.balanceOf(address(gateway)), 0);

      // @todo check event
    }
  }

  function testFinalizeDepositERC20Failed() public {
    // should revert, called by non-messenger
    hevm.expectRevert("only messenger can call");
    gateway.finalizeDepositERC20(address(0), address(0), address(0), address(0), 0, "");

    // should revert, called by messenger, xDomainMessageSender not set
    hevm.expectRevert("only call by conterpart");
    messenger.callTarget(
      address(gateway),
      abi.encodeWithSelector(
        L2StandardERC20Gateway.finalizeDepositERC20.selector,
        address(0),
        address(0),
        address(0),
        address(0),
        0,
        ""
      )
    );

    // should revert, called by messenger, xDomainMessageSender set wrong
    messenger.setXDomainMessageSender(address(2));
    hevm.expectRevert("only call by conterpart");
    messenger.callTarget(
      address(gateway),
      abi.encodeWithSelector(
        L2StandardERC20Gateway.finalizeDepositERC20.selector,
        address(0),
        address(0),
        address(0),
        address(0),
        0,
        ""
      )
    );

    // should revert, called by messenger, xDomainMessageSender set,nonzero msg.value
    messenger.setXDomainMessageSender(address(counterpart));
    hevm.expectRevert("nonzero msg.value");
    messenger.callTarget{ value: 1 }(
      address(gateway),
      abi.encodeWithSelector(
        L2StandardERC20Gateway.finalizeDepositERC20.selector,
        address(0),
        address(0),
        address(0),
        address(0),
        1,
        ""
      )
    );

    // should revert, called by messenger, xDomainMessageSender set, l2 token mismatch
    messenger.setXDomainMessageSender(address(counterpart));
    hevm.expectRevert("l2 token mismatch");
    messenger.callTarget(
      address(gateway),
      abi.encodeWithSelector(
        L2StandardERC20Gateway.finalizeDepositERC20.selector,
        address(0),
        address(0),
        address(0),
        address(0),
        1,
        ""
      )
    );
  }

  function testFinalizeDepositERC20FirstCall(
    address from,
    address to,
    uint256 amount
  ) public {
    if (to == address(0)) to = address(1);

    // will deploy token and mint
    address l2Address = gateway.getL2ERC20Address(address(badToken));
    assertEq(gateway.getL1ERC20Address(l2Address), address(0));
    messenger.setXDomainMessageSender(address(counterpart));
    messenger.callTarget(
      address(gateway),
      abi.encodeWithSelector(
        L2StandardERC20Gateway.finalizeDepositERC20.selector,
        address(badToken),
        l2Address,
        from,
        to,
        amount,
        abi.encode("", abi.encode("symbol", "name", 18))
      )
    );
    assertEq(ScrollStandardERC20(l2Address).balanceOf(to), amount);
    assertEq(gateway.getL1ERC20Address(l2Address), address(badToken));
    assertEq(ScrollStandardERC20(l2Address).symbol(), "symbol");
    assertEq(ScrollStandardERC20(l2Address).name(), "name");
    assertEq(ScrollStandardERC20(l2Address).decimals(), 18);
  }
}
