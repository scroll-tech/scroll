// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import {IERC20} from "@openzeppelin/contracts/token/ERC20/IERC20.sol";

// solhint-disable no-empty-blocks

contract MockGasSwapTarget {
    address public token;

    uint256 public amountIn;

    receive() external payable {}

    function setToken(address _token) external {
        token = _token;
    }

    function setAmountIn(uint256 _amountIn) external {
        amountIn = _amountIn;
    }

    function swap() external {
        IERC20(token).transferFrom(msg.sender, address(this), amountIn);
        (bool success, ) = msg.sender.call{value: address(this).balance}("");
        require(success, "transfer ETH failed");
    }
}
