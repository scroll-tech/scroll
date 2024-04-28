// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {MockERC20} from "solmate/test/utils/mocks/MockERC20.sol";

// solhint-disable no-empty-blocks

contract RevertOnTransferToken is MockERC20 {
    bool private revertOnTransfer;
    bool private transferReturn;

    constructor(
        string memory _name,
        string memory _symbol,
        uint8 _decimals
    ) MockERC20(_name, _symbol, _decimals) {
        transferReturn = true;
    }

    function setRevertOnTransfer(bool _revertOnTransfer) external payable {
        revertOnTransfer = _revertOnTransfer;
    }

    function setTransferReturn(bool _transferReturn) external payable {
        transferReturn = _transferReturn;
    }

    function transfer(address to, uint256 amount) public virtual override returns (bool) {
        if (revertOnTransfer) revert();
        if (!transferReturn) return false;

        balanceOf[msg.sender] -= amount;

        // Cannot overflow because the sum of all user
        // balances can't exceed the max uint256 value.
        unchecked {
            balanceOf[to] += amount;
        }

        emit Transfer(msg.sender, to, amount);

        return true;
    }
}
