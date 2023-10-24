// SPDX-License-Identifier: AGPL-3.0

pragma solidity 0.8.17;

import { ReentrancyGuardUpgradeable } from "@openzeppelin/contracts-upgradeable/security/ReentrancyGuardUpgradeable.sol";
import { OwnableUpgradeable } from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import { Address } from "@openzeppelin/contracts/utils/Address.sol";

import { IOpsideSlots } from "./interfaces/IOpsideSlots.sol";
import { IOpsideErrors } from "./interfaces/IOpsideErrors.sol";
import { IGlobalPool } from "./interfaces/IGlobalPool.sol";
import { IRewardDistribution } from "./interfaces/IRewardDistribution.sol";
import { ISlotAdapter } from "./interfaces/ISlotAdapter.sol";

import { RewardType } from "./util/Structs.sol";

contract GlobalRewardPool is IGlobalPool, IOpsideErrors, ReentrancyGuardUpgradeable, OwnableUpgradeable {
    uint256 public totalBalanceReward;
    uint256 public claimed;

    address public opsideSlots;

    IRewardDistribution public rewardDistribution;

    mapping(address => bool) public slotAdapters;

    function initialize(address _opsideSlots, address _rewardDistribution) external virtual initializer {
        opsideSlots = _opsideSlots;
        rewardDistribution = IRewardDistribution(_rewardDistribution);

        __Ownable_init_unchained();
    }

    function setRewardDistribution(address _rewardDistribution) external onlyOwner {
        rewardDistribution = IRewardDistribution(_rewardDistribution);
    }

    modifier onlyOpsideSlots() {
        if (opsideSlots != msg.sender) {
            revert OnlyOpsideSlots();
        }
        _;
    }

    modifier onlySlotAdapter() {
        if (!slotAdapters[msg.sender]) {
            revert OnlySlotAdapter();
        }
        _;
    }

    /**
     * @param _reward Add reward to the global rewards pool
     */
    function addGlobalRewards(uint256 _reward) external onlyOpsideSlots {
        totalBalanceReward += _reward;
    }

    function addSlotAdapter(address _slotAdapter) external onlyOwner {
        require(!slotAdapters[_slotAdapter], "SlotAdapter already exists");

        slotAdapters[_slotAdapter] = true;
    }

    /**
        @param _index prover index
        @param _rewardType reward type
     */
    function distributionReward(uint8 _index, RewardType _rewardType) external onlySlotAdapter returns (uint256) {
        require(_rewardType == RewardType.PROVER || _rewardType == RewardType.SEQUENCER, "Invalid reward type");
        uint256 _reward = rewardDistribution.calcProverReward(_index);
        if (_rewardType == RewardType.PROVER) {
            _reward = _reward;
        }

        if (_rewardType == RewardType.SEQUENCER) {
            _reward = _reward;
        }

        claimed += _reward;
        return _reward;
    }


    function getAvailableRewards() external view returns (uint256) {
        return totalBalanceReward - claimed;
    }

    function calcSlotReward(uint256 _slotId) external view returns (uint256) {
        return rewardDistribution.getRewardPerBlock() / _slotId;
    }
}
