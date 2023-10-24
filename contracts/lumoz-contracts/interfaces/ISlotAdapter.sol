// SPDX-License-Identifier: AGPL-3.0

pragma solidity 0.8.17;

import { IDeposit } from "./IDeposit.sol";

import "../util/Structs.sol";

interface ISlotAdapter {
    /**
     * @notice Set slotId
     */
     function setSlotId(uint256 _slotId) external;

    /**
     * @notice Get slotId
     */
     function getSlotId() external returns (uint256);


    /**
     * @notice Set the zkEvm contract
     */
    function setZKEvmContract(address _zkEvmContract) external;

    function getZKEvmContract() external returns (address);

    /**
     * @notice Start the bound chain
     */
    function start() external;

    /**
     * @notice Stop the bound chain
     */
    function stop() external;

    /**
     * @notice Pause the bound chain
     */
    function pause() external;

    /**
     * @notice Unpause the bound chain
     */
    function unpause() external;

    /**
     * @notice distribute reward
     */
    function distributeRewards(address _recipient, uint64 _initNumBatch, uint64 _finalNewBatch, IDeposit _iDeposit) external;

    /**
     * @notice Change slot manager
     */
    function changeSlotManager(address _manager) external;

    function punish(address _recipient, IDeposit _iDeposit, uint256 _punishAmount) external;

    function calcSlotReward(uint64 _batchNum, IDeposit _iDeposit) external;

    function calcCurrentTotalDeposit(uint64 _batchNum, IDeposit _iDeposit, address _account, bool _reset) external;

    function setPledgeStatus(address _account) external;

    function getPledgeStatus(address _account) external view returns (uint256);

    function delPledge(address _account) external;

    function getIDEToken() external view returns (address);
}
