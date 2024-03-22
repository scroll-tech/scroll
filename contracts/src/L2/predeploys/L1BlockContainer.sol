// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {IL1BlockContainer} from "./IL1BlockContainer.sol";
import {IL1GasPriceOracle} from "./IL1GasPriceOracle.sol";

import {OwnableBase} from "../../libraries/common/OwnableBase.sol";
import {IWhitelist} from "../../libraries/common/IWhitelist.sol";
import {ScrollPredeploy} from "../../libraries/constants/ScrollPredeploy.sol";

/// @title L1BlockContainer
/// @notice This contract will maintain the list of blocks proposed in L1.
contract L1BlockContainer is OwnableBase, IL1BlockContainer {
    /**********
     * Events *
     **********/

    /// @notice Emitted when owner updates whitelist contract.
    /// @param _oldWhitelist The address of old whitelist contract.
    /// @param _newWhitelist The address of new whitelist contract.
    event UpdateWhitelist(address _oldWhitelist, address _newWhitelist);

    /***********
     * Structs *
     ***********/

    /// @dev Compiler will pack this into single `uint256`.
    struct BlockMetadata {
        // The block height.
        uint64 height;
        // The block timestamp.
        uint64 timestamp;
        // The base fee in the block.
        uint128 baseFee;
    }

    /*************
     * Variables *
     *************/

    /// @notice The address of whitelist contract.
    IWhitelist public whitelist;

    /// @inheritdoc IL1BlockContainer
    bytes32 public override latestBlockHash;

    /// @notice Mapping from block hash to corresponding state root.
    mapping(bytes32 => bytes32) public stateRoot;

    /// @notice Mapping from block hash to corresponding block metadata,
    /// including timestamp and height.
    mapping(bytes32 => BlockMetadata) public metadata;

    /***************
     * Constructor *
     ***************/

    constructor(address _owner) {
        _transferOwnership(_owner);
    }

    function initialize(
        bytes32 _startBlockHash,
        uint64 _startBlockHeight,
        uint64 _startBlockTimestamp,
        uint128 _startBlockBaseFee,
        bytes32 _startStateRoot
    ) external onlyOwner {
        require(latestBlockHash == bytes32(0), "already initialized");

        latestBlockHash = _startBlockHash;
        stateRoot[_startBlockHash] = _startStateRoot;
        metadata[_startBlockHash] = BlockMetadata(_startBlockHeight, _startBlockTimestamp, _startBlockBaseFee);

        emit ImportBlock(_startBlockHash, _startBlockHeight, _startBlockTimestamp, _startBlockBaseFee, _startStateRoot);
    }

    /*************************
     * Public View Functions *
     *************************/

    /// @inheritdoc IL1BlockContainer
    function latestBaseFee() external view override returns (uint256) {
        return metadata[latestBlockHash].baseFee;
    }

    /// @inheritdoc IL1BlockContainer
    function latestBlockNumber() external view override returns (uint256) {
        return metadata[latestBlockHash].height;
    }

    /// @inheritdoc IL1BlockContainer
    function latestBlockTimestamp() external view override returns (uint256) {
        return metadata[latestBlockHash].timestamp;
    }

    /// @inheritdoc IL1BlockContainer
    function getStateRoot(bytes32 _blockHash) external view returns (bytes32) {
        return stateRoot[_blockHash];
    }

    /// @inheritdoc IL1BlockContainer
    function getBlockTimestamp(bytes32 _blockHash) external view returns (uint256) {
        return metadata[_blockHash].timestamp;
    }

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @inheritdoc IL1BlockContainer
    function importBlockHeader(
        bytes32 _blockHash,
        bytes calldata _blockHeaderRLP,
        bool _updateGasPriceOracle
    ) external {
        require(whitelist.isSenderAllowed(msg.sender), "Not whitelisted sender");

        // The encoding order in block header is
        // 1. ParentHash: 32 bytes
        // 2. UncleHash: 32 bytes
        // 3. Coinbase: 20 bytes
        // 4. StateRoot: 32 bytes
        // 5. TransactionsRoot: 32 bytes
        // 6. ReceiptsRoot: 32 bytes
        // 7. LogsBloom: 256 bytes
        // 8. Difficulty: uint
        // 9. BlockHeight: uint
        // 10. GasLimit: uint64
        // 11. GasUsed: uint64
        // 12. BlockTimestamp: uint64
        // 13. ExtraData: several bytes
        // 14. MixHash: 32 bytes
        // 15. BlockNonce: 8 bytes
        // 16. BaseFee: uint // optional
        bytes32 _parentHash;
        bytes32 _stateRoot;
        uint64 _height;
        uint64 _timestamp;
        uint128 _baseFee;

        assembly {
            // reverts with error `msg`.
            // make sure the length of error string <= 32
            function revertWith(msg) {
                // keccak("Error(string)")
                mstore(0x00, shl(224, 0x08c379a0))
                mstore(0x04, 0x20) // str.offset
                mstore(0x44, msg)
                let msgLen
                for {

                } msg {

                } {
                    msg := shl(8, msg)
                    msgLen := add(msgLen, 1)
                }
                mstore(0x24, msgLen) // str.length
                revert(0x00, 0x64)
            }
            // reverts with `msg` when condition is not matched.
            // make sure the length of error string <= 32
            function require(cond, msg) {
                if iszero(cond) {
                    revertWith(msg)
                }
            }
            // returns the calldata offset of the value and the length in bytes
            // for the RLP encoded data item at `ptr`. used in `decodeFlat`
            function decodeValue(ptr) -> dataLen, valueOffset {
                let b0 := byte(0, calldataload(ptr))

                // 0x00 - 0x7f, single byte
                if lt(b0, 0x80) {
                    // for a single byte whose value is in the [0x00, 0x7f] range,
                    // that byte is its own RLP encoding.
                    dataLen := 1
                    valueOffset := ptr
                    leave
                }

                // 0x80 - 0xb7, short string/bytes, length <= 55
                if lt(b0, 0xb8) {
                    // the RLP encoding consists of a single byte with value 0x80
                    // plus the length of the string followed by the string.
                    dataLen := sub(b0, 0x80)
                    valueOffset := add(ptr, 1)
                    leave
                }

                // 0xb8 - 0xbf, long string/bytes, length > 55
                if lt(b0, 0xc0) {
                    // the RLP encoding consists of a single byte with value 0xb7
                    // plus the length in bytes of the length of the string in binary form,
                    // followed by the length of the string, followed by the string.
                    let lengthBytes := sub(b0, 0xb7)
                    if gt(lengthBytes, 4) {
                        invalid()
                    }

                    // load the extended length
                    valueOffset := add(ptr, 1)
                    let extendedLen := calldataload(valueOffset)
                    let bits := sub(256, mul(lengthBytes, 8))
                    extendedLen := shr(bits, extendedLen)

                    dataLen := extendedLen
                    valueOffset := add(valueOffset, lengthBytes)
                    leave
                }

                revertWith("Not value")
            }

            let ptr := _blockHeaderRLP.offset
            let headerPayloadLength
            {
                let b0 := byte(0, calldataload(ptr))
                // the input should be a long list
                if lt(b0, 0xf8) {
                    invalid()
                }
                let lengthBytes := sub(b0, 0xf7)
                if gt(lengthBytes, 32) {
                    invalid()
                }
                // load the extended length
                ptr := add(ptr, 1)
                headerPayloadLength := calldataload(ptr)
                let bits := sub(256, mul(lengthBytes, 8))
                // compute payload length: extended length + length bytes + 1
                headerPayloadLength := shr(bits, headerPayloadLength)
                headerPayloadLength := add(headerPayloadLength, lengthBytes)
                headerPayloadLength := add(headerPayloadLength, 1)
                ptr := add(ptr, lengthBytes)
            }

            let memPtr := mload(0x40)
            calldatacopy(memPtr, _blockHeaderRLP.offset, headerPayloadLength)
            let _computedBlockHash := keccak256(memPtr, headerPayloadLength)
            require(eq(_blockHash, _computedBlockHash), "Block hash mismatch")

            // load 16 vaules
            for {
                let i := 0
            } lt(i, 16) {
                i := add(i, 1)
            } {
                let len, offset := decodeValue(ptr)
                // the value we care must have at most 32 bytes
                if lt(len, 33) {
                    let bits := mul(sub(32, len), 8)
                    let value := calldataload(offset)
                    value := shr(bits, value)
                    mstore(memPtr, value)
                }
                memPtr := add(memPtr, 0x20)
                ptr := add(len, offset)
            }
            require(eq(ptr, add(_blockHeaderRLP.offset, _blockHeaderRLP.length)), "Header RLP length mismatch")

            memPtr := mload(0x40)
            // load parent hash, 1-st entry
            _parentHash := mload(memPtr)
            // load state root, 4-th entry
            _stateRoot := mload(add(memPtr, 0x60))
            // load block height, 9-th entry
            _height := mload(add(memPtr, 0x100))
            // load block timestamp, 12-th entry
            _timestamp := mload(add(memPtr, 0x160))
            // load base fee, 16-th entry
            _baseFee := mload(add(memPtr, 0x1e0))
        }
        require(stateRoot[_parentHash] != bytes32(0), "Parent not imported");
        BlockMetadata memory _parentMetadata = metadata[_parentHash];
        require(_parentMetadata.height + 1 == _height, "Block height mismatch");
        require(_parentMetadata.timestamp <= _timestamp, "Parent block has larger timestamp");

        latestBlockHash = _blockHash;
        stateRoot[_blockHash] = _stateRoot;
        metadata[_blockHash] = BlockMetadata(_height, _timestamp, _baseFee);

        emit ImportBlock(_blockHash, _height, _timestamp, _baseFee, _stateRoot);

        if (_updateGasPriceOracle) {
            IL1GasPriceOracle(ScrollPredeploy.L1_GAS_PRICE_ORACLE).setL1BaseFee(_baseFee);
        }
    }

    /************************
     * Restricted Functions *
     ************************/

    /// @notice Update whitelist contract.
    /// @dev This function can only called by contract owner.
    /// @param _newWhitelist The address of new whitelist contract.
    function updateWhitelist(address _newWhitelist) external onlyOwner {
        address _oldWhitelist = address(whitelist);

        whitelist = IWhitelist(_newWhitelist);
        emit UpdateWhitelist(_oldWhitelist, _newWhitelist);
    }
}
