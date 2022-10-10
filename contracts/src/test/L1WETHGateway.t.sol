// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { DSTestPlus } from "solmate/test/utils/DSTestPlus.sol";
import { WETH } from "solmate/tokens/WETH.sol";

import { L1GatewayRouter } from "../L1/gateways/L1GatewayRouter.sol";
import { L1WETHGateway } from "../L1/gateways/L1WETHGateway.sol";
import { L2WETHGateway } from "../L2/gateways/L2WETHGateway.sol";
import { MockScrollMessenger } from "./mocks/MockScrollMessenger.sol";

contract L1WETHGatewayTest is DSTestPlus {
  WETH private l1weth;
  WETH private l2weth;

  MockScrollMessenger private messenger;
  L1WETHGateway private gateway;
  L2WETHGateway private counterpart;
  L1GatewayRouter private router;

  function setUp() public {
    l1weth = new WETH();
    l2weth = new WETH();

    messenger = new MockScrollMessenger();
    router = new L1GatewayRouter();
    router.initialize(address(0), address(1), address(messenger));

    counterpart = new L2WETHGateway();
    gateway = new L1WETHGateway();
    gateway.initialize(address(counterpart), address(router), address(messenger), address(l1weth), address(l2weth));

    {
      address[] memory _tokens = new address[](1);
      address[] memory _gateways = new address[](1);
      _tokens[0] = address(l1weth);
      _gateways[0] = address(gateway);
      router.setERC20Gateway(_tokens, _gateways);
    }

    l1weth.deposit{ value: address(this).balance / 2 }();
    l1weth.approve(address(gateway), type(uint256).max);
  }

  function testReinitilize() public {
    hevm.expectRevert("Initializable: contract is already initialized");
    gateway.initialize(address(1), address(router), address(messenger), address(l1weth), address(l2weth));
  }

  function testDirectTransferETH(uint256 amount) public {
    amount = bound(amount, 0, address(this).balance);
    // solhint-disable-next-line avoid-low-level-calls
    (bool success, bytes memory result) = address(gateway).call{ value: amount }("");
    assertBoolEq(success, false);
    assertEq(string(result), string(abi.encodeWithSignature("Error(string)", "only WETH")));
  }

  function testDepositERC20WithRouter(uint256 amount) public {
    amount = bound(amount, 0, l1weth.balanceOf(address(this)));

    if (amount == 0) {
      hevm.expectRevert("deposit zero amount");
      router.depositERC20(address(l1weth), amount, 0);
    } else {
      uint256 messengerBalance = address(messenger).balance;
      router.depositERC20(address(l1weth), amount, 0);
      assertEq(amount + messengerBalance, address(messenger).balance);
    }
  }

  function testDepositERC20WithRouter(uint256 amount, address to) public {
    amount = bound(amount, 0, l1weth.balanceOf(address(this)));
    if (to == address(0)) to = address(1);

    if (amount == 0) {
      hevm.expectRevert("deposit zero amount");
      router.depositERC20(address(l1weth), to, amount, 0);
    } else {
      uint256 messengerBalance = address(messenger).balance;
      router.depositERC20(address(l1weth), to, amount, 0);
      assertEq(amount + messengerBalance, address(messenger).balance);
    }
  }

  function testDepositERC20AndCallWithRouter(uint256 amount, address to) public {
    amount = bound(amount, 0, l1weth.balanceOf(address(this)));
    if (to == address(0)) to = address(1);

    if (amount == 0) {
      hevm.expectRevert("deposit zero amount");
      router.depositERC20AndCall(address(l1weth), to, amount, "", 0);
    } else {
      uint256 messengerBalance = address(messenger).balance;
      router.depositERC20AndCall(address(l1weth), to, amount, "", 0);
      assertEq(amount + messengerBalance, address(messenger).balance);
    }
  }

  function testDepositERC20WithGateway(uint256 amount) public {
    amount = bound(amount, 0, l1weth.balanceOf(address(this)));

    if (amount == 0) {
      hevm.expectRevert("deposit zero amount");
      gateway.depositERC20(address(l1weth), amount, 0);
    } else {
      uint256 messengerBalance = address(messenger).balance;
      gateway.depositERC20(address(l1weth), amount, 0);
      assertEq(amount + messengerBalance, address(messenger).balance);
    }
  }

  function testDepositERC20WithGateway(uint256 amount, address to) public {
    amount = bound(amount, 0, l1weth.balanceOf(address(this)));
    if (to == address(0)) to = address(1);

    if (amount == 0) {
      hevm.expectRevert("deposit zero amount");
      gateway.depositERC20(address(l1weth), to, amount, 0);
    } else {
      uint256 messengerBalance = address(messenger).balance;
      gateway.depositERC20(address(l1weth), to, amount, 0);
      assertEq(amount + messengerBalance, address(messenger).balance);
    }
  }

  function testDepositERC20AndCallWithGateway(uint256 amount, address to) public {
    amount = bound(amount, 0, l1weth.balanceOf(address(this)));
    if (to == address(0)) to = address(1);

    if (amount == 0) {
      hevm.expectRevert("deposit zero amount");
      gateway.depositERC20AndCall(address(l1weth), to, amount, "", 0);
    } else {
      uint256 messengerBalance = address(messenger).balance;
      gateway.depositERC20AndCall(address(l1weth), to, amount, "", 0);
      assertEq(amount + messengerBalance, address(messenger).balance);
    }
  }

  function testDepositERC20WithGatewayFailed(address token) public {
    if (token == address(l1weth)) return;
    // token is not l1WETH
    hevm.expectRevert("only WETH is allowed");
    gateway.depositERC20(token, 1, 0);
  }

  function testFinalizeWithdrawERC20Failed() public {
    // called by non-messenger
    hevm.expectRevert("only messenger can call");
    gateway.finalizeWithdrawERC20(address(0), address(0), address(0), address(0), 0, "");

    // called by messenger, xDomainMessageSender not set
    hevm.expectRevert("only call by conterpart");
    messenger.callTarget(
      address(gateway),
      abi.encodeWithSelector(
        L1WETHGateway.finalizeWithdrawERC20.selector,
        address(0),
        address(0),
        address(0),
        address(0),
        0,
        ""
      )
    );

    // called by messenger, xDomainMessageSender set wrong
    messenger.setXDomainMessageSender(address(2));
    hevm.expectRevert("only call by conterpart");
    messenger.callTarget(
      address(gateway),
      abi.encodeWithSelector(
        L1WETHGateway.finalizeWithdrawERC20.selector,
        address(0),
        address(0),
        address(0),
        address(0),
        0,
        ""
      )
    );

    // called by messenger, xDomainMessageSender set, wrong l1 token
    messenger.setXDomainMessageSender(address(counterpart));
    hevm.expectRevert("l1 token not WETH");
    messenger.callTarget(
      address(gateway),
      abi.encodeWithSelector(
        L1WETHGateway.finalizeWithdrawERC20.selector,
        address(0),
        address(0),
        address(0),
        address(0),
        0,
        ""
      )
    );

    // called by messenger, xDomainMessageSender set, wrong l2 token
    messenger.setXDomainMessageSender(address(counterpart));
    hevm.expectRevert("l2 token not WETH");
    messenger.callTarget(
      address(gateway),
      abi.encodeWithSelector(
        L1WETHGateway.finalizeWithdrawERC20.selector,
        address(l1weth),
        address(0),
        address(0),
        address(0),
        0,
        ""
      )
    );

    // called by messenger, xDomainMessageSender set, mismatch amount
    messenger.setXDomainMessageSender(address(counterpart));
    hevm.expectRevert("msg.value mismatch");
    messenger.callTarget(
      address(gateway),
      abi.encodeWithSelector(
        L1WETHGateway.finalizeWithdrawERC20.selector,
        address(l1weth),
        address(l2weth),
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
    amount = bound(amount, 0, address(this).balance);

    messenger.setXDomainMessageSender(address(counterpart));
    uint256 balanceBefore = l1weth.balanceOf(to);
    messenger.callTarget{ value: amount }(
      address(gateway),
      abi.encodeWithSelector(
        L1WETHGateway.finalizeWithdrawERC20.selector,
        address(l1weth),
        address(l2weth),
        from,
        to,
        amount,
        ""
      )
    );
    assertEq(l1weth.balanceOf(to), balanceBefore + amount);
  }
}
