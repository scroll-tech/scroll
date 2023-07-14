// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

import {DSTestPlus} from "solmate/test/utils/DSTestPlus.sol";

import {IL1BlockContainer, L1BlockContainer} from "../L2/predeploys/L1BlockContainer.sol";
import {IL1GasPriceOracle, L1GasPriceOracle} from "../L2/predeploys/L1GasPriceOracle.sol";
import {L2MessageQueue} from "../L2/predeploys/L2MessageQueue.sol";
import {Whitelist} from "../L2/predeploys/Whitelist.sol";
import {L1ScrollMessenger} from "../L1/L1ScrollMessenger.sol";
import {L2ScrollMessenger} from "../L2/L2ScrollMessenger.sol";

abstract contract L2GatewayTestBase is DSTestPlus {
    // from L2MessageQueue
    event AppendMessage(uint256 index, bytes32 messageHash);

    // from L2ScrollMessenger
    event SentMessage(
        address indexed sender,
        address indexed target,
        uint256 value,
        uint256 messageNonce,
        uint256 gasLimit,
        bytes message
    );
    event RelayedMessage(bytes32 indexed messageHash);
    event FailedRelayedMessage(bytes32 indexed messageHash);

    // pay 0.1 extra ETH to test refund
    uint256 internal constant extraValue = 1e17;

    L1ScrollMessenger internal l1Messenger;

    address internal feeVault;
    Whitelist private whitelist;

    L2ScrollMessenger internal l2Messenger;
    L1BlockContainer internal l1BlockContainer;
    L2MessageQueue internal l2MessageQueue;
    L1GasPriceOracle internal l1GasOracle;

    function setUpBase() internal {
        feeVault = address(uint160(address(this)) - 1);

        // Deploy L1 contracts
        l1Messenger = new L1ScrollMessenger();

        // Deploy L2 contracts
        whitelist = new Whitelist(address(this));
        l1BlockContainer = new L1BlockContainer(address(this));
        l2MessageQueue = new L2MessageQueue(address(this));
        l1GasOracle = new L1GasPriceOracle(address(this));
        l2Messenger = new L2ScrollMessenger(address(l1BlockContainer), address(l1GasOracle), address(l2MessageQueue));

        // Initialize L2 contracts
        l2Messenger.initialize(address(l1Messenger), feeVault);
        l2MessageQueue.initialize();
        l2MessageQueue.updateMessenger(address(l2Messenger));
        l1GasOracle.updateWhitelist(address(whitelist));

        address[] memory _accounts = new address[](1);
        _accounts[0] = address(this);
        whitelist.updateWhitelistStatus(_accounts, true);
    }

    function setL1BaseFee(uint256 baseFee) internal {
        l1GasOracle.setL1BaseFee(baseFee);
    }
}
