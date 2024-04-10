// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {DSTestPlus} from "solmate/test/utils/DSTestPlus.sol";
import {MockERC721} from "solmate/test/utils/mocks/MockERC721.sol";

import {ERC1967Proxy} from "@openzeppelin/contracts/proxy/ERC1967/ERC1967Proxy.sol";

import {L1ERC721Gateway} from "../L1/gateways/L1ERC721Gateway.sol";
import {L2ERC721Gateway} from "../L2/gateways/L2ERC721Gateway.sol";
import {MockScrollMessenger} from "./mocks/MockScrollMessenger.sol";
import {MockERC721Recipient} from "./mocks/MockERC721Recipient.sol";

contract L2ERC721GatewayTest is DSTestPlus {
    /**********
     * Errors *
     **********/

    // from IScrollGateway
    error ErrorZeroAddress();
    error ErrorCallerIsNotMessenger();
    error ErrorCallerIsNotCounterpartGateway();
    error ErrorNotInDropMessageContext();

    uint256 private constant TOKEN_COUNT = 100;
    uint256 private constant NOT_OWNED_TOKEN_ID = 233333;

    MockScrollMessenger private messenger;
    L1ERC721Gateway private counterpart;
    L2ERC721Gateway private gateway;

    MockERC721 private token;
    MockERC721Recipient private mockRecipient;

    function setUp() public {
        messenger = new MockScrollMessenger();
        counterpart = new L1ERC721Gateway(address(1), address(1));

        gateway = _deployGateway(address(messenger));
        gateway.initialize(address(counterpart), address(messenger));

        token = new MockERC721("Mock", "M");
        for (uint256 i = 0; i < TOKEN_COUNT; i++) {
            token.mint(address(this), i);
        }

        mockRecipient = new MockERC721Recipient();
        token.mint(address(mockRecipient), NOT_OWNED_TOKEN_ID);
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

        // l1 token is zero, should revert
        hevm.expectRevert("token address cannot be 0");
        gateway.updateTokenMapping(token1, address(0));
    }

    function testUpdateTokenMappingSuccess(address token1, address token2) public {
        if (token2 == address(0)) token2 = address(1);

        assertEq(gateway.tokenMapping(token1), address(0));
        gateway.updateTokenMapping(token1, token2);
        assertEq(gateway.tokenMapping(token1), token2);
    }

    /// @dev failed to withdraw erc721
    function testWithdrawERC721WithGatewayFailed(address to) public {
        // token not support
        hevm.expectRevert("no corresponding l1 token");
        if (to == address(0)) {
            gateway.withdrawERC721(address(token), 0, 0);
        } else {
            gateway.withdrawERC721(address(token), to, 0, 0);
        }

        // token not owned
        gateway.updateTokenMapping(address(token), address(token));
        hevm.expectRevert("token not owned");
        if (to == address(0)) {
            gateway.withdrawERC721(address(token), NOT_OWNED_TOKEN_ID, 0);
        } else {
            gateway.withdrawERC721(address(token), to, NOT_OWNED_TOKEN_ID, 0);
        }
    }

    /// @dev withdraw erc721 without recipient
    function testWithdrawERC721WithGatewaySuccess(uint256 tokenId) public {
        tokenId = bound(tokenId, 0, TOKEN_COUNT - 1);
        gateway.updateTokenMapping(address(token), address(token));

        gateway.withdrawERC721(address(token), tokenId, 0);
        hevm.expectRevert("NOT_MINTED");
        token.ownerOf(tokenId);
        assertEq(token.balanceOf(address(this)), TOKEN_COUNT - 1);

        // @todo check event
    }

    /// @dev withdraw erc721 with recipient
    function testWithdrawERC721WithGatewaySuccess(uint256 tokenId, address to) public {
        tokenId = bound(tokenId, 0, TOKEN_COUNT - 1);
        gateway.updateTokenMapping(address(token), address(token));

        gateway.withdrawERC721(address(token), to, tokenId, 0);
        hevm.expectRevert("NOT_MINTED");
        token.ownerOf(tokenId);
        assertEq(token.balanceOf(address(this)), TOKEN_COUNT - 1);

        // @todo check event
    }

    /// @dev failed to batch withdraw erc721
    function testBatchWithdrawERC721WithGatewayFailed(address to) public {
        // token not support
        hevm.expectRevert("no corresponding l1 token");
        if (to == address(0)) {
            gateway.batchWithdrawERC721(address(token), new uint256[](1), 0);
        } else {
            gateway.batchWithdrawERC721(address(token), to, new uint256[](1), 0);
        }

        // no token to withdraw
        hevm.expectRevert("no token to withdraw");
        if (to == address(0)) {
            gateway.batchWithdrawERC721(address(token), new uint256[](0), 0);
        } else {
            gateway.batchWithdrawERC721(address(token), to, new uint256[](0), 0);
        }

        // token not owned
        gateway.updateTokenMapping(address(token), address(token));
        uint256[] memory tokenIds = new uint256[](1);
        tokenIds[0] = NOT_OWNED_TOKEN_ID;
        hevm.expectRevert("token not owned");
        if (to == address(0)) {
            gateway.batchWithdrawERC721(address(token), tokenIds, 0);
        } else {
            gateway.batchWithdrawERC721(address(token), to, tokenIds, 0);
        }
    }

    /// @dev batch withdraw erc721 without recipient
    function testBatchWithdrawERC721WithGatewaySuccess(uint256 count) public {
        count = bound(count, 1, TOKEN_COUNT);
        gateway.updateTokenMapping(address(token), address(token));

        uint256[] memory _tokenIds = new uint256[](count);
        for (uint256 i = 0; i < count; i++) {
            _tokenIds[i] = i;
        }

        gateway.batchWithdrawERC721(address(token), _tokenIds, 0);
        for (uint256 i = 0; i < count; i++) {
            hevm.expectRevert("NOT_MINTED");
            token.ownerOf(i);
        }
        assertEq(token.balanceOf(address(this)), TOKEN_COUNT - count);

        // @todo check event
    }

    /// @dev batch withdraw erc721 with recipient
    function testBatchWithdrawERC721WithGatewaySuccess(uint256 count, address to) public {
        count = bound(count, 1, TOKEN_COUNT);
        gateway.updateTokenMapping(address(token), address(token));

        uint256[] memory _tokenIds = new uint256[](count);
        for (uint256 i = 0; i < count; i++) {
            _tokenIds[i] = i;
        }

        gateway.batchWithdrawERC721(address(token), to, _tokenIds, 0);
        for (uint256 i = 0; i < count; i++) {
            hevm.expectRevert("NOT_MINTED");
            token.ownerOf(i);
        }
        assertEq(token.balanceOf(address(this)), TOKEN_COUNT - count);

        // @todo check event
    }

    /// @dev failed to finalize withdraw erc721
    function testFinalizeDepositERC721Failed() public {
        // should revert, called by non-messenger
        hevm.expectRevert(ErrorCallerIsNotMessenger.selector);
        gateway.finalizeDepositERC721(address(0), address(0), address(0), address(0), 0);

        // should revert, called by messenger, xDomainMessageSender not set
        hevm.expectRevert(ErrorCallerIsNotCounterpartGateway.selector);
        messenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(
                L2ERC721Gateway.finalizeDepositERC721.selector,
                address(0),
                address(0),
                address(0),
                address(0),
                0
            )
        );

        // should revert, called by messenger, xDomainMessageSender set wrong
        messenger.setXDomainMessageSender(address(2));
        hevm.expectRevert(ErrorCallerIsNotCounterpartGateway.selector);
        messenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(
                L2ERC721Gateway.finalizeDepositERC721.selector,
                address(0),
                address(0),
                address(0),
                address(0),
                0
            )
        );
    }

    /// @dev finalize withdraw erc721
    function testFinalizeDepositERC721(
        address from,
        address to,
        uint256 tokenId
    ) public {
        hevm.assume(to != address(0));
        hevm.assume(to.code.length == 0);

        tokenId = bound(tokenId, NOT_OWNED_TOKEN_ID + 1, type(uint256).max);
        gateway.updateTokenMapping(address(token), address(token));

        // finalize deposit
        messenger.setXDomainMessageSender(address(counterpart));
        messenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(
                L2ERC721Gateway.finalizeDepositERC721.selector,
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
    function testFinalizeBatchDepositERC721Failed() public {
        // should revert, called by non-messenger
        hevm.expectRevert(ErrorCallerIsNotMessenger.selector);
        gateway.finalizeBatchDepositERC721(address(0), address(0), address(0), address(0), new uint256[](0));

        // should revert, called by messenger, xDomainMessageSender not set
        hevm.expectRevert(ErrorCallerIsNotCounterpartGateway.selector);
        messenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(
                L2ERC721Gateway.finalizeBatchDepositERC721.selector,
                address(0),
                address(0),
                address(0),
                address(0),
                new uint256[](0)
            )
        );

        // should revert, called by messenger, xDomainMessageSender set wrong
        messenger.setXDomainMessageSender(address(2));
        hevm.expectRevert(ErrorCallerIsNotCounterpartGateway.selector);
        messenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(
                L2ERC721Gateway.finalizeBatchDepositERC721.selector,
                address(0),
                address(0),
                address(0),
                address(0),
                new uint256[](0)
            )
        );
    }

    /// @dev finalize batch withdraw erc721
    function testFinalizeBatchDepositERC721(
        address from,
        address to,
        uint256 count
    ) public {
        if (to == address(0)) to = address(1);
        if (to == address(mockRecipient)) to = address(1);
        if (to == address(this)) to = address(1);

        gateway.updateTokenMapping(address(token), address(token));

        // deposit first
        count = bound(count, 1, TOKEN_COUNT);
        uint256[] memory _tokenIds = new uint256[](count);
        for (uint256 i = 0; i < count; i++) {
            _tokenIds[i] = i + NOT_OWNED_TOKEN_ID + 1;
        }

        // then withdraw
        messenger.setXDomainMessageSender(address(counterpart));
        messenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(
                L2ERC721Gateway.finalizeBatchDepositERC721.selector,
                address(token),
                address(token),
                from,
                to,
                _tokenIds
            )
        );
        assertEq(token.balanceOf(to), count);
        for (uint256 i = 0; i < count; i++) {
            assertEq(token.ownerOf(_tokenIds[i]), to);
        }
    }

    function _deployGateway(address _messenger) internal returns (L2ERC721Gateway) {
        return
            L2ERC721Gateway(
                address(
                    new ERC1967Proxy(
                        address(new L2ERC721Gateway(address(counterpart), address(_messenger))),
                        new bytes(0)
                    )
                )
            );
    }
}
