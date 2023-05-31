// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

contract Forwarder {

    address public admin;
    address public superAdmin;

    event Forwarded(address indexed target, uint256 value, bytes data);
    event SetAdmin(address indexed admin);
    event SetSuperAdmin(address indexed superAdmin);

    constructor(address _admin, address _superAdmin) {
        admin = _admin;
        superAdmin = _superAdmin;
    }

    function setAdmin(address _admin) public {
        require(msg.sender == superAdmin, "only superAdmin");
        admin = _admin;
        emit SetAdmin(_admin);
    }

    function setSuperAdmin(address _superAdmin) public {
        require(msg.sender == superAdmin, "only superAdmin");
        superAdmin = _superAdmin;
        emit SetSuperAdmin(_superAdmin);
    }

    function forward(address _target, bytes memory _data) public payable {
        require(msg.sender == superAdmin || msg.sender == admin, "only admin or superAdmin");
        (bool success, ) = _target.call{value: msg.value}(_data);
        // bubble up revert reason
        if (!success) {
            assembly {
                let ptr := mload(0x40)
                let size := returndatasize()
                returndatacopy(ptr, 0, size)
                revert(ptr, size)
            }
        }
        emit Forwarded(_target, msg.value, _data);
    }
}
