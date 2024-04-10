// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

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

        chain = new MockScrollChain(address(1), address(1));
        uint256[] memory _versions = new uint256[](1);
        address[] memory _verifiers = new address[](1);
        _versions[0] = 0;
        _verifiers[0] = address(v0);
        verifier = new MultipleVersionRollupVerifier(address(chain), _versions, _verifiers);
    }

    function testUpdateVerifierVersion0(address _newVerifier) external {
        hevm.assume(_newVerifier != address(0));

        // set by non-owner, should revert
        hevm.startPrank(address(1));
        hevm.expectRevert("Ownable: caller is not the owner");
        verifier.updateVerifier(0, 0, address(0));
        hevm.stopPrank();

        // start batch index finalized, revert
        hevm.expectRevert(MultipleVersionRollupVerifier.ErrorStartBatchIndexFinalized.selector);
        verifier.updateVerifier(0, 0, address(1));

        // zero verifier address, revert
        hevm.expectRevert(MultipleVersionRollupVerifier.ErrorZeroAddress.selector);
        verifier.updateVerifier(0, 1, address(0));

        // change to random operator
        assertEq(verifier.legacyVerifiersLength(0), 0);
        verifier.updateVerifier(0, uint64(100), _newVerifier);
        assertEq(verifier.legacyVerifiersLength(0), 1);
        (uint64 _startBatchIndex, address _verifier) = verifier.latestVerifier(0);
        assertEq(_startBatchIndex, uint64(100));
        assertEq(_verifier, _newVerifier);
        (_startBatchIndex, _verifier) = verifier.legacyVerifiers(0, 0);
        assertEq(_startBatchIndex, uint64(0));
        assertEq(_verifier, address(v0));

        // change to same batch index
        verifier.updateVerifier(0, uint64(100), address(v1));
        (_startBatchIndex, _verifier) = verifier.latestVerifier(0);
        assertEq(_startBatchIndex, uint64(100));
        assertEq(_verifier, address(v1));
        (_startBatchIndex, _verifier) = verifier.legacyVerifiers(0, 0);
        assertEq(_startBatchIndex, uint64(0));
        assertEq(_verifier, address(v0));

        // start batch index too small, revert
        hevm.expectRevert(MultipleVersionRollupVerifier.ErrorStartBatchIndexTooSmall.selector);
        verifier.updateVerifier(0, 99, _newVerifier);
    }

    function testUpdateVerifierVersion(uint256 version, address _newVerifier) external {
        hevm.assume(version != 0);
        hevm.assume(_newVerifier != address(0));

        // set v0
        assertEq(verifier.legacyVerifiersLength(version), 0);
        verifier.updateVerifier(version, 1, address(v0));
        assertEq(verifier.legacyVerifiersLength(version), 0);
        (uint64 _startBatchIndex, address _verifier) = verifier.latestVerifier(version);
        assertEq(_startBatchIndex, 1);
        assertEq(_verifier, address(v0));

        // set by non-owner, should revert
        hevm.startPrank(address(1));
        hevm.expectRevert("Ownable: caller is not the owner");
        verifier.updateVerifier(version, 0, address(0));
        hevm.stopPrank();

        // start batch index finalized, revert
        hevm.expectRevert(MultipleVersionRollupVerifier.ErrorStartBatchIndexFinalized.selector);
        verifier.updateVerifier(version, 0, address(1));

        // zero verifier address, revert
        hevm.expectRevert(MultipleVersionRollupVerifier.ErrorZeroAddress.selector);
        verifier.updateVerifier(version, 1, address(0));

        // change to random operator
        assertEq(verifier.legacyVerifiersLength(version), 0);
        verifier.updateVerifier(version, uint64(100), _newVerifier);
        assertEq(verifier.legacyVerifiersLength(version), 1);
        (_startBatchIndex, _verifier) = verifier.latestVerifier(version);
        assertEq(_startBatchIndex, uint64(100));
        assertEq(_verifier, _newVerifier);
        (_startBatchIndex, _verifier) = verifier.legacyVerifiers(version, 0);
        assertEq(_startBatchIndex, uint64(1));
        assertEq(_verifier, address(v0));

        // change to same batch index
        verifier.updateVerifier(version, uint64(100), address(v1));
        (_startBatchIndex, _verifier) = verifier.latestVerifier(version);
        assertEq(_startBatchIndex, uint64(100));
        assertEq(_verifier, address(v1));
        (_startBatchIndex, _verifier) = verifier.legacyVerifiers(version, 0);
        assertEq(_startBatchIndex, uint64(1));
        assertEq(_verifier, address(v0));

        // start batch index too small, revert
        hevm.expectRevert(MultipleVersionRollupVerifier.ErrorStartBatchIndexTooSmall.selector);
        verifier.updateVerifier(version, 99, _newVerifier);
    }

    function testGetVerifierV0() external {
        verifier.updateVerifier(0, 100, address(v1));
        verifier.updateVerifier(0, 300, address(v2));

        assertEq(verifier.getVerifier(0, 0), address(v0));
        assertEq(verifier.getVerifier(0, 1), address(v0));
        assertEq(verifier.getVerifier(0, 99), address(v0));
        assertEq(verifier.getVerifier(0, 100), address(v1));
        assertEq(verifier.getVerifier(0, 101), address(v1));
        assertEq(verifier.getVerifier(0, 299), address(v1));
        assertEq(verifier.getVerifier(0, 300), address(v2));
        assertEq(verifier.getVerifier(0, 301), address(v2));
        assertEq(verifier.getVerifier(0, 10000), address(v2));
    }

    function testGetVerifier(uint256 version) external {
        hevm.assume(version != 0);

        verifier.updateVerifier(version, 1, address(v0));
        verifier.updateVerifier(version, 100, address(v1));
        verifier.updateVerifier(version, 300, address(v2));

        assertEq(verifier.getVerifier(version, 1), address(v0));
        assertEq(verifier.getVerifier(version, 99), address(v0));
        assertEq(verifier.getVerifier(version, 100), address(v1));
        assertEq(verifier.getVerifier(version, 101), address(v1));
        assertEq(verifier.getVerifier(version, 299), address(v1));
        assertEq(verifier.getVerifier(version, 300), address(v2));
        assertEq(verifier.getVerifier(version, 301), address(v2));
        assertEq(verifier.getVerifier(version, 10000), address(v2));
    }

    function testVerifyAggregateProofV0() external {
        verifier.updateVerifier(0, 100, address(v1));
        verifier.updateVerifier(0, 300, address(v2));

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

    function testVerifyAggregateProof(uint256 version) external {
        hevm.assume(version != 0);

        verifier.updateVerifier(version, 1, address(v0));
        verifier.updateVerifier(version, 100, address(v1));
        verifier.updateVerifier(version, 300, address(v2));

        hevm.expectRevert(abi.encode(address(v0)));
        verifier.verifyAggregateProof(version, 1, new bytes(0), bytes32(0));
        hevm.expectRevert(abi.encode(address(v0)));
        verifier.verifyAggregateProof(version, 99, new bytes(0), bytes32(0));
        hevm.expectRevert(abi.encode(address(v1)));
        verifier.verifyAggregateProof(version, 100, new bytes(0), bytes32(0));
        hevm.expectRevert(abi.encode(address(v1)));
        verifier.verifyAggregateProof(version, 101, new bytes(0), bytes32(0));
        hevm.expectRevert(abi.encode(address(v1)));
        verifier.verifyAggregateProof(version, 299, new bytes(0), bytes32(0));
        hevm.expectRevert(abi.encode(address(v2)));
        verifier.verifyAggregateProof(version, 300, new bytes(0), bytes32(0));
        hevm.expectRevert(abi.encode(address(v2)));
        verifier.verifyAggregateProof(version, 301, new bytes(0), bytes32(0));
        hevm.expectRevert(abi.encode(address(v2)));
        verifier.verifyAggregateProof(version, 10000, new bytes(0), bytes32(0));
    }
}
