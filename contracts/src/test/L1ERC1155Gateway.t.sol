// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { console2 } from "forge-std/console2.sol";
import { DSTestPlus } from "solmate/test/utils/DSTestPlus.sol";
import { MockERC1155 } from "solmate/test/utils/mocks/MockERC1155.sol";
import { ERC1155TokenReceiver } from "solmate/tokens/ERC1155.sol";

import { L1ERC1155Gateway } from "../L1/gateways/L1ERC1155Gateway.sol";
import { L2ERC1155Gateway } from "../L2/gateways/L2ERC1155Gateway.sol";
import { MockScrollMessenger } from "./mocks/MockScrollMessenger.sol";
import { MockERC1155Recipient } from "./mocks/MockERC1155Recipient.sol";

contract L1ERC1155GatewayTest is DSTestPlus, ERC1155TokenReceiver {
  uint256 private constant TOKEN_COUNT = 100;

  MockScrollMessenger private messenger;
  L2ERC1155Gateway private counterpart;
  L1ERC1155Gateway private gateway;

  MockERC1155 private token;
  MockERC1155Recipient private mockRecipient;

  function setUp() public {
    messenger = new MockScrollMessenger();

    counterpart = new L2ERC1155Gateway();
    gateway = new L1ERC1155Gateway();
    gateway.initialize(address(counterpart), address(messenger));

    token = new MockERC1155();
    for (uint256 i = 0; i < TOKEN_COUNT; i++) {
      token.mint(address(this), i, type(uint256).max, "");
    }
    token.setApprovalForAll(address(gateway), true);

    mockRecipient = new MockERC1155Recipient();
  }

  function testReinitilize() public {
    hevm.expectRevert("Initializable: contract is already initialized");
    gateway.initialize(address(1), address(messenger));
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

    assertEq(gateway.tokenMapping(token1), address(0));
    gateway.updateTokenMapping(token1, token2);
    assertEq(gateway.tokenMapping(token1), token2);
  }

  /// @dev failed to deposit erc1155
  function testDepositERC1155WithGatewayFailed(address to) public {
    // token not support
    hevm.expectRevert("token not supported");
    if (to == address(0)) {
      gateway.depositERC1155(address(token), 0, 1, 0);
    } else {
      gateway.depositERC1155(address(token), to, 0, 1, 0);
    }

    // deposit zero amount
    hevm.expectRevert("deposit zero amount");
    if (to == address(0)) {
      gateway.depositERC1155(address(token), 0, 0, 0);
    } else {
      gateway.depositERC1155(address(token), to, 0, 0, 0);
    }
  }

  /// @dev deposit erc1155 without recipient
  function testDepositERC1155WithGatewaySuccess(uint256 tokenId, uint256 amount) public {
    tokenId = bound(tokenId, 0, TOKEN_COUNT - 1);
    amount = bound(amount, 1, type(uint256).max);
    gateway.updateTokenMapping(address(token), address(token));

    gateway.depositERC1155(address(token), tokenId, amount, 0);
    assertEq(token.balanceOf(address(gateway), tokenId), amount);

    // @todo check event
  }

  /// @dev deposit erc1155 with recipient
  function testDepositERC1155WithGatewaySuccess(
    uint256 tokenId,
    uint256 amount,
    address to
  ) public {
    tokenId = bound(tokenId, 0, TOKEN_COUNT - 1);
    amount = bound(amount, 1, type(uint256).max);
    gateway.updateTokenMapping(address(token), address(token));

    gateway.depositERC1155(address(token), to, tokenId, amount, 0);
    assertEq(token.balanceOf(address(gateway), tokenId), amount);

    // @todo check event
  }

  /// @dev failed to batch deposit erc1155
  function testBatchDepositERC1155WithGatewayFailed(address to) public {
    // no token to deposit
    hevm.expectRevert("no token to deposit");
    if (to == address(0)) {
      gateway.batchDepositERC1155(address(token), new uint256[](0), new uint256[](0), 0);
    } else {
      gateway.batchDepositERC1155(address(token), to, new uint256[](0), new uint256[](0), 0);
    }

    // length mismatch
    hevm.expectRevert("length mismatch");
    if (to == address(0)) {
      gateway.batchDepositERC1155(address(token), new uint256[](1), new uint256[](0), 0);
    } else {
      gateway.batchDepositERC1155(address(token), to, new uint256[](1), new uint256[](0), 0);
    }

    uint256[] memory amounts = new uint256[](1);
    // deposit zero amount
    hevm.expectRevert("deposit zero amount");
    if (to == address(0)) {
      gateway.batchDepositERC1155(address(token), new uint256[](1), amounts, 0);
    } else {
      gateway.batchDepositERC1155(address(token), to, new uint256[](1), amounts, 0);
    }

    // token not support
    amounts[0] = 1;
    hevm.expectRevert("token not supported");
    if (to == address(0)) {
      gateway.batchDepositERC1155(address(token), new uint256[](1), amounts, 0);
    } else {
      gateway.batchDepositERC1155(address(token), to, new uint256[](1), amounts, 0);
    }
  }

  /// @dev batch deposit erc1155 without recipient
  function testBatchDepositERC1155WithGatewaySuccess(uint256 count, uint256 amount) public {
    count = bound(count, 1, TOKEN_COUNT);
    amount = bound(amount, 1, type(uint256).max);
    gateway.updateTokenMapping(address(token), address(token));

    uint256[] memory _tokenIds = new uint256[](count);
    uint256[] memory _amounts = new uint256[](count);
    for (uint256 i = 0; i < count; i++) {
      _tokenIds[i] = i;
      _amounts[i] = amount;
    }

    gateway.batchDepositERC1155(address(token), _tokenIds, _amounts, 0);
    for (uint256 i = 0; i < count; i++) {
      assertEq(token.balanceOf(address(gateway), i), _amounts[i]);
    }

    // @todo check event
  }

  /// @dev batch deposit erc1155 with recipient
  function testBatchDepositERC1155WithGatewaySuccess(
    uint256 count,
    uint256 amount,
    address to
  ) public {
    count = bound(count, 1, TOKEN_COUNT);
    amount = bound(amount, 1, type(uint256).max);
    gateway.updateTokenMapping(address(token), address(token));

    uint256[] memory _tokenIds = new uint256[](count);
    uint256[] memory _amounts = new uint256[](count);
    for (uint256 i = 0; i < count; i++) {
      _tokenIds[i] = i;
      _amounts[i] = amount;
    }

    gateway.batchDepositERC1155(address(token), to, _tokenIds, _amounts, 0);
    for (uint256 i = 0; i < count; i++) {
      assertEq(token.balanceOf(address(gateway), i), _amounts[i]);
    }

    // @todo check event
  }

  /// @dev failed to finalize withdraw erc1155
  function testFinalizeWithdrawERC1155Failed() public {
    // should revert, called by non-messenger
    hevm.expectRevert("only messenger can call");
    gateway.finalizeWithdrawERC1155(address(0), address(0), address(0), address(0), 0, 1);

    // should revert, called by messenger, xDomainMessageSender not set
    hevm.expectRevert("only call by conterpart");
    messenger.callTarget(
      address(gateway),
      abi.encodeWithSelector(
        L1ERC1155Gateway.finalizeWithdrawERC1155.selector,
        address(0),
        address(0),
        address(0),
        address(0),
        0,
        1
      )
    );

    // should revert, called by messenger, xDomainMessageSender set wrong
    messenger.setXDomainMessageSender(address(2));
    hevm.expectRevert("only call by conterpart");
    messenger.callTarget(
      address(gateway),
      abi.encodeWithSelector(
        L1ERC1155Gateway.finalizeWithdrawERC1155.selector,
        address(0),
        address(0),
        address(0),
        address(0),
        0,
        1
      )
    );
  }

  /// @dev finalize withdraw erc1155
  function testFinalizeWithdrawERC1155(
    address from,
    address to,
    uint256 tokenId,
    uint256 amount
  ) public {
    if (to == address(0) || to.code.length > 0) to = address(1);

    // deposit first
    gateway.updateTokenMapping(address(token), address(token));
    tokenId = bound(tokenId, 0, TOKEN_COUNT - 1);
    amount = bound(amount, 1, type(uint256).max);
    gateway.depositERC1155(address(token), tokenId, amount, 0);

    // then withdraw
    messenger.setXDomainMessageSender(address(counterpart));
    messenger.callTarget(
      address(gateway),
      abi.encodeWithSelector(
        L1ERC1155Gateway.finalizeWithdrawERC1155.selector,
        address(token),
        address(token),
        from,
        to,
        tokenId,
        amount
      )
    );
    assertEq(token.balanceOf(to, tokenId), amount);
  }

  /// @dev failed to finalize batch withdraw erc1155
  function testFinalizeBatchWithdrawERC1155Failed() public {
    // should revert, called by non-messenger
    hevm.expectRevert("only messenger can call");
    gateway.finalizeBatchWithdrawERC1155(
      address(0),
      address(0),
      address(0),
      address(0),
      new uint256[](0),
      new uint256[](0)
    );

    // should revert, called by messenger, xDomainMessageSender not set
    hevm.expectRevert("only call by conterpart");
    messenger.callTarget(
      address(gateway),
      abi.encodeWithSelector(
        L1ERC1155Gateway.finalizeBatchWithdrawERC1155.selector,
        address(0),
        address(0),
        address(0),
        address(0),
        new uint256[](0),
        new uint256[](0)
      )
    );

    // should revert, called by messenger, xDomainMessageSender set wrong
    messenger.setXDomainMessageSender(address(2));
    hevm.expectRevert("only call by conterpart");
    messenger.callTarget(
      address(gateway),
      abi.encodeWithSelector(
        L1ERC1155Gateway.finalizeBatchWithdrawERC1155.selector,
        address(0),
        address(0),
        address(0),
        address(0),
        new uint256[](0),
        new uint256[](0)
      )
    );
  }

  /// @dev finalize batch withdraw erc1155
  function testFinalizeBatchWithdrawERC1155(
    address from,
    address to,
    uint256 count,
    uint256 amount
  ) public {
    if (to == address(0) || to.code.length > 0) to = address(1);
    gateway.updateTokenMapping(address(token), address(token));

    // deposit first
    count = bound(count, 1, TOKEN_COUNT);
    amount = bound(amount, 1, type(uint256).max);
    uint256[] memory _tokenIds = new uint256[](count);
    uint256[] memory _amounts = new uint256[](count);
    for (uint256 i = 0; i < count; i++) {
      _tokenIds[i] = i;
      _amounts[i] = amount;
    }
    gateway.batchDepositERC1155(address(token), _tokenIds, _amounts, 0);

    // then withdraw
    messenger.setXDomainMessageSender(address(counterpart));
    messenger.callTarget(
      address(gateway),
      abi.encodeWithSelector(
        L1ERC1155Gateway.finalizeBatchWithdrawERC1155.selector,
        address(token),
        address(token),
        from,
        to,
        _tokenIds,
        _amounts
      )
    );
    for (uint256 i = 0; i < count; i++) {
      assertEq(token.balanceOf(to, i), _amounts[i]);
    }
  }

  /// @dev should detect reentrance
  function testReentranceWhenFinalizeWithdraw(
    address from,
    uint256 tokenId,
    uint256 amount
  ) public {
    // deposit first
    gateway.updateTokenMapping(address(token), address(token));
    tokenId = bound(tokenId, 0, TOKEN_COUNT - 1);
    amount = bound(amount, 1, type(uint256).max);
    gateway.depositERC1155(address(token), tokenId, amount, 0);

    mockRecipient.setCall(
      address(gateway),
      0,
      abi.encodeWithSignature("depositERC1155(address,uint256,uint256,uint256)", address(token), tokenId, amount, 0)
    );

    // finalize withdraw
    messenger.setXDomainMessageSender(address(counterpart));
    hevm.expectRevert("ReentrancyGuard: reentrant call");
    messenger.callTarget(
      address(gateway),
      abi.encodeWithSelector(
        L1ERC1155Gateway.finalizeWithdrawERC1155.selector,
        address(token),
        address(token),
        from,
        address(mockRecipient),
        tokenId,
        amount
      )
    );

    // finalize batch withdraw
    messenger.setXDomainMessageSender(address(counterpart));
    hevm.expectRevert("ReentrancyGuard: reentrant call");
    uint256[] memory tokenIds = new uint256[](1);
    uint256[] memory amounts = new uint256[](1);
    tokenIds[0] = tokenId;
    amounts[0] = amount;
    messenger.callTarget(
      address(gateway),
      abi.encodeWithSelector(
        L1ERC1155Gateway.finalizeBatchWithdrawERC1155.selector,
        address(token),
        address(token),
        from,
        address(mockRecipient),
        tokenIds,
        amounts
      )
    );
  }
}
