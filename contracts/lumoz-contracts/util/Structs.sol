// SPDX-License-Identifier: AGPL-3.0

pragma solidity 0.8.17;

/**
 * @notice Slot data
 * @param id Request id
 * @param name Slot name
 * @param manager Address of the slot manager
 */
struct Request {
    uint256 id;
    uint256 value;
    string name;
    address manager;
}

enum SlotStatus {
    Created,
    Ready,
    Running,
    Paused,
    Stopped
}

/**
 * @notice Slot data
 * @param id Slot id
 * @param name Slot name
 * @param chainId The chain id of the rollup bound to this slot
 * @param manager Address of the slot manager
 * @param rewardsPool The private rewards pool of this slot
 * @param issuedReward Number of awards issued
 */
struct Slot {
    uint256 id;
    string name;
    uint256 chainId;
    address manager;
    uint256 rewardsPool;
    uint256 issuedReward;
}

enum RewardType {PROVER, SEQUENCER}

enum RewardDistributionType { Automatic, Manual }
