// SPDX-License-Identifier: AGPL-3.0

pragma solidity 0.8.16;

contract MockOpenRegistrar {
    uint256 public regId;

    function initialize(address _opsideSlots) external virtual {

    }

    function request(string calldata _name, address _manager, uint16 _period, uint256 _amount) external payable {
     
    }

    function accept(uint256 _regId) external {

    }

    function addRegistrant(address _registrant) external {

    }

    function setRent(uint16 _period, uint256 _amount) external {

    }
}
