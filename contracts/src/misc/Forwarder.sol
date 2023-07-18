// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";

// solhint-disable no-inline-assembly

/// @notice This contract is designed as a proxy contract to forward contract call to some other
/// target contract. There are two roles in the contract: the `owner` is the super admin and the
/// `admin`. The `owner` can change the address of the `admin`. Both roles can forward contract call.
contract Forwarder is Ownable {
    /**********
     * Events *
     **********/

    /// @notice Emitted when forward call happens.
    /// @param caller The address of the caller.
    /// @param target The address of the target contract.
    /// @param value The value passed to the target contract.
    /// @param data The calldata passed to the target contract.
    event Forwarded(address indexed caller, address indexed target, uint256 value, bytes data);

    /// @notice Emitted when the address of admin is updated.
    /// @param oldAdmin The address of the old admin.
    /// @param newAdmin The address of the new admin.
    event SetAdmin(address indexed oldAdmin, address indexed newAdmin);

    /*************
     * Variables *
     *************/

    /// @notice The address of contract admin.
    address public admin;

    /***************
     * Constructor *
     ***************/

    constructor(address _admin) {
        admin = _admin;

        emit SetAdmin(address(0), _admin);
    }

    /************************
     * Restricted Functions *
     ************************/

    /// @notice Update the address of admin.
    /// @param _newAdmin The address of the new admin.
    function setAdmin(address _newAdmin) external onlyOwner {
        address _oldAdmin = admin;
        admin = _newAdmin;

        emit SetAdmin(_oldAdmin, _newAdmin);
    }

    /// @notice Forward calldata to some target contract.
    /// @param _target The address of the target contract.
    /// @param _data The data forwarded to the target contract.
    function forward(address _target, bytes calldata _data) external payable {
        require(msg.sender == owner() || msg.sender == admin, "only owner or admin");

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

        emit Forwarded(msg.sender, _target, msg.value, _data);
    }
}
