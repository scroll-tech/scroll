// SPDX-License-Identifier: AGPL-3.0

pragma solidity 0.8.17;

import { Ownable } from "@openzeppelin/contracts/access/Ownable.sol";

import { ISlotAdapter } from "../interfaces/ISlotAdapter.sol";
import { IDEDeposit } from "../IDEDeposit.sol";

contract ZkEvmContract is Ownable {
    ISlotAdapter public slotAdapter;
    IDEDeposit public ideDeposit;
    error OnlyDeposit();
    modifier onlyDeposit() {
        if (address(ideDeposit) != msg.sender) {
            revert OnlyDeposit();
        }
        _;
    }

    function setSlotAdapter(address _slotAdapter) external onlyOwner {
        require(_slotAdapter != address(0), "Set slotAdapter zero address");
        slotAdapter = ISlotAdapter(_slotAdapter);
    }

    function setDeposit(IDEDeposit _ideDeposit) public onlyOwner {
        ideDeposit = _ideDeposit;
    }

    function sequenceBatches(uint64 batchNum) public onlyOwner{
        slotAdapter.calcSlotReward(batchNum, ideDeposit);
    }

    function submitProofHash(uint64 initNumBatch, uint64 finalNewBatch, bytes32 _proofHash) external {
        slotAdapter.calcCurrentTotalDeposit(finalNewBatch, ideDeposit, msg.sender, false);
    }

    function distributeRewards(address _recipient, uint64 _initbatchNum, uint64 _finalbatchNum) external onlyOwner {
        slotAdapter.distributeRewards(_recipient, _initbatchNum, _finalbatchNum, ideDeposit);
    }

    function settle(address _account) external onlyDeposit {

    }
}
