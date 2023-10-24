// SPDX-License-Identifier: AGPL-3.0

pragma solidity 0.8.17;

interface IRewardDistribution {
    function calcProverReward(uint8 _index) external returns (uint256);
    function setRewardPerBlock(uint256 _reward) external;
    function getRewardPerBlock() external view returns (uint256);
}
