// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { console2 } from "forge-std/console2.sol";
import { DSTestPlus } from "solmate/test/utils/DSTestPlus.sol";
import { MockERC20 } from "solmate/test/utils/mocks/MockERC20.sol";
import { WETH } from "solmate/tokens/WETH.sol";

import { L1GatewayRouter } from "../L1/gateways/L1GatewayRouter.sol";
import { L1StandardERC20Gateway } from "../L1/gateways/L1StandardERC20Gateway.sol";
import { L2StandardERC20Gateway } from "../L2/gateways/L2StandardERC20Gateway.sol";
import { ScrollStandardERC20 } from "../libraries/token/ScrollStandardERC20.sol";
import { ScrollStandardERC20Factory } from "../libraries/token/ScrollStandardERC20Factory.sol";
import { MockScrollMessenger } from "./mocks/MockScrollMessenger.sol";
import { TransferReentrantToken } from "./mocks/tokens/TransferReentrantToken.sol";
import { FeeOnTransferToken } from "./mocks/tokens/FeeOnTransferToken.sol";

contract L1StandardERC20GatewayTest is DSTestPlus {
  ScrollStandardERC20 private template;
  ScrollStandardERC20Factory private factory;

  MockScrollMessenger private messenger;
  L2StandardERC20Gateway private counterpart;
  L1StandardERC20Gateway private gateway;
  L1GatewayRouter private router;

  MockERC20 private token;
  TransferReentrantToken private reentrantToken;
  FeeOnTransferToken private feeToken;

  function setUp() public {
    template = new ScrollStandardERC20();
    factory = new ScrollStandardERC20Factory(address(template));

    messenger = new MockScrollMessenger();
    router = new L1GatewayRouter();
    router.initialize(address(0), address(1), address(messenger));

    counterpart = new L2StandardERC20Gateway();
    gateway = new L1StandardERC20Gateway();
    gateway.initialize(address(counterpart), address(router), address(messenger), address(template), address(factory));

    router.setDefaultERC20Gateway(address(gateway));

    token = new MockERC20("Mock", "M", 18);
    token.mint(address(this), type(uint256).max);
    token.approve(address(gateway), type(uint256).max);

    reentrantToken = new TransferReentrantToken("Reentrant", "R", 18);
    reentrantToken.mint(address(this), type(uint256).max);
    reentrantToken.approve(address(gateway), type(uint256).max);

    feeToken = new FeeOnTransferToken("Fee", "F", 18);
    // use uint128.max to avoid multiplication overflow
    feeToken.mint(address(this), type(uint128).max);
    feeToken.approve(address(gateway), type(uint256).max);
  }

  function testReinitilize() public {
    hevm.expectRevert("Initializable: contract is already initialized");
    gateway.initialize(address(1), address(router), address(messenger), address(template), address(factory));
  }

  function testGetL2ERC20Address(address l1Address) public {
    assertEq(gateway.getL2ERC20Address(l1Address), factory.computeL2TokenAddress(address(counterpart), l1Address));
  }

  function testDepositERC20WithRouter(uint256 amount) public {
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

  function testDepositERC20WithGateway(uint256 amount) public {
    amount = bound(amount, 0, token.balanceOf(address(this)));

    if (amount == 0) {
      hevm.expectRevert("deposit zero amount");
      gateway.depositERC20(address(token), amount, 0);
    } else {
      uint256 gatewayBalance = token.balanceOf(address(gateway));
      gateway.depositERC20(address(token), amount, 0);
      assertEq(amount + gatewayBalance, token.balanceOf(address(gateway)));

      // @todo check event
    }
  }

  function testDepositERC20WithGateway(uint256 amount, address to) public {
    amount = bound(amount, 0, token.balanceOf(address(this)));
    if (to == address(0)) to = address(1);

    if (amount == 0) {
      hevm.expectRevert("deposit zero amount");
      gateway.depositERC20(address(token), to, amount, 0);
    } else {
      uint256 gatewayBalance = token.balanceOf(address(gateway));
      gateway.depositERC20(address(token), to, amount, 0);
      assertEq(amount + gatewayBalance, token.balanceOf(address(gateway)));

      // @todo check event
    }
  }

  function testDepositERC20AndCallWithGateway(uint256 amount, address to) public {
    amount = bound(amount, 0, token.balanceOf(address(this)));
    if (to == address(0)) to = address(1);

    if (amount == 0) {
      // should revert, when amount is zero
      hevm.expectRevert("deposit zero amount");
      gateway.depositERC20AndCall(address(token), to, amount, "", 0);
    } else {
      // should succeed, for valid amount
      uint256 gatewayBalance = token.balanceOf(address(gateway));
      gateway.depositERC20AndCall(address(token), to, amount, "", 0);
      assertEq(amount + gatewayBalance, token.balanceOf(address(gateway)));

      // @todo check event
    }
  }

  function testDepositReentrantToken(uint256 amount) public {
    // should revert, reentrant before transfer
    reentrantToken.setReentrantCall(
      address(gateway),
      0,
      abi.encodeWithSignature("depositERC20(address,uint256,uint256)", address(0), 1, 0),
      true
    );
    amount = bound(amount, 1, reentrantToken.balanceOf(address(this)));
    hevm.expectRevert("ReentrancyGuard: reentrant call");
    gateway.depositERC20(address(reentrantToken), amount, 0);

    // should revert, reentrant after transfer
    reentrantToken.setReentrantCall(
      address(gateway),
      0,
      abi.encodeWithSignature("depositERC20(address,uint256,uint256)", address(0), 1, 0),
      false
    );
    amount = bound(amount, 1, reentrantToken.balanceOf(address(this)));
    hevm.expectRevert("ReentrancyGuard: reentrant call");
    gateway.depositERC20(address(reentrantToken), amount, 0);
  }

  function testFeeOnTransferTokenFailed(uint256 amount) public {
    feeToken.setFeeRate(1e9);
    amount = bound(amount, 1, feeToken.balanceOf(address(this)));
    hevm.expectRevert("deposit zero amount");
    gateway.depositERC20(address(feeToken), amount, 0);
  }

  function testFeeOnTransferTokenSucceed(uint256 amount, uint256 feeRate) public {
    feeRate = bound(feeRate, 0, 1e9 - 1);
    amount = bound(amount, 1e9, feeToken.balanceOf(address(this)));
    feeToken.setFeeRate(feeRate);

    // should succeed, for valid amount
    uint256 balanceBefore = feeToken.balanceOf(address(gateway));
    uint256 fee = (amount * feeRate) / 1e9;
    gateway.depositERC20(address(feeToken), amount, 0);
    uint256 balanceAfter = feeToken.balanceOf(address(gateway));
    assertEq(balanceBefore + amount - fee, balanceAfter);

    // @todo check event
  }

  function testFinalizeWithdrawERC20Failed() public {
    // should revert, called by non-messenger
    hevm.expectRevert("only messenger can call");
    gateway.finalizeWithdrawERC20(address(0), address(0), address(0), address(0), 0, "");

    // should revert, called by messenger, xDomainMessageSender not set
    hevm.expectRevert("only call by conterpart");
    messenger.callTarget(
      address(gateway),
      abi.encodeWithSelector(
        L1StandardERC20Gateway.finalizeWithdrawERC20.selector,
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
        L1StandardERC20Gateway.finalizeWithdrawERC20.selector,
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
        L1StandardERC20Gateway.finalizeWithdrawERC20.selector,
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
        L1StandardERC20Gateway.finalizeWithdrawERC20.selector,
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
