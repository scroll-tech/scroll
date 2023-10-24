// SPDX-License-Identifier: AGPL-3.0

pragma solidity 0.8.17;

import "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import "../interfaces/ISlotAdapter.sol";
import "../interfaces/IDeposit.sol";

contract SlotAdapterMock is ISlotAdapter ,OwnableUpgradeable {
    address public zkEvmContract;
    error OnlyZkEvmContract();

    mapping(address =>uint256 ) public reward_number ;

    function initialize() public initializer {

        // Initialize OZ contracts
        __Ownable_init_unchained();
    }

    modifier onlyZkEvmContract() {
        if (zkEvmContract != msg.sender) {
            revert OnlyZkEvmContract();
        }
        _;
    }

    function setZKEvmContract(address _zkEvmContract) external onlyOwner {
        zkEvmContract = _zkEvmContract;
    }
    function getSlotId() external returns (uint256){
        return 1;
    }
    function getZKEvmContract() external returns (address){
        return msg.sender;
    }
    function pause() external{}

    function setSlotId(uint256 _slotId) external{
    }

    function distributeRewards(address _recipient, uint64 _initNumBatch, uint64 _finalNewBatch, IDeposit _iDeposit) external onlyZkEvmContract {
        reward_number[_recipient]+=1;
    }
    function calcSlotReward(uint64 _batchNum, IDeposit _iDeposit) external onlyZkEvmContract {

    }

    function calcCurrentTotalDeposit(uint64 _batchNum, IDeposit _iDeposit, address _account, bool _reset) external onlyZkEvmContract {

    }

    function changeSlotManager(address _manager) external{}

//    function punish(address _recipient, IDeposit _iDeposit) external onlyZkEvmContract {
//
//    }

    function punish(address _recipient, IDeposit _iDeposit, uint256 _punishAmount) external{

//        require(!Address.isContract(_recipient), "Account not EOA");
        _iDeposit.punish(_recipient, _punishAmount);
//        emit PunishAmount(_recipient, _punishAmount);
    }

    function start() external{

    }
    function stop() external{

    }
    function unpause() external{

    }

    function setPledgeStatus(address _account) external {

    }

    function getPledgeStatus(address _account) external view returns(uint256) {
        return 1;
    }

    function delPledge(address _account) external {

    }

    function getIDEToken() external view returns (address) {
        
    }
}