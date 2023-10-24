// SPDX-License-Identifier: AGPL-3.0

pragma solidity 0.8.17;

import "./ISlotAdapter.sol";
import "../util/Structs.sol";

interface IOpsideSlots {
    /**
     * @notice Register a new slot
     * @param name The name of the new slot
     * @param manager The manager of the new slot
     * @param _amount amount for erc20
     */
    function register(
        string calldata name,
        address manager,
        uint256 _amount
    ) external payable returns (uint256);

    /**
     * @notice Setup a slot with configurations
     * @param _slotId The slot to setup
     * @param chainId The chain id of the chain bound to the slot
     * @param adapter The adapter between the slot and the bound chain
     */
    function setup(
        uint256 _slotId,
        uint256 chainId,
        ISlotAdapter adapter
    ) external;

    /**
     * @notice Start a slot
     */
    function start(uint256 _slotId) external;

    /**
     * @notice Start a slot
     */
    function stop(uint256 _slotId) external;

    /**
     * @notice Deregister a slot
     */
    function deregister(uint256 _slotId) external;

    /**
     * @notice Pause a slot
     * @dev A slot can be paused by its manager
     */
    function pause(uint256 _slotId) external;

    /**
     * @notice Unpause a slot
     * @dev A slot can be unpaused by its manager
     */
    function unpause(uint256 _slotId) external;

    /**
     * @notice Change slot manager
     */
    function setSlotManager(uint256 _slotId, address manager) external;

    /**
     * @notice Fund the global rewards pool
     */
    function fundGlobalRewards() external payable;

    /**
     * @notice Fund a slot rewards pool
     */
    function fundSlotRewards(uint256 _slotId) external payable;

    /**
     * @notice Get slot data
     */
    function getSlot(uint256 _slotId) external view returns (Slot memory slot);

    /**
     * @notice Get slot adapter
     */
    function getSlotAdapter(uint256 _slotId) external view returns (address);

    /**
     * @notice Get global rewards pool size
     */
    function getGlobalRewards() external view returns (uint256 rewards);

    /**
     * @notice Get slot by chain id
     */
    function chainToSlot(
        uint256 _chainId
    ) external view returns (Slot memory slot);

    /**
     * @notice Get slot status
     */
    function slotStatus(
        uint256 _slotId
    ) external view returns (SlotStatus status);

    function distributeReward(address payable _to, uint64 _initNumBatch, uint64 _finalNewBatch, uint256 _amount) external;

    function calcSlotReward() external returns (uint256);

    function getBlockReward() external view returns (uint256);

    function commitCount() external;

    function getCommitCount(uint256 _blockNumber) external view returns (uint256);

    function setMinerToSlot(address _account) external;

    function getMinerSlot(address _account) external view returns (uint256) ;

    function deleteMinerSlot(address _account) external;

    function getIDEToken() external view returns (address);
}
