// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {IRollupVerifier} from "../../libraries/verifier/IRollupVerifier.sol";

contract MockRollupVerifier is IRollupVerifier {
    /// @inheritdoc IRollupVerifier
    function verifyAggregateProof(bytes calldata, bytes32) external view {}
}
