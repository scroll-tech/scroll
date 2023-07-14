// SPDX-License-Identifier: MIT

pragma solidity =0.8.16;

import {FeeVault} from "../../libraries/FeeVault.sol";

/// @title L2TxFeeVault
/// @notice The `L2TxFeeVault` contract collects all L2 transaction fees and allows withdrawing these fees to a predefined L1 address.
/// The minimum withdrawal amount is 10 ether.
contract L2TxFeeVault is FeeVault {
    /// @param _owner The owner of the contract.
    /// @param _recipient The fee recipient address on L1.
    constructor(address _owner, address _recipient) FeeVault(_owner, _recipient, 10 ether) {}
}
