// File: src/libraries/common/OwnableBase.sol

// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

abstract contract OwnableBase {
  /**************************************** Events ****************************************/

  /// @notice Emitted when owner is changed by current owner.
  /// @param _oldOwner The address of previous owner.
  /// @param _newOwner The address of new owner.
  event OwnershipTransferred(address indexed _oldOwner, address indexed _newOwner);

  /**************************************** Variables ****************************************/

  /// @notice The address of the current owner.
  address public owner;

  /// @dev Throws if called by any account other than the owner.
  modifier onlyOwner() {
    require(owner == msg.sender, "caller is not the owner");
    _;
  }

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

  /// @dev Transfers ownership of the contract to a new account (`newOwner`).
  /// Internal function without access restriction.
  function _transferOwnership(address _newOwner) internal {
    address _oldOwner = owner;
    owner = _newOwner;
    emit OwnershipTransferred(_oldOwner, _newOwner);
  }
}

// File: src/libraries/common/IWhitelist.sol



pragma solidity ^0.8.0;

interface IWhitelist {
  /// @notice Check whether the sender is allowed to do something.
  /// @param _sender The address of sender.
  function isSenderAllowed(address _sender) external view returns (bool);
}

// File: src/L2/predeploys/Whitelist.sol



pragma solidity ^0.8.0;


contract Whitelist is OwnableBase, IWhitelist {
  /// @notice Emitted when account whitelist status changed.
  /// @param _account The address of account whose status is changed.
  /// @param _status The current whitelist status.
  event WhitelistStatusChanged(address indexed _account, bool _status);

  /// @notice Keep track whether the account is whitelisted.
  mapping(address => bool) private isWhitelisted;

  constructor(address _owner) {
    owner = _owner;
  }

  /// @notice See {IWhitelist-isSenderAllowed}
  function isSenderAllowed(address _sender) external view returns (bool) {
    return isWhitelisted[_sender];
  }

  /// @notice Update the whitelist status
  /// @param _accounts The list of addresses to update.
  /// @param _status The whitelist status to update.
  function updateWhitelistStatus(address[] memory _accounts, bool _status) external onlyOwner {
    for (uint256 i = 0; i < _accounts.length; i++) {
      isWhitelisted[_accounts[i]] = _status;
      emit WhitelistStatusChanged(_accounts[i], _status);
    }
  }
}
