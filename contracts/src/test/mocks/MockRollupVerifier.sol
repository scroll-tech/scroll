// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {IRollupVerifier} from "../../libraries/verifier/IRollupVerifier.sol";

contract MockRollupVerifier is IRollupVerifier {
    /// @inheritdoc IRollupVerifier
    function verifyAggregateProof(
        uint256,
        bytes calldata,
        bytes32
    ) external view {}

    /// @inheritdoc IRollupVerifier
    function verifyAggregateProof(
        uint256,
        uint256,
        bytes calldata,
        bytes32
    ) external view {}
}
