// SPDX-License-Identifier: AGPL-3.0

pragma solidity 0.8.17;

import "../util/Structs.sol";

interface IGlobalPool {
    function addGlobalRewards(uint256 _reward) external;
    function distributionReward(uint8 _index, RewardType _rewardType) external returns (uint256);
    function getAvailableRewards() external view returns (uint256);
    function setRewardDistribution(address _rewardDistribution) external;
}
