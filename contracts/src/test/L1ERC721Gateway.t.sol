// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { console2 } from "forge-std/console2.sol";
import { DSTestPlus } from "solmate/test/utils/DSTestPlus.sol";
import { MockERC721 } from "solmate/test/utils/mocks/MockERC721.sol";

import { L1ERC721Gateway } from "../L1/gateways/L1ERC721Gateway.sol";
import { L2ERC721Gateway } from "../L2/gateways/L2ERC721Gateway.sol";
import { MockScrollMessenger } from "./mocks/MockScrollMessenger.sol";
import { MockERC721Recipient } from "./mocks/MockERC721Recipient.sol";

contract L1ERC721GatewayTest is DSTestPlus {
  uint256 private constant TOKEN_COUNT = 100;

  MockScrollMessenger private messenger;
  L2ERC721Gateway private counterpart;
  L1ERC721Gateway private gateway;

  MockERC721 private token;
  MockERC721Recipient private mockRecipient;

  function setUp() public {
    messenger = new MockScrollMessenger();

    counterpart = new L2ERC721Gateway();
    gateway = new L1ERC721Gateway();
    gateway.initialize(address(counterpart), address(messenger));

    token = new MockERC721("Mock", "M");
    for (uint256 i = 0; i < TOKEN_COUNT; i++) {
      token.mint(address(this), i);
    }
    token.setApprovalForAll(address(gateway), true);

    mockRecipient = new MockERC721Recipient();
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

  /// @dev failed to deposit erc721
  function testDepositERC721WithGatewayFailed(address to) public {
    // token not support
    hevm.expectRevert("token not supported");
    if (to == address(0)) {
      gateway.depositERC721(address(token), 0, 0);
    } else {
      gateway.depositERC721(address(token), to, 0, 0);
    }
  }

  /// @dev deposit erc721 without recipient
  function testDepositERC721WithGatewaySuccess(uint256 tokenId) public {
    tokenId = bound(tokenId, 0, TOKEN_COUNT - 1);
    gateway.updateTokenMapping(address(token), address(token));

    gateway.depositERC721(address(token), tokenId, 0);
    assertEq(token.ownerOf(tokenId), address(gateway));
    assertEq(token.balanceOf(address(gateway)), 1);

    // @todo check event
  }

  /// @dev deposit erc721 with recipient
  function testDepositERC721WithGatewaySuccess(uint256 tokenId, address to) public {
    tokenId = bound(tokenId, 0, TOKEN_COUNT - 1);
    gateway.updateTokenMapping(address(token), address(token));

    gateway.depositERC721(address(token), to, tokenId, 0);
    assertEq(token.ownerOf(tokenId), address(gateway));
    assertEq(token.balanceOf(address(gateway)), 1);

    // @todo check event
  }

  /// @dev failed to batch deposit erc721
  function testBatchDepositERC721WithGatewayFailed(address to) public {
    // token not support
    hevm.expectRevert("token not supported");
    if (to == address(0)) {
      gateway.batchDepositERC721(address(token), new uint256[](1), 0);
    } else {
      gateway.batchDepositERC721(address(token), to, new uint256[](1), 0);
    }

    // no token to deposit
    hevm.expectRevert("no token to deposit");
    if (to == address(0)) {
      gateway.batchDepositERC721(address(token), new uint256[](0), 0);
    } else {
      gateway.batchDepositERC721(address(token), to, new uint256[](0), 0);
    }
  }

  /// @dev batch deposit erc721 without recipient
  function testBatchDepositERC721WithGatewaySuccess(uint256 count) public {
    count = bound(count, 1, TOKEN_COUNT);
    gateway.updateTokenMapping(address(token), address(token));

    uint256[] memory _tokenIds = new uint256[](count);
    for (uint256 i = 0; i < count; i++) {
      _tokenIds[i] = i;
    }

    gateway.batchDepositERC721(address(token), _tokenIds, 0);
    for (uint256 i = 0; i < count; i++) {
      assertEq(token.ownerOf(i), address(gateway));
    }
    assertEq(token.balanceOf(address(gateway)), count);

    // @todo check event
  }

  /// @dev batch deposit erc721 with recipient
  function testBatchDepositERC721WithGatewaySuccess(uint256 count, address to) public {
    count = bound(count, 1, TOKEN_COUNT);
    gateway.updateTokenMapping(address(token), address(token));

    uint256[] memory _tokenIds = new uint256[](count);
    for (uint256 i = 0; i < count; i++) {
      _tokenIds[i] = i;
    }

    gateway.batchDepositERC721(address(token), to, _tokenIds, 0);
    for (uint256 i = 0; i < count; i++) {
      assertEq(token.ownerOf(i), address(gateway));
    }
    assertEq(token.balanceOf(address(gateway)), count);

    // @todo check event
  }

  /// @dev failed to finalize withdraw erc721
  function testFinalizeWithdrawERC721Failed() public {
    // should revert, called by non-messenger
    hevm.expectRevert("only messenger can call");
    gateway.finalizeWithdrawERC721(address(0), address(0), address(0), address(0), 0);

    // should revert, called by messenger, xDomainMessageSender not set
    hevm.expectRevert("only call by conterpart");
    messenger.callTarget(
      address(gateway),
      abi.encodeWithSelector(
        L1ERC721Gateway.finalizeWithdrawERC721.selector,
        address(0),
        address(0),
        address(0),
        address(0),
        0
      )
    );

    // should revert, called by messenger, xDomainMessageSender set wrong
    messenger.setXDomainMessageSender(address(2));
    hevm.expectRevert("only call by conterpart");
    messenger.callTarget(
      address(gateway),
      abi.encodeWithSelector(
        L1ERC721Gateway.finalizeWithdrawERC721.selector,
        address(0),
        address(0),
        address(0),
        address(0),
        0
      )
    );
  }

  /// @dev finalize withdraw erc721
  function testFinalizeWithdrawERC721(
    address from,
    address to,
    uint256 tokenId
  ) public {
    if (to == address(0) || to.code.length > 0) to = address(1);

    // deposit first
    gateway.updateTokenMapping(address(token), address(token));
    tokenId = bound(tokenId, 0, TOKEN_COUNT - 1);
    gateway.depositERC721(address(token), tokenId, 0);

    // then withdraw
    messenger.setXDomainMessageSender(address(counterpart));
    messenger.callTarget(
      address(gateway),
      abi.encodeWithSelector(
        L1ERC721Gateway.finalizeWithdrawERC721.selector,
        address(token),
        address(token),
        from,
        to,
        tokenId
      )
    );
    assertEq(token.balanceOf(to), 1);
    assertEq(token.ownerOf(tokenId), to);
  }

  /// @dev failed to finalize batch withdraw erc721
  function testFinalizeBatchWithdrawERC721Failed() public {
    // should revert, called by non-messenger
    hevm.expectRevert("only messenger can call");
    gateway.finalizeBatchWithdrawERC721(address(0), address(0), address(0), address(0), new uint256[](0));

    // should revert, called by messenger, xDomainMessageSender not set
    hevm.expectRevert("only call by conterpart");
    messenger.callTarget(
      address(gateway),
      abi.encodeWithSelector(
        L1ERC721Gateway.finalizeBatchWithdrawERC721.selector,
        address(0),
        address(0),
        address(0),
        address(0),
        new uint256[](0)
      )
    );

    // should revert, called by messenger, xDomainMessageSender set wrong
    messenger.setXDomainMessageSender(address(2));
    hevm.expectRevert("only call by conterpart");
    messenger.callTarget(
      address(gateway),
      abi.encodeWithSelector(
        L1ERC721Gateway.finalizeBatchWithdrawERC721.selector,
        address(0),
        address(0),
        address(0),
        address(0),
        new uint256[](0)
      )
    );
  }

  /// @dev finalize batch withdraw erc721
  function testFinalizeBatchWithdrawERC721(
    address from,
    address to,
    uint256 count
  ) public {
    if (to == address(0) || to.code.length > 0) to = address(1);
    gateway.updateTokenMapping(address(token), address(token));

    // deposit first
    count = bound(count, 1, TOKEN_COUNT);
    uint256[] memory _tokenIds = new uint256[](count);
    for (uint256 i = 0; i < count; i++) {
      _tokenIds[i] = i;
    }
    gateway.batchDepositERC721(address(token), _tokenIds, 0);

    // then withdraw
    messenger.setXDomainMessageSender(address(counterpart));
    messenger.callTarget(
      address(gateway),
      abi.encodeWithSelector(
        L1ERC721Gateway.finalizeBatchWithdrawERC721.selector,
        address(token),
        address(token),
        from,
        to,
        _tokenIds
      )
    );
    assertEq(token.balanceOf(to), count);
    for (uint256 i = 0; i < count; i++) {
      assertEq(token.ownerOf(i), to);
    }
  }

  /// @dev should detect reentrance
  function testReentranceWhenFinalizeWithdraw(address from, uint256 tokenId) public {
    // deposit first
    gateway.updateTokenMapping(address(token), address(token));
    tokenId = bound(tokenId, 0, TOKEN_COUNT - 1);
    gateway.depositERC721(address(token), tokenId, 0);

    mockRecipient.setCall(
      address(gateway),
      0,
      abi.encodeWithSignature("depositERC721(address,uint256,uint256)", address(token), tokenId, 0)
    );
    // finalize withdraw
    messenger.setXDomainMessageSender(address(counterpart));
    hevm.expectRevert("ReentrancyGuard: reentrant call");
    messenger.callTarget(
      address(gateway),
      abi.encodeWithSelector(
        L1ERC721Gateway.finalizeWithdrawERC721.selector,
        address(token),
        address(token),
        from,
        address(mockRecipient),
        tokenId
      )
    );

    // finalize batch withdraw
    messenger.setXDomainMessageSender(address(counterpart));
    hevm.expectRevert("ReentrancyGuard: reentrant call");
    uint256[] memory tokenIds = new uint256[](1);
    tokenIds[0] = tokenId;
    messenger.callTarget(
      address(gateway),
      abi.encodeWithSelector(
        L1ERC721Gateway.finalizeBatchWithdrawERC721.selector,
        address(token),
        address(token),
        from,
        address(mockRecipient),
        tokenIds
      )
    );
  }
}
