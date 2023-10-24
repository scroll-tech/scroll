// SPDX-License-Identifier: AGPL-3.0

pragma solidity 0.8.17;

import "@openzeppelin/contracts-upgradeable/token/ERC20/extensions/IERC20MetadataUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/token/ERC20/utils/SafeERC20Upgradeable.sol";

import "./lib/DepositContract.sol";
import "./lib/TokenWrapped.sol";
import "./interfaces/IBasePolygonZkEVMGlobalExitRoot.sol";
import "./interfaces/IBridgeMessageReceiver.sol";
import "./interfaces/IPolygonZkEVMBridge.sol";
import "./interfaces/IRollupIdInfo.sol";
import "./lib/EmergencyManager.sol";
import "./lib/GlobalExitRootLib.sol";

/**
 * PolygonZkEVMBridge that will be deployed on both networks Ethereum and Polygon zkEVM
 * Contract responsible to manage the token interactions with other networks
 */
contract PolygonZkEVMGasTokenBridge is
    DepositContract,
    EmergencyManager,
    IPolygonZkEVMBridge
{
    using SafeERC20Upgradeable for IERC20Upgradeable;

    // Wrapped Token information struct
    struct TokenInformation {
        uint32 originNetwork;
        address originTokenAddress;
    }

    // bytes4(keccak256(bytes("permit(address,address,uint256,uint256,uint8,bytes32,bytes32)")));
    bytes4 private constant _PERMIT_SIGNATURE = 0xd505accf;

    // bytes4(keccak256(bytes("permit(address,address,uint256,uint256,bool,uint8,bytes32,bytes32)")));
    bytes4 private constant _PERMIT_SIGNATURE_DAI = 0x8fcbaf0c;

    // Mainnet identifier
    uint32 private constant _MAINNET_NETWORK_ID = 0;

    // Number of networks supported by the bridge
    uint32 private constant _CURRENT_SUPPORTED_NETWORKS = 2;

    // Leaf type asset
    uint8 private constant _LEAF_TYPE_ASSET = 0;

    // Leaf type message
    uint8 private constant _LEAF_TYPE_MESSAGE = 1;

    // Network identifier
    uint32 public networkID;

    // Global Exit Root address
    IBasePolygonZkEVMGlobalExitRoot public globalExitRootManager;

    // Last updated deposit count to the global exit root manager
    uint32 public lastUpdatedDepositCount;

    // Leaf index --> claimed bit map
    mapping(uint256 => uint256) public claimedBitMap;

    // keccak256(OriginNetwork || tokenAddress) --> Wrapped token address
    mapping(bytes32 => address) public tokenInfoToWrappedToken;

    // Wrapped token Address --> Origin token information
    mapping(address => TokenInformation) public wrappedTokenToTokenInfo;

    // PolygonZkEVM address
    address public polygonZkEVMaddress;

    uint256 public bridgeFee;
    address public feeAddress;
    address public admin;
    address public gasTokenAddress;
    bytes public gasTokenMetadata;
    
    // Obtain the corresponding L1 bridge contract address according to the rollup id
    address public rollupIdInfoAddress;
    
    // Self L2 identifier
    // uint32 private constant _SELF_ROLLUP_NETWORK_ID = 1;
    uint256 public bridgeCrossFee;
    
    /**
     * @param _networkID networkID
     * @param _globalExitRootManager global exit root manager address
     * @param _polygonZkEVMaddress polygonZkEVM address
     * @notice The value of `_polygonZkEVMaddress` on the L2 deployment of the contract will be address(0), so
     * emergency state is not possible for the L2 deployment of the bridge, intentionally
     */
    function initialize(
        uint32 _networkID,
        IBasePolygonZkEVMGlobalExitRoot _globalExitRootManager,
        address _polygonZkEVMaddress,
        address _admin,
        uint256  _bridgeFee,
        address _gasTokenAddress,
        bytes memory _gasTokenMetadata,
        address _rollupIdInfoAddress
    ) external virtual initializer {
        networkID = _networkID;
        globalExitRootManager = _globalExitRootManager;
        polygonZkEVMaddress = _polygonZkEVMaddress;
        bridgeFee = _bridgeFee;
        feeAddress = _admin;
        admin =  _admin;
        gasTokenAddress = _gasTokenAddress;
        gasTokenMetadata = _gasTokenMetadata;
        rollupIdInfoAddress = _rollupIdInfoAddress;
        // Initialize OZ contracts
        __ReentrancyGuard_init();
    }

    modifier onlyPolygonZkEVM() {
        if (polygonZkEVMaddress != msg.sender) {
            revert OnlyPolygonZkEVM();
        }
        _;
    }

    error OnlyAdmin();

    modifier onlyAdmin() {
        if (admin != msg.sender) {
            revert OnlyAdmin();
        }
        _;
    }

    /**
     * @dev Emitted when bridge assets or messages to another network
     */
    event BridgeEvent(
        uint8 leafType,
        uint32 originNetwork,
        address originAddress,
        uint32 destinationNetwork,
        address destinationAddress,
        uint256 amount,
        bytes metadata,
        uint32 depositCount
    );

    /**
     * @dev Emitted when a claim is done from another network
     */
    event ClaimEvent(
        uint32 index,
        uint32 originNetwork,
        address originAddress,
        uint32 destinationNetwork,
        address destinationAddress,
        uint256 amount
    );

    /**
     * @dev Emitted when a new wrapped token is created
     */
    event NewWrappedToken(
        uint32 originNetwork,
        address originTokenAddress,
        address wrappedTokenAddress,
        bytes metadata
    );

    /**
     * @notice Deposit add a new leaf to the merkle tree
     * @param destinationNetwork Network destination
     * @param destinationAddress Address destination
     * @param amount Amount of tokens
     * @param token Token address, 0 address is reserved for ether
     * @param forceUpdateGlobalExitRoot Indicates if the new global exit root is updated or not
     */
    function bridgeAsset(
        uint32 destinationNetwork,
        address destinationAddress,
        uint256 amount,
        address token,
        bool forceUpdateGlobalExitRoot
    ) public payable virtual ifNotEmergencyState nonReentrant {
        if (
            destinationNetwork == networkID || (networkID == _MAINNET_NETWORK_ID && destinationNetwork >= _CURRENT_SUPPORTED_NETWORKS)
        ) {
            revert DestinationNetworkInvalid();
        }

        address originTokenAddress;
        uint32 originNetwork;
        bytes memory metadata;
        uint256 leafAmount = amount;

        uint256 fee = bridgeFee;
        if (destinationNetwork >= 1 && bridgeCrossFee != 0) {
            fee = bridgeCrossFee;
        }

        if (token == address(0)) {
            // Ether transfer
            if ((msg.value - fee) != amount) {
                revert AmountDoesNotMatchMsgValue();
            }

            // Ether is treated as ether from mainnet
            originNetwork = _MAINNET_NETWORK_ID;
        } else {
            // Check msg.value is 0 if tokens are bridged
            if (msg.value != fee) {
                revert MsgValueNotZero();
            }

            TokenInformation memory tokenInfo = wrappedTokenToTokenInfo[token];

            if (tokenInfo.originTokenAddress != address(0)) {
                // The token is a wrapped token from another network

                // Burn tokens
                TokenWrapped(token).burn(msg.sender, amount);

                originTokenAddress = tokenInfo.originTokenAddress;
                originNetwork = tokenInfo.originNetwork;
            } else {
                // In order to support fee tokens check the amount received, not the transferred
                uint256 balanceBefore = IERC20Upgradeable(token).balanceOf(
                    address(this)
                );
                IERC20Upgradeable(token).safeTransferFrom(
                    msg.sender,
                    address(this),
                    amount
                );
                uint256 balanceAfter = IERC20Upgradeable(token).balanceOf(
                    address(this)
                );

                // Override leafAmount with the received amount
                leafAmount = balanceAfter - balanceBefore;

                originTokenAddress = token;
                originNetwork = networkID;

                // Encode metadata
                metadata = IRollupIdInfo(rollupIdInfoAddress).encodeMetadata(token);
            }
        }

       if (gasTokenAddress != address (0)) {
            if (token == address(0)) {
                originTokenAddress = gasTokenAddress;
                metadata = gasTokenMetadata;
            } else if (originTokenAddress == gasTokenAddress) {
                originTokenAddress = address(0);
            }
        }

        emit BridgeEvent(
            _LEAF_TYPE_ASSET,
            originNetwork,
            originTokenAddress,
            destinationNetwork,
            destinationAddress,
            leafAmount,
            metadata,
            uint32(depositCount)
        );

        _deposit(
            getLeafValue(
                _LEAF_TYPE_ASSET,
                originNetwork,
                originTokenAddress,
                destinationNetwork,
                destinationAddress,
                leafAmount,
                keccak256(metadata)
            )
        );

        (bool success, ) = feeAddress.call{value: fee}(new bytes(0));
        if (!success) {
            revert EtherTransferFailed();
        }

        // Update the new root to the global exit root manager if set by the user
        if (forceUpdateGlobalExitRoot) {
            _updateGlobalExitRoot();
        }
    }

    /**
     * @notice Bridge message and send ETH value
     * @param destinationNetwork Network destination
     * @param destinationAddress Address destination
     * @param forceUpdateGlobalExitRoot Indicates if the new global exit root is updated or not
     * @param metadata Message metadata
     */
    function bridgeMessage(
        uint32 destinationNetwork,
        address destinationAddress,
        bool forceUpdateGlobalExitRoot,
        bytes calldata metadata
    ) external payable ifNotEmergencyState {
        if (
            destinationNetwork == networkID ||
            destinationNetwork >= _CURRENT_SUPPORTED_NETWORKS
        ) {
            revert DestinationNetworkInvalid();
        }

        emit BridgeEvent(
            _LEAF_TYPE_MESSAGE,
            networkID,
            msg.sender,
            destinationNetwork,
            destinationAddress,
            msg.value,
            metadata,
            uint32(depositCount)
        );

        _deposit(
            getLeafValue(
                _LEAF_TYPE_MESSAGE,
                networkID,
                msg.sender,
                destinationNetwork,
                destinationAddress,
                msg.value,
                keccak256(metadata)
            )
        );

        // Update the new root to the global exit root manager if set by the user
        if (forceUpdateGlobalExitRoot) {
            _updateGlobalExitRoot();
        }
    }

    function isCrossRollup(uint32 destinationNetwork) private view returns (bool) {
        return (networkID == _MAINNET_NETWORK_ID && destinationNetwork > 1);
    }

    /**
     * @notice Verify merkle proof and withdraw tokens/ether
     * @param smtProof Smt proof
     * @param index Index of the leaf
     * @param mainnetExitRoot Mainnet exit root
     * @param rollupExitRoot Rollup exit root
     * @param originNetwork Origin network
     * @param originTokenAddress  Origin token address, 0 address is reserved for ether
     * @param destinationNetwork Network destination
     * @param destinationAddress Address destination
     * @param amount Amount of tokens
     * @param metadata Abi encoded metadata if any, empty otherwise
     */
    function claimAsset(
        bytes32[_DEPOSIT_CONTRACT_TREE_DEPTH] calldata smtProof,
        uint32 index,
        bytes32 mainnetExitRoot,
        bytes32 rollupExitRoot,
        uint32 originNetwork,
        address originTokenAddress,
        uint32 destinationNetwork,
        address destinationAddress,
        uint256 amount,
        bytes calldata metadata
    ) external payable ifNotEmergencyState {
        // Verify leaf exist and it does not have been claimed
        _verifyLeaf(
            smtProof,
            index,
            mainnetExitRoot,
            rollupExitRoot,
            originNetwork,
            originTokenAddress,
            destinationNetwork,
            destinationAddress,
            amount,
            metadata,
            _LEAF_TYPE_ASSET
        );

        address destinationTokenAddress = address(0);

        // Transfer funds
        if (originTokenAddress == address(0)) {
            if (!isCrossRollup(destinationNetwork)) {
                // Transfer ether
                /* solhint-disable avoid-low-level-calls */
                (bool success, ) = destinationAddress.call{value: amount}(
                    new bytes(0)
                );
                if (!success) {
                    revert EtherTransferFailed();
                }
            }
        } else {
            // Transfer tokens
            if (originNetwork == networkID) {
                // The token is an ERC20 from this network
                if (!isCrossRollup(destinationNetwork)) {
                    IERC20Upgradeable(originTokenAddress).safeTransfer(
                        destinationAddress,
                        amount
                    );
                }
                destinationTokenAddress = originTokenAddress;
            } else {
                // The tokens is not from this network
                // Create a wrapper for the token if not exist yet
                bytes32 tokenInfoHash = keccak256(
                    abi.encodePacked(originNetwork, originTokenAddress)
                );
                address wrappedToken = tokenInfoToWrappedToken[tokenInfoHash];
                destinationTokenAddress = wrappedToken;
                if (wrappedToken == address(0)) {
                    // Get ERC20 metadata
                    (
                        string memory name,
                        string memory symbol,
                        uint8 decimals
                    ) = abi.decode(metadata, (string, string, uint8));

                    // Create a new wrapped erc20 using create2
                    TokenWrapped newWrappedToken = (new TokenWrapped){
                        salt: tokenInfoHash
                    }(name, symbol, decimals);

                    destinationTokenAddress = address(newWrappedToken);

                    // Mint tokens for the destination address
                    if (isCrossRollup(destinationNetwork)) {
                        newWrappedToken.mint(address(this), amount);
                    } else {
                        newWrappedToken.mint(destinationAddress, amount);
                    }

                    // Create mappings
                    tokenInfoToWrappedToken[tokenInfoHash] = address(
                        newWrappedToken
                    );

                    wrappedTokenToTokenInfo[address(newWrappedToken)] = TokenInformation(originNetwork, originTokenAddress);

                    emit NewWrappedToken(
                        originNetwork,
                        originTokenAddress,
                        address(newWrappedToken),
                        metadata
                    );
                } else {
                    // Use the existing wrapped erc20
                    if (isCrossRollup(destinationNetwork)) {
                        TokenWrapped(wrappedToken).mint(address(this), amount);
                    } else {
                        TokenWrapped(wrappedToken).mint(destinationAddress, amount);
                    }
                }
            }
        }
        // L1->L2
        if (isCrossRollup(destinationNetwork)) {
            if (rollupIdInfoAddress == address(0)) {
                revert EtherTransferFailed();
            }
            address destinationL1BridgeAddress =  getL1BridgeAddress(destinationNetwork);
            if (destinationL1BridgeAddress == address(0)) {
                revert EtherTransferFailed();
            }

            uint256 fee = IPolygonZkEVMBridge(destinationL1BridgeAddress).bridgeFee();
            if (msg.value != fee) {
                revert MsgValueNotZero();
            }
            uint256 transferAmount = fee;
            if (destinationTokenAddress != address(0)) {
                IERC20Upgradeable(destinationTokenAddress).approve(destinationL1BridgeAddress, amount);
            } else {
                transferAmount += amount;
            }
            IPolygonZkEVMBridge(destinationL1BridgeAddress).bridgeAsset{value: transferAmount}(1, destinationAddress, amount, destinationTokenAddress, true);
        }

        emit ClaimEvent(
            index,
            originNetwork,
            originTokenAddress,
            destinationNetwork,
            destinationAddress,
            amount
        );
    }

    /**
     * @notice Verify merkle proof and execute message
     * If the receiving address is an EOA, the call will result as a success
     * Which means that the amount of ether will be transferred correctly, but the message
     * will not trigger any execution
     * @param smtProof Smt proof
     * @param index Index of the leaf
     * @param mainnetExitRoot Mainnet exit root
     * @param rollupExitRoot Rollup exit root
     * @param originNetwork Origin network
     * @param originAddress Origin address
     * @param destinationNetwork Network destination
     * @param destinationAddress Address destination
     * @param amount message value
     * @param metadata Abi encoded metadata if any, empty otherwise
     */
    function claimMessage(
        bytes32[_DEPOSIT_CONTRACT_TREE_DEPTH] calldata smtProof,
        uint32 index,
        bytes32 mainnetExitRoot,
        bytes32 rollupExitRoot,
        uint32 originNetwork,
        address originAddress,
        uint32 destinationNetwork,
        address destinationAddress,
        uint256 amount,
        bytes calldata metadata
    ) external ifNotEmergencyState {
        // Verify leaf exist and it does not have been claimed
        _verifyLeaf(
            smtProof,
            index,
            mainnetExitRoot,
            rollupExitRoot,
            originNetwork,
            originAddress,
            destinationNetwork,
            destinationAddress,
            amount,
            metadata,
            _LEAF_TYPE_MESSAGE
        );

        // Execute message
        // Transfer ether
        /* solhint-disable avoid-low-level-calls */
        (bool success, ) = destinationAddress.call{value: amount}(
            abi.encodeCall(
                IBridgeMessageReceiver.onMessageReceived,
                (originAddress, originNetwork, metadata)
            )
        );
        if (!success) {
            revert MessageFailed();
        }

        emit ClaimEvent(
            index,
            originNetwork,
            originAddress,
            destinationNetwork,
            destinationAddress,
            amount
        );
    }

    /**
     * @notice Returns the precalculated address of a wrapper using the token information
     * Note Updating the metadata of a token is not supported.
     * Since the metadata has relevance in the address deployed, this function will not return a valid
     * wrapped address if the metadata provided is not the original one.
     * @param originNetwork Origin network
     * @param originTokenAddress Origin token address, 0 address is reserved for ether
     * @param name Name of the token
     * @param symbol Symbol of the token
     * @param decimals Decimals of the token
     */
    function precalculatedWrapperAddress(
        uint32 originNetwork,
        address originTokenAddress,
        string calldata name,
        string calldata symbol,
        uint8 decimals
    ) external view returns (address) {
        bytes32 salt = keccak256(
            abi.encodePacked(originNetwork, originTokenAddress)
        );

        bytes32 hashCreate2 = keccak256(
            abi.encodePacked(
                bytes1(0xff),
                address(this),
                salt,
                keccak256(
                    abi.encodePacked(
                        type(TokenWrapped).creationCode,
                        abi.encode(name, symbol, decimals)
                    )
                )
            )
        );

        // last 20 bytes of hash to address
        return address(uint160(uint256(hashCreate2)));
    }

    /**
     * @notice Returns the address of a wrapper using the token information if already exist
     * @param originNetwork Origin network
     * @param originTokenAddress Origin token address, 0 address is reserved for ether
     */
    function getTokenWrappedAddress(
        uint32 originNetwork,
        address originTokenAddress
    ) external view returns (address) {
        return
            tokenInfoToWrappedToken[
                keccak256(abi.encodePacked(originNetwork, originTokenAddress))
            ];
    }

    /**
     * @notice Function to activate the emergency state
     " Only can be called by the Polygon ZK-EVM in extreme situations
     */
    function activateEmergencyState() external onlyPolygonZkEVM {
        _activateEmergencyState();
    }

    /**
     * @notice Function to deactivate the emergency state
     " Only can be called by the Polygon ZK-EVM
     */
    function deactivateEmergencyState() external onlyPolygonZkEVM {
        _deactivateEmergencyState();
    }

    /**
     * @notice Verify leaf and checks that it has not been claimed
     * @param smtProof Smt proof
     * @param index Index of the leaf
     * @param mainnetExitRoot Mainnet exit root
     * @param rollupExitRoot Rollup exit root
     * @param originNetwork Origin network
     * @param originAddress Origin address
     * @param destinationNetwork Network destination
     * @param destinationAddress Address destination
     * @param amount Amount of tokens
     * @param metadata Abi encoded metadata if any, empty otherwise
     * @param leafType Leaf type -->  [0] transfer Ether / ERC20 tokens, [1] message
     */
    function _verifyLeaf(
        bytes32[_DEPOSIT_CONTRACT_TREE_DEPTH] calldata smtProof,
        uint32 index,
        bytes32 mainnetExitRoot,
        bytes32 rollupExitRoot,
        uint32 originNetwork,
        address originAddress,
        uint32 destinationNetwork,
        address destinationAddress,
        uint256 amount,
        bytes calldata metadata,
        uint8 leafType
    ) internal {
        // Set and check nullifier
        _setAndCheckClaimed(index);

        // Check timestamp where the global exit root was set
        uint256 timestampGlobalExitRoot = globalExitRootManager
            .globalExitRootMap(
                GlobalExitRootLib.calculateGlobalExitRoot(
                    mainnetExitRoot,
                    rollupExitRoot
                )
            );

        if (timestampGlobalExitRoot == 0) {
            revert GlobalExitRootInvalid();
        }

        // Destination network compliance check
        if (destinationNetwork != networkID && destinationNetwork < _CURRENT_SUPPORTED_NETWORKS) {
            revert DestinationNetworkInvalid();
        }

        bytes32 claimRoot;
        if (networkID == _MAINNET_NETWORK_ID) {
            // Verify merkle proof using rollup exit root
            claimRoot = rollupExitRoot;
        } else {
            // Verify merkle proof using mainnet exit root
            claimRoot = mainnetExitRoot;
        }
        if (
            !verifyMerkleProof(
                getLeafValue(
                    leafType,
                    originNetwork,
                    originAddress,
                    destinationNetwork,
                    destinationAddress,
                    amount,
                    keccak256(metadata)
                ),
                smtProof,
                index,
                claimRoot
            )
        ) {
            revert InvalidSmtProof();
        }
    }

    /**
     * @notice Function to check if an index is claimed or not
     * @param index Index
     */
    function isClaimed(uint256 index) external view returns (bool) {
        (uint256 wordPos, uint256 bitPos) = _bitmapPositions(index);
        uint256 mask = (1 << bitPos);
        return (claimedBitMap[wordPos] & mask) == mask;
    }

    /**
     * @notice Function to check that an index is not claimed and set it as claimed
     * @param index Index
     */
    function _setAndCheckClaimed(uint256 index) private {
        (uint256 wordPos, uint256 bitPos) = _bitmapPositions(index);
        uint256 mask = 1 << bitPos;
        uint256 flipped = claimedBitMap[wordPos] ^= mask;
        if (flipped & mask == 0) {
            revert AlreadyClaimed();
        }
    }

    /**
     * @notice Function to update the globalExitRoot if the last deposit is not submitted
     */
    function updateGlobalExitRoot() external {
        if (lastUpdatedDepositCount < depositCount) {
            _updateGlobalExitRoot();
        }
    }

    /**
     * @notice Function to update the globalExitRoot
     */
    function _updateGlobalExitRoot() internal {
        lastUpdatedDepositCount = uint32(depositCount);
        globalExitRootManager.updateExitRoot(getDepositRoot());
    }

    /**
     * @notice Function decode an index into a wordPos and bitPos
     * @param index Index
     */
    function _bitmapPositions(
        uint256 index
    ) private pure returns (uint256 wordPos, uint256 bitPos) {
        wordPos = uint248(index >> 8);
        bitPos = uint8(index);
    }

    function setSettings(uint256 _bridgeFee, address _feeAddress, uint256 _bridgeCrossFee, address _rollupIdInfoAddress) external onlyAdmin {
        if(_bridgeFee > 0) {
            bridgeFee = _bridgeFee;
        }
        if(_feeAddress != address(0)) {
            feeAddress = _feeAddress;
        }
        if(_bridgeCrossFee > 0) {
            bridgeCrossFee = _bridgeCrossFee;
        }
        if(_rollupIdInfoAddress != address(0)) {
            rollupIdInfoAddress = _rollupIdInfoAddress;
        }
    }

    function getL1BridgeAddress(uint32 _rollupId) public view returns (address) {
        return IRollupIdInfo(rollupIdInfoAddress).getL1BridgeAddress(_rollupId);
    }
}
