// SPDX-License-Identifier: AGPL-3.0

pragma solidity 0.8.17;

interface IZKEVMContract {
    function settle(address _account) external;
    function ideDeposit() external view returns(address);
}