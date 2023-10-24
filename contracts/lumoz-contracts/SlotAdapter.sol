// SPDX-License-Identifier: AGPL-3.0

pragma solidity 0.8.17;

import { OwnableUpgradeable } from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import { Address } from "@openzeppelin/contracts/utils/Address.sol";



import { ISlotAdapter } from "./interfaces/ISlotAdapter.sol";
import { IOpsideSlots } from  "./interfaces/IOpsideSlots.sol";
import { IOpsideErrors } from "./interfaces/IOpsideErrors.sol";
import { IRewardDistribution } from "./interfaces/IRewardDistribution.sol";
import { IGlobalPool } from "./interfaces/IGlobalPool.sol";
import { IDeposit } from "./interfaces/IDeposit.sol";

import { SlotStatus, RewardType, RewardDistributionType } from "./util/Structs.sol";
import { IZKEVMContract } from "./interfaces/IZKEVMContract.sol";

contract SlotAdapter is ISlotAdapter, IOpsideErrors, OwnableUpgradeable {
    struct CurrentProverReward {
        uint256 reward;
        uint256 currentTotalDeposit;
    }

    struct gasFreeContractSet {
        address[] _addressList;
        mapping(address => uint) indexOf;
        mapping(address => bool) inserted;
    }

    event RewardFromGlobalPool(uint256 indexed _slotId, address indexed _to, uint256 _amount, RewardType _rewardType);
    event Withdrawal(uint256 indexed _slotId, address indexed _caller, address indexed _recipient, uint256 _amount);
    event DistributeRewards(uint256 indexed _slotId, address indexed _caller, address indexed _recipient, uint256 _amount);
    event Initialize(address _manager, address _opsideSlots, address _globalPool);
    event PunishAmount(address indexed _account, uint256 _amount);
    event Settle(address _account);
    event UpdateGasFreeContractBatch(address[] _addrList, bool isAdd);

    uint256 public slotId;

    address public slotManager;

    address public zkEvmContract;

    uint256 public punishAmount;

    uint256 internal constant _UINT_AMOUNT = 1 ether;

    IOpsideSlots public opsideSlots;

    IGlobalPool public globalPool;

    SlotStatus public status;

    mapping(uint64 => CurrentProverReward) public proverReward;

    mapping(uint64 => uint256) public finalNumToBlock;

    // batchnum -> address -> deposit
    mapping(uint64 => mapping(address => uint256)) public batchNumToAddressPledge;

    gasFreeContractSet private gasFreeContracts;

    constructor() {
        status = SlotStatus.Created;
    }

    function initialize(address _manager, address _opsideSlots, address _globalPool) external virtual initializer {
        slotManager = _manager;
        opsideSlots = IOpsideSlots(_opsideSlots);
        globalPool = IGlobalPool(_globalPool);
        status = SlotStatus.Ready;
        punishAmount = 1000 ether;

        // Initialize OZ contracts
        __Ownable_init_unchained();

        status = SlotStatus.Running;

        emit Initialize(_manager, _opsideSlots, _globalPool);
    }

    modifier onlyOpsideSlots() {
        if (address(opsideSlots) != msg.sender) {
            revert OnlyOpsideSlots();
        }
        _;
    }

    modifier onlyManager() {
        if (slotManager != msg.sender) {
            revert OnlyManager();
        }
        _;
    }

    modifier onlyZkEvmContract() {
        if (zkEvmContract != msg.sender) {
            revert OnlyZkEvmContract();
        }
        _;
    }

    /**
     * @notice Set slot id
     */
     function setSlotId(uint256 _slotId) external onlyOpsideSlots {
        slotId = _slotId;
     }

    /**
     * @notice Get slotId
     */
     function getSlotId() external view returns (uint256) {
        return slotId;
     }

    /**
     * @notice Set the zkEvm contract
     */
    function setZKEvmContract(address _zkEvmContract) external onlyOwner {
        zkEvmContract = _zkEvmContract;
    }

    function getZKEvmContract() external view returns (address) {
        return zkEvmContract;
    }

    function setPunishAmount(uint256 amount) external onlyManager  {
        punishAmount = amount;
    }

    /**
     * @notice Start the bound chain
     */
    function start() external onlyManager {
        require(status == SlotStatus.Ready , "Invalid status");
        SlotStatus _slotStatus = opsideSlots.slotStatus(slotId);
        require(_slotStatus == SlotStatus.Running , "Invalid status");

        status = SlotStatus.Running;
    }

    /**
     * @notice Stop the bound chain
     */
    function stop() external onlyManager {
        require(status != SlotStatus.Stopped , "Invalid status");
        status = SlotStatus.Stopped;
    }

    /**
     * @notice Pause the bound chain
     */
    function pause() external onlyManager {
        require(status == SlotStatus.Running , "Invalid status");
        status = SlotStatus.Paused;
    }

    /**
     * @notice Unpause the bound chain
     */
    function unpause() external onlyManager {
        require(status == SlotStatus.Paused , "Invalid status");

        SlotStatus _slotStatus = opsideSlots.slotStatus(slotId);
        require(_slotStatus == SlotStatus.Running , "Invalid status");

        status = SlotStatus.Running;
    }

    function distributeRewards(address _recipient, uint64 _initNumBatch, uint64 _finalNewBatch, IDeposit _iDeposit) external onlyZkEvmContract {
        require(!Address.isContract(_recipient), "Account not EOA");
        require(opsideSlots.slotStatus(slotId) == SlotStatus.Running, "Slot must be running");
        require(status == SlotStatus.Running, "ZkEvm must be running");

        uint256 _deposit = batchNumToAddressPledge[_finalNewBatch][_recipient] / _UINT_AMOUNT;
        CurrentProverReward memory _currentReward = proverReward[_finalNewBatch];
        uint256 _totalDeposit = _currentReward.currentTotalDeposit / _UINT_AMOUNT;

        uint256 _count =  opsideSlots.getCommitCount(finalNumToBlock[_finalNewBatch]);
        require(_count != 0, "invaild reward");

        uint256 _reward = _deposit * (_currentReward.reward/_count) / _totalDeposit;

        opsideSlots.distributeReward(payable(_recipient), _initNumBatch, _finalNewBatch, _reward);

        emit DistributeRewards(slotId, address(this), _recipient, _reward);
    }

    /**
     * @notice Change slot manager
     */
    function changeSlotManager(address _manager) external onlyManager {
        slotManager = _manager;

        opsideSlots.setSlotManager(slotId, _manager);
    }

    function calcSlotReward(uint64 _batchNum, IDeposit _iDeposit) external onlyZkEvmContract {
        proverReward[_batchNum].reward = opsideSlots.getBlockReward(); // get reward per block
        opsideSlots.commitCount();
        finalNumToBlock[_batchNum] = block.number;
    }

    function calcCurrentTotalDeposit(uint64 _batchNum, IDeposit _iDeposit, address _account, bool _reset) external onlyZkEvmContract {
        if (_reset) {
            batchNumToAddressPledge[_batchNum][_account] = 0;
            proverReward[_batchNum].currentTotalDeposit = 0;
        } else {
            batchNumToAddressPledge[_batchNum][_account] = _iDeposit.depositOf(_account);
            proverReward[_batchNum].currentTotalDeposit += _iDeposit.depositOf(_account);
        }
    }

    function punish(address _recipient, IDeposit _iDeposit, uint256 _punishAmount) external onlyZkEvmContract {
        require(!Address.isContract(_recipient), "Account not EOA");

        _iDeposit.punish(_recipient, _punishAmount);

        emit PunishAmount(_recipient, _punishAmount);
    }

    function setPledgeStatus(address _account) external {
        IZKEVMContract zKEVMContract = IZKEVMContract(zkEvmContract);
        require(zKEVMContract.ideDeposit() == msg.sender, "only deposit");

        opsideSlots.setMinerToSlot(_account);
    }

    function getPledgeStatus(address _account) external view returns(uint256) {
        return opsideSlots.getMinerSlot(_account);
    }

    function delPledge(address _account) external {
        IZKEVMContract zKEVMContract = IZKEVMContract(zkEvmContract);
        require(zKEVMContract.ideDeposit() == msg.sender, "only deposit");
        opsideSlots.deleteMinerSlot(_account);
    }

    function addGasFreeContractBatch(address[] calldata addrList) external {
        require(slotManager == msg.sender || owner() == msg.sender, "only owner or manager");

        address _addr;
        for(uint32 i = 0; i < addrList.length; i++) {
            _addr = addrList[i];
            _freeContractAddOne(_addr);
        }
        emit UpdateGasFreeContractBatch(addrList, true);
    }

    function delGasFreeContractBatch(address[] calldata addrList) external {
        require(slotManager == msg.sender || owner() == msg.sender, "only owner or manager");

        address _addr;
        for(uint32 i = 0; i < addrList.length; i++) {
            _addr = addrList[i];
            _freeContractDelOne(_addr);
        }
        emit UpdateGasFreeContractBatch(addrList, false);
    }

    function getGasFreeContracts() public view returns (address[] memory){
        return gasFreeContracts._addressList;
    }

    function isGasFreeContract(address _addr) public view returns (bool) {
        return gasFreeContracts.inserted[_addr];
    }

    function _freeContractAddOne(address _addr) private {
        if (!gasFreeContracts.inserted[_addr]) { // addr not exists, add
            gasFreeContracts.inserted[_addr] = true;
            gasFreeContracts.indexOf[_addr] = gasFreeContracts._addressList.length;
            gasFreeContracts._addressList.push(_addr);
        }
    }

    function _freeContractDelOne(address _addr) private {
        if (gasFreeContracts.inserted[_addr]) {  // addr exists, delete
            delete gasFreeContracts.inserted[_addr];

            uint index = gasFreeContracts.indexOf[_addr];
            address lastKey = gasFreeContracts._addressList[gasFreeContracts._addressList.length - 1];

            gasFreeContracts.indexOf[lastKey] = index;
            delete gasFreeContracts.indexOf[_addr];

            gasFreeContracts._addressList[index] = lastKey;
            gasFreeContracts._addressList.pop();
        }
    }

    function getIDEToken() external view returns (address) {
        return opsideSlots.getIDEToken();
    }
}
