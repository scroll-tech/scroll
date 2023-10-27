// SPDX-License-Identifier: AGPL-3.0

pragma solidity 0.8.16;

import { Address } from "@openzeppelin/contracts/utils/Address.sol";
import { OwnableUpgradeable } from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";

import { IDeposit } from "./interfaces/IDeposit.sol";

contract IDEDeposit is IDeposit, OwnableUpgradeable {
    event Deposit(address indexed user, uint256 amount);
    event Withdraw(address indexed user, uint256 amount);

    uint256 public totalDeposits;

    mapping(address => bool) public slotAdapters;
    mapping(address => uint256) public depositAmounts;
    mapping(address => uint256) public punishAmounts;

    uint private _locked;
    
    modifier lock() {
        require(_locked == 0, "LOCKED");
        _locked = 1;
        _;
        _locked = 0;
    }

    function initialize() external virtual initializer {
        // Initialize OZ contracts
        __Ownable_init_unchained();
    }


    function deposit(uint256 amount) external payable lock {
        require(msg.value > 0, "zero");

        totalDeposits = totalDeposits + msg.value;

        depositAmounts[msg.sender] += msg.value;
        emit Deposit(msg.sender, msg.value);
    }


    function withdraw(uint256 amount) external lock {
        require(amount > 0, "zero");
        require(amount <= depositAmounts[msg.sender], "Invalid amount");
        require(!Address.isContract(msg.sender), "Account not EOA");

        // update
        totalDeposits = totalDeposits - amount;
        depositAmounts[msg.sender] -= amount;
        // transfer
        (bool success, ) = msg.sender.call{ value: amount }(new bytes(0));
        require(success, "withdrawal: transfer failed");
        emit Withdraw(msg.sender, amount);
    }

    function punish(address account, uint256 amount) external lock {
        require(slotAdapters[msg.sender], "Not slotAdapter");
        depositAmounts[account] -= amount; // TODO
        punishAmounts[account] += amount;
    }

    function depositOf(address account) external view returns(uint256){
        return depositAmounts[account];
    }

    function setSlotAdapter(address _slotAdapter) external onlyOwner {
        require(Address.isContract(_slotAdapter), "Account EOA");
        slotAdapters[_slotAdapter] = true;
    }
}