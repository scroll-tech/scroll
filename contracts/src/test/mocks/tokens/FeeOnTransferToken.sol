// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {MockERC20} from "solmate/test/utils/mocks/MockERC20.sol";

// solhint-disable no-empty-blocks

contract FeeOnTransferToken is MockERC20 {
    uint256 private feeRate;

    constructor(
        string memory _name,
        string memory _symbol,
        uint8 _decimals
    ) MockERC20(_name, _symbol, _decimals) {}

    function setFeeRate(uint256 _feeRate) external payable {
        feeRate = _feeRate;
    }

    function transfer(address to, uint256 amount) public virtual override returns (bool) {
        balanceOf[msg.sender] -= amount;

        uint256 fee = (amount * feeRate) / 1e9;
        amount -= fee;

        // Cannot overflow because the sum of all user
        // balances can't exceed the max uint256 value.
        unchecked {
            balanceOf[to] += amount;
        }

        emit Transfer(msg.sender, to, amount);

        return true;
    }

    function transferFrom(
        address from,
        address to,
        uint256 amount
    ) public virtual override returns (bool) {
        uint256 allowed = allowance[from][msg.sender]; // Saves gas for limited approvals.

        if (allowed != type(uint256).max) allowance[from][msg.sender] = allowed - amount;

        balanceOf[from] -= amount;

        uint256 fee = (amount * feeRate) / 1e9;
        amount -= fee;

        // Cannot overflow because the sum of all user
        // balances can't exceed the max uint256 value.
        unchecked {
            balanceOf[to] += amount;
        }

        emit Transfer(from, to, amount);

        return true;
    }
}
