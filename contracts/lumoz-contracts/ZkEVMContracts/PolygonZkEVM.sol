// SPDX-License-Identifier: AGPL-3.0
pragma solidity 0.8.17;

import "@openzeppelin/contracts-upgradeable/token/ERC20/utils/SafeERC20Upgradeable.sol";
import "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";

import "./interfaces/IVerifierRollup.sol";
import "./interfaces/IPolygonZkEVMGlobalExitRoot.sol";
import "./interfaces/IPolygonZkEVMBridge.sol";
import "./lib/EmergencyManager.sol";
import "./interfaces/IPolygonZkEVMErrors.sol";
import "../interfaces/ISlotAdapter.sol";
import "../interfaces/IDeposit.sol";

/**
 * Contract responsible for managing the states and the updates of L2 network.
 * There will be a trusted sequencer, which is able to send transactions.
 * Any user can force some transaction and the sequencer will have a timeout to add them in the queue.
 * The sequenced state is deterministic and can be precalculated before it's actually verified by a zkProof.
 * The aggregators will be able to verify the sequenced state with zkProofs and therefore make available the withdrawals from L2 network.
 * To enter and exit of the L2 network will be used a PolygonZkEVMBridge smart contract that will be deployed in both networks.
 */
contract PolygonZkEVM is
OwnableUpgradeable,
EmergencyManager,
IPolygonZkEVMErrors
{
    using SafeERC20Upgradeable for IERC20Upgradeable;

    /**
     * @notice Struct which will be used to call sequenceBatches
     * @param transactions L2 ethereum transactions EIP-155 or pre-EIP-155 with signature:
     * EIP-155: rlp(nonce, gasprice, gasLimit, to, value, data, chainid, 0, 0,) || v || r || s
     * pre-EIP-155: rlp(nonce, gasprice, gasLimit, to, value, data) || v || r || s
     * @param globalExitRoot Global exit root of the batch
     * @param timestamp Sequenced timestamp of the batch
     * @param minForcedTimestamp Minimum timestamp of the force batch data, empty when non forced batch
     */
    struct BatchData {
        bytes transactions;
        bytes32 globalExitRoot;
        uint64 timestamp;
        uint64 minForcedTimestamp;
    }

    /**
     * @notice Struct which will be used to call sequenceForceBatches
     * @param transactions L2 ethereum transactions EIP-155 or pre-EIP-155 with signature:
     * EIP-155: rlp(nonce, gasprice, gasLimit, to, value, data, chainid, 0, 0,) || v || r || s
     * pre-EIP-155: rlp(nonce, gasprice, gasLimit, to, value, data) || v || r || s
     * @param globalExitRoot Global exit root of the batch
     * @param minForcedTimestamp Indicates the minimum sequenced timestamp of the batch
     */
    struct ForcedBatchData {
        bytes transactions;
        bytes32 globalExitRoot;
        uint64 minForcedTimestamp;
    }

    /**
     * @notice Struct which will be stored for every batch sequence
     * @param accInputHash Hash chain that contains all the information to process a batch:
     *  keccak256(bytes32 oldAccInputHash, keccak256(bytes transactions), bytes32 globalExitRoot, uint64 timestamp, address seqAddress)
     * @param sequencedTimestamp Sequenced timestamp
     * @param previousLastBatchSequenced Previous last batch sequenced before the current one, this is used to properly calculate the fees
     */
    struct SequencedBatchData {
        bytes32 accInputHash;
        uint64 sequencedTimestamp;
        uint64 previousLastBatchSequenced;
        uint256 blockNumber;
        bool proofSubmitted;
    }

    struct ProofHashData {
        bytes32 proofHash;
        uint256 blockNumber;
        bool proof;
    }

    /**
     * @notice Struct to store the pending states
     * Pending state will be an intermediary state, that after a timeout can be consolidated, which means that will be added
     * to the state root mapping, and the global exit root will be updated
     * This is a protection mechanism against soundness attacks, that will be turned off in the future
     * @param timestamp Timestamp where the pending state is added to the queue
     * @param lastVerifiedBatch Last batch verified batch of this pending state
     * @param exitRoot Pending exit root
     * @param stateRoot Pending state root
     */
    struct PendingState {
        uint64 timestamp;
        uint64 lastVerifiedBatch;
        bytes32 exitRoot;
        bytes32 stateRoot;
    }

    /**
     * @notice Struct to call initialize, this saves gas because pack the parameters and avoid stack too deep errors.
     * @param admin Admin address
     * @param trustedSequencer Trusted sequencer address
     * @param pendingStateTimeout Pending state timeout
     * @param trustedAggregator Trusted aggregator
     * @param trustedAggregatorTimeout Trusted aggregator timeout
     */
    struct InitializePackedParameters {
        address admin;
        address trustedSequencer;
        uint64 pendingStateTimeout;
        address trustedAggregator;
        uint64 trustedAggregatorTimeout;
    }


    struct ProverLiquidationInfo {
        address prover;
        bool isSubmittedProofHash;
        uint256 submitHashBlockNumber;
        bool isSubmittedProof;
        uint256 submitProofBlockNumber;
        bool isLiquidated;
        uint64 finalNewBatch;
    }


    // An mapping records the record of miners submitting proofhash and proof
    mapping(address => ProverLiquidationInfo[])    public proverLiquidation;
    //The array position of the prover's final liquidation
    mapping(address => uint256) public  proverLastLiquidated;

    mapping(bytes32 => uint256)public  proverPosition;

    // Modulus zkSNARK
    uint256 internal constant _RFIELD =
        21888242871839275222246405745257275088548364400416034343698204186575808495617;

    // Max transactions bytes that can be added in a single batch
    // Max keccaks circuit = (2**23 / 155286) * 44 = 2376
    // Bytes per keccak = 136
    // Minimum Static keccaks batch = 2
    // Max bytes allowed = (2376 - 2) * 136 = 322864 bytes - 1 byte padding
    // Rounded to 300000 bytes
    // In order to process the transaction, the data is approximately hashed twice for ecrecover:
    // 300000 bytes / 2 = 150000 bytes
    // Since geth pool currently only accepts at maximum 128kb transactions:
    // https://github.com/ethereum/go-ethereum/blob/master/core/txpool/txpool.go#L54
    // We will limit this length to be compliant with the geth restrictions since our node will use it
    // We let 8kb as a sanity margin
    uint256 internal constant _MAX_TRANSACTIONS_BYTE_LENGTH = 120000;

    // Max force batch transaction length
    // This is used to avoid huge calldata attacks, where the attacker call force batches from another contract
    uint256 internal constant _MAX_FORCE_BATCH_BYTE_LENGTH = 5000;

    // If a sequenced batch exceeds this timeout without being verified, the contract enters in emergency mode
    uint64 internal constant _HALT_AGGREGATION_TIMEOUT = 1 weeks;

    // Maximum batches that can be verified in one call. It depends on our current metrics
    // This should be a protection against someone that tries to generate huge chunk of invalid batches, and we can't prove otherwise before the pending timeout expires
    uint64 internal constant _MAX_VERIFY_BATCHES = 1000;

    // Max batch multiplier per verification
    uint256 internal constant _MAX_BATCH_MULTIPLIER = 12;

    // Max batch fee value
    uint256 internal constant _MAX_BATCH_FEE = 1000 ether;

    // Min value batch fee
    uint256 internal constant _MIN_BATCH_FEE = 1 gwei;

    // Goldilocks prime field
    uint256 internal constant _GOLDILOCKS_PRIME_FIELD = 0xFFFFFFFF00000001; // 2 ** 64 - 2 ** 32 + 1

    // Max uint64
    uint256 internal constant _MAX_UINT_64 = type(uint64).max; // 0xFFFFFFFFFFFFFFFF

    uint256 public minDeposit;

    uint256 public noProofPunishAmount;

    uint256 public incorrectProofHashPunishAmount;

    ISlotAdapter public slotAdapter;

    IDeposit public ideDeposit;

    // Rollup verifier interface
    IVerifierRollup public immutable rollupVerifier;

    // Global Exit Root interface
    IPolygonZkEVMGlobalExitRoot public immutable globalExitRootManager;

    // PolygonZkEVM Bridge Address
    IPolygonZkEVMBridge public immutable bridgeAddress;

    // L2 chain identifier
    uint64 public immutable chainID;

    // L2 chain identifier
    uint64 public immutable forkID;

    // Time target of the verification of a batch
    // Adaptatly the batchFee will be updated to achieve this target
    uint64 public verifyBatchTimeTarget;

    // Batch fee multiplier with 3 decimals that goes from 1000 - 1023
    uint16 public multiplierBatchFee;

    // Trusted sequencer address
    address public trustedSequencer;


    // Queue of forced batches with their associated data
    // ForceBatchNum --> hashedForcedBatchData
    // hashedForcedBatchData: hash containing the necessary information to force a batch:
    // keccak256(keccak256(bytes transactions), bytes32 globalExitRoot, unint64 minForcedTimestamp)
    mapping(uint64 => bytes32) public forcedBatches;

    // Queue of batches that defines the virtual state
    // SequenceBatchNum --> SequencedBatchData
    mapping(uint64 => SequencedBatchData) public sequencedBatches;

    // Last sequenced timestamp
    uint64 public lastTimestamp;

    // Last batch sent by the sequencers
    uint64 public lastBatchSequenced;

    // Last forced batch included in the sequence
    uint64 public lastForceBatchSequenced;

    // Last forced batch
    uint64 public lastForceBatch;

    // Last batch verified by the aggregators
    uint64 public lastVerifiedBatch;


    // State root mapping
    // BatchNum --> state root
    mapping(uint64 => bytes32) public batchNumToStateRoot;

    // blocknumber --> true
    mapping(uint => bool) public blockCommitBatches;

    // finalNewBatch --> proofHash
    mapping(uint64 => mapping(address => ProofHashData)) public proverCommitProofHash;
    // mapping(uint64 => uint256) public commitBatchBlock;
    mapping(address => uint256) public proofNum;

    // Trusted sequencer URL
    string public trustedSequencerURL;

    // L2 network name
    string public networkName;

    // Pending state mapping
    // pendingStateNumber --> PendingState
    mapping(uint256 => PendingState) public pendingStateTransitions;

    // Last pending state
    uint64 public lastPendingState;

    // Last pending state consolidated
    uint64 public lastPendingStateConsolidated;

    // Once a pending state exceeds this timeout it can be consolidated
    uint64 public pendingStateTimeout;

    // Trusted aggregator timeout, if a sequence is not verified in this time frame,
    // everyone can verify that sequence
    uint64 public trustedAggregatorTimeout;

    // Address that will be able to adjust contract parameters or stop the emergency state
    address public admin;

    // This account will be able to accept the admin role
    address public pendingAdmin;

    // Force batch timeout
    uint64 public forceBatchTimeout;

    // Indicates if forced batches are disallowed
    bool public isForcedBatchDisallowed;

    uint8 public proofHashCommitEpoch;

    // here presents time to submit proof since the first proof hash, would be proofHashCommitEpoch + set proofEpoch
    /*         {--proofHashCommitEpoch---}{--proofCommitEpoch--}
    *          |-------------------------|---------------------|
    *        proofHash              proof start         proof end   */
    uint8 public proofCommitEpoch;

    /**
     * @dev Emitted when the trusted sequencer sends a new batch of transactions
     */
    event SequenceBatches(uint64 indexed numBatch);

    /**
     * @dev Emitted when a batch is forced
     */
    event ForceBatch(
        uint64 indexed forceBatchNum,
        bytes32 lastGlobalExitRoot,
        address sequencer,
        bytes transactions
    );

    /**
     * @dev Emitted when forced batches are sequenced by not the trusted sequencer
     */
    event SequenceForceBatches(uint64 indexed numBatch);

    /**
     * @dev Emitted when a aggregator verifies batches
     */
    event VerifyBatches(
        uint64 indexed numBatch,
        bytes32 stateRoot,
        address indexed aggregator
    );

    /**
     * @dev Emitted when the trusted aggregator verifies batches
     */
    event VerifyBatchesTrustedAggregator(
        uint64 indexed numBatch,
        bytes32 stateRoot,
        address indexed aggregator
    );

    /**
     * @dev Emitted when pending state is consolidated
     */
    event ConsolidatePendingState(
        uint64 indexed numBatch,
        bytes32 stateRoot,
        uint64 indexed pendingStateNum
    );

    /**
     * @dev Emitted when the admin updates the trusted sequencer address
     */
    event SetTrustedSequencer(address newTrustedSequencer);

    /**
     * @dev Emitted when the admin updates the sequencer URL
     */
    event SetTrustedSequencerURL(string newTrustedSequencerURL);

    /**
     * @dev Emitted when the admin updates the trusted aggregator timeout
     */
    event SetTrustedAggregatorTimeout(uint64 newTrustedAggregatorTimeout);

    /**
     * @dev Emitted when the admin updates the pending state timeout
     */
    event SetPendingStateTimeout(uint64 newPendingStateTimeout);

    /**
     * @dev Emitted when the admin updates the trusted aggregator address
     */
    event SetTrustedAggregator(address newTrustedAggregator);

    /**
     * @dev Emitted when the admin updates the multiplier batch fee
     */
    event SetMultiplierBatchFee(uint16 newMultiplierBatchFee);

    /**
     * @dev Emitted when the admin updates the verify batch timeout
     */
    event SetVerifyBatchTimeTarget(uint64 newVerifyBatchTimeTarget);

    /**
     * @dev Emitted when the admin update the force batch timeout
     */
    event SetForceBatchTimeout(uint64 newforceBatchTimeout);

    /**
     * @dev Emitted when activate force batches
     */
    event ActivateForceBatches();

    /**
     * @dev Emitted when the admin starts the two-step transfer role setting a new pending admin
     */
    event TransferAdminRole(address newPendingAdmin);

    /**
     * @dev Emitted when the pending admin accepts the admin role
     */
    event AcceptAdminRole(address newAdmin);

    /**
     * @dev Emitted when is proved a different state given the same batches
     */
    event ProveNonDeterministicPendingState(
        bytes32 storedStateRoot,
        bytes32 provedStateRoot
    );

    /**
     * @dev Emitted when the trusted aggregator overrides pending state
     */
    event OverridePendingState(
        uint64 indexed numBatch,
        bytes32 stateRoot,
        address indexed aggregator
    );

    /**
     * @dev Emitted everytime the forkID is updated, this includes the first initialization of the contract
     * This event is intended to be emitted for every upgrade of the contract with relevant changes for the nodes
     */
    event UpdateZkEVMVersion(uint64 numBatch, uint64 forkID, string version);

    event SubmitProofHash(address _prover, uint64 initNumBatch, uint64 finalNewBatch, bytes32 _proofHash);

    event SetProofHashCommitEpoch(uint8 newProofHashCommitEpoch);

    event SetProofCommitEpoch(uint8 newProofCommitEpoch);

    /**
     * @param _globalExitRootManager Global exit root manager address
     * @param _rollupVerifier Rollup verifier address
     * @param _bridgeAddress Bridge address
     * @param _chainID L2 chainID
     * @param _forkID Fork Id
     */
    constructor(
        IPolygonZkEVMGlobalExitRoot _globalExitRootManager,
        IVerifierRollup _rollupVerifier,
        IPolygonZkEVMBridge _bridgeAddress,
        uint64 _chainID,
        uint64 _forkID
    ) {
        globalExitRootManager = _globalExitRootManager;
        rollupVerifier = _rollupVerifier;
        bridgeAddress = _bridgeAddress;
        chainID = _chainID;
        forkID = _forkID;
    }

    /**
     * @param initializePackedParameters Struct to save gas and avoid stack too deep errors
     * @param genesisRoot Rollup genesis root
     * @param _trustedSequencerURL Trusted sequencer URL
     * @param _networkName L2 network name
     */
    function initialize(
        InitializePackedParameters calldata initializePackedParameters,
        bytes32 genesisRoot,
        string memory _trustedSequencerURL,
        string memory _networkName,
        string calldata _version
    ) external initializer {
        admin = initializePackedParameters.admin;
        trustedSequencer = initializePackedParameters.trustedSequencer;
        batchNumToStateRoot[0] = genesisRoot;
        trustedSequencerURL = _trustedSequencerURL;
        networkName = _networkName;

        // Check initialize parameters
        if (
            initializePackedParameters.pendingStateTimeout >
            _HALT_AGGREGATION_TIMEOUT
        ) {
            revert PendingStateTimeoutExceedHaltAggregationTimeout();
        }
        pendingStateTimeout = initializePackedParameters.pendingStateTimeout;

        if (
            initializePackedParameters.trustedAggregatorTimeout >
            _HALT_AGGREGATION_TIMEOUT
        ) {
            revert TrustedAggregatorTimeoutExceedHaltAggregationTimeout();
        }

        trustedAggregatorTimeout = initializePackedParameters
            .trustedAggregatorTimeout;

        // Constant deployment variables
        verifyBatchTimeTarget = 30 minutes;
        multiplierBatchFee = 1002;
        forceBatchTimeout = 5 days;
        isForcedBatchDisallowed = true;
        minDeposit = 50000 ether;
        noProofPunishAmount = 1000 ether;
        incorrectProofHashPunishAmount = 10000 ether;
        proofHashCommitEpoch = 20;
        proofCommitEpoch = 32;
        // Initialize OZ contracts
        __Ownable_init_unchained();

        // emit version event
        emit UpdateZkEVMVersion(0, forkID, _version);
    }

    modifier onlyAdmin() {
        if (admin != msg.sender) {
            revert OnlyAdmin();
        }
        _;
    }

    modifier onlyTrustedSequencer() {
        if (trustedSequencer != msg.sender) {
            revert OnlyTrustedSequencer();
        }
        _;
    }

    modifier isForceBatchAllowed() {
        if (isForcedBatchDisallowed) {
            revert ForceBatchNotAllowed();
        }
        _;
    }

    modifier isSlotAdapterEmpty() {
        if (address(slotAdapter) == address(0)) {
            revert SlotAdapterEmpty();
        }
        _;
    }

    modifier isCommitProofHash(uint64 finalNewBatch) {
        ProofHashData memory proofHashData = proverCommitProofHash[finalNewBatch][msg.sender];
        if (lastVerifiedBatch >= finalNewBatch || proofHashData.proof) {
            revert CommittedProof();
        }
        if (proofHashData.proofHash != bytes32(0) && (proofHashData.blockNumber + proofHashCommitEpoch + proofCommitEpoch) > block.number) {
            revert CommittedProofHash();
        }

        _;
    }

    modifier commitProof(uint64 finalNewBatch) {
        SequencedBatchData memory sequencedBatch = sequencedBatches[finalNewBatch];
        if (sequencedBatch.blockNumber + proofHashCommitEpoch > block.number) {
            revert SubmitProofEarly();
        }

        ProofHashData memory _proofHashData = proverCommitProofHash[finalNewBatch][msg.sender];
        if (_proofHashData.blockNumber != sequencedBatch.blockNumber) {
            revert ErrCommitProof();
        }

        if (!sequencedBatch.proofSubmitted && (_proofHashData.blockNumber + proofHashCommitEpoch + proofCommitEpoch) < block.number) {
            revert SubmitProofTooLate();
        }

        if (_proofHashData.proofHash == bytes32(0)) {
            revert CommittedProofHash();
        }

        if (_proofHashData.proof == true) {
            revert CommittedProof();
        }

        _;
    }

    modifier isZeroAddress(address account) {
        if ( account == address(0) ) {
            revert ZeroAddress();
        }
        _;
    }

    modifier onlyDeposit() {
        if (address(ideDeposit) != msg.sender) {
            revert OnlyDeposit();
        }
        _;
    }

    /////////////////////////////////////
    // Sequence/Verify batches functions
    ////////////////////////////////////

    /**
     * @notice Allows a sequencer to send multiple batches
     * @param batches Struct array which holds the necessary data to append new batches to the sequence
     * @param l2Coinbase Address that will receive the fees from L2
     */
    function sequenceBatches(
        BatchData[] calldata batches,
        address l2Coinbase
    ) external ifNotEmergencyState onlyTrustedSequencer {
        if (blockCommitBatches[block.number]) {
            revert CommittedBatches();
        }

        uint256 batchesNum = batches.length;
        if (batchesNum == 0) {
            revert SequenceZeroBatches();
        }

        if (batchesNum > _MAX_VERIFY_BATCHES) {
            revert ExceedMaxVerifyBatches();
        }


        // Store storage variables in memory, to save gas, because will be overrided multiple times
        uint64 currentTimestamp = lastTimestamp;
        uint64 currentBatchSequenced = lastBatchSequenced;
        uint64 currentLastForceBatchSequenced = lastForceBatchSequenced;
        SequencedBatchData memory sequencedBatchData = sequencedBatches[currentBatchSequenced];
        bytes32 currentAccInputHash = sequencedBatchData.accInputHash;
        // bytes32 currentAccInputHash = sequencedBatches[currentBatchSequenced]
        //     .accInputHash;

        // Store in a temporal variable, for avoid access again the storage slot
        uint64 initLastForceBatchSequenced = currentLastForceBatchSequenced;

        for (uint256 i = 0; i < batchesNum; i++) {
            // Load current sequence
            BatchData memory currentBatch = batches[i];

            // Store the current transactions hash since can be used more than once for gas saving
            bytes32 currentTransactionsHash = keccak256(
                currentBatch.transactions
            );

            // Check if it's a forced batch
            if (currentBatch.minForcedTimestamp > 0) {
                currentLastForceBatchSequenced++;

                // Check forced data matches
                bytes32 hashedForcedBatchData = keccak256(
                    abi.encodePacked(
                        currentTransactionsHash,
                        currentBatch.globalExitRoot,
                        currentBatch.minForcedTimestamp
                    )
                );

                if (
                    hashedForcedBatchData !=
                    forcedBatches[currentLastForceBatchSequenced]
                ) {
                    revert ForcedDataDoesNotMatch();
                }

                // Delete forceBatch data since won't be used anymore
                delete forcedBatches[currentLastForceBatchSequenced];

                // Check timestamp is bigger than min timestamp
                if (currentBatch.timestamp < currentBatch.minForcedTimestamp) {
                    revert SequencedTimestampBelowForcedTimestamp();
                }
            } else {
                // Check global exit root exists with proper batch length. These checks are already done in the forceBatches call
                // Note that the sequencer can skip setting a global exit root putting zeros
                if (
                    currentBatch.globalExitRoot != bytes32(0) &&
                    globalExitRootManager.globalExitRootMap(
                        currentBatch.globalExitRoot
                    ) ==
                    0
                ) {
                    revert GlobalExitRootNotExist();
                }

                if (
                    currentBatch.transactions.length >
                    _MAX_TRANSACTIONS_BYTE_LENGTH
                ) {
                    revert TransactionsLengthAboveMax();
                }
            }

            // Check Batch timestamps are correct
            if (
                currentBatch.timestamp < currentTimestamp ||
                currentBatch.timestamp > block.timestamp
            ) {
                revert SequencedTimestampInvalid();
            }

            // Calculate next accumulated input hash
            currentAccInputHash = keccak256(
                abi.encodePacked(
                    currentAccInputHash,
                    currentTransactionsHash,
                    currentBatch.globalExitRoot,
                    currentBatch.timestamp,
                    l2Coinbase
                )
            );

            // Update timestamp
            currentTimestamp = currentBatch.timestamp;
        }
        // Update currentBatchSequenced
        currentBatchSequenced += uint64(batchesNum);

        // Sanity check, should be unreachable
        if (currentLastForceBatchSequenced > lastForceBatch) {
            revert ForceBatchesOverflow();
        }

        // Update sequencedBatches mapping
        sequencedBatches[currentBatchSequenced] = SequencedBatchData({
            accInputHash: currentAccInputHash,
            sequencedTimestamp: uint64(block.timestamp),
            previousLastBatchSequenced: lastBatchSequenced,
            blockNumber: 0,
            proofSubmitted: false
        });

        // Store back the storage variables
        lastTimestamp = currentTimestamp;
        lastBatchSequenced = currentBatchSequenced;

        if (currentLastForceBatchSequenced != initLastForceBatchSequenced)
            lastForceBatchSequenced = currentLastForceBatchSequenced;


        // Update global exit root if there are new deposits
        bridgeAddress.updateGlobalExitRoot();

        blockCommitBatches[block.number] = true;
        // calc slot reward
        slotAdapter.calcSlotReward(currentBatchSequenced, ideDeposit);

        emit SequenceBatches(currentBatchSequenced);
    }


    function submitProofHash(uint64 initNumBatch, uint64 finalNewBatch, bytes32 _proofHash) external ifNotEmergencyState isCommitProofHash(finalNewBatch) {
        if (ideDeposit.depositOf(msg.sender) < minDeposit) {
            revert InsufficientPledge();
        }

        if (initNumBatch != sequencedBatches[finalNewBatch].previousLastBatchSequenced) {
            revert ErrSequencerRange(initNumBatch, sequencedBatches[finalNewBatch].previousLastBatchSequenced);
        }

        if (initNumBatch != 0 && sequencedBatches[initNumBatch].accInputHash == bytes32(0)) {
            revert OldAccInputHashDoesNotExist();
        }

        if (sequencedBatches[finalNewBatch].accInputHash == bytes32(0)) {
            revert NewAccInputHashDoesNotExist();
        }

        uint256 _finalNewBatchNumber = sequencedBatches[finalNewBatch].blockNumber;
        if ( _finalNewBatchNumber > 0 && (block.number - _finalNewBatchNumber) > (proofHashCommitEpoch + proofCommitEpoch)) {
            if (!sequencedBatches[finalNewBatch].proofSubmitted) {
                sequencedBatches[finalNewBatch].blockNumber = 0;
                slotAdapter.calcCurrentTotalDeposit(finalNewBatch, ideDeposit, msg.sender, true);
            }
        }

        uint256 number = sequencedBatches[finalNewBatch].blockNumber;
        if (number > 0 && (block.number - number) > proofHashCommitEpoch) {
            revert CommittedTimeout();
        }

        if (number == 0) {
            sequencedBatches[finalNewBatch].blockNumber = block.number;
        }

        // store hash finalNewBatch -> msg.sender -> ProofHashData
        proverCommitProofHash[finalNewBatch][msg.sender] = ProofHashData(
            _proofHash,
            sequencedBatches[finalNewBatch].blockNumber,
            false
        );
        slotAdapter.calcCurrentTotalDeposit(finalNewBatch, ideDeposit, msg.sender, false);
        updateProofHashLiquidation(_proofHash, finalNewBatch);
        emit SubmitProofHash(msg.sender, initNumBatch, finalNewBatch, _proofHash);
    }


    /**
     * @notice Allows an aggregator to verify multiple batches
     * @param initNumBatch Batch which the aggregator starts the verification
     * @param finalNewBatch Last batch aggregator intends to verify
     * @param newLocalExitRoot  New local exit root once the batch is processed
     * @param newStateRoot New State root once the batch is processed
     * @param proof fflonk proof
     */
    function verifyBatches(
        uint64 initNumBatch,
        uint64 finalNewBatch,
        bytes32 newLocalExitRoot,
        bytes32 newStateRoot,
        bytes calldata proof
    ) external ifNotEmergencyState {
        if (finalNewBatch - initNumBatch > _MAX_VERIFY_BATCHES) {
            revert ExceedMaxVerifyBatches();
        }
        bytes32 proofHash = _verifyAndRewardBatches(initNumBatch, finalNewBatch, newLocalExitRoot, newStateRoot, proof);
        if (proofHash != bytes32(0)) {
            proofNum[msg.sender]++;
            slotAdapter.distributeRewards(msg.sender, initNumBatch, finalNewBatch, ideDeposit);
            updateProofLiquidation(proofHash, false);

            if (!sequencedBatches[finalNewBatch].proofSubmitted) {
                // Consolidate state
                lastVerifiedBatch = finalNewBatch;
                batchNumToStateRoot[finalNewBatch] = newStateRoot;
                sequencedBatches[finalNewBatch].proofSubmitted = true;

                // Interact with globalExitRootManager
                globalExitRootManager.updateExitRoot(newLocalExitRoot);
                emit VerifyBatchesTrustedAggregator(finalNewBatch, newStateRoot, msg.sender);
            }

            proverCommitProofHash[finalNewBatch][msg.sender].proof = true;
        }
    }

    /**
     * @notice Verify and reward batches internal function
     * @param initNumBatch Batch which the aggregator starts the verification
     * @param finalNewBatch Last batch aggregator intends to verify
     * @param newLocalExitRoot  New local exit root once the batch is processed
     * @param newStateRoot New State root once the batch is processed
     * @param proof fflonk proof
     */
    function _verifyAndRewardBatches(
        uint64 initNumBatch,
        uint64 finalNewBatch,
        bytes32 newLocalExitRoot,
        bytes32 newStateRoot,
        bytes calldata proof
    ) internal isSlotAdapterEmpty commitProof(finalNewBatch) returns (bytes32) {
        bytes32 oldStateRoot;
        uint64 currentLastVerifiedBatch = getLastVerifiedBatch();

        // Use consolidated state
        oldStateRoot = batchNumToStateRoot[initNumBatch];

        if (oldStateRoot == bytes32(0)) {
            revert OldStateRootDoesNotExist();
        }

        // Check initNumBatch is inside the range, sanity check
        if (initNumBatch > currentLastVerifiedBatch) {
            revert InitNumBatchAboveLastVerifiedBatch();
        }

        // check proof hash
        bytes32 proofHash = keccak256(abi.encodePacked(keccak256(proof), msg.sender));
        if (proverCommitProofHash[finalNewBatch][msg.sender].proofHash != proofHash) {
            slotAdapter.punish(msg.sender, ideDeposit, incorrectProofHashPunishAmount);
            updateProofLiquidation(proverCommitProofHash[finalNewBatch][msg.sender].proofHash, true);
            return bytes32(0);
        }

        // Get snark bytes
        bytes memory snarkHashBytes = getInputSnarkBytes(
            initNumBatch,
            finalNewBatch,
            newLocalExitRoot,
            oldStateRoot,
            newStateRoot
        );

        // Calculate the snark input
        uint256 inputSnark = uint256(sha256(snarkHashBytes)) % _RFIELD;
        // Verify proof
        if (!rollupVerifier.verifyProof(proof, [inputSnark])) {
            slotAdapter.punish(msg.sender, ideDeposit, incorrectProofHashPunishAmount);
            updateProofLiquidation(proofHash, true);
            return bytes32(0);
        }

        return proofHash;
    }

    ////////////////////////////
    // Force batches functions
    ////////////////////////////

    /**
     * @notice Allows a sequencer/user to force a batch of L2 transactions.
     * This should be used only in extreme cases where the trusted sequencer does not work as expected
     * Note The sequencer has certain degree of control on how non-forced and forced batches are ordered
     * In order to assure that users force transactions will be processed properly, user must not sign any other transaction
     * with the same nonce
     * @param transactions L2 ethereum transactions EIP-155 or pre-EIP-155 with signature:
     * @param maticAmount Max amount of MATIC tokens that the sender is willing to pay
     */
     // todo
    function forceBatch(
        bytes calldata transactions,
        uint256 maticAmount
    ) public isForceBatchAllowed ifNotEmergencyState {

        if (transactions.length > _MAX_FORCE_BATCH_BYTE_LENGTH) {
            revert TransactionsLengthAboveMax();
        }

        // Get globalExitRoot global exit root
        bytes32 lastGlobalExitRoot = globalExitRootManager
            .getLastGlobalExitRoot();

        // Update forcedBatches mapping
        lastForceBatch++;

        forcedBatches[lastForceBatch] = keccak256(
            abi.encodePacked(
                keccak256(transactions),
                lastGlobalExitRoot,
                uint64(block.timestamp)
            )
        );

        // blockCommitBatches[block.number] = true;
        // // calc slot reward
        // slotAdapter.calcSlotReward(currentBatchSequenced);

        if (msg.sender == tx.origin) {
            // Getting the calldata from an EOA is easy so no need to put the `transactions` in the event
            emit ForceBatch(lastForceBatch, lastGlobalExitRoot, msg.sender, "");
        } else {
            // Getting internal transaction calldata is complicated (because it requires an archive node)
            // Therefore it's worth it to put the `transactions` in the event, which is easy to query
            emit ForceBatch(
                lastForceBatch,
                lastGlobalExitRoot,
                msg.sender,
                transactions
            );
        }
    }

    /**
     * @notice Allows anyone to sequence forced Batches if the trusted sequencer has not done so in the timeout period
     * @param batches Struct array which holds the necessary data to append force batches
     */
    function sequenceForceBatches(
        ForcedBatchData[] calldata batches
    ) external isForceBatchAllowed ifNotEmergencyState {
        uint256 batchesNum = batches.length;

        if (batchesNum == 0) {
            revert SequenceZeroBatches();
        }

        if (batchesNum > _MAX_VERIFY_BATCHES) {
            revert ExceedMaxVerifyBatches();
        }

        if (
            uint256(lastForceBatchSequenced) + batchesNum >
            uint256(lastForceBatch)
        ) {
            revert ForceBatchesOverflow();
        }

        // Store storage variables in memory, to save gas, because will be overrided multiple times
        uint64 currentBatchSequenced = lastBatchSequenced;
        uint64 currentLastForceBatchSequenced = lastForceBatchSequenced;
        bytes32 currentAccInputHash = sequencedBatches[currentBatchSequenced]
            .accInputHash;

        // Sequence force batches
        for (uint256 i = 0; i < batchesNum; i++) {
            // Load current sequence
            ForcedBatchData memory currentBatch = batches[i];
            currentLastForceBatchSequenced++;

            // Store the current transactions hash since it's used more than once for gas saving
            bytes32 currentTransactionsHash = keccak256(
                currentBatch.transactions
            );

            // Check forced data matches
            bytes32 hashedForcedBatchData = keccak256(
                abi.encodePacked(
                    currentTransactionsHash,
                    currentBatch.globalExitRoot,
                    currentBatch.minForcedTimestamp
                )
            );

            if (
                hashedForcedBatchData !=
                forcedBatches[currentLastForceBatchSequenced]
            ) {
                revert ForcedDataDoesNotMatch();
            }

            // Delete forceBatch data since won't be used anymore
            delete forcedBatches[currentLastForceBatchSequenced];

            if (i == (batchesNum - 1)) {
                // The last batch will have the most restrictive timestamp
                if (
                    currentBatch.minForcedTimestamp + forceBatchTimeout >
                    block.timestamp
                ) {
                    revert ForceBatchTimeoutNotExpired();
                }
            }
            // Calculate next acc input hash
            currentAccInputHash = keccak256(
                abi.encodePacked(
                    currentAccInputHash,
                    currentTransactionsHash,
                    currentBatch.globalExitRoot,
                    uint64(block.timestamp),
                    msg.sender
                )
            );
        }
        // Update currentBatchSequenced
        currentBatchSequenced += uint64(batchesNum);

        lastTimestamp = uint64(block.timestamp);

        // Store back the storage variables
        sequencedBatches[currentBatchSequenced] = SequencedBatchData({
            accInputHash: currentAccInputHash,
            sequencedTimestamp: uint64(block.timestamp),
            previousLastBatchSequenced: lastBatchSequenced,
            blockNumber: 0,
            proofSubmitted: false
        });
        lastBatchSequenced = currentBatchSequenced;
        lastForceBatchSequenced = currentLastForceBatchSequenced;

        emit SequenceForceBatches(currentBatchSequenced);
    }

    //////////////////
    // admin functions
    //////////////////

    function setSlotAdapter(ISlotAdapter _slotAdapter) public onlyAdmin isZeroAddress(address(_slotAdapter)) {
        // require(address(_slotAdapter) != address(0), "set 0 address");
        slotAdapter = _slotAdapter;
    }

    function setDeposit(IDeposit _ideDeposit) public onlyAdmin isZeroAddress(address(_ideDeposit)) {
        // require(address(_ideDeposit) != address(0), "Set 0 address");
        ideDeposit = _ideDeposit;
    }

    function setProofHashCommitEpoch(uint8 _newCommitEpoch) external onlyAdmin {
        proofHashCommitEpoch = _newCommitEpoch;
        emit SetProofHashCommitEpoch(_newCommitEpoch);
    }

    function setProofCommitEpoch(uint8 _newCommitEpoch) external onlyAdmin {
        proofCommitEpoch = _newCommitEpoch;
        emit SetProofCommitEpoch(_newCommitEpoch);
    }

    function setMinDeposit(uint256 _amount) external onlyAdmin {
        minDeposit = _amount;
    }

    function setNoProofPunishAmount(uint256 _amount) external onlyAdmin {
        noProofPunishAmount = _amount;
    }

    function setIncorrectProofPunishAmount(uint256 _amount) external onlyAdmin {
        incorrectProofHashPunishAmount = _amount;
    }


    /**
     * @notice Allow the admin to set a new trusted sequencer
     * @param newTrustedSequencer Address of the new trusted sequencer
     */
    function setTrustedSequencer(
        address newTrustedSequencer
    ) external onlyAdmin {
        trustedSequencer = newTrustedSequencer;

        emit SetTrustedSequencer(newTrustedSequencer);
    }

    /**
     * @notice Allow the admin to set the trusted sequencer URL
     * @param newTrustedSequencerURL URL of trusted sequencer
     */
    function setTrustedSequencerURL(
        string memory newTrustedSequencerURL
    ) external onlyAdmin {
        trustedSequencerURL = newTrustedSequencerURL;

        emit SetTrustedSequencerURL(newTrustedSequencerURL);
    }

    /**
     * @notice Allow the admin to set a new multiplier batch fee
     * @param newMultiplierBatchFee multiplier batch fee
     */
    function setMultiplierBatchFee(
        uint16 newMultiplierBatchFee
    ) external onlyAdmin {
        if (newMultiplierBatchFee < 1000 || newMultiplierBatchFee > 1023) {
            revert InvalidRangeMultiplierBatchFee();
        }

        multiplierBatchFee = newMultiplierBatchFee;
        emit SetMultiplierBatchFee(newMultiplierBatchFee);
    }

    /**
     * @notice Allow the admin to set a new verify batch time target
     * This value will only be relevant once the aggregation is decentralized, so
     * the trustedAggregatorTimeout should be zero or very close to zero
     * @param newVerifyBatchTimeTarget Verify batch time target
     */
    function setVerifyBatchTimeTarget(
        uint64 newVerifyBatchTimeTarget
    ) external onlyAdmin {
        if (newVerifyBatchTimeTarget > 1 days) {
            revert InvalidRangeBatchTimeTarget();
        }
        verifyBatchTimeTarget = newVerifyBatchTimeTarget;
        emit SetVerifyBatchTimeTarget(newVerifyBatchTimeTarget);
    }

    /**
     * @notice Allow the admin to set the forcedBatchTimeout
     * The new value can only be lower, except if emergency state is active
     * @param newforceBatchTimeout New force batch timeout
     */
    function setForceBatchTimeout(
        uint64 newforceBatchTimeout
    ) external onlyAdmin {
        if (newforceBatchTimeout > _HALT_AGGREGATION_TIMEOUT) {
            revert InvalidRangeForceBatchTimeout();
        }

        if (!isEmergencyState) {
            if (newforceBatchTimeout >= forceBatchTimeout) {
                revert InvalidRangeForceBatchTimeout();
            }
        }

        forceBatchTimeout = newforceBatchTimeout;
        emit SetForceBatchTimeout(newforceBatchTimeout);
    }

    /**
     * @notice Allow the admin to turn on the force batches
     * This action is not reversible
     */
    function activateForceBatches() external onlyAdmin {
        if (!isForcedBatchDisallowed) {
            revert ForceBatchesAlreadyActive();
        }
        isForcedBatchDisallowed = false;
        emit ActivateForceBatches();
    }

    /**
     * @notice Starts the admin role transfer
     * This is a two step process, the pending admin must accepted to finalize the process
     * @param newPendingAdmin Address of the new pending admin
     */
    function transferAdminRole(address newPendingAdmin) external onlyAdmin {
        pendingAdmin = newPendingAdmin;
        emit TransferAdminRole(newPendingAdmin);
    }

    /**
     * @notice Allow the current pending admin to accept the admin role
     */
    function acceptAdminRole() external {
        if (pendingAdmin != msg.sender) {
            revert OnlyPendingAdmin();
        }

        admin = pendingAdmin;
        emit AcceptAdminRole(pendingAdmin);
    }

    /////////////////////////////////
    // Soundness protection functions
    /////////////////////////////////

    /**
     * @notice Function to activate emergency state, which also enables the emergency mode on both PolygonZkEVM and PolygonZkEVMBridge contracts
     * If not called by the owner must be provided a batcnNum that does not have been aggregated in a _HALT_AGGREGATION_TIMEOUT period
     * @param sequencedBatchNum Sequenced batch number that has not been aggreagated in _HALT_AGGREGATION_TIMEOUT
     */
    function activateEmergencyState(uint64 sequencedBatchNum) external {
        if (msg.sender != owner()) {
            // Only check conditions if is not called by the owner
            uint64 currentLastVerifiedBatch = getLastVerifiedBatch();

            // Check that the batch has not been verified
            if (sequencedBatchNum <= currentLastVerifiedBatch) {
                revert BatchAlreadyVerified();
            }

            // Check that the batch has been sequenced and this was the end of a sequence
            if (
                sequencedBatchNum > lastBatchSequenced ||
                sequencedBatches[sequencedBatchNum].sequencedTimestamp == 0
            ) {
                revert BatchNotSequencedOrNotSequenceEnd();
            }

            // Check that has been passed _HALT_AGGREGATION_TIMEOUT since it was sequenced
            if (
                sequencedBatches[sequencedBatchNum].sequencedTimestamp +
                    _HALT_AGGREGATION_TIMEOUT >
                block.timestamp
            ) {
                revert HaltTimeoutNotExpired();
            }
        }
        _activateEmergencyState();
    }

    /**
     * @notice Function to deactivate emergency state on both PolygonZkEVM and PolygonZkEVMBridge contracts
     */
    function deactivateEmergencyState() external onlyAdmin {
        // Deactivate emergency state on PolygonZkEVMBridge
        bridgeAddress.deactivateEmergencyState();

        // Deactivate emergency state on this contract
        super._deactivateEmergencyState();
    }

    /**
     * @notice Internal function to activate emergency state on both PolygonZkEVM and PolygonZkEVMBridge contracts
     */
    function _activateEmergencyState() internal override {
        // Activate emergency state on PolygonZkEVM Bridge
        bridgeAddress.activateEmergencyState();

        // Activate emergency state on this contract
        super._activateEmergencyState();
    }

    ////////////////////////
    // public/view functions
    ////////////////////////

    /**
     * @notice Get the last verified batch
     */
    function getLastVerifiedBatch() public view returns (uint64) {
        if (lastPendingState > 0) {
            return pendingStateTransitions[lastPendingState].lastVerifiedBatch;
        } else {
            return lastVerifiedBatch;
        }
    }

    /**
     * @notice Function to calculate the input snark bytes
     * @param initNumBatch Batch which the aggregator starts the verification
     * @param finalNewBatch Last batch aggregator intends to verify
     * @param newLocalExitRoot New local exit root once the batch is processed
     * @param oldStateRoot State root before batch is processed
     * @param newStateRoot New State root once the batch is processed
     */
    function getInputSnarkBytes(
        uint64 initNumBatch,
        uint64 finalNewBatch,
        bytes32 newLocalExitRoot,
        bytes32 oldStateRoot,
        bytes32 newStateRoot
    ) public view returns (bytes memory) {
        // sanity checks
        bytes32 oldAccInputHash = sequencedBatches[initNumBatch].accInputHash;
        bytes32 newAccInputHash = sequencedBatches[finalNewBatch].accInputHash;

        if (initNumBatch != 0 && oldAccInputHash == bytes32(0)) {
            revert OldAccInputHashDoesNotExist();
        }

        if (newAccInputHash == bytes32(0)) {
            revert NewAccInputHashDoesNotExist();
        }

        // Check that new state root is inside goldilocks field
        if (!checkStateRootInsidePrime(uint256(newStateRoot))) {
            revert NewStateRootNotInsidePrime();
        }

        return
            abi.encodePacked(
                msg.sender,
                oldStateRoot,
                oldAccInputHash,
                initNumBatch,
                chainID,
                forkID,
                newStateRoot,
                newAccInputHash,
                newLocalExitRoot,
                finalNewBatch
            );
    }

    function checkStateRootInsidePrime(
        uint256 newStateRoot
    ) public pure returns (bool) {
        if (
            ((newStateRoot & _MAX_UINT_64) < _GOLDILOCKS_PRIME_FIELD) &&
            (((newStateRoot >> 64) & _MAX_UINT_64) < _GOLDILOCKS_PRIME_FIELD) &&
            (((newStateRoot >> 128) & _MAX_UINT_64) <
                _GOLDILOCKS_PRIME_FIELD) &&
            ((newStateRoot >> 192) < _GOLDILOCKS_PRIME_FIELD)
        ) {
            return true;
        } else {
            return false;
        }
    }


    function updateProofHashLiquidation( bytes32 _proofHash, uint64 finalNewBatch) internal {
        // uint256 position = proverPosition[_proofHash];
        ProverLiquidationInfo[] storage proverLiquidations = proverLiquidation[msg.sender];
        proverLiquidations.push(ProverLiquidationInfo({
            prover: msg.sender,
            isSubmittedProofHash: true,
            submitHashBlockNumber: block.number,
            isSubmittedProof: false,
            submitProofBlockNumber: block.number,
            isLiquidated: false,
            finalNewBatch: finalNewBatch
        }));
        proverPosition[_proofHash] = proverLiquidations.length - 1;
        updateLiquidation(msg.sender);
    }


    function updateProofLiquidation(bytes32 _proofHash, bool _punished) internal {
        uint256 position = proverPosition[_proofHash];
        ProverLiquidationInfo[] storage proverLiquidations = proverLiquidation[msg.sender];
        ProverLiquidationInfo storage proverLiquidationInfo = proverLiquidations[position];
        proverLiquidationInfo.submitProofBlockNumber = block.number;
        proverLiquidationInfo.isSubmittedProof = true;
        if (_punished) {
            proverLiquidationInfo.isLiquidated = true;
        }
        updateLiquidation(msg.sender);
    }

    function updateLiquidation(address _account) internal {
        uint256 proverLastLiquidatedPosition = proverLastLiquidated[_account];
        ProverLiquidationInfo[] storage proverLiquidations = proverLiquidation[_account];
        for (uint256 i = proverLastLiquidatedPosition; i < proverLiquidations.length; i++) {
            ProverLiquidationInfo storage proverLiquidationInfo = proverLiquidations[i];
            if (!proverLiquidationInfo.isLiquidated) {
                if (proverLiquidationInfo.isSubmittedProof) {
                    if (!sequencedBatches[proverLiquidationInfo.finalNewBatch].proofSubmitted) {
                        if ((proverLiquidationInfo.submitProofBlockNumber - proverLiquidationInfo.submitHashBlockNumber) > (proofHashCommitEpoch + proofCommitEpoch)) {
                            proverLiquidationInfo.isLiquidated = true;
                            proverLastLiquidated[_account]++;
                            slotAdapter.punish(_account, ideDeposit, noProofPunishAmount);
                        }
                    } else {
                        proverLiquidationInfo.isLiquidated = true;
                        proverLastLiquidated[_account]++;
                    }
                } else {
                    if ((block.number - proverLiquidationInfo.submitHashBlockNumber) > (proofHashCommitEpoch + proofCommitEpoch)) {
                        proverLiquidationInfo.isLiquidated = true;
                        proverLastLiquidated[_account]++;
                        slotAdapter.punish(_account, ideDeposit, noProofPunishAmount);
                    }
                }

            } else {
                proverLastLiquidated[_account]++;
            }
        }
    }

    function isAllLiquidated() external view  returns(bool) {
        ProverLiquidationInfo[] storage proverLiquidations = proverLiquidation[msg.sender];
        return proverLiquidations[proverLiquidations.length-1].isLiquidated;
    }

    function settle(address _account) external onlyDeposit {
        updateLiquidation(_account);
    }

}
