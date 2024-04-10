// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

contract MockCaller {
    function callTarget(address to, bytes calldata data) external payable {
        // solhint-disable-next-line avoid-low-level-calls
        (bool success, ) = address(to).call{value: msg.value}(data);
        if (!success) {
            // solhint-disable-next-line no-inline-assembly
            assembly {
                let ptr := mload(0x40)
                let size := returndatasize()
                returndatacopy(ptr, 0, size)
                revert(ptr, size)
            }
        }
    }
}
