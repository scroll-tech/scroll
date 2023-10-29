// SPDX-License-Identifier: AGPL-3.0

pragma solidity 0.8.16;

contract MockMinerDeposit {

    function initialize() external virtual {

    }


    function deposit(uint256 amount) external payable {
       
    }

    function punish(address account, uint256 amount) external {

    }

    function setSlotAdapter(address _slotAdapter) external {

    }
    function depositOf(address account) external view returns(uint256){
        return 10000 ether;
    }

    fallback() external payable {

    }

    receive() external payable {
        
    }
}