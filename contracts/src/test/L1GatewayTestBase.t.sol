// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {DSTestPlus} from "solmate/test/utils/DSTestPlus.sol";

import {EnforcedTxGateway} from "../L1/gateways/EnforcedTxGateway.sol";
import {L1MessageQueue} from "../L1/rollup/L1MessageQueue.sol";
import {L2GasPriceOracle} from "../L1/rollup/L2GasPriceOracle.sol";
import {ScrollChain, IScrollChain} from "../L1/rollup/ScrollChain.sol";
import {Whitelist} from "../L2/predeploys/Whitelist.sol";
import {L1ScrollMessenger} from "../L1/L1ScrollMessenger.sol";
import {L2ScrollMessenger} from "../L2/L2ScrollMessenger.sol";

abstract contract L1GatewayTestBase is DSTestPlus {
    // from L1MessageQueue
    event QueueTransaction(
        address indexed sender,
        address indexed target,
        uint256 value,
        uint256 queueIndex,
        uint256 gasLimit,
        bytes data
    );

    // from L1ScrollMessenger
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
    L1MessageQueue internal messageQueue;
    L2GasPriceOracle internal gasOracle;
    EnforcedTxGateway internal enforcedTxGateway;
    ScrollChain internal rollup;

    address internal feeVault;
    Whitelist private whitelist;

    L2ScrollMessenger internal l2Messenger;

    function setUpBase() internal {
        feeVault = address(uint160(address(this)) - 1);

        // Deploy L1 contracts
        l1Messenger = new L1ScrollMessenger();
        messageQueue = new L1MessageQueue();
        gasOracle = new L2GasPriceOracle();
        rollup = new ScrollChain(1233);
        enforcedTxGateway = new EnforcedTxGateway();
        whitelist = new Whitelist(address(this));

        // Deploy L2 contracts
        l2Messenger = new L2ScrollMessenger(address(0), address(0), address(0));

        // Initialize L1 contracts
        l1Messenger.initialize(address(l2Messenger), feeVault, address(rollup), address(messageQueue));
        messageQueue.initialize(address(l1Messenger), address(enforcedTxGateway), address(gasOracle), 10000000);
        gasOracle.initialize(0, 0, 0, 0);
        gasOracle.updateWhitelist(address(whitelist));
        rollup.initialize(address(messageQueue), address(0));

        address[] memory _accounts = new address[](1);
        _accounts[0] = address(this);
        whitelist.updateWhitelistStatus(_accounts, true);
    }

    function prepareL2MessageRoot(bytes32 messageHash) internal {
        bytes memory _batchHeader = new bytes(121);
        assembly {
            mstore(add(_batchHeader, 57), 1)
            mstore(add(_batchHeader, 89), 1)
        }

        rollup.importGenesisBatch(_batchHeader, bytes32(uint256(1)), messageHash);
    }
}
