// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {ERC20} from "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import {ERC20Permit} from "@openzeppelin/contracts/token/ERC20/extensions/draft-ERC20Permit.sol";

// solhint-disable reason-string
// solhint-disable no-empty-blocks

/// @author Inspired by WETH9 (https://github.com/dapphub/ds-weth/blob/master/src/weth9.sol)
contract WrappedEther is ERC20Permit {
    /// @notice Emitted when user deposits Ether to this contract.
    /// @param dst The address of depositor.
    /// @param wad The amount of Ether in wei deposited.
    event Deposit(address indexed dst, uint256 wad);

    /// @notice Emitted when user withdraws some Ether from this contract.
    /// @param src The address of caller.
    /// @param wad The amount of Ether in wei withdrawn.
    event Withdrawal(address indexed src, uint256 wad);

    constructor() ERC20Permit("Wrapped Ether") ERC20("Wrapped Ether", "WETH") {}

    receive() external payable {
        deposit();
    }

    function deposit() public payable {
        address _sender = _msgSender();

        _mint(_sender, msg.value);

        emit Deposit(_sender, msg.value);
    }

    function withdraw(uint256 wad) external {
        address _sender = _msgSender();

        _burn(_sender, wad);

        (bool success, ) = _sender.call{value: wad}("");
        require(success, "withdraw ETH failed");

        emit Withdrawal(_sender, wad);
    }
}
