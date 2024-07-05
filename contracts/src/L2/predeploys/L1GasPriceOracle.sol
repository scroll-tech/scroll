// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {OwnableBase} from "../../libraries/common/OwnableBase.sol";
import {IWhitelist} from "../../libraries/common/IWhitelist.sol";

import {IL1GasPriceOracle} from "./IL1GasPriceOracle.sol";

contract L1GasPriceOracle is OwnableBase, IL1GasPriceOracle {
    /**********
     * Events *
     **********/

    /// @notice Emitted when owner updates whitelist contract.
    /// @param _oldWhitelist The address of old whitelist contract.
    /// @param _newWhitelist The address of new whitelist contract.
    event UpdateWhitelist(address _oldWhitelist, address _newWhitelist);

    /**********
     * Errors *
     **********/

    /// @dev Thrown when the blob fee scalar exceed `MAX_BLOB_SCALAR`.
    error ErrExceedMaxBlobScalar();

    /// @dev Thrown when the commit fee scalar exceed `MAX_COMMIT_SCALAR`.
    error ErrExceedMaxCommitScalar();

    /// @dev Thrown when the l1 fee overhead exceed `MAX_OVERHEAD`.
    error ErrExceedMaxOverhead();

    /// @dev Thrown when the l1 fee scalar exceed `MAX_SCALAR`.
    error ErrExceedMaxScalar();

    /// @dev Thrown when the caller is not whitelisted.
    error ErrCallerNotWhitelisted();

    /// @dev Thrown when we enable Curie fork after Curie fork.
    error ErrAlreadyInCurieFork();

    /*************
     * Constants *
     *************/

    /// @dev The precision used in the scalar.
    uint256 private constant PRECISION = 1e9;

    /// @dev The maximum possible l1 fee overhead.
    ///      Computed based on current l1 block gas limit.
    uint256 private constant MAX_OVERHEAD = 30000000 / 16;

    /// @dev The maximum possible l1 fee scale before Curie.
    ///      x1000 should be enough.
    uint256 private constant MAX_SCALAR = 1000 * PRECISION;

    /// @dev The maximum possible l1 commit fee scalar after Curie.
    /// We derive the commit scalar by
    /// ```
    /// commit_scalar = commit_gas_per_tx * fluctuation_multiplier * 1e9
    /// ```
    /// So, the value should not exceed 10^9 * 1e9 normally.
    uint256 private constant MAX_COMMIT_SCALAR = 10**9 * PRECISION;

    /// @dev The maximum possible l1 blob fee scalar after Curie.
    /// We derive the blob scalar by
    /// ```
    /// blob_scalar = fluctuation_multiplier / compression_ratio / blob_util_ratio * 1e9
    /// ```
    /// So, the value should not exceed 10^9 * 1e9 normally.
    uint256 private constant MAX_BLOB_SCALAR = 10**9 * PRECISION;

    /*************
     * Variables *
     *************/

    /// @inheritdoc IL1GasPriceOracle
    uint256 public l1BaseFee;

    /// @inheritdoc IL1GasPriceOracle
    uint256 public override overhead;

    /// @inheritdoc IL1GasPriceOracle
    uint256 public override scalar;

    /// @notice The address of whitelist contract.
    IWhitelist public whitelist;

    /// @inheritdoc IL1GasPriceOracle
    uint256 public override l1BlobBaseFee;

    /// @inheritdoc IL1GasPriceOracle
    uint256 public override commitScalar;

    /// @inheritdoc IL1GasPriceOracle
    uint256 public override blobScalar;

    /// @notice Indicates whether the network has gone through the Curie upgrade.
    bool public isCurie;

    /*************
     * Modifiers *
     *************/

    modifier onlyWhitelistedSender() {
        if (!whitelist.isSenderAllowed(msg.sender)) revert ErrCallerNotWhitelisted();
        _;
    }

    /***************
     * Constructor *
     ***************/

    constructor(address _owner) {
        _transferOwnership(_owner);
    }

    /*************************
     * Public View Functions *
     *************************/

    /// @inheritdoc IL1GasPriceOracle
    function getL1Fee(bytes memory _data) external view override returns (uint256) {
        if (isCurie) {
            return _getL1FeeCurie(_data);
        } else {
            return _getL1FeeBeforeCurie(_data);
        }
    }

    /// @inheritdoc IL1GasPriceOracle
    function getL1GasUsed(bytes memory _data) public view override returns (uint256) {
        if (isCurie) {
            // It is near zero since we put all transactions to blob.
            return 0;
        } else {
            return _getL1GasUsedBeforeCurie(_data);
        }
    }

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @inheritdoc IL1GasPriceOracle
    function setL1BaseFee(uint256 _l1BaseFee) external override onlyWhitelistedSender {
        l1BaseFee = _l1BaseFee;

        emit L1BaseFeeUpdated(_l1BaseFee);
    }

    /// @inheritdoc IL1GasPriceOracle
    function setL1BaseFeeAndBlobBaseFee(uint256 _l1BaseFee, uint256 _l1BlobBaseFee)
        external
        override
        onlyWhitelistedSender
    {
        l1BaseFee = _l1BaseFee;
        l1BlobBaseFee = _l1BlobBaseFee;

        emit L1BaseFeeUpdated(_l1BaseFee);
        emit L1BlobBaseFeeUpdated(_l1BlobBaseFee);
    }

    /************************
     * Restricted Functions *
     ************************/

    /// @notice Allows the owner to modify the overhead.
    /// @param _overhead New overhead
    function setOverhead(uint256 _overhead) external onlyOwner {
        if (_overhead > MAX_OVERHEAD) revert ErrExceedMaxOverhead();

        overhead = _overhead;
        emit OverheadUpdated(_overhead);
    }

    /// Allows the owner to modify the scalar.
    /// @param _scalar New scalar
    function setScalar(uint256 _scalar) external onlyOwner {
        if (_scalar > MAX_SCALAR) revert ErrExceedMaxScalar();

        scalar = _scalar;
        emit ScalarUpdated(_scalar);
    }

    /// Allows the owner to modify the commit scalar.
    /// @param _scalar New scalar
    function setCommitScalar(uint256 _scalar) external onlyOwner {
        if (_scalar > MAX_COMMIT_SCALAR) revert ErrExceedMaxCommitScalar();

        commitScalar = _scalar;
        emit CommitScalarUpdated(_scalar);
    }

    /// Allows the owner to modify the blob scalar.
    /// @param _scalar New scalar
    function setBlobScalar(uint256 _scalar) external onlyOwner {
        if (_scalar > MAX_BLOB_SCALAR) revert ErrExceedMaxBlobScalar();

        blobScalar = _scalar;
        emit BlobScalarUpdated(_scalar);
    }

    /// @notice Update whitelist contract.
    /// @dev This function can only called by contract owner.
    /// @param _newWhitelist The address of new whitelist contract.
    function updateWhitelist(address _newWhitelist) external onlyOwner {
        address _oldWhitelist = address(whitelist);

        whitelist = IWhitelist(_newWhitelist);
        emit UpdateWhitelist(_oldWhitelist, _newWhitelist);
    }

    /// @notice Enable the Curie fork (callable by contract owner).
    ///
    /// @dev Since this is a predeploy contract, we will directly set the slot while hard fork
    /// to avoid external owner operations.
    /// The reason that we keep this function is for easy unit testing.
    function enableCurie() external onlyOwner {
        if (isCurie) revert ErrAlreadyInCurieFork();
        isCurie = true;
    }

    /**********************
     * Internal Functions *
     **********************/

    /// @dev Internal function to computes the amount of L1 gas used for a transaction before Curie fork.
    ///   The `_data` is the RLP-encoded transaction with signature. And we also reserve additional
    ///   4 bytes in the non-zero bytes to store the number of bytes in the RLP-encoded transaction.
    /// @param _data Signed fully RLP-encoded transaction to get the L1 gas for.
    /// @return Amount of L1 gas used to publish the transaction.
    function _getL1GasUsedBeforeCurie(bytes memory _data) private view returns (uint256) {
        uint256 _total = 0;
        uint256 _length = _data.length;
        unchecked {
            for (uint256 i = 0; i < _length; i++) {
                if (_data[i] == 0) {
                    _total += 4;
                } else {
                    _total += 16;
                }
            }
            return _total + overhead + (4 * 16);
        }
    }

    /// @dev Internal function to compute the L1 portion of the fee based on the size of the rlp encoded input
    ///   transaction, the current L1 base fee, and the various dynamic parameters, before Curie fork.
    /// @param _data Signed fully RLP-encoded transaction to get the L1 fee for.
    /// @return L1 fee that should be paid for the tx
    function _getL1FeeBeforeCurie(bytes memory _data) private view returns (uint256) {
        uint256 _l1GasUsed = _getL1GasUsedBeforeCurie(_data);
        uint256 _l1Fee = _l1GasUsed * l1BaseFee;
        return (_l1Fee * scalar) / PRECISION;
    }

    /// @dev Internal function to compute the L1 portion of the fee based on the size of the rlp encoded input
    ///   transaction, the current L1 base fee, and the various dynamic parameters, after Curie fork.
    /// @param _data Signed fully RLP-encoded transaction to get the L1 fee for.
    /// @return L1 fee that should be paid for the tx
    function _getL1FeeCurie(bytes memory _data) private view returns (uint256) {
        // We have bounded the value of `commitScalar` and `blobScalar`, the whole expression won't overflow.
        return (commitScalar * l1BaseFee + blobScalar * _data.length * l1BlobBaseFee) / PRECISION;
    }
}
