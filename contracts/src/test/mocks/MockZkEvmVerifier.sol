// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {IZkEvmVerifier} from "../../libraries/verifier/IZkEvmVerifier.sol";

contract MockZkEvmVerifier is IZkEvmVerifier {
    event Called(address);

    /// @inheritdoc IZkEvmVerifier
    function verify(bytes calldata, bytes32) external view {
        revert(string(abi.encode(address(this))));
    }
}
