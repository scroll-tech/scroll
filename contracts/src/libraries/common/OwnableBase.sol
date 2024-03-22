// SPDX-License-Identifier: MIT

pragma solidity ^0.8.24;

abstract contract OwnableBase {
    /**********
     * Events *
     **********/

    /// @notice Emitted when owner is changed by current owner.
    /// @param _oldOwner The address of previous owner.
    /// @param _newOwner The address of new owner.
    event OwnershipTransferred(address indexed _oldOwner, address indexed _newOwner);

    /*************
     * Variables *
     *************/

    /// @notice The address of the current owner.
    address public owner;

    /**********************
     * Function Modifiers *
     **********************/

    /// @dev Throws if called by any account other than the owner.
    modifier onlyOwner() {
        require(owner == msg.sender, "caller is not the owner");
        _;
    }

    /************************
     * Restricted Functions *
     ************************/

    /// @notice Leaves the contract without owner. It will not be possible to call
    /// `onlyOwner` functions anymore. Can only be called by the current owner.
    ///
    /// @dev Renouncing ownership will leave the contract without an owner,
    /// thereby removing any functionality that is only available to the owner.
    function renounceOwnership() public onlyOwner {
        _transferOwnership(address(0));
    }

    /// @notice Transfers ownership of the contract to a new account (`newOwner`).
    /// Can only be called by the current owner.
    function transferOwnership(address _newOwner) public onlyOwner {
        require(_newOwner != address(0), "new owner is the zero address");
        _transferOwnership(_newOwner);
    }

    /**********************
     * Internal Functions *
     **********************/

    /// @dev Transfers ownership of the contract to a new account (`newOwner`).
    /// Internal function without access restriction.
    function _transferOwnership(address _newOwner) internal {
        address _oldOwner = owner;
        owner = _newOwner;
        emit OwnershipTransferred(_oldOwner, _newOwner);
    }
}
