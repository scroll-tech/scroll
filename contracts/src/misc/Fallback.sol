// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {IERC20} from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import {SafeERC20} from "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";

contract Fallback is Ownable {
    using SafeERC20 for IERC20;

    /// @notice Withdraw stucked token from this contract.
    /// @param _token The address of token to withdraw, use `address(0)` if withdraw ETH.
    /// @param _amount The amount of token to withdraw.
    /// @param _recipient The address of receiver.
    function withdraw(
        address _token,
        uint256 _amount,
        address _recipient
    ) external onlyOwner {
        if (_token == address(0)) {
            (bool _success, ) = _recipient.call{value: _amount}("");
            require(_success, "transfer ETH failed");
        } else {
            IERC20(_token).safeTransfer(_recipient, _amount);
        }
    }

    /// @notice Execute an arbitrary message.
    /// @param _target The address of contract to call.
    /// @param _data The calldata passed to target contract.
    function execute(address _target, bytes calldata _data) external payable onlyOwner {
        (bool _success, ) = _target.call{value: msg.value}(_data);
        require(_success, "call failed");
    }
}
