// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

import { ScrollChain } from "../../L1/rollup/ScrollChain.sol";

contract MockScrollChain is ScrollChain {
  constructor() ScrollChain(0, 4, 0xb5baa665b2664c3bfed7eb46e00ebc110ecf2ebd257854a9bf2b9dbc9b2c08f6) {}

  function computePublicInputHash(uint64 accTotalL1Messages, Batch memory batch)
    external
    view
    returns (
      bytes32,
      uint64,
      uint64,
      uint64
    )
  {
    return _computePublicInputHash(accTotalL1Messages, batch);
  }
}
