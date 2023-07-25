// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

import {DSTestPlus} from "solmate/test/utils/DSTestPlus.sol";

import {ERC1967Proxy} from "@openzeppelin/contracts/proxy/ERC1967/ERC1967Proxy.sol";

import {L1BlockContainer} from "../L2/predeploys/L1BlockContainer.sol";
import {L1GasPriceOracle} from "../L2/predeploys/L1GasPriceOracle.sol";
import {L2MessageQueue} from "../L2/predeploys/L2MessageQueue.sol";
import {Whitelist} from "../L2/predeploys/Whitelist.sol";
import {L1ScrollMessenger} from "../L1/L1ScrollMessenger.sol";
import {L2ScrollMessenger} from "../L2/L2ScrollMessenger.sol";

import {AddressAliasHelper} from "../libraries/common/AddressAliasHelper.sol";

contract L2ScrollMessengerTest is DSTestPlus {
    L1ScrollMessenger internal l1Messenger;

    address internal feeVault;
    Whitelist private whitelist;

    L2ScrollMessenger internal l2Messenger;
    L1BlockContainer internal l1BlockContainer;
    L2MessageQueue internal l2MessageQueue;
    L1GasPriceOracle internal l1GasOracle;

    function setUp() public {
        // Deploy L1 contracts
        l1Messenger = new L1ScrollMessenger();

        // Deploy L2 contracts
        whitelist = new Whitelist(address(this));
        l1BlockContainer = new L1BlockContainer(address(this));
        l2MessageQueue = new L2MessageQueue(address(this));
        l1GasOracle = new L1GasPriceOracle(address(this));
        l2Messenger = L2ScrollMessenger(
            payable(new ERC1967Proxy(address(new L2ScrollMessenger(address(l2MessageQueue))), new bytes(0)))
        );

        // Initialize L2 contracts
        l2Messenger.initialize(address(l1Messenger), feeVault);
        l2MessageQueue.initialize(address(l2Messenger));
        l1GasOracle.updateWhitelist(address(whitelist));
    }

    function testRelayByCounterparty() external {
        hevm.expectRevert("Caller is not L1ScrollMessenger");
        l2Messenger.relayMessage(address(this), address(this), 0, 0, new bytes(0));
    }

    function testForbidCallFromL1() external {
        hevm.startPrank(AddressAliasHelper.applyL1ToL2Alias(address(l1Messenger)));
        hevm.expectRevert("Forbid to call message queue");
        l2Messenger.relayMessage(address(this), address(l2MessageQueue), 0, 0, new bytes(0));

        hevm.expectRevert("Forbid to call self");
        l2Messenger.relayMessage(address(this), address(l2Messenger), 0, 0, new bytes(0));
        hevm.stopPrank();
    }

    function testSendMessage(address refundAddress) external {
        hevm.assume(refundAddress.code.length == 0);
        hevm.assume(uint256(uint160(refundAddress)) > 100); // ignore some precompile contracts
        hevm.assume(refundAddress != address(0x000000000000000000636F6e736F6c652e6c6f67)); // ignore console/console2
        hevm.assume(refundAddress != address(this));

        // Insufficient msg.value
        hevm.expectRevert("msg.value mismatch");
        l2Messenger.sendMessage(address(0), 1, new bytes(0), 21000, refundAddress);

        // succeed normally
        uint256 balanceBefore = refundAddress.balance;
        l2Messenger.sendMessage{value: 1}(address(0), 1, new bytes(0), 21000, refundAddress);
        assertEq(balanceBefore, refundAddress.balance);
    }
}
