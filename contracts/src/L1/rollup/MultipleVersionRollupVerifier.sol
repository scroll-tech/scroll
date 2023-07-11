// SPDX-License-Identifier: MIT

pragma solidity =0.8.20;

import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";

import {IRollupVerifier} from "../../libraries/verifier/IRollupVerifier.sol";
import {IZkEvmVerifier} from "../../libraries/verifier/IZkEvmVerifier.sol";

contract MultipleVersionRollupVerifier is IRollupVerifier, Ownable {
    /**********
     * Events *
     **********/

    /// @notice Emitted when the address of verifier is updated.
    /// @param startBatchIndex The start batch index when the verifier will be used.
    /// @param verifier The address of new verifier.
    event UpdateVerifier(uint256 startBatchIndex, address verifier);

    /***********
     * Structs *
     ***********/

    struct Verifier {
        // The start batch index for the verifier.
        uint64 startBatchIndex;
        // The address of zkevm verifier.
        address verifier;
    }

    /*************
     * Variables *
     *************/

    /// @notice The list of legacy zkevm verifier, sorted by batchIndex in increasing order.
    Verifier[] public legacyVerifiers;

    /// @notice The lastest used zkevm verifier.
    Verifier public latestVerifier;

    /***************
     * Constructor *
     ***************/

    constructor(address _verifier) {
        require(_verifier != address(0), "zero verifier address");

        latestVerifier.verifier = _verifier;
    }

    /*************************
     * Public View Functions *
     *************************/

    /// @notice Return the number of legacy verifiers.
    function legacyVerifiersLength() external view returns (uint256) {
        return legacyVerifiers.length;
    }

    /// @notice Compute the verifier should be used for specific batch.
    /// @param _batchIndex The batch index to query.
    function getVerifier(uint256 _batchIndex) public view returns (address) {
        // Normally, we will use the latest verifier.
        Verifier memory _verifier = latestVerifier;

        if (_verifier.startBatchIndex > _batchIndex) {
            uint256 _length = legacyVerifiers.length;
            // In most case, only last few verifier will be used by `ScrollChain`.
            // So, we use linear search instead of binary search.
            unchecked {
                for (uint256 i = _length; i > 0; --i) {
                    _verifier = legacyVerifiers[i - 1];
                    if (_verifier.startBatchIndex <= _batchIndex) break;
                }
            }
        }

        return _verifier.verifier;
    }

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @inheritdoc IRollupVerifier
    function verifyAggregateProof(
        uint256 _batchIndex,
        bytes calldata _aggrProof,
        bytes32 _publicInputHash
    ) external view override {
        address _verifier = getVerifier(_batchIndex);

        IZkEvmVerifier(_verifier).verify(_aggrProof, _publicInputHash);
    }

    /************************
     * Restricted Functions *
     ************************/

    /// @notice Update the address of zkevm verifier.
    /// @param _startBatchIndex The start batch index when the verifier will be used.
    /// @param _verifier The address of new verifier.
    function updateVerifier(uint64 _startBatchIndex, address _verifier) external onlyOwner {
        Verifier memory _latestVerifier = latestVerifier;
        require(_startBatchIndex >= _latestVerifier.startBatchIndex, "start batch index too small");
        require(_verifier != address(0), "zero verifier address");

        if (_latestVerifier.startBatchIndex < _startBatchIndex) {
            legacyVerifiers.push(_latestVerifier);
            _latestVerifier.startBatchIndex = _startBatchIndex;
        }
        _latestVerifier.verifier = _verifier;

        latestVerifier = _latestVerifier;

        emit UpdateVerifier(_startBatchIndex, _verifier);
    }
}
