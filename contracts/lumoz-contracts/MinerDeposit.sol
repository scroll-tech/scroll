// SPDX-License-Identifier: AGPL-3.0

pragma solidity 0.8.17;

import { Address } from "@openzeppelin/contracts/utils/Address.sol";
import { OwnableUpgradeable } from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import { SafeERC20Upgradeable } from "@openzeppelin/contracts-upgradeable/token/ERC20/utils/SafeERC20Upgradeable.sol";
import "@openzeppelin/contracts-upgradeable/token/ERC20/extensions/IERC20MetadataUpgradeable.sol";

import { IDeposit } from "./interfaces/IDeposit.sol";
import { ISlotAdapter } from "./interfaces/ISlotAdapter.sol";
import { IZKEVMContract } from "./interfaces/IZKEVMContract.sol";

contract MinerDeposit is IDeposit, OwnableUpgradeable {
    using SafeERC20Upgradeable for IERC20Upgradeable;

    struct PunishAmount {
        uint256 expect;
        uint256 real;
    }
    event Deposit(address indexed user, uint256 amount);
    event Withdraw(address indexed user, uint256 amount);

    uint256 public totalDeposits;

    ISlotAdapter public slotAdapter;
    mapping(address => uint256) public depositAmounts;
    mapping(address => PunishAmount) public punishAmounts;

    uint private _locked;

    function initialize() external virtual initializer {
        // Initialize OZ contracts
        __Ownable_init_unchained();
    }


    function deposit(uint256 amount) external payable {
        address _token = slotAdapter.getIDEToken();
        if (_token == address(0)) {
            amount = msg.value;
        } else {
            IERC20Upgradeable(_token).safeTransferFrom(msg.sender, address(this), amount);
        }

        require(amount > 0, "zero");
        
        uint256 _slotId = slotAdapter.getPledgeStatus(msg.sender);
        if (_slotId != 0) {
            require(slotAdapter.getSlotId() == _slotId, "invalid deposit");
        }

        totalDeposits = totalDeposits + amount;

        depositAmounts[msg.sender] += amount;
        slotAdapter.setPledgeStatus(msg.sender);
        emit Deposit(msg.sender, amount);
    }


    function withdraw(uint256 amount) external {
        require(!Address.isContract(msg.sender), "Account not EOA");
        // settle
        IZKEVMContract zKEVMContract = IZKEVMContract(slotAdapter.getZKEvmContract());

        zKEVMContract.settle(msg.sender);

        // transfer
        uint256 _depositAmount = depositAmounts[msg.sender];
        if (_depositAmount < amount) {
            amount = _depositAmount;
            _depositAmount = 0;
        } else {
            _depositAmount -= amount;
        }
        depositAmounts[msg.sender] = _depositAmount;
        totalDeposits = totalDeposits - amount;

        address _token = slotAdapter.getIDEToken();
        if (_token == address(0)) {
            (bool success, ) = msg.sender.call{ value: amount }(new bytes(0));
            require(success, "withdrawal: transfer failed");
        } else {
            IERC20Upgradeable(_token).safeTransfer(msg.sender, amount);
        }

        if (_depositAmount == 0 && slotAdapter.getSlotId() == slotAdapter.getPledgeStatus(msg.sender)) {
            slotAdapter.delPledge(msg.sender);
        }

        emit Withdraw(msg.sender, amount);
    }

    function punish(address account, uint256 amount) external {
        require(msg.sender == address(slotAdapter), "Not slotAdapter");
        uint256 _depositAmount = depositAmounts[account];
        uint256 _real = amount;
        if (_depositAmount < amount) {
            _real = _depositAmount;
            _depositAmount = 0;
        } else {
            _depositAmount -= amount;
        }
        depositAmounts[account] = _depositAmount;
        punishAmounts[account].real += _real;
        punishAmounts[account].expect += amount;
    }

    function depositOf(address account) external view returns(uint256){
        return depositAmounts[account];
    }

    function setSlotAdapter(ISlotAdapter _slotAdapter) external onlyOwner {
        require(Address.isContract(address(_slotAdapter)), "Account EOA");
        slotAdapter = _slotAdapter;
    }
}