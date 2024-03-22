// SPDX-License-Identifier: MIT

pragma solidity =0.8.24;

import {ERC721TokenReceiver} from "solmate/tokens/ERC721.sol";

contract MockERC721Recipient is ERC721TokenReceiver {
    address private target;
    uint256 private value;
    bytes private data;

    function setCall(
        address _target,
        uint256 _value,
        bytes calldata _data
    ) external payable {
        target = _target;
        value = _value;
        data = _data;
    }

    function onERC721Received(
        address,
        address,
        uint256,
        bytes calldata
    ) external override returns (bytes4) {
        // solhint-disable-next-line avoid-low-level-calls
        (bool success, ) = target.call{value: value}(data);
        if (!success) {
            // solhint-disable-next-line no-inline-assembly
            assembly {
                let ptr := mload(0x40)
                let size := returndatasize()
                returndatacopy(ptr, 0, size)
                revert(ptr, size)
            }
        }
        return ERC721TokenReceiver.onERC721Received.selector;
    }
}
