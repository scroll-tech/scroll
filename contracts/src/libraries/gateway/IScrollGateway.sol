// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

interface IScrollGateway {
  function counterpart() external view returns (address);

  function finalizeDropMessage() external payable;
}
