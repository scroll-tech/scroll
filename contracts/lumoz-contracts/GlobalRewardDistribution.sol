// SPDX-License-Identifier: AGPL-3.0

pragma solidity 0.8.17;

import { Ownable } from "@openzeppelin/contracts/access/Ownable.sol";

import { IRewardDistribution } from "./interfaces/IRewardDistribution.sol";
import { IOpsideErrors } from "./interfaces/IOpsideErrors.sol";

contract GlobalRewardDistribution is IRewardDistribution, IOpsideErrors, Ownable{
    uint256 public rewardPerBlock = 0.1 ether;
    uint8 internal constant _FACTOR = 2;

    function calcProverReward(uint8 _index) external view returns (uint256) {
        uint256 reward = rewardPerBlock;
         for (uint8 i = 1; i <=  _index; i++) {
            reward = reward / _FACTOR;
         }

         return reward;
    }

    function setRewardPerBlock(uint256 _reward) external onlyOwner {
        rewardPerBlock = _reward;
    }

    function getRewardPerBlock() external view returns (uint256) {
        return rewardPerBlock;
    }
}
