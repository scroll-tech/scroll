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

import {MockRollupVerifier} from "./mocks/MockRollupVerifier.sol";

// solhint-disable no-inline-assembly

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

    MockRollupVerifier internal verifier;

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
        verifier = new MockRollupVerifier();

        // Deploy L2 contracts
        l2Messenger = new L2ScrollMessenger(address(0), address(0), address(0));

        // Initialize L1 contracts
        l1Messenger.initialize(address(l2Messenger), feeVault, address(rollup), address(messageQueue));
        messageQueue.initialize(
            address(l1Messenger),
            address(rollup),
            address(enforcedTxGateway),
            address(gasOracle),
            10000000
        );
        gasOracle.initialize(0, 0, 0, 0);
        gasOracle.updateWhitelist(address(whitelist));
        rollup.initialize(address(messageQueue), address(verifier), 44);

        address[] memory _accounts = new address[](1);
        _accounts[0] = address(this);
        whitelist.updateWhitelistStatus(_accounts, true);
    }

    function prepareL2MessageRoot(bytes32 messageHash) internal {
        rollup.updateSequencer(address(this), true);

        // import genesis batch
        bytes memory batchHeader0 = new bytes(89);
        assembly {
            mstore(add(batchHeader0, add(0x20, 25)), 1)
        }
        rollup.importGenesisBatch(batchHeader0, bytes32(uint256(1)));
        bytes32 batchHash0 = rollup.committedBatches(0);

        // commit one batch
        bytes[] memory chunks = new bytes[](1);
        bytes memory chunk0 = new bytes(1 + 60);
        chunk0[0] = bytes1(uint8(1)); // one block in this chunk
        chunks[0] = chunk0;
        rollup.commitBatch(0, batchHeader0, chunks, new bytes(0));

        bytes memory batchHeader1 = new bytes(89);
        assembly {
            mstore(add(batchHeader1, 0x20), 0) // version
            mstore(add(batchHeader1, add(0x20, 1)), shl(192, 1)) // batchIndex
            mstore(add(batchHeader1, add(0x20, 9)), 0) // l1MessagePopped
            mstore(add(batchHeader1, add(0x20, 17)), 0) // totalL1MessagePopped
            mstore(add(batchHeader1, add(0x20, 25)), 0x246394445f4fe64ed5598554d55d1682d6fb3fe04bf58eb54ef81d1189fafb51) // dataHash
            mstore(add(batchHeader1, add(0x20, 57)), batchHash0) // parentBatchHash
        }

        rollup.finalizeBatchWithProof(
            batchHeader1,
            bytes32(uint256(1)),
            bytes32(uint256(2)),
            messageHash,
            new bytes(0)
        );
    }
}
