// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

import {DSTestPlus} from "solmate/test/utils/DSTestPlus.sol";

import {L1MessageQueue} from "../L1/rollup/L1MessageQueue.sol";
import {MultipleVersionRollupVerifier} from "../L1/rollup/MultipleVersionRollupVerifier.sol";

import {MockScrollChain} from "./mocks/MockScrollChain.sol";
import {MockZkEvmVerifier} from "./mocks/MockZkEvmVerifier.sol";

contract MultipleVersionRollupVerifierTest is DSTestPlus {
    // from MultipleVersionRollupVerifier
    event UpdateVerifier(uint256 startBatchIndex, address verifier);

    MultipleVersionRollupVerifier private verifier;
    MockZkEvmVerifier private v0;
    MockZkEvmVerifier private v1;
    MockZkEvmVerifier private v2;
    MockScrollChain private chain;

    function setUp() external {
        v0 = new MockZkEvmVerifier();
        v1 = new MockZkEvmVerifier();
        v2 = new MockZkEvmVerifier();
        chain = new MockScrollChain();

        verifier = new MultipleVersionRollupVerifier(address(v0));
    }

    function testInitialize(address _chain) external {
        hevm.assume(_chain != address(0));

        // set by non-owner, should revert
        hevm.startPrank(address(1));
        hevm.expectRevert("Ownable: caller is not the owner");
        verifier.initialize(_chain);
        hevm.stopPrank();

        // succeed
        assertEq(verifier.scrollChain(), address(0));
        verifier.initialize(_chain);
        assertEq(verifier.scrollChain(), _chain);

        // initialized, revert
        hevm.expectRevert("initialized");
        verifier.initialize(_chain);
    }

    function testUpdateVerifier(address _newVerifier) external {
        hevm.assume(_newVerifier != address(0));

        verifier.initialize(address(chain));

        // set by non-owner, should revert
        hevm.startPrank(address(1));
        hevm.expectRevert("Ownable: caller is not the owner");
        verifier.updateVerifier(0, address(0));
        hevm.stopPrank();

        // start batch index finalized, revert
        hevm.expectRevert("start batch index finalized");
        verifier.updateVerifier(0, address(1));

        // zero verifier address, revert
        hevm.expectRevert("zero verifier address");
        verifier.updateVerifier(1, address(0));

        // change to random operator
        assertEq(verifier.legacyVerifiersLength(), 0);
        verifier.updateVerifier(uint64(100), _newVerifier);
        assertEq(verifier.legacyVerifiersLength(), 1);
        (uint64 _startBatchIndex, address _verifier) = verifier.latestVerifier();
        assertEq(_startBatchIndex, uint64(100));
        assertEq(_verifier, _newVerifier);
        (_startBatchIndex, _verifier) = verifier.legacyVerifiers(0);
        assertEq(_startBatchIndex, uint64(0));
        assertEq(_verifier, address(v0));

        // change to same batch index
        verifier.updateVerifier(uint64(100), address(v1));
        (_startBatchIndex, _verifier) = verifier.latestVerifier();
        assertEq(_startBatchIndex, uint64(100));
        assertEq(_verifier, address(v1));
        (_startBatchIndex, _verifier) = verifier.legacyVerifiers(0);
        assertEq(_startBatchIndex, uint64(0));
        assertEq(_verifier, address(v0));

        // start batch index too small, revert
        hevm.expectRevert("start batch index too small");
        verifier.updateVerifier(99, _newVerifier);
    }

    function testGetVerifier() external {
        verifier.initialize(address(chain));

        verifier.updateVerifier(100, address(v1));
        verifier.updateVerifier(300, address(v2));

        assertEq(verifier.getVerifier(0), address(v0));
        assertEq(verifier.getVerifier(1), address(v0));
        assertEq(verifier.getVerifier(99), address(v0));
        assertEq(verifier.getVerifier(100), address(v1));
        assertEq(verifier.getVerifier(101), address(v1));
        assertEq(verifier.getVerifier(299), address(v1));
        assertEq(verifier.getVerifier(300), address(v2));
        assertEq(verifier.getVerifier(301), address(v2));
        assertEq(verifier.getVerifier(10000), address(v2));
    }

    function testVerifyAggregateProof() external {
        verifier.initialize(address(chain));

        verifier.updateVerifier(100, address(v1));
        verifier.updateVerifier(300, address(v2));

        hevm.expectRevert(abi.encode(address(v0)));
        verifier.verifyAggregateProof(0, new bytes(0), bytes32(0));
        hevm.expectRevert(abi.encode(address(v0)));
        verifier.verifyAggregateProof(1, new bytes(0), bytes32(0));
        hevm.expectRevert(abi.encode(address(v0)));
        verifier.verifyAggregateProof(99, new bytes(0), bytes32(0));
        hevm.expectRevert(abi.encode(address(v1)));
        verifier.verifyAggregateProof(100, new bytes(0), bytes32(0));
        hevm.expectRevert(abi.encode(address(v1)));
        verifier.verifyAggregateProof(101, new bytes(0), bytes32(0));
        hevm.expectRevert(abi.encode(address(v1)));
        verifier.verifyAggregateProof(299, new bytes(0), bytes32(0));
        hevm.expectRevert(abi.encode(address(v2)));
        verifier.verifyAggregateProof(300, new bytes(0), bytes32(0));
        hevm.expectRevert(abi.encode(address(v2)));
        verifier.verifyAggregateProof(301, new bytes(0), bytes32(0));
        hevm.expectRevert(abi.encode(address(v2)));
        verifier.verifyAggregateProof(10000, new bytes(0), bytes32(0));
    }
}
