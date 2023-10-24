// SPDX-License-Identifier: AGPL-3.0

pragma solidity 0.8.17;

import "../interfaces/IVerifierRollup.sol";

contract VerifierRollupHelperMock is IVerifierRollup {
    function verifyProof(
        bytes memory proof, 
        uint256[1] memory pubSignals
    ) public view override returns (bool) {
        return true;
    }
}
