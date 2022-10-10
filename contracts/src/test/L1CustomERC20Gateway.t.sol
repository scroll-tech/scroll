// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { DSTestPlus } from "solmate/test/utils/DSTestPlus.sol";
import { MockERC20 } from "solmate/test/utils/mocks/MockERC20.sol";

import { L1GatewayRouter } from "../L1/gateways/L1GatewayRouter.sol";
import { L1CustomERC20Gateway } from "../L1/gateways/L1CustomERC20Gateway.sol";
import { L2CustomERC20Gateway } from "../L2/gateways/L2CustomERC20Gateway.sol";
import { MockScrollMessenger } from "./mocks/MockScrollMessenger.sol";

contract L1CustomERC20GatewayTest is DSTestPlus {
  MockScrollMessenger private messenger;
  L1CustomERC20Gateway private gateway;
  L2CustomERC20Gateway private counterpart;
  L1GatewayRouter private router;

  MockERC20 private token;

  function setUp() public {
    messenger = new MockScrollMessenger();
    router = new L1GatewayRouter();

    counterpart = new L2CustomERC20Gateway();
    gateway = new L1CustomERC20Gateway();

    gateway.initialize(address(counterpart), address(router), address(messenger));
    router.initialize(address(gateway), address(1), address(messenger));

    token = new MockERC20("Mock", "M", 18);
    token.mint(address(this), type(uint256).max);
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

    assertEq(gateway.getL2ERC20Address(token1), address(0));
    gateway.updateTokenMapping(token1, token2);
    assertEq(gateway.getL2ERC20Address(token1), token2);
  }

  function testDepositERC20WithRouter(uint256 amount) public {
    gateway.updateTokenMapping(address(token), address(token));
    amount = bound(amount, 0, token.balanceOf(address(this)));

    if (amount == 0) {
      hevm.expectRevert("deposit zero amount");
      router.depositERC20(address(token), amount, 0);
    } else {
      uint256 gatewayBalance = token.balanceOf(address(gateway));
      router.depositERC20(address(token), amount, 0);
      assertEq(amount + gatewayBalance, token.balanceOf(address(gateway)));

      // @todo check event
    }
  }

  function testDepositERC20WithRouter(uint256 amount, address to) public {
    gateway.updateTokenMapping(address(token), address(token));
    amount = bound(amount, 0, token.balanceOf(address(this)));
    if (to == address(0)) to = address(1);

    if (amount == 0) {
      hevm.expectRevert("deposit zero amount");
      router.depositERC20(address(token), to, amount, 0);
    } else {
      uint256 gatewayBalance = token.balanceOf(address(gateway));
      router.depositERC20(address(token), to, amount, 0);
      assertEq(amount + gatewayBalance, token.balanceOf(address(gateway)));

      // @todo check event
    }
  }

  function testDepositERC20AndCallWithRouter(uint256 amount, address to) public {
    gateway.updateTokenMapping(address(token), address(token));
    amount = bound(amount, 0, token.balanceOf(address(this)));
    if (to == address(0)) to = address(1);

    if (amount == 0) {
      hevm.expectRevert("deposit zero amount");
      router.depositERC20AndCall(address(token), to, amount, "", 0);
    } else {
      uint256 gatewayBalance = token.balanceOf(address(gateway));
      router.depositERC20AndCall(address(token), to, amount, "", 0);
      assertEq(amount + gatewayBalance, token.balanceOf(address(gateway)));

      // @todo check event
    }
  }

  /// @dev failed to deposit erc20
  function testDepositERC20WithGatewayFailed(address to) public {
    // token not support
    hevm.expectRevert("no corresponding l2 token");
    if (to == address(0)) {
      gateway.depositERC20(address(token), 0, 0);
    } else {
      gateway.depositERC20(address(token), to, 0, 0);
    }
  }

  function testDepositERC20ZeroAmount() public {
    gateway.updateTokenMapping(address(token), address(token));

    hevm.expectRevert("deposit zero amount");
    gateway.depositERC20(address(token), 0, 0);

    hevm.expectRevert("deposit zero amount");
    gateway.depositERC20(address(token), address(this), 0, 0);

    hevm.expectRevert("deposit zero amount");
    gateway.depositERC20AndCall(address(token), address(this), 0, "", 0);
  }

  function testDepositERC20(uint256 amount) public {
    gateway.updateTokenMapping(address(token), address(token));
    if (amount == 0) amount = 1;

    uint256 gatewayBalance = token.balanceOf(address(gateway));
    gateway.depositERC20(address(token), amount, 0);
    assertEq(amount + gatewayBalance, token.balanceOf(address(gateway)));
  }

  function testDepositERC20(uint256 amount, address to) public {
    gateway.updateTokenMapping(address(token), address(token));
    if (amount == 0) amount = 1;

    uint256 gatewayBalance = token.balanceOf(address(gateway));
    gateway.depositERC20(address(token), to, amount, 0);
    assertEq(amount + gatewayBalance, token.balanceOf(address(gateway)));
  }

  function testDepositERC20AndCall(
    uint256 amount,
    address to,
    bytes calldata data
  ) public {
    gateway.updateTokenMapping(address(token), address(token));
    if (amount == 0) amount = 1;

    uint256 gatewayBalance = token.balanceOf(address(gateway));
    gateway.depositERC20AndCall(address(token), to, amount, data, 0);
    assertEq(amount + gatewayBalance, token.balanceOf(address(gateway)));
  }

  function testFinalizeWithdrawERC20Failed() public {
    gateway.updateTokenMapping(address(token), address(token));
    // should revert, called by non-messenger
    hevm.expectRevert("only messenger can call");
    gateway.finalizeWithdrawERC20(address(0), address(0), address(0), address(0), 0, "");

    // should revert, called by messenger, xDomainMessageSender not set
    hevm.expectRevert("only call by conterpart");
    messenger.callTarget(
      address(gateway),
      abi.encodeWithSelector(
        L1CustomERC20Gateway.finalizeWithdrawERC20.selector,
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
        L1CustomERC20Gateway.finalizeWithdrawERC20.selector,
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
        L1CustomERC20Gateway.finalizeWithdrawERC20.selector,
        address(0),
        address(0),
        address(0),
        address(0),
        1,
        ""
      )
    );
  }

  function testFinalizeWithdrawERC20WithoutData(
    address from,
    address to,
    uint256 amount
  ) public {
    gateway.updateTokenMapping(address(token), address(token));
    // this should not happen in unit tests
    if (to == address(gateway)) return;

    // deposit first
    amount = bound(amount, 1, token.balanceOf(address(this)));
    gateway.depositERC20(address(token), amount, 0);

    // then withdraw
    messenger.setXDomainMessageSender(address(counterpart));
    uint256 balanceBefore = token.balanceOf(to);
    messenger.callTarget(
      address(gateway),
      abi.encodeWithSelector(
        L1CustomERC20Gateway.finalizeWithdrawERC20.selector,
        address(token),
        address(token),
        from,
        to,
        amount,
        ""
      )
    );
    assertEq(token.balanceOf(to), balanceBefore + amount);
  }
}
