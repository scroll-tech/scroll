// SPDX-License-Identifier: AGPL-3.0

pragma solidity 0.8.17;

interface IDeposit {
    function deposit(uint256 amount) external payable;
    function withdraw(uint256 amount) external;
    function depositOf(address account) external view returns(uint256);
    function punish(address account, uint256 amount) external;
    function totalDeposits() external view returns(uint256);
}
