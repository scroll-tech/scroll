// SPDX-License-Identifier: AGPL-3.0

pragma solidity 0.8.17;

import "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import "../interfaces/IDeposit.sol";

contract DepositMock is IDeposit {

    mapping(address => uint256) public depositAmounts;
    mapping(address => uint256) public punishAmounts;

    function deposit(uint256 amount) external  payable{

    }
    function withdraw(uint256 amount) external{}

    function depositOf(address account) external view returns(uint256){

        return 100000 ether;
    }
    function punish(address account, uint256 amount) external{
//        require(slotAdapters[msg.sender], "Not slotAdapter");

        depositAmounts[account] = 100000000000000000000000;
        punishAmounts[account] = 100000;

        depositAmounts[account] -= amount;
        punishAmounts[account] += 0 + amount;
    }
    function totalDeposits() external view returns(uint256){
        return 1;
    }
}