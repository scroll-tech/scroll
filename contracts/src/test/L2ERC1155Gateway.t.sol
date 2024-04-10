// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {DSTestPlus} from "solmate/test/utils/DSTestPlus.sol";
import {MockERC1155} from "solmate/test/utils/mocks/MockERC1155.sol";
import {ERC1155TokenReceiver} from "solmate/tokens/ERC1155.sol";

import {ERC1967Proxy} from "@openzeppelin/contracts/proxy/ERC1967/ERC1967Proxy.sol";

import {L1ERC1155Gateway} from "../L1/gateways/L1ERC1155Gateway.sol";
import {L2ERC1155Gateway} from "../L2/gateways/L2ERC1155Gateway.sol";
import {MockScrollMessenger} from "./mocks/MockScrollMessenger.sol";
import {MockERC1155Recipient} from "./mocks/MockERC1155Recipient.sol";

contract L2ERC1155GatewayTest is DSTestPlus, ERC1155TokenReceiver {
    /**********
     * Errors *
     **********/

    // from IScrollGateway
    error ErrorZeroAddress();
    error ErrorCallerIsNotMessenger();
    error ErrorCallerIsNotCounterpartGateway();
    error ErrorNotInDropMessageContext();

    uint256 private constant TOKEN_COUNT = 100;

    MockScrollMessenger private messenger;
    L1ERC1155Gateway private counterpart;
    L2ERC1155Gateway private gateway;

    MockERC1155 private token;
    MockERC1155Recipient private mockRecipient;

    function setUp() public {
        messenger = new MockScrollMessenger();
        counterpart = new L1ERC1155Gateway(address(1), address(1));

        gateway = _deployGateway(address(messenger));
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
        hevm.expectRevert("token address cannot be 0");
        gateway.updateTokenMapping(token1, address(0));
    }

    function testUpdateTokenMappingSuccess(address token1, address token2) public {
        if (token2 == address(0)) token2 = address(1);

        assertEq(gateway.tokenMapping(token1), address(0));
        gateway.updateTokenMapping(token1, token2);
        assertEq(gateway.tokenMapping(token1), token2);
    }

    /// @dev failed to withdraw erc1155
    function testWithdrawERC1155WithGatewayFailed(address to) public {
        // token not support
        hevm.expectRevert("no corresponding l1 token");
        if (to == address(0)) {
            gateway.withdrawERC1155(address(token), 0, 1, 0);
        } else {
            gateway.withdrawERC1155(address(token), to, 0, 1, 0);
        }

        // withdraw zero amount
        hevm.expectRevert("withdraw zero amount");
        if (to == address(0)) {
            gateway.withdrawERC1155(address(token), 0, 0, 0);
        } else {
            gateway.withdrawERC1155(address(token), to, 0, 0, 0);
        }
    }

    /// @dev withdraw erc1155 without recipient
    function testWithdrawERC1155WithGatewaySuccess(uint256 tokenId, uint256 amount) public {
        tokenId = bound(tokenId, 0, TOKEN_COUNT - 1);
        amount = bound(amount, 1, type(uint256).max);
        gateway.updateTokenMapping(address(token), address(token));

        gateway.withdrawERC1155(address(token), tokenId, amount, 0);
        assertEq(token.balanceOf(address(gateway), tokenId), 0);
        assertEq(token.balanceOf(address(this), tokenId), type(uint256).max - amount);

        // @todo check event
    }

    /// @dev withdraw erc1155 with recipient
    function testWithdrawERC1155WithGatewaySuccess(
        uint256 tokenId,
        uint256 amount,
        address to
    ) public {
        tokenId = bound(tokenId, 0, TOKEN_COUNT - 1);
        amount = bound(amount, 1, type(uint256).max);
        gateway.updateTokenMapping(address(token), address(token));

        gateway.withdrawERC1155(address(token), to, tokenId, amount, 0);
        assertEq(token.balanceOf(address(gateway), tokenId), 0);
        assertEq(token.balanceOf(address(this), tokenId), type(uint256).max - amount);

        // @todo check event
    }

    /// @dev failed to batch withdraw erc1155
    function testBatchWithdrawERC1155WithGatewayFailed(address to) public {
        // no token to withdraw
        hevm.expectRevert("no token to withdraw");
        if (to == address(0)) {
            gateway.batchWithdrawERC1155(address(token), new uint256[](0), new uint256[](0), 0);
        } else {
            gateway.batchWithdrawERC1155(address(token), to, new uint256[](0), new uint256[](0), 0);
        }

        // length mismatch
        hevm.expectRevert("length mismatch");
        if (to == address(0)) {
            gateway.batchWithdrawERC1155(address(token), new uint256[](1), new uint256[](0), 0);
        } else {
            gateway.batchWithdrawERC1155(address(token), to, new uint256[](1), new uint256[](0), 0);
        }

        uint256[] memory amounts = new uint256[](1);
        // withdraw zero amount
        hevm.expectRevert("withdraw zero amount");
        if (to == address(0)) {
            gateway.batchWithdrawERC1155(address(token), new uint256[](1), amounts, 0);
        } else {
            gateway.batchWithdrawERC1155(address(token), to, new uint256[](1), amounts, 0);
        }

        // token not support
        amounts[0] = 1;
        hevm.expectRevert("no corresponding l1 token");
        if (to == address(0)) {
            gateway.batchWithdrawERC1155(address(token), new uint256[](1), amounts, 0);
        } else {
            gateway.batchWithdrawERC1155(address(token), to, new uint256[](1), amounts, 0);
        }
    }

    /// @dev batch withdraw erc1155 without recipient
    function testBatchWithdrawERC1155WithGatewaySuccess(uint256 count, uint256 amount) public {
        count = bound(count, 1, TOKEN_COUNT);
        amount = bound(amount, 1, type(uint256).max);
        gateway.updateTokenMapping(address(token), address(token));

        uint256[] memory _tokenIds = new uint256[](count);
        uint256[] memory _amounts = new uint256[](count);
        for (uint256 i = 0; i < count; i++) {
            _tokenIds[i] = i;
            _amounts[i] = amount;
        }

        gateway.batchWithdrawERC1155(address(token), _tokenIds, _amounts, 0);
        for (uint256 i = 0; i < count; i++) {
            assertEq(token.balanceOf(address(gateway), i), 0);
            assertEq(token.balanceOf(address(this), i), type(uint256).max - amount);
        }

        // @todo check event
    }

    /// @dev batch withdraw erc1155 with recipient
    function testBatchWithdrawERC1155WithGatewaySuccess(
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

        gateway.batchWithdrawERC1155(address(token), to, _tokenIds, _amounts, 0);
        for (uint256 i = 0; i < count; i++) {
            assertEq(token.balanceOf(address(gateway), i), 0);
            assertEq(token.balanceOf(address(this), i), type(uint256).max - _amounts[i]);
        }

        // @todo check event
    }

    /// @dev failed to finalize deposit erc1155
    function testFinalizeDepositERC1155Failed() public {
        // should revert, called by non-messenger
        hevm.expectRevert(ErrorCallerIsNotMessenger.selector);
        gateway.finalizeDepositERC1155(address(0), address(0), address(0), address(0), 0, 1);

        // should revert, called by messenger, xDomainMessageSender not set
        hevm.expectRevert(ErrorCallerIsNotCounterpartGateway.selector);
        messenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(
                L2ERC1155Gateway.finalizeDepositERC1155.selector,
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
        hevm.expectRevert(ErrorCallerIsNotCounterpartGateway.selector);
        messenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(
                L2ERC1155Gateway.finalizeDepositERC1155.selector,
                address(0),
                address(0),
                address(0),
                address(0),
                0,
                1
            )
        );
    }

    /// @dev finalize deposit erc1155
    function testFinalizeDepositERC1155(
        address from,
        address to,
        uint256 tokenId,
        uint256 amount
    ) public {
        hevm.assume(to != address(0));
        hevm.assume(to.code.length == 0);

        gateway.updateTokenMapping(address(token), address(token));
        tokenId = bound(tokenId, 0, TOKEN_COUNT - 1);
        amount = bound(amount, 1, type(uint256).max);

        // finalize deposit
        messenger.setXDomainMessageSender(address(counterpart));
        messenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(
                L2ERC1155Gateway.finalizeDepositERC1155.selector,
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

    /// @dev failed to finalize batch deposit erc1155
    function testFinalizeBatchDepositERC1155Failed() public {
        // should revert, called by non-messenger
        hevm.expectRevert(ErrorCallerIsNotMessenger.selector);
        gateway.finalizeBatchDepositERC1155(
            address(0),
            address(0),
            address(0),
            address(0),
            new uint256[](0),
            new uint256[](0)
        );

        // should revert, called by messenger, xDomainMessageSender not set
        hevm.expectRevert(ErrorCallerIsNotCounterpartGateway.selector);
        messenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(
                L2ERC1155Gateway.finalizeBatchDepositERC1155.selector,
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
        hevm.expectRevert(ErrorCallerIsNotCounterpartGateway.selector);
        messenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(
                L2ERC1155Gateway.finalizeBatchDepositERC1155.selector,
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

        count = bound(count, 1, TOKEN_COUNT);
        amount = bound(amount, 1, type(uint256).max);
        uint256[] memory _tokenIds = new uint256[](count);
        uint256[] memory _amounts = new uint256[](count);
        for (uint256 i = 0; i < count; i++) {
            _tokenIds[i] = i;
            _amounts[i] = amount;
        }

        // finalzie batch deposit
        messenger.setXDomainMessageSender(address(counterpart));
        messenger.callTarget(
            address(gateway),
            abi.encodeWithSelector(
                L2ERC1155Gateway.finalizeBatchDepositERC1155.selector,
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

    function _deployGateway(address _messenger) internal returns (L2ERC1155Gateway) {
        return
            L2ERC1155Gateway(
                address(
                    new ERC1967Proxy(
                        address(new L2ERC1155Gateway(address(counterpart), address(_messenger))),
                        new bytes(0)
                    )
                )
            );
    }
}
