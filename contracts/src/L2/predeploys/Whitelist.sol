// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {OwnableBase} from "../../libraries/common/OwnableBase.sol";
import {IWhitelist} from "../../libraries/common/IWhitelist.sol";

contract Whitelist is OwnableBase, IWhitelist {
    /// @notice Emitted when account whitelist status changed.
    /// @param _account The address of account whose status is changed.
    /// @param _status The current whitelist status.
    event WhitelistStatusChanged(address indexed _account, bool _status);

    /// @notice Keep track whether the account is whitelisted.
    mapping(address => bool) private isWhitelisted;

    constructor(address _owner) {
        _transferOwnership(_owner);
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
