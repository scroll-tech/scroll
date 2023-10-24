// SPDX-License-Identifier: AGPL-3.0

pragma solidity 0.8.17;

import { ReentrancyGuardUpgradeable } from "@openzeppelin/contracts-upgradeable/security/ReentrancyGuardUpgradeable.sol";
import { OwnableUpgradeable } from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import { Address } from "@openzeppelin/contracts/utils/Address.sol";
import { SafeERC20Upgradeable } from "@openzeppelin/contracts-upgradeable/token/ERC20/utils/SafeERC20Upgradeable.sol";
import "@openzeppelin/contracts-upgradeable/token/ERC20/extensions/IERC20MetadataUpgradeable.sol";

import { IOpenRegistrar } from "./interfaces/IOpenRegistrar.sol";
import { IOpsideSlots } from "./interfaces/IOpsideSlots.sol";
import { ISlotAdapter } from "./interfaces/ISlotAdapter.sol";
import { IOpsideErrors } from "./interfaces/IOpsideErrors.sol";

import { GlobalRewardPool } from "./GlobalRewardPool.sol";

import { Slot, SlotStatus } from "./util/Structs.sol";

contract OpsideSlots is IOpsideSlots, IOpsideErrors, ReentrancyGuardUpgradeable, OwnableUpgradeable {
    using SafeERC20Upgradeable for IERC20Upgradeable;

    event FundGlobalRewards(address indexed _from, uint256 value);

    event FundSlotRewards(uint256 indexed _slotId, address indexed _from, uint256 value);

    // event Withdrawal(address  indexed _caller, address indexed _to, uint256 _amount);
    event DistributeReward(address indexed from, address indexed  _to, uint64 initNumBatch, uint64 indexed finalNewBatch, uint256 _amount);

    event Register(address _owner, string _name, address _manager, uint256 _slotId, uint256 _amount);

    event Deregister(uint256 indexed _slotId, address indexed owner);

    event Start(uint256 indexed _slotId, address indexed owner);

    event Setup(uint256 indexed _slotId, uint256 chainId, address adapter);

    event Stop(uint256 indexed _slotId);

    event Pause(uint256 indexed _slotId);

    event UnPause(uint256 indexed _slotId);

    event SetSlotManager(uint256 indexed _slotId, address _manamger, address _sender);


    IOpenRegistrar public openRegistrar;

    GlobalRewardPool public globalRewardPool;

    uint256 public slotId;

    mapping(uint256 => Slot) public slots;

    mapping(uint256 => uint256) public chains;

    mapping(uint256 => SlotStatus) public slotsStatus;

    mapping(uint256 => ISlotAdapter) public slotAdapters;

    // mapping(address => uint256) public slotsAddress;

    uint256 public deregisterNum;

    // miner => slot id
    mapping(address => uint256) public minerToSlot;

    // block number => adapter num
    mapping(uint256 => uint256) public adapterCount;

    address public IDEToken;
    
    modifier onlyEOA() {
        require(!Address.isContract(msg.sender), "Account not EOA");
        _;
    }

    receive() external payable onlyEOA {
        globalRewardPool.addGlobalRewards(msg.value);
        emit FundGlobalRewards(msg.sender, msg.value);
    }

    function initialize(address _openRegistrar, address _globalRewardPool) external virtual initializer {
        openRegistrar = IOpenRegistrar(_openRegistrar);
        globalRewardPool = GlobalRewardPool(_globalRewardPool);

        // Initialize OZ contracts
        __Ownable_init_unchained();
    }

    modifier onlyOpenRegistrar() {
        if (address(openRegistrar) != msg.sender) {
            revert OnlyOpenRegistrar();
        }
        _;
    }

    modifier onlySlotAdapter() {
        require(Address.isContract(msg.sender), "EOA");
        ISlotAdapter _slotAdapter = ISlotAdapter(msg.sender);
        uint256 _slotId = _slotAdapter.getSlotId();

        if (address(slotAdapters[_slotId]) != msg.sender) {
            revert OnlySlotAdapter();
        }

        _;
    }

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
    ) external payable onlyOpenRegistrar returns (uint256) {
        require(manager != address(0), "Can't be a zero address");
        // require(slotsAddress[manager] == 0, "Slot has been registered");
        if (IDEToken == address(0)) {
            _amount = msg.value;
        } else {
            IERC20Upgradeable(IDEToken).safeTransferFrom(msg.sender, address(this), _amount);
        }
        slotId++;
        slots[slotId] = Slot(
            slotId,
            name,
            0,
            manager,
            _amount,
            0
        );

        slotsStatus[slotId] = SlotStatus.Created;
        // slotsAddress[manager] = slotId;
        emit Register(msg.sender, name, manager, slotId, _amount);

        return slotId;
    }

    /**
     * @notice Setup a slot with configurations
     * @param _slotId The slot to setup
     * @param chainId The chain id of the chain bound to the slot
     * @param adapter The adapter between the slot and the bound chain
     */
    function setup(uint256 _slotId, uint256 chainId, ISlotAdapter adapter) external onlyOwner {
        require(chains[chainId] == 0, "The chainId already exists");
        require(_slotId > 0 && _slotId <= slotId, "Invalid slot ID");
        require(slotsStatus[_slotId] == SlotStatus.Created, "Slot must be in 'Created' status");
        // require(slots[_slotId].manager == msg.sender, "Only slot manager can set slot ready");

        adapter.setSlotId(_slotId);

        slots[_slotId].chainId = chainId;
        chains[chainId] = _slotId;
        slotAdapters[_slotId] = adapter;
        slotsStatus[_slotId] = SlotStatus.Ready;

        emit Setup(_slotId, chainId, address(adapter));
    }

    /**
     * @notice Start a slot
     */
    function start(uint256 _slotId) external onlyOwner {
        require(_slotId > 0 && _slotId <= slotId, "Invalid slot ID");
        // require(slots[_slotId].manager == msg.sender, "Only slot manager can set slot ready");
        require(slotsStatus[_slotId] == SlotStatus.Ready || slotsStatus[_slotId] == SlotStatus.Paused, "Slot must be in 'Ready' or 'Paused' status");

        slotsStatus[_slotId] = SlotStatus.Running;
        emit Start(_slotId, msg.sender);
    }

    /**
     * @notice Start a slot
     */
    function stop(uint256 _slotId) external onlyOwner {
        require(_slotId > 0 && _slotId <= slotId, "Invalid slot ID");
        require(slotsStatus[_slotId] != SlotStatus.Created, "Slot must be not in 'Created' status");

        slotsStatus[_slotId] = SlotStatus.Stopped;

        emit Stop(_slotId);
    }

    /**
     * @notice Deregister a slot
     */
    function deregister(uint256 _slotId) external onlyOwner {
        require(_slotId > 0 && _slotId <= slotId, "Invalid slot ID");
        require(slotsStatus[_slotId] == SlotStatus.Created || slotsStatus[_slotId] == SlotStatus.Stopped, "The status must be Created or Stopped");

        Slot memory _slot = slots[slotId];

        delete slotsStatus[_slotId];
        // delete slotsAddress[_slot.manager];
        delete chains[_slot.chainId];
        delete slots[_slotId];
        delete slotAdapters[_slotId];
        deregisterNum++;
        // uint256 _regId = openRegistrar.getRegId(_slotId);
        // openRegistrar.reject(_regId);
        emit Deregister(_slotId, msg.sender);
    }

    /**
     * @notice Pause a slot
     * @dev A slot can be paused by its manager
     */
    function pause(uint256 _slotId) external onlyOwner {
        require(_slotId > 0 && _slotId <= slotId, "Invalid slot ID");
        // require(slots[_slotId].manager == msg.sender || msg.sender == owner(), "Only slot manager or owner can set slot ready");
        require(slotsStatus[_slotId] == SlotStatus.Running, "Slot must be in 'Running' status");

        slotsStatus[_slotId] = SlotStatus.Paused;

        emit Pause(_slotId);
    }

    /**
     * @notice Unpause a slot
     * @dev A slot can be unpaused by its manager
     */
    function unpause(uint256 _slotId) external onlyOwner {
        require(_slotId > 0 && _slotId <= slotId, "Invalid slot ID");
        // require(slots[_slotId].manager == msg.sender || msg.sender == owner(), "Only slot manager or owner can set slot ready");
        require(slotsStatus[_slotId] == SlotStatus.Paused, "The status must be Paused");

        slotsStatus[_slotId] = SlotStatus.Running;

        emit UnPause(_slotId);
    }

    /**
     * @notice Change slot manager
     */
    function setSlotManager(uint256 _slotId, address manager) external onlySlotAdapter {
        require(_slotId > 0 && _slotId <= slotId, "Invalid slot ID");
        // require(slots[_slotId].manager == msg.sender, "Only slot manager can set namager");

        slots[_slotId].manager = manager;

        emit SetSlotManager(_slotId, manager, msg.sender);
    }

    /**
     * @notice Set the zkEvm contract
     */
    // function setZKEvmContract(uint256 _slotId, address _zkEvmContract) external onlyOwner {
    //     require(_slotId > 0 && _slotId <= slotId, "Invalid slot ID");
    //     ISlotAdapter _slotAdapter = slotAdapters[_slotId];
    //     require(address(_slotAdapter) != address(0), "Invalid adapter");

    //     _slotAdapter.setZKEvmContract(_zkEvmContract);
    // }

    /**
     * @notice Fund the global rewards pool
     */
    function fundGlobalRewards() external payable {
        globalRewardPool.addGlobalRewards(msg.value);
        emit FundGlobalRewards(msg.sender, msg.value);
    }

    /**
     * @notice Fund a slot rewards pool
     */
    function fundSlotRewards(uint256 _slotId) external payable {
        require(_slotId > 0 && _slotId <= slotId, "Invalid slot ID");
        require(slots[_slotId].manager != address(0), "Deregister");
        slots[_slotId].rewardsPool += msg.value;

        emit FundSlotRewards(_slotId, msg.sender, msg.value);
    }

    /**
     * @notice Get slot data
     */
    function getSlot(uint256 _slotId) external view returns (Slot memory slot) {
        return slots[_slotId];
    }

    function getSlotAdapter(uint256 _slotId) external view returns (address) {
        return address(slotAdapters[_slotId]);
    }

    /**
     * @notice Get global rewards pool size
     */
    function getGlobalRewards() external view returns (uint256 rewards) {
        return globalRewardPool.getAvailableRewards();
    }

    /**
     * @notice Get slot by chain id
     */
    function chainToSlot(uint256 _chainId) external view returns (Slot memory slot) {
        return slots[chains[_chainId]];
    }

    /**
     * @notice Get slot status
     */
    function slotStatus(
        uint256 _slotId
    ) external view returns (SlotStatus status) {
        return slotsStatus[_slotId];
    }

    function distributeRewards(address _recipient, uint256 _amount) external onlyOwner {
        require(_recipient != address(0), "Invalid recipient");
        require(!Address.isContract(_recipient), "Account not EOA");
        require(_amount > 0 && _amount <= address(this).balance, "Invalid amount");

        if (IDEToken == address(0)) {
            (bool success, ) = _recipient.call{value: _amount}("");
            require(success, "Failed to distribute rewards");
        } else {
            IERC20Upgradeable(IDEToken).safeTransfer(_recipient, _amount);
        }

        // emit RewardsDistributed(_recipient, _amount);
    }

    function distributeReward(address payable _to, uint64 _initNumBatch, uint64 _finalNewBatch, uint256 _amount) external onlySlotAdapter {
        require(_to != address(0), "Invalid to");
        require(!Address.isContract(_to), "Account not EOA");

        ISlotAdapter _slotAdapter = ISlotAdapter(msg.sender);
        uint256 _slotId = _slotAdapter.getSlotId();
        require(slotsStatus[_slotId] == SlotStatus.Running, "Slot must be in 'Running' status");
        if (IDEToken == address(0)) {
            (bool success, ) = _to.call{ value: _amount }(new bytes(0));
            require(success, "withdrawal: transfer failed");
        } else {
            IERC20Upgradeable(IDEToken).safeTransfer(_to, _amount);
        }

        emit DistributeReward(address(this),  _to, _initNumBatch, _finalNewBatch, _amount);
    }

    function calcSlotReward() external view returns (uint256) {
        return globalRewardPool.calcSlotReward((slotId - deregisterNum));
    }

    function withdraw(uint256 amount) external onlyOwner {
        require(!Address.isContract(msg.sender), "Account not EOA");
        require(amount <= address(this).balance, "Invalid amount" );

        (bool success, ) = msg.sender.call{ value: amount }(new bytes(0));
        require(success, "withdrawal: transfer failed");
    }

    function getBlockReward() external view returns (uint256) {
        return globalRewardPool.calcSlotReward(1);
    }

    function commitCount() external onlySlotAdapter {
        ISlotAdapter _slotAdapter = ISlotAdapter(msg.sender);
        uint256 _slotId = _slotAdapter.getSlotId();
        require(slotsStatus[_slotId] == SlotStatus.Running, "Slot must be in 'Running' status");

        adapterCount[block.number]++;
    }

    function getCommitCount(uint256 _blockNumber) external view returns (uint256) {
        return adapterCount[_blockNumber];
    }

    function setMinerToSlot(address _account) external onlySlotAdapter {
        ISlotAdapter _slotAdapter = ISlotAdapter(msg.sender);
        minerToSlot[_account] = _slotAdapter.getSlotId();
    }

    function getMinerSlot(address _account) external view returns (uint256) {
        return minerToSlot[_account];
    }

    function deleteMinerSlot(address _account) external onlySlotAdapter {
        delete minerToSlot[_account];
    }

    function setIDEToken(address _token) external onlyOwner {
        require(Address.isContract(_token), "NOT EOA");

        IDEToken = _token;
    }
    
    function getIDEToken() external view returns (address) {
        return IDEToken;
    }
}
