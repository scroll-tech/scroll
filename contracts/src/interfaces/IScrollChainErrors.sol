// SPDX-License-Identifier: AGPL-3.0

pragma solidity 0.8.16;

interface IScrollChainErrors {
    /**
     * @dev Thrown when the SlotAdapter address is ZeroAddress
     */
    error SlotAdapterEmpty();

    /**
     * @dev Thrown when the address is ZeroAddress
     */
    error ZeroAddress();

    /**
     * @dev Thrown when Caller is not IDEposit contract
     */
    error OnlyDeposit();

    /**
     * @dev Thrown when Caller has not deposited
     */
    error InsufficientPledge();

    /**
     * @dev Thrown when commit wrong batch hash
     */
    error ErrorBatchHash(bytes32);

    /**
     * @dev Thrown when commit time out
     */
    error CommittedTimeout();

    /**
     * @dev Thrown when prover already committed proof
     */
    error CommittedProof();

    /**
     * @dev Thrown when prover already committed proof hash
     */
    error CommittedProofHash();

    /**
     * @dev Thrown when prover submit proof early
     */
    error SubmitProofEarly();

    /**
     * @dev Thrown when prover submitted invalid proof
     */
    error ErrCommitProof();

    /**
     * @dev Thrown when prover submitted proof too late
     */
    error SubmitProofTooLate();
}
