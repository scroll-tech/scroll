// SPDX-License-Identifier: GPL-3.0

pragma solidity =0.8.24;

contract Hash {
    function sha256(bytes memory input) public view returns (bytes memory out) {
        (bool ok, bytes memory out) = address(2).staticcall(input);
        require(ok);
    }

    function sha256Yul(bytes memory input) public view returns (bytes memory out) {
        assembly {
            // mstore(0, input)
            if iszero(staticcall(gas(), 2, 0, 32, 0, 32)) {
                revert(0, 0)
            }
            // return(0, 32)
        }
    }

    function sha256s(uint256 n) public {
        bytes memory input = abi.encode(999);
        for (uint256 i = 0; i < n; i++) {
            sha256(input);
        }
    }

    function keccak256s(uint256 n) public {
        bytes32[] memory output = new bytes32[](n);
        for (uint256 i = 0; i < n; i++) {
            bytes memory input = abi.encode(i);
            output[i] = keccak256(input);
        }
    }
}
