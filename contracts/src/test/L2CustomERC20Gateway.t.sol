// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { DSTestPlus } from "solmate/test/utils/DSTestPlus.sol";
import { MockERC20 } from "solmate/test/utils/mocks/MockERC20.sol";

import { L2GatewayRouter } from "../L2/gateways/L2GatewayRouter.sol";
import { L1CustomERC20Gateway } from "../L1/gateways/L1CustomERC20Gateway.sol";
import { L2CustomERC20Gateway } from "../L2/gateways/L2CustomERC20Gateway.sol";
import { MockScrollMessenger } from "./mocks/MockScrollMessenger.sol";

contract L2CustomERC20GatewayTest is DSTestPlus {
  MockScrollMessenger private messenger;
  L2CustomERC20Gateway private gateway;
  L1CustomERC20Gateway private counterpart;
  L2GatewayRouter private router;
  MockERC20 private token;

  function setUp() public {
    messenger = new MockScrollMessenger();
    router = new L2GatewayRouter();

    counterpart = new L1CustomERC20Gateway();
    gateway = new L2CustomERC20Gateway();

    gateway.initialize(address(counterpart), address(router), address(messenger));
    router.initialize(address(gateway), address(1), address(messenger));

    // deploy l2 token
    token = new MockERC20("L2", "L2", 18);
    token.mint(address(this), type(uint256).max / 2);
    token.approve(address(gateway), type(uint256).max);
  }

  function testInitialized() public {
    assertEq(address(this), gateway.owner());
    assertEq(address(counterpart), gateway.counterpart());
    assertEq(address(router), gateway.router());
    assertEq(address(messenger), gateway.messenger());

    hevm.expectRevert("Initializable: contract is already initialized");
    gateway.initialize(address(1), address(1), address(messenger));
  }

  function testUpdateTokenMappingFailed(address token1) public {
    // call by non-owner, should revert
    hevm.startPrank(address(1));
    hevm.expectRevert("Ownable: caller is not the owner");
    gateway.updateTokenMapping(token1, token1);
    hevm.stopPrank();

    // l2 token is zero, should revert
    hevm.expectRevert("map to zero address");
    gateway.updateTokenMapping(token1, address(0));
  }

  function testUpdateTokenMappingSuccess(address token1, address token2) public {
    if (token2 == address(0)) token2 = address(1);

    assertEq(gateway.getL1ERC20Address(token1), address(0));
    gateway.updateTokenMapping(token1, token2);
    assertEq(gateway.getL1ERC20Address(token1), token2);
  }

  function testWithdrawERC20WithFailed() public {
    hevm.expectRevert("no corresponding l1 token");
    gateway.withdrawERC20(address(token), 1, 0);
  }

  function testWithdrawERC20WithRouter(uint256 amount) public {
    gateway.updateTokenMapping(address(token), address(token));
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
    gateway.updateTokenMapping(address(token), address(token));
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
    gateway.updateTokenMapping(address(token), address(token));
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
    gateway.updateTokenMapping(address(token), address(token));
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
    gateway.updateTokenMapping(address(token), address(token));
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
    gateway.updateTokenMapping(address(token), address(token));
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
    gateway.updateTokenMapping(address(token), address(token));
    // should revert, called by non-messenger
    hevm.expectRevert("only messenger can call");
    gateway.finalizeDepositERC20(address(0), address(0), address(0), address(0), 0, "");

    // should revert, called by messenger, xDomainMessageSender not set
    hevm.expectRevert("only call by conterpart");
    messenger.callTarget(
      address(gateway),
      abi.encodeWithSelector(
        L2CustomERC20Gateway.finalizeDepositERC20.selector,
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
        L2CustomERC20Gateway.finalizeDepositERC20.selector,
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
        L2CustomERC20Gateway.finalizeDepositERC20.selector,
        address(0),
        address(0),
        address(0),
        address(0),
        1,
        ""
      )
    );
  }

  function testFinalizeDepositERC20Success(
    address from,
    address to,
    uint256 amount
  ) public {
    gateway.updateTokenMapping(address(token), address(token));
    if (to == address(0)) to = address(1);
    if (to == address(this)) to = address(1);
    amount = bound(amount, 0, type(uint256).max - token.totalSupply());

    // will deploy token and mint
    messenger.setXDomainMessageSender(address(counterpart));
    messenger.callTarget(
      address(gateway),
      abi.encodeWithSelector(
        L2CustomERC20Gateway.finalizeDepositERC20.selector,
        address(token),
        address(token),
        from,
        to,
        amount,
        ""
      )
    );
    assertEq(token.balanceOf(to), amount);
  }
}
