// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {ScrollChain} from "../L1/rollup/ScrollChain.sol";

import {BatchHeaderV0Codec} from "../libraries/codec/BatchHeaderV0Codec.sol";
import {BatchHeaderV1Codec} from "../libraries/codec/BatchHeaderV1Codec.sol";

contract ScrollChainMockFinalize is ScrollChain {
    /***************
     * Constructor *
     ***************/

    /// @notice Constructor for `ScrollChain` implementation contract.
    ///
    /// @param _chainId The chain id of L2.
    /// @param _messageQueue The address of `L1MessageQueue` contract.
    /// @param _verifier The address of zkevm verifier contract.
    constructor(
        uint64 _chainId,
        address _messageQueue,
        address _verifier
    ) ScrollChain(_chainId, _messageQueue, _verifier) {}

    /*****************************
     * Public Mutating Functions *
     *****************************/

    function finalizeBatch(
        bytes calldata _batchHeader,
        bytes32 _prevStateRoot,
        bytes32 _postStateRoot,
        bytes32 _withdrawRoot
    ) external OnlyProver whenNotPaused {
        require(_prevStateRoot != bytes32(0), "previous state root is zero");
        require(_postStateRoot != bytes32(0), "new state root is zero");

        // compute batch hash and verify
        (uint256 memPtr, bytes32 _batchHash, uint256 _batchIndex, ) = _loadBatchHeader(_batchHeader);

        // verify previous state root.
        require(finalizedStateRoots[_batchIndex - 1] == _prevStateRoot, "incorrect previous state root");

        // avoid duplicated verification
        require(finalizedStateRoots[_batchIndex] == bytes32(0), "batch already verified");

        // check and update lastFinalizedBatchIndex
        unchecked {
            require(lastFinalizedBatchIndex + 1 == _batchIndex, "incorrect batch index");
            lastFinalizedBatchIndex = _batchIndex;
        }

        // record state root and withdraw root
        finalizedStateRoots[_batchIndex] = _postStateRoot;
        withdrawRoots[_batchIndex] = _withdrawRoot;

        // Pop finalized and non-skipped message from L1MessageQueue.
        _popL1Messages(
            BatchHeaderV0Codec.getSkippedBitmapPtr(memPtr),
            BatchHeaderV0Codec.getTotalL1MessagePopped(memPtr),
            BatchHeaderV0Codec.getL1MessagePopped(memPtr)
        );

        emit FinalizeBatch(_batchIndex, _batchHash, _postStateRoot, _withdrawRoot);
    }

    function finalizeBatch4844(
        bytes calldata _batchHeader,
        bytes32 _prevStateRoot,
        bytes32 _postStateRoot,
        bytes32 _withdrawRoot,
        bytes calldata _blobDataProof
    ) external OnlyProver whenNotPaused {
        if (_prevStateRoot == bytes32(0)) revert ErrorPreviousStateRootIsZero();
        if (_postStateRoot == bytes32(0)) revert ErrorStateRootIsZero();

        // compute batch hash and verify
        (uint256 memPtr, bytes32 _batchHash, uint256 _batchIndex, ) = _loadBatchHeader(_batchHeader);
        bytes32 _blobVersionedHash = BatchHeaderV1Codec.getBlobVersionedHash(memPtr);

        // Calls the point evaluation precompile and verifies the output
        {
            (bool success, bytes memory data) = POINT_EVALUATION_PRECOMPILE_ADDR.staticcall(
                abi.encodePacked(_blobVersionedHash, _blobDataProof)
            );
            // We verify that the point evaluation precompile call was successful by testing the latter 32 bytes of the
            // response is equal to BLS_MODULUS as defined in https://eips.ethereum.org/EIPS/eip-4844#point-evaluation-precompile
            if (!success) revert ErrorCallPointEvaluationPrecompileFailed();
            (, uint256 result) = abi.decode(data, (uint256, uint256));
            if (result != BLS_MODULUS) revert ErrorUnexpectedPointEvaluationPrecompileOutput();
        }

        // verify previous state root.
        if (finalizedStateRoots[_batchIndex - 1] != _prevStateRoot) revert ErrorIncorrectPreviousStateRoot();

        // avoid duplicated verification
        if (finalizedStateRoots[_batchIndex] != bytes32(0)) revert ErrorBatchIsAlreadyVerified();

        // check and update lastFinalizedBatchIndex
        unchecked {
            if (lastFinalizedBatchIndex + 1 != _batchIndex) revert ErrorIncorrectBatchIndex();
            lastFinalizedBatchIndex = _batchIndex;
        }

        // record state root and withdraw root
        finalizedStateRoots[_batchIndex] = _postStateRoot;
        withdrawRoots[_batchIndex] = _withdrawRoot;

        // Pop finalized and non-skipped message from L1MessageQueue.
        _popL1Messages(
            BatchHeaderV1Codec.getSkippedBitmapPtr(memPtr),
            BatchHeaderV1Codec.getTotalL1MessagePopped(memPtr),
            BatchHeaderV1Codec.getL1MessagePopped(memPtr)
        );

        emit FinalizeBatch(_batchIndex, _batchHash, _postStateRoot, _withdrawRoot);
    }
}
