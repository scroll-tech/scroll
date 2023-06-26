// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {AccessControlEnumerable} from "@openzeppelin/contracts/access/AccessControlEnumerable.sol";
import {EnumerableSet} from "@openzeppelin/contracts/utils/structs/EnumerableSet.sol";

// solhint-disable no-empty-blocks

contract ScrollOwner is AccessControlEnumerable {
    using EnumerableSet for EnumerableSet.Bytes32Set;

    /*************
     * Constants *
     *************/

    /// @notice The role for governance.
    bytes32 public constant GOVERNANCE_ROLE = keccak256("Governance");

    /// @notice The role for owner.
    bytes32 public constant OWNER_ROLE = keccak256("Owner");

    /*************
     * Variables *
     *************/

    /// @notice Mapping from target address to selector to the list of accessible roles.
    mapping(address => mapping(bytes4 => EnumerableSet.Bytes32Set)) private targetAccess;

    /**********************
     * Function Modifiers *
     **********************/

    modifier hasAccess(
        address _target,
        bytes4 _selector,
        bytes32 _role
    ) {
        require(targetAccess[_target][_selector].contains(_role), "no access");
        _;
    }

    /***************
     * Constructor *
     ***************/

    constructor() {
        _grantRole(DEFAULT_ADMIN_ROLE, msg.sender);
    }

    /*************************
     * Public View Functions *
     *************************/

    /// @notice Return a list of roles which has access to the function.
    /// @param _target The address of target contract.
    /// @param _selector The function selector to query.
    /// @return _roles The list of roles.
    function callable(address _target, bytes4 _selector) external view returns (bytes32[] memory _roles) {
        EnumerableSet.Bytes32Set storage _lists = targetAccess[_target][_selector];
        _roles = new bytes32[](_lists.length());
        for (uint256 i = 0; i < _roles.length; i++) {
            _roles[i] = _lists.at(i);
        }
    }

    /*****************************
     * Public Mutating Functions *
     *****************************/

    /// @notice Owner perform a function call.
    /// @param _target The address of target contract.
    /// @param _value The value passing to target contract.
    /// @param _data The calldata passing to target contract.
    function ownerExecute(
        address _target,
        uint256 _value,
        bytes calldata _data
    ) external payable {
        execute(_target, _value, _data, OWNER_ROLE);
    }

    /// @notice Governor perform a function call.
    /// @param _target The address of target contract.
    /// @param _value The value passing to target contract.
    /// @param _data The calldata passing to target contract.
    function governaceExecute(
        address _target,
        uint256 _value,
        bytes calldata _data
    ) external payable {
        execute(_target, _value, _data, GOVERNANCE_ROLE);
    }

    /// @notice Perform a function call from arbitrary role.
    /// @param _target The address of target contract.
    /// @param _value The value passing to target contract.
    /// @param _data The calldata passing to target contract.
    /// @param _role The expected role of the caller.
    function execute(
        address _target,
        uint256 _value,
        bytes calldata _data,
        bytes32 _role
    ) public payable onlyRole(_role) hasAccess(_target, bytes4(_data[0:4]), _role) {
        _execute(_target, _value, _data);
    }

    // allow others to send ether to this contract.
    receive() external payable {}

    /************************
     * Restricted Functions *
     ************************/

    /// @notice Update the access to target contract.
    /// @param _target The address of target contract.
    /// @param _selectors The list of function selectors to update.
    /// @param _role The role to change.
    /// @param _status True if we are going to add the role, otherwise remove the role.
    function updateAccess(
        address _target,
        bytes4[] memory _selectors,
        bytes32 _role,
        bool _status
    ) external onlyRole(DEFAULT_ADMIN_ROLE) {
        if (_status) {
            for (uint256 i = 0; i < _selectors.length; i++) {
                targetAccess[_target][_selectors[i]].add(_role);
            }
        } else {
            for (uint256 i = 0; i < _selectors.length; i++) {
                targetAccess[_target][_selectors[i]].remove(_role);
            }
        }
    }

    /**********************
     * Internal Functions *
     **********************/

    /// @dev Internal function to call contract. If the call reverted, the error will be popped up.
    /// @param _target The address of target contract.
    /// @param _value The value passing to target contract.
    /// @param _data The calldata passing to target contract.
    function _execute(
        address _target,
        uint256 _value,
        bytes calldata _data
    ) internal {
        // solhint-disable-next-line avoid-low-level-calls
        (bool success, ) = address(_target).call{value: _value}(_data);
        if (!success) {
            // solhint-disable-next-line no-inline-assembly
            assembly {
                let ptr := mload(0x40)
                let size := returndatasize()
                returndatacopy(ptr, 0, size)
                revert(ptr, size)
            }
        }
    }
}
